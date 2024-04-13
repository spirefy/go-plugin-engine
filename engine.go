package pluginengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	extism "github.com/extism/go-sdk"
	"github.com/spirefy/go-pdk/types"
	gopdk "github.com/spirefy/go-pdk/types"
	"github.com/tetratelabs/wazero"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type (
	ExtensionPoint struct {
		types.ExtensionPoint
		// Because this outer ExtensionPoint wrapper allows for host extension points, which are native to Go, a func pointer
		// to call upon that extension point is necessary. This is not the typical wasm string func name to call, but an
		// actual Go function provided by the host to be called
		Func       func([]*Extension) error
		Extensions []*Extension
		Plugin     Plugin
	}

	Extension struct {
		types.Extension
		Plugin   Plugin
		Resolved bool `json:"resolved" yaml:"resolved"`
	}

	Plugin struct {
		Details  types.Plugin
		Plugin   *extism.Plugin
		Resolved bool
	}

	Engine struct {
		plugins         map[string]map[string]*Plugin
		extensionPoints map[string][]*ExtensionPoint
		extensions      map[string]*Extension
		unresolved      []*Extension
		hostFuncs       []extism.HostFunction
	}
)

func contains(arr []string, str string) bool {
	for _, s := range arr {
		if s == str {
			return true
		}
	}
	return false
}

func findFilesWithExtensions(root string, extensions []string) ([]string, error) {
	var matchingFiles []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			ext := filepath.Ext(path)
			if contains(extensions, ext) {
				// fileName := filepath.Base(path)
				matchingFiles = append(matchingFiles, path)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return matchingFiles, nil
}

// addPlugin
//
// This method will add the plugin passed in to the engine's plugins property. It will ensure that if a plugin
// at the name and version provided does not yet exist, the map of internalPlugin objects is created.
// It's important to note that if a plugin already exists at the name and version intersection, it is replaced. This
// should allow for reloading (and eventual GC of old plugins as they are replaced) if need be.
func (e *Engine) addPlugin(p *Plugin) {
	if nil != e.plugins && nil != p {
		pv := e.plugins[p.Details.Name]

		if nil == pv {
			pv = make(map[string]*Plugin, 0)
			e.plugins[p.Details.Name] = pv
		}

		pv[p.Details.Version] = p

		// now add all of this plugins extensions to the unresolved list... a call to engine.resolve() will then try to
		// find/resolve all extensions and subsequently resolve all plugins
		if nil != p.Details.Extensions && len(p.Details.Extensions) > 0 {
			for _, ex := range p.Details.Extensions {
				ee := &Extension{
					Extension: ex,
					Plugin:    *p,
					Resolved:  false,
				}

				e.unresolved = append(e.unresolved, ee)
			}
		}

		// now add all the plugins extension points to the engines extension points using the ExtensionPoint object
		// that will tie this plugin instance to it as well.
		if nil != p.Details.ExtensionPoints && len(p.Details.ExtensionPoints) > 0 {
			for _, ep := range p.Details.ExtensionPoints {
				eep := &ExtensionPoint{
					ExtensionPoint: ep,
					Func:           nil,
					Extensions:     nil,
					Plugin:         *p,
				}

				eps := e.extensionPoints[ep.Id]
				if nil == eps {
					eps = make([]*ExtensionPoint, 0)
				}

				eps = append(eps, eep)
				// reassign because exps may be a new larger ref.. has to be reassigned
				e.extensionPoints[ep.Id] = eps
			}
		}
	}
}

// validateVersion
//
// TODO: For now this just returns true. It needs to add a check to make sure a version string matches a semver version
// value
func isSemverValid(version string) bool {
	// Split the version string into major, minor, and patch components
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}

	// Check if each component is a non-negative integer
	for _, part := range parts {
		if !isValidNumber(part) {
			return false
		}
	}
	return true
}

// isValidNumber
// helper func used by isSemverValid
func isValidNumber(str string) bool {
	if len(str) == 0 || str[0] == '-' {
		return false
	}

	for _, c := range str {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

// GetExtensionsForExtensionPoint
//
// This method will look for a matching endpoint in the map of endpoints and if found and the version provided is not
// nil, look for a matching version (TODO: version range may be added in future). If version is nil, the first
// extension point's extensions are returned.
func (e *Engine) GetExtensionsForExtensionPoint(epoint string, versions []string) ([]*gopdk.Extension, error) {
	eps := e.extensionPoints[epoint]
	var lowerVersion, upperVersion string

	if len(versions) > 0 {
		lowerVersion = versions[0]
		if len(versions) > 1 {
			upperVersion = versions[1]
		}

		if !isSemverValid(lowerVersion) {
			return nil, errors.New("version or lower bound version is not a valid SemVer: " + lowerVersion)
		}

		if len(upperVersion) > 0 && !isSemverValid(upperVersion) {
			return nil, errors.New("version or upper bound version is not a valid SemVer: " + upperVersion)
		}
	}

	if nil != eps && len(eps) > 0 {
		if len(lowerVersion) > 0 {
			for _, epVer := range eps {
				// if only one version we need an exact match
				if len(upperVersion) <= 0 {
					if epVer.Version == lowerVersion {
						exts := make([]*gopdk.Extension, 0)
						for _, epex := range epVer.Extensions {
							exts = append(exts, &epex.Extension)
						}
						return exts, nil
					}
				} else {
					// we have a range of versions so the epVer.Version needs to be >= lowerVersion and <= upperVersion
				}
			}
		} else {
			exts := make([]*gopdk.Extension, 0)
			for _, epex := range eps[0].Extensions {
				exts = append(exts, &epex.Extension)
			}
			return exts, nil
		}
	}

	return nil, errors.New("no extensions found for extension point")
}

// Load
func (e *Engine) Load(path string) error {
	// First make sure that path is NOT a URL to a single plugin file
	lower := strings.ToLower(path)
	if strings.HasPrefix(lower, "http") {
		// This is a URL
		u, err := url.Parse(lower)
		// todo: load URL path
		fmt.Println("err,u: ", u, err)
	}

	base, er := os.Getwd()
	if er != nil {
		return er
	}

	newpath := filepath.Join(base, lower)

	// Hardcode WASM extension as it's the only plugin module format supported.
	files, err := findFilesWithExtensions(newpath, []string{".wasm"})

	if err != nil {
		// Handle error
		return err
	}

	ctx := context.Background()
	cache := wazero.NewCompilationCache()
	defer func(cache wazero.CompilationCache, ctx context.Context) {
		err := cache.Close(ctx)
		if err != nil {

		}
	}(cache, ctx)

	config := extism.PluginConfig{
		EnableWasi:    true,
		ModuleConfig:  wazero.NewModuleConfig(),
		RuntimeConfig: wazero.NewRuntimeConfig().WithCompilationCache(cache),
		LogLevel:      extism.LogLevelDebug,
	}

	// this is defined here because when a plugin module is loaded the extism.Plugin instance is assigned during the
	// host func registerPlugin call.. but as that is a call back, the definition has no instance of the actual
	// extism.Plugin to use. The extism.CurrentPlugin that IS available in the host/callback func has a private
	// extism.Plugin property that can not be accessed in the callback registerPlugin func. So we need to keep the
	// instance of it in this variable, defined outside of the host func declarations because the registerPlugin call
	// SHOULD happen inside the plugin's start() exported function. If it does not happen there, the plugin wont
	// ever be added/registered. So it is 100% required for the plugin's exported start() func to call the host
	// function regstierPlugin() function. The Go PDK (and other language PDKs) should provide an idiomatic wrapper
	// around the extism/wasm requirements for the registerPlugin func.
	//
	// NOTE: This is done in synchronous.. not async so should not have any concerns with this "global" variable being
	// used by more than one thread and assigning the incorrect calling plugin instance.
	var callingPlugin *extism.Plugin

	getExtensionsForExtensionPoint := extism.NewHostFunctionWithStack(
		"getExtensionsForExtensionPoint",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			ep, err2 := p.ReadString(stack[0])

			fmt.Println("WE ARE CALLING GET EXTENSION FOR EXTENSION POINT.. EP IS ", ep)
			if nil != err2 {
				// TODO: Figure out how to handle this correctly
				fmt.Println("ERROR CALLING FROM PLUGIN TO HOST getExtensionsForExtensionPoint FUNCTION: ", err2)
			}

			ver, err2 := p.ReadString(stack[1])
			if nil != err2 {
				// TODO: Figure out how to handle this correctly
				fmt.Println("ERROR CALLING FROM PLUGIN TO HOST getExtensionsForExtensionPoint FUNCTION: ", err2)
			}

			fmt.Println("VERSION FOR EP IS ", ver)
			exts, err := e.GetExtensionsForExtensionPoint(ep, nil)
			if nil != err {
				fmt.Println("ERROR IN HOST FUNC: ", err)
			}
			if nil != exts && len(exts) > 0 {
				for _, ext := range exts {
					fmt.Println("We're seeing if ext makes sense: ", ext.Id, ext.Name, ext.Func)
				}
			}
		},
		[]extism.ValueType{extism.ValueTypeI64, extism.ValueTypeI64}, []extism.ValueType{extism.ValueTypeI64},
	)
	getExtensionsForExtensionPoint.SetNamespace("extism:host/user")

	callExtension := extism.NewHostFunctionWithStack(
		"callExtension",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			_, err2 := p.ReadBytes(stack[0])
			if nil != err2 {
				// TODO: Figure out how to handle this correctly
				fmt.Println("ERROR CALLING FROM PLUGIN TO HOST callExtension FUNCTION: ", err2)
			}
		},
		[]extism.ValueType{extism.ValueTypeI64}, []extism.ValueType{extism.ValueTypeI64},
	)
	callExtension.SetNamespace("extism:host/user")

	registerPlugin := extism.NewHostFunctionWithStack("registerPlugin",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			data, err := p.ReadBytes(stack[0])
			if nil != err {
				// TODO: Figure out how to handle this correctly
				fmt.Println("ERROR CALLING FROM PLUGIN TO HOST registerPlugin FUNCTION: ", err)
			}

			plugin := &types.Plugin{}
			err = json.Unmarshal(data, plugin)
			if nil != err {
				fmt.Println("A PLUGIN IS BEING REGISTERED VIA CALLBACK HOST FUNC: ", plugin.Id, plugin.Name)
			} else {
				newPlugin := &Plugin{
					Details:  *plugin,
					Plugin:   callingPlugin,
					Resolved: false,
				}

				e.addPlugin(newPlugin)
			}
		},

		[]extism.ValueType{extism.ValueTypeI64}, []extism.ValueType{extism.ValueTypeI64},
	)
	registerPlugin.SetNamespace("extism:host/user")

	// add host functions
	hostFuncs := []extism.HostFunction{getExtensionsForExtensionPoint, callExtension, registerPlugin}

	for _, file := range files {
		fmt.Println(file)
		manifest := extism.Manifest{
			Wasm: []extism.Wasm{
				extism.WasmFile{
					Path: file,
				},
			},
		}

		callingPlugin, err = extism.NewPlugin(ctx, manifest, config, hostFuncs)

		if err != nil {
			fmt.Printf("Failed to initialize plugin: %v\n", err)
			continue
		}

		_, _, err = callingPlugin.Call("start", nil)

		if nil != err {
			fmt.Println("Error calling plugin: ", err)
			continue
		} else {
			fmt.Println("WE CALLED IT")
		}
	}

	e.resolve()
	return nil
}

// resolve
//
// This method will loop through all plugins and unresolved extensions, attempting to ensure all extensions of a given
// plugin have been resolved to loaded plugins with matching extension points. Only when all extensions of a plugin
// are resolved will a plugin's status change to resolved.
func (e *Engine) resolve() {
	if nil != e.unresolved && len(e.unresolved) > 0 {
		leftover := make([]*Extension, 0)
		for _, v := range e.unresolved {
			// mkae sure the status is unresolved
			if !v.Resolved {
				// find extension point this extension anchors to
				eps := e.extensionPoints[v.ExtensionPoint]
				if nil != eps && len(eps) > 0 {
					for _, ep := range eps {
						fmt.Println("Checking if ep resolves: ", v.ExtensionPoint, ep.Id)
						if v.ExtensionPoint == ep.Id {
							fmt.Println("We found a match: ", ep.Name)
							ep.Extensions = append(ep.Extensions, v)
							v.Resolved = true
							e.extensions[v.Name] = v
						} else {
							// not found, append to leftover
							leftover = append(leftover, v)
						}
					}
				} else {
					fmt.Println("Looks like extension point not yet loaded: ", v.ExtensionPoint)
				}
			}
		}

		// set the leftover unresolved
		e.unresolved = leftover
	}
}

// RegisterHostExtensionPoint
//
// This method allows a host/client application that is using the Plugin Engine to register extension points. This is
// useful if the host/client app has some specific things it wants to allow anchor points for plugins to attach to.
// Ideally a host/client app may ship/install/start with plugins already, but this gives the ability for the host/client
// to have native code functions tied to extension points that are then filled by plugin extensions.
func (e *Engine) RegisterHostExtensionPoint(id, name, version, description string) {
	ep := &ExtensionPoint{
		ExtensionPoint: types.ExtensionPoint{
			Id:          id,
			Description: description,
			Name:        name,
			Version:     version,
		},
	}

	exps := e.extensionPoints[id]
	if nil == exps {
		exps = make([]*ExtensionPoint, 0)
	}

	exps = append(exps, ep)
	// reassign because exps may be a new larger ref.. has to be reassigned
	e.extensionPoints[id] = exps
}

func (e *Engine) GetPlugins() map[string]map[string]*Plugin {
	return e.plugins
}

func (e *Engine) CallExtensionFunc(ex gopdk.Extension, data []byte) ([]byte, error) {
	for _, ext := range e.extensions {
		if ext.Name == ex.Name {

			p := ext.Plugin

			if nil == p.Plugin {
				fmt.Println("UH OH the plugin instance is nil")
			}

			_, data, err := p.Plugin.Call(ext.Func, data)
			if nil != err {
				fmt.Println("Error calling extension func: ", ex.Name, err)
				return nil, err
			}

			return data, nil
		}
	}

	return nil, nil
}

// NewPluginEngine
//
// This function will create a new plugin engine instance. Passed in are host functions per the Extism (WASI)
// Host Function spec. This allows consumers of this engine to provide its own host functions that plugins will be
// able to utilize along with the plugin engine host functions.
func NewPluginEngine(hostFuncs []extism.HostFunction) *Engine {
	plugins := make(map[string]map[string]*Plugin)
	unresolved := make([]*Extension, 0)
	extensionPoints := make(map[string][]*ExtensionPoint)
	extensions := make(map[string]*Extension)

	// instantiate as we need this in the host functions
	engine := &Engine{
		plugins:         plugins,
		unresolved:      unresolved,
		hostFuncs:       hostFuncs,
		extensions:      extensions,
		extensionPoints: extensionPoints,
	}

	return engine
}

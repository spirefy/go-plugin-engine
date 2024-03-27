package pluginengine

import (
	"context"
	"encoding/json"
	"fmt"
	extism "github.com/extism/go-sdk"
	"github.com/spirefy/go-pdk/types"
	gopdk "github.com/spirefy/go-pdk/types"
	"github.com/tetratelabs/wazero"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	fmt.Print("ADDING PLUGIN")
	if nil != e.plugins && nil != p {
		fmt.Println(": ", p.Details.Name)
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

// GetExtensionsForExtensionPoint
//
// This method will look for a matching endpoint in the map of endpoints and if found and the version provided is not
// nil, look for a matching version (TODO: version range may be added in future). If version is nil, the first
// extension point's extensions are returned.
func (e *Engine) GetExtensionsForExtensionPoint(epoint string, version *string) []*gopdk.Extension {
	eps := e.extensionPoints[epoint]
	if nil != eps && len(eps) > 0 {
		if nil != version && len(*version) > 0 {
			for _, epVer := range eps {
				if epVer.Version == *version {
					exts := make([]*gopdk.Extension, 0)
					for _, epex := range epVer.Extensions {
						exts = append(exts, &epex.Extension)
					}
					return exts
				}
			}
		} else {
			exts := make([]*gopdk.Extension, 0)
			for _, epex := range eps[0].Extensions {
				exts = append(exts, &epex.Extension)
			}
			return exts
		}
	}

	return nil
}

// Load
func (e *Engine) Load(path string) error {
	fmt.Println("LOADING...")
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

	sendEvent := extism.NewHostFunctionWithStack(
		"sendEvent",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			event, err := p.ReadString(stack[0])
			if nil != err {
				// TODO: Figure out how to handle this correctly
				fmt.Println("ERROR CALLING FROM PLUGIN TO HOST sendEvent FUNCTION: ", err)
			}

			data, err2 := p.ReadBytes(stack[1])
			if nil != err2 {
				// TODO: Figure out how to handle this correctly
				fmt.Println("ERROR CALLING FROM PLUGIN TO HOST sendEvent FUNCTION: ", err2)
			}

			if nil != err {
				fmt.Println("ERROR SENDING EVENT: ", event, err)
			}

			fmt.Println("Data: ", data)
		},
		[]extism.ValueType{extism.ValueTypeI64, extism.ValueTypeI64}, []extism.ValueType{extism.ValueTypeI64},
	)
	sendEvent.SetNamespace("extism:host/user")

	callExtension := extism.NewHostFunctionWithStack(
		"callExtension",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			data, err2 := p.ReadBytes(stack[0])
			if nil != err2 {
				// TODO: Figure out how to handle this correctly
				fmt.Println("ERROR CALLING FROM PLUGIN TO HOST callExtension FUNCTION: ", err2)
			}

			if nil != err {
				fmt.Println("ERROR SENDING EVENT: ", data, err)
			}

			fmt.Println("Data: ", data)
		},
		[]extism.ValueType{extism.ValueTypeI64}, []extism.ValueType{extism.ValueTypeI64},
	)
	callExtension.SetNamespace("extism:host/user")

	registerPlugin := extism.NewHostFunctionWithStack("registerPlugin",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			fmt.Println("REGISTER HOST FUNC CALLED")
			data, err := p.ReadBytes(stack[0])
			if nil != err {
				// TODO: Figure out how to handle this correctly
				fmt.Println("ERROR CALLING FROM PLUGIN TO HOST registerPlugin FUNCTION: ", err)
			}

			plugin := types.Plugin{}
			err = json.Unmarshal(data, &plugin)
			if nil != err {
				fmt.Println("A PLUGIN IS BEING REGISTERED VIA CALLBACK HOST FUNC: ", plugin.Id, plugin.Name)
			}
			fmt.Println("Plugin: ", plugin.Id, plugin.Name, plugin.Version, plugin.MinVersion, plugin.Description)
			for _, ep := range plugin.ExtensionPoints {
				fmt.Println("EP: ", ep.Id, ep.Name)
			}
		},

		[]extism.ValueType{extism.ValueTypeI64}, []extism.ValueType{extism.ValueTypeI64},
	)
	registerPlugin.SetNamespace("extism:host/user")

	// add host functions
	hostFuncs := []extism.HostFunction{sendEvent, callExtension, registerPlugin}

	for _, file := range files {
		manifest := extism.Manifest{
			Wasm: []extism.Wasm{
				extism.WasmFile{
					Path: file,
				},
			},
		}

		plug, err := extism.NewPlugin(ctx, manifest, config, hostFuncs)

		if err != nil {
			fmt.Printf("Failed to initialize plugin: %v\n", err)
			continue
		}

		_, _, err = plug.Call("start", nil)

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

/**

THIS CODE IS TO BE PUT IN REGISTER PLUGIN TO ADD PLUGIN TO UNRESOLVED
if nil != data && len(data) > 0 {
	fmt.Println(string(data))
	p := &types.Plugin{}
	er := json.Unmarshal(data, p)

	if nil != er {
		fmt.Println("Error unmarshalling plugin data: ", er)
		continue
	} else {
		ip := &Plugin{
			Details:  *p,
			Plugin:   plug,
			Resolved: false,
		}

		e.addPlugin(ip)
	}
}
*/

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
			fmt.Println("We got the FULL extension with plugin")
			p := ext.Plugin
			fmt.Println("About to call func: ", ext.Func)
			s, data, err := p.Plugin.Call(ext.Func, data)
			if nil != err {
				fmt.Println("Error calling extension func: ", ex.Name, err)
				return nil, err
			}

			fmt.Println("Status: ", s)
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

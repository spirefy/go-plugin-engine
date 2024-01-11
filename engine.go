package pluginengine

import (
	"context"
	"encoding/json"
	"fmt"
	extism "github.com/extism/go-sdk"
	"github.com/spirefy/go-plugin-engine/types"
	"github.com/tetratelabs/wazero"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type ExtensionPoint struct {
	types.ExtensionPoint
	// Because this outer ExtensionPoint wrapper allows for host extension points, which are native to Go, a func pointer
	// to call upon that extension point is necessary. This is not the typical wasm string func name to call, but an
	// actual Go function provided by the host to be called
	Func       func([]*Extension) error
	Extensions []*Extension
	Plugin     *extism.Plugin
}

type Extension struct {
	types.Extension
	Plugin   *extism.Plugin
	Resolved bool `json:"resolved" yaml:"resolved"`
}

type Plugin struct {
	Details  types.Plugin
	Plugin   *extism.Plugin
	Resolved bool
}

type Engine struct {
	plugins         map[string]map[string]*Plugin
	extensionPoints []*ExtensionPoint
	unresolved      map[string][]*Extension
	hostFuncs       []extism.HostFunction
}

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

		// now add all of this plugins extensions to the unresolved list.. a call to engine.resolve() will then try to
		// find/resolve all extensions and subsequently resolve all plugins
		if nil != p.Details.Extensions && len(p.Details.Extensions) > 0 {
			exs := make([]*Extension, 0)
			for _, ex := range p.Details.Extensions {
				ee := &Extension{
					Extension: ex,
					Plugin:    p.Plugin,
					Resolved:  false,
				}
				exs = append(exs, ee)
			}

			e.unresolved[p.Details.Id] = exs
		}

		// now add all the plugins extension points to the engines extension points using the ExtensionPoint object
		// that will tie this plugin instance to it as well.
		if nil != p.Details.ExtensionPoints && len(p.Details.ExtensionPoints) > 0 {
			for _, ep := range p.Details.ExtensionPoints {
				eep := &ExtensionPoint{
					ExtensionPoint: ep,
					Func:           nil,
					Extensions:     nil,
					Plugin:         p.Plugin,
				}

				e.extensionPoints = append(e.extensionPoints, eep)
			}
		}
	}
}

type mytyp struct {
	Name  string
	Age   int64
	Alive bool
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

	for _, file := range files {
		manifest := extism.Manifest{
			Wasm: []extism.Wasm{
				extism.WasmFile{
					Path: file,
				},
			},
		}

		sendEvent := extism.NewHostFunctionWithStack(
			"sendevent",
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

				m := mytyp{}
				json.Unmarshal(data, &m)
				fmt.Println("event: ", event)
				if nil != data {
					fmt.Println("DATA: ", m)
				}
				/*
					 err = SendEvent(event, data)

					 if nil != err {
						fmt.Println("ERROR SENDING EVENT: ", event, err)
					}

				*/
			},
			[]extism.ValueType{extism.ValueTypeI64, extism.ValueTypeI64}, []extism.ValueType{extism.ValueTypeI64},
		)
		sendEvent.SetNamespace("extism:host/user")

		// add host functions
		hostFuncs := []extism.HostFunction{sendEvent}

		plug, err := extism.NewPlugin(ctx, manifest, config, hostFuncs)

		if err != nil {
			fmt.Printf("Failed to initialize plugin: %v\n", err)
			continue
		}

		_, data, err := plug.Call("register", nil)

		if nil != err {
			fmt.Println("Error calling plugin: ", err)
			continue
		}

		if nil != data && len(data) > 0 {
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
		for _, v := range e.unresolved {
			for _, ex := range v {
				for _, ep := range e.extensionPoints {
					if ep.Id == ex.ExtensionPoint {
						// add the extension to the EPs extensions list and set to resolved
						ep.Extensions = append(ep.Extensions, ex)
					}
				}
			}
		}
	}
}

// RegisterHostExtensionPoint
//
// This method allows a host/client application that is using the Plugin Engine to register extension points. This is
// useful if the host/client app has some specific things it wants to allow anchor points for plugins to attach to.
// Ideally a host/client app may ship/install/start with plugins already, but this gives the ability for the host/client
// to have native code functions tied to extension points that are then filled by plugin extensions.
func (e *Engine) RegisterHostExtensionPoint(id, name, description string, f func([]*Extension) error) {
	ep := &ExtensionPoint{
		ExtensionPoint: types.ExtensionPoint{
			Id:          id,
			Description: description,
			Name:        name,
		},
		Func: f,
	}

	e.extensionPoints = append(e.extensionPoints, ep)
}

// Start
// This receiver function is called whenever the consuming application is ready to start any plugins that are set to
// startOnLoad true. This will kick off extension point extensions being executed for those with start on load
func (e *Engine) Start() {
	fmt.Println("Plugin Engine starting")

	for _, ep := range e.extensionPoints {
		if nil != ep.Func {
			fmt.Println("Host Func for EP")
			err := ep.Func(ep.Extensions)
			if nil != err {
				fmt.Println("We got an error calling host extension point function: ", err)
			}
		} else if len(ep.FuncName) > 0 {
			fmt.Println("We have an EP func name in the plugin to call: ", ep.FuncName)
		}
	}
}

func (e *Engine) GetPlugins() map[string]map[string]*Plugin {
	return e.plugins
}

func (e *Engine) SendEvent(event string, data []byte) error {
	fmt.Println("Sending data to event: ", event)
	if nil != data && len(data) > 0 {
		fmt.Println("Data size: ", len(data))
	}

	return nil
}

func (e *Engine) CallExtensionFunc(ex types.Extension, data []byte) error {
	return nil
}

// NewPluginEngine
//
// This function will create a new plugin engine instance. Passed in are host functions per the Extism (WASI)
// Host Function spec. This allows consumers of this engine to provide its own host functions that plugins will be
// able to utilize along with the plugin engine host functions.
func NewPluginEngine(hostFuncs []extism.HostFunction) *Engine {
	plugins := make(map[string]map[string]*Plugin, 0)
	unresolved := make(map[string][]*Extension, 0)

	// instantiate as we need this in the host functions
	engine := &Engine{
		plugins:    plugins,
		unresolved: unresolved,
		hostFuncs:  hostFuncs,
	}

	sendEvent := extism.NewHostFunctionWithStack(
		"sendevent",
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

			err = engine.SendEvent(event, data)

			if nil != err {
				fmt.Println("ERROR SENDING EVENT: ", event, err)
			}
		},
		[]extism.ValueType{extism.ValueTypeI64, extism.ValueTypeI64}, []extism.ValueType{extism.ValueTypeI64},
	)
	sendEvent.SetNamespace("host/user")

	// add host functions
	engine.hostFuncs = append([]extism.HostFunction{sendEvent})

	/**
	kvStore := make(map[string][]byte)

	kvRead := extism.NewHostFunctionWithStack(
		"kv_read",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			key, err := p.ReadString(stack[0])
			if err != nil {
				panic(err)
			}

			value, success := kvStore[key]
			if !success {
				value = []byte{0, 0, 0, 0}
			}

			stack[0], err = p.WriteBytes(value)
		},
		[]ValueType{ValueTypePTR},
		[]ValueType{ValueTypePTR},
	)

	kvWrite := extism.NewHostFunctionWithStack(
		"kv_write",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			key, err := p.ReadString(stack[0])
			if err != nil {
				panic(err)
			}

			value, err := p.ReadBytes(stack[1])
			if err != nil {
				panic(err)
			}

			kvStore[key] = value
		},
		[]ValueType{ValueTypePTR, ValueTypePTR},
		[]ValueType{},
	)

	*/

	return engine
}

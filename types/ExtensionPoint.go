package types

type ExtensionPoint struct {
	// a unique identifier such as a name made up of the company, org, project and category separated by periods, or just
	// a simple string. The point is that this is to be matched by Extension's ExtensionPoint property in order for
	// extension to resolve to extension points when loaded.
	Id string `json:"id"`

	// a meaningful description of this extension point that could be displayed in a plugin store for example. This
	// should probably provide details as to how the extension point will be called, when and what expectations if
	// any should be performed or provided by extensions
	Description string `json:"description"`

	// a display or friendly name for this extension point, not to be confused with the Id.
	Name string `json:"name"`

	// the name of a WASI/WASM exported function within the plugin to call after all plugins are resolved. This would
	// be called after all plugin register() exported functions are called for all plugins. This gives the chance for
	// each extension point to iterate through all resolved extensions and call those extensions if that is the way
	// the extension point works.
	FuncName string `json:"funcName"`

	// if true, after all plugins are loaded and all extensions resolved, this extension point (and any others set to
	// true) have their FuncName functions called. This allows ExtensionPoints to be executed right away rather than
	// dynamically via some extension code that calls them.
	StartOnLoad bool `json:"startOnLoad"`
}

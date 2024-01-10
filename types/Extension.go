package types

type Extension struct {
	// the extension point Id that this extension provides functionality to/for.
	ExtensionPoint string `json:"extensionPoint"`

	// a meaningful description of this extension that could be displayed in a plugin store for example
	Description string `json:"description"`

	// a display or friendly name for this extension
	Name string `json:"name"`

	// a func IN the WASM plugin that matches this value that would be called by the plugin engine when the extension
	// point is executed
	FuncName string `json:"funcName"`
}

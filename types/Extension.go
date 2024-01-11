package types

type Extension struct {
	// the extension point Id that this extension provides functionality to/for.
	ExtensionPoint string `json:"extensionPoint"`

	// a meaningful description of this extension that could be displayed in a plugin store for example
	Description string `json:"description"`

	// a display or friendly name for this extension
	Name string `json:"name"`

	// a func IN the WASM plugin that matches this value that would be called by the extension point using the plugin
	// engine's host function to call extension functions.
	FuncName string `json:"funcName"`

	// This property can hold custom data for an extension point to use without having to call the extension func
	// to have data returned. This is useful when an ExtnesionPoint want's to build up say.. a Menu system or a Help
	// system that is composed of static data and does not require the Extnesion func to execute to return such data.
	// Each ExtensionPoint would define the format (structure) that this metadata would need to be in and it is up
	// to the exported register() function to marshal the data into a []byte before returning the Extension(s) as
	// part of the register return []byte data. ExtensionPoint funcs would be called to process the metadata however
	// necessary.
	MetaData []byte `json:"metadata"`
}

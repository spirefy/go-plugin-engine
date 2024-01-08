package types

type Schema struct {
	Type   string `json:"type" yaml:"type"`
	Schema []byte `json:"schema" yaml:"schema"`
}

type ExtensionPoint struct {
	// a unique identifier such as a name made up of the company, org, project and category separated by periods, or just
	// a simple string. The point is that this is to be matched by Extension's ExtensionPoint property in order for
	// extension to resolve to extension points when loaded.
	Id string `json:"id" yaml:"id"`

	// a meaningful description of this extension point that could be displayed in a plugin store for example. This
	// should probably provide details as to how the extension point will be called, when and what expectations if
	// any should be performed or provided by extensions
	Description string `json:"description" yaml:"description"`

	// a display or friendly name for this extension point, not to be confused with the Id.
	Name string `json:"name" yaml:"name"`

	// ExtensionPoint's could be part of a plugin that ALSO has extensions to other ExtensionPoints... those extensions
	// when called by their extensionpoint might in turns use another extension point.. call it dynamically at runtime.
	// In other cases, it is necessary for some extension points to start after the engine is done loading all plugins.
	// This property indicates it should start on load or wait for dynamic activation.
	StartOnLoad bool `json:"startOnLoad" yaml:"startOnLoad"`

	// Schema is a json schema definition that if provided is the expected payload and response object (as a []byte)
	// that a func in an extension will accept and/or return.
	Schema Schema `json:"schema" yaml:"schema"`
}

type Extension struct {
	// the extension point Id that this extension provides functionality to/for.
	ExtensionPoint string `json:"extensionPoint" yaml:"extensionPoint"`

	// a meaningful description of this extension that could be displayed in a plugin store for example
	Description string `json:"description" yaml:"description"`

	// a display or friendly name for this extension
	Name string `json:"name" yaml:"name"`

	// a func IN the WASM plugin that matches this value that would be called by the plugin engine when the extension
	// point is executed
	FuncName string `json:"func" yaml:"func"`
}

type Plugin struct {
	Id              string           `json:"id" yaml:"id"`
	Name            string           `json:"name" yaml:"name"`
	Version         string           `json:"version" yaml:"versioin"`
	Description     string           `json:"description" yaml:"description"`
	ExtensionPoints []ExtensionPoint `json:"extensionPoints" yaml:"extensionPoints"`
	Extensions      []Extension      `json:"extensions" yaml:"extensions"`
}

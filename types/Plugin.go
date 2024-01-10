package types

type Plugin struct {
	Id              string           `json:"id"`
	Name            string           `json:"name"`
	Version         string           `json:"version"`
	Description     string           `json:"description"`
	ExtensionPoints []ExtensionPoint `json:"extensionPoints"`
	Extensions      []Extension      `json:"extensions"`
	Listeners       []Listener       `json:"listeners"`
}

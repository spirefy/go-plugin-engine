package types

type Event struct {
	Id     string `json:"id"`
	Data   []byte `json:"data"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type Listener struct {
	Event string `json:"event"`
	Func  string `json:"func"`
}

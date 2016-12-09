package debug

type Dispatcher struct {
	Name       string
	ID         string
	Procedures []Procedure
	Inbounds   []Inbound
	Outbounds  []Outbound
}

type Procedure struct {
	Service   string
	Name      string
	Flavor    string
	Encoding  string
	Signature string
}

type Inbound struct {
	Transport string
	Endpoint  string
	State     string
}

type Outbound struct {
	Name      string
	Flavor    string
	Transport string
	Endpoint  string
	State     string
	Peers     []Peer
}

type Peer struct {
	Identifier string
	Status     string
}

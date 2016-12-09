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
	Peer      string
	State     string
}

type Outbound struct {
	Name      string
	Transport string
	Endpoint  string
	Peer      string
	State     string
}

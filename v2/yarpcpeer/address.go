package yarpcpeer

// Address is a peer identified by its host:port string.
//
// Address is the least elaborate peer identifier implementation.
type Address string

// Identifier returns the peer identifier as a string.
func (a Address) Identifier() string {
	return string(a)
}

// Addresses lifts a list of string addresses to Address peer identifiers.
func Addresses(addrs []string) []Identifier {
	ids := make([]Identifier, 0, len(addrs))
	for _, addr := range addrs {
		ids = append(ids, Address(addr))
	}
	return ids
}

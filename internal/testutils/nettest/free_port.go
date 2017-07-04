package nettest

import "net"

func getClosedTCPAddr() (*net.TCPAddr, error) {
	address, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return nil, err
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr), nil
}

// MustGetFreeHostPort returns a TCP host:port that is free for unit tests
// that cannot use port 0.
func MustGetFreeHostPort() string {
	addr, err := getClosedTCPAddr()
	if err != nil {
		panic(err)
	}
	return addr.String()
}

// MustGetFreePort returns a TCP port that is free for unit tests that cannot
// use port 0.
func MustGetFreePort() uint16 {
	addr, err := getClosedTCPAddr()
	if err != nil {
		panic(err)
	}
	return uint16(addr.Port)
}

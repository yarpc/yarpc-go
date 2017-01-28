package hashring

import (
	"fmt"
	"sync"
)

type fakeMember struct {
	address  string
	identity string
}

func (f fakeMember) GetAddress() string {
	return f.address
}

func (f fakeMember) Label(key string) (value string, has bool) {
	return "", false
}

func (f fakeMember) Identity() string {
	if f.identity != "" {
		return f.identity
	}
	return f.address
}

func genMembers(host, fromPort, toPort int, overrideIdentity bool) (members []Member) {
	for i := fromPort; i <= toPort; i++ {
		member := fakeMember{
			address: fmt.Sprintf("127.0.0.%v:%v", host, 3000+i),
		}
		if overrideIdentity {
			member.identity = fmt.Sprintf("identity%v", i)
		}
		members = append(members, member)
	}
	return members
}

// fake event listener
type dummyEvents struct {
	l      sync.Mutex
	events int
}

func (d *dummyEvents) EventCount() int {
	d.l.Lock()
	events := d.events
	d.l.Unlock()
	return events
}

func (d *dummyEvents) inc() {
	d.l.Lock()
	d.events++
	d.l.Unlock()
}

func (d *dummyEvents) RingChanged(RingChange) {
	d.inc()
}

func (d *dummyEvents) RingChecksum(RingChecksum) {
	d.inc()
}

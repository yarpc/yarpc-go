package hashring

type RingChecksum struct {
	OldChecksum  uint32
	NewChecksum  uint32
	OldChecksums map[string]uint32
	NewChecksums map[string]uint32
}

type RingChange struct {
	ServersAdded   []string
	ServersRemoved []string
	ServersUpdated []string
}

type EventHandler interface {
	RingChanged(RingChange)
	RingChecksum(RingChecksum)
}

type noEventHandler struct{}

func (noEventHandler) RingChecksum(RingChecksum) {
}

func (noEventHandler) RingChanged(RingChange) {
}

var NoEventHandler = noEventHandler{}

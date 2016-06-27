package yarpc

import (
	"time"

	"github.com/yarpc/yarpc-go"
)

// SleepRaw responds to raw requests over any transport by sleeping for one
// second.
func SleepRaw(reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
	time.Sleep(1 * time.Second)
	return nil, nil, nil
}

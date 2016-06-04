package yarpc

import (
	"time"

	"github.com/yarpc/yarpc-go/encoding/raw"
)

// SleepRaw responds to raw requests over any transport by sleeping for one
// second.
func SleepRaw(reqMeta *raw.ReqMeta, body []byte) ([]byte, *raw.ResMeta, error) {
	time.Sleep(1 * time.Second)
	return nil, nil, nil
}

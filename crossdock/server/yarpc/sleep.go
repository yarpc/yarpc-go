package yarpc

import (
	"time"

	"github.com/yarpc/yarpc-go/encoding/raw"
)

func SleepRaw(reqMeta *raw.ReqMeta, body []byte) ([]byte, *raw.ResMeta, error) {
	time.Sleep(1 * time.Second)
	return nil, nil, nil
}

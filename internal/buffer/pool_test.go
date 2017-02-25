package buffer

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuffers(t *testing.T) {
	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 100; i++ {
				buf := Get()
				assert.Zero(t, buf.Len(), "Expected truncated buffer")

				b := getRandBytes()
				_, err := buf.Write(b)
				assert.NoError(t, err, "Unexpected error from buffer.Write")

				assert.Equal(t, buf.Len(), len(b), "Expected same buffer size")

				Put(buf)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func getRandBytes() []byte {
	b := make([]byte, rand.Intn(5000))
	rand.Read(b)
	return b
}

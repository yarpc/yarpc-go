package iopool

import (
	"bytes"
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
				inputBytes := make([]byte, rand.Intn(5000)+20)
				_, err := rand.Read(inputBytes)
				assert.NoError(t, err, "Unexpected error from rand.Read")
				reader := bytes.NewReader(inputBytes)

				outputBytes := make([]byte, 0, len(inputBytes))
				writer := bytes.NewBuffer(outputBytes)

				Copy(writer, reader)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

package joinencodings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinEncodings(t *testing.T) {
	tests := []struct {
		encodings []string
		want      string
	}{
		{
			want: `no encodings`,
		},
		{
			encodings: []string{
				"json",
			},
			want: `"json"`,
		},
		{
			encodings: []string{
				"json",
				"thrift",
			},
			want: `"json" or "thrift"`,
		},
		{
			encodings: []string{
				"json",
				"thrift",
				"proto",
			},
			want: `"json", "thrift", or "proto"`,
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, Join(tt.encodings), "join %+v", tt.encodings)
	}
}

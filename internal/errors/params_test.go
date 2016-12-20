package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMissingParameters(t *testing.T) {
	tests := []struct {
		params []string
		want   string
	}{
		{},
		{
			[]string{"x"},
			"missing x",
		},
		{
			[]string{"x", "y"},
			"missing x and y",
		},
		{
			[]string{"x", "y", "z"},
			"missing x, y, and z",
		},
	}

	for _, tt := range tests {
		err := MissingParameters(tt.params)
		if tt.want != "" {
			if assert.Error(t, err) {
				assert.Equal(t, tt.want, err.Error())
			}
		} else {
			assert.NoError(t, err)
		}

	}
}

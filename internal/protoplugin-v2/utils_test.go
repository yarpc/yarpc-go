package protopluginv2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoCamelCase(t *testing.T) {
	tests := []struct {
		Arg        string
		WantOutput string
	}{
		{"", ""},
		{"one", "One"},
		{"one_two", "OneTwo"},
		{"One_Two", "One_Two"},
		{"my_Name", "My_Name"},
		{"OneTwo", "OneTwo"},
		{"one.two", "OneTwo"},
		{"one.Two", "One_Two"},
		{"one_two.three_four", "OneTwoThreeFour"},
		{"one_two.Three_four", "OneTwo_ThreeFour"},
		{"ONE_TWO", "ONE_TWO"},
		{"one__two", "One_Two"},
		{"camelCase", "CamelCase"},
		{"go2proto", "Go2Proto"},
	}

	for _, test := range tests {
		camelCase := GoCamelCase(test.Arg)
		assert.Equal(t, camelCase, test.WantOutput)
	}
}

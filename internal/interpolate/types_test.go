package interpolate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func mapResolver(m map[string]string) VariableResolver {
	return func(name string) (value string, ok bool) {
		if m == nil {
			return "", false
		}
		value, ok = m[name]
		return
	}
}

func TestRender(t *testing.T) {
	tests := []struct {
		give String
		vars map[string]string

		want    string
		wantErr string
	}{
		{
			give: String{literal("foo "), literal("bar")},
			want: "foo bar",
		},
		{
			give: String{literal("foo"), variable{Name: "bar"}},
			vars: map[string]string{"bar": "baz"},
			want: "foobaz",
		},
	}

	for _, tt := range tests {
		got, err := tt.give.Render(mapResolver(tt.vars))
		if tt.wantErr != "" {
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
			continue
		}

		if assert.NoError(t, err) {
			assert.Equal(t, tt.want, got)
		}
	}
}

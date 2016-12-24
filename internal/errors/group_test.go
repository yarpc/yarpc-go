package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiError(t *testing.T) {
	tests := []struct {
		give        []error
		want        error
		wantMessage string
	}{
		{
			give: []error{},
			want: nil,
		},
		{
			give:        []error{errors.New("great sadness")},
			want:        errors.New("great sadness"),
			wantMessage: "great sadness",
		},
		{
			give: []error{
				errors.New("foo"),
				errors.New("bar"),
			},
			want: ErrorGroup{
				errors.New("foo"),
				errors.New("bar"),
			},
			wantMessage: "the following errors occurred:\n\tfoo\n\tbar",
		},
		{
			give: []error{
				errors.New("great sadness"),
				errors.New("multi\n  line\nerror message"),
				errors.New("single line error message"),
			},
			want: ErrorGroup{
				errors.New("great sadness"),
				errors.New("multi\n  line\nerror message"),
				errors.New("single line error message"),
			},
			wantMessage: "the following errors occurred:\n" +
				"\tgreat sadness\n" +
				"\tmulti\n" +
				"  line\n" +
				"error message\n" +
				"\tsingle line error message",
		},
		{
			give: []error{
				errors.New("foo"),
				ErrorGroup{
					errors.New("bar"),
					errors.New("baz"),
				},
				errors.New("qux"),
			},
			want: ErrorGroup{
				errors.New("foo"),
				errors.New("bar"),
				errors.New("baz"),
				errors.New("qux"),
			},
			wantMessage: "the following errors occurred:\n" +
				"\tfoo\n" +
				"\tbar\n" +
				"\tbaz\n" +
				"\tqux",
		},
	}

	for _, tt := range tests {
		err := MultiError(tt.give)
		if assert.Equal(t, tt.want, err) && tt.wantMessage != "" {
			assert.Equal(t, tt.wantMessage, err.Error())
		}
	}
}

func TestCombineErrors(t *testing.T) {
	tests := []struct {
		give []error
		want error
	}{
		{
			give: []error{
				errors.New("foo"),
				nil,
				ErrorGroup{
					errors.New("bar"),
				},
				nil,
			},
			want: ErrorGroup{
				errors.New("foo"),
				errors.New("bar"),
			},
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, CombineErrors(tt.give...))
	}
}

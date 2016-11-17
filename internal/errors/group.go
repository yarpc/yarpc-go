package errors

import "strings"

// ErrorGroup represents a collection of errors.
type ErrorGroup []error

func (e ErrorGroup) Error() string {
	messages := make([]string, 0, len(e)+1)
	messages = append(messages, "the following errors occurred:")
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "\n\t")
}

package joinencodings

import "fmt"

// Join joins a list of encodings as a English (Chicago Style Manual)
// string for presentation in diagnostic messages.
func Join(encs []string) string {
	switch len(encs) {
	case 0:
		return "no encodings"
	case 1:
		return fmt.Sprintf("%q", encs[0])
	case 2:
		return fmt.Sprintf("%q or %q", encs[0], encs[1])
	default:
		i := 1
		inner := ""
		for ; i < len(encs)-1; i++ {
			inner = fmt.Sprintf("%s, %q", inner, encs[i])
		}
		// first, inner, inner, or last
		return fmt.Sprintf("%q%s, or %q", encs[0], inner, encs[i])
	}
}

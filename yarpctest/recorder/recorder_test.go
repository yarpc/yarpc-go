package recorder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeFilename(t *testing.T) {
	assert.EqualValues(t, sanitizeFilename(`hello`), `hello`)
	assert.EqualValues(t, sanitizeFilename(`h/e\l?l%o*`), `h_e_l_l_o_`)
	assert.EqualValues(t, sanitizeFilename(`:h|e"l<l>o.`), `_h_e_l_l_o.`)
	assert.EqualValues(t, sanitizeFilename(`10€|çí¹`), `10__çí¹`)
	assert.EqualValues(t, sanitizeFilename("hel\x00lo"), `hel_lo`)
}

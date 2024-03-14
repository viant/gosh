package term

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCleanXtermControlChars(t *testing.T) {
	var testCases = []struct {
		orignal string
		cleaned string
	}{
		{
			orignal: "\x1b[31mThis is red text\x1b[0m, \x1b[1;34mthis is blue and bold.\x1b[0m",
			cleaned: "This is red text, this is blue and bold.",
		},
		{
			orignal: "\x1b[31mThis is red text\x1b[0m, and this is normal.",
			cleaned: "This is red text, and this is normal.",
		},
	}
	for _, testCase := range testCases {
		assert.Equal(t, testCase.cleaned, Clean(testCase.orignal))
	}
}

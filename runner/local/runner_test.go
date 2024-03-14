package local

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestService_Run(t *testing.T) {
	runner := New()
	output, code, err := runner.Run("ls /")
	assert.Nil(t, err)
	assert.Equal(t, 0, code)
	assert.Truef(t, len(output) > 0, "output was empty")
	assert.True(t, runner.PID() > 0)
}

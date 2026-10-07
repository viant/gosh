package runner

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipeline_extractStatusCode_WithoutNewlineBeforeMarker(t *testing.T) {
	p := &Pipeline{options: NewOptions(nil)}
	marker := "__gosh_status__:test1:"
	p.setStatusMarker(marker)
	chunk := "hello" + marker + "7\n"
	tail := ""

	code := p.extractStatusCode(chunk, &tail)
	require.NotNil(t, code)
	assert.Equal(t, 7, *code)
	assert.Empty(t, tail)
}

func TestPipeline_extractStatusCode_PartialMarkerAcrossChunks(t *testing.T) {
	p := &Pipeline{options: NewOptions(nil)}
	marker := "__gosh_status__:test2:"
	p.setStatusMarker(marker)
	tail := ""
	assert.Nil(t, p.extractStatusCode("data __gosh_sta", &tail))

	code := p.extractStatusCode("tus__:test2:3\n", &tail)
	require.NotNil(t, code)
	assert.Equal(t, 3, *code)
	assert.Empty(t, tail)
}

func TestPipeline_Read_DetectsCompletionForOutputWithoutTrailingNewline(t *testing.T) {
	p := &Pipeline{
		running: 1,
		options: NewOptions(nil),
		output:  make(chan string, 2),
		error:   make(chan string, 1),
		done:    make(chan bool, 1),
	}
	marker := "__gosh_status__:test3:"
	p.setStatusMarker(marker)
	p.output <- "value"
	p.output <- marker + "0\n"

	out, has, code, err := p.Read(context.Background(), WithTimeout(500))
	require.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, 0, code)
	assert.Equal(t, "value", out)
}

func TestPipeline_nextStatusMarker_ChangesPerCommand(t *testing.T) {
	p := &Pipeline{options: NewOptions(nil)}
	first := p.nextStatusMarker()
	second := p.nextStatusMarker()
	assert.NotEqual(t, first, second)
	assert.Equal(t, second, p.statusMarker())
	assert.Contains(t, first, "__gosh_status__:")
	assert.Contains(t, first, "_")
	assert.Contains(t, second, "__gosh_status__:")
}

func TestPipeline_stripStatusToken(t *testing.T) {
	p := &Pipeline{options: NewOptions(nil)}
	marker := "__gosh_status__:test4:"
	p.setStatusMarker(marker)
	output := "line1\n" + marker + "0\nline2\n"
	assert.Equal(t, "line1\nline2\n", p.stripStatusToken(output))
}

func TestPipelineReadTimeoutDoesNotReportSuccess(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(fmt.Sprintf("partial=%t", partial), func(t *testing.T) {
			p := &Pipeline{options: NewOptions(nil), output: make(chan string, 1), error: make(chan string, 1)}
			p.setStatusMarker("__gosh_status__:timeout:")
			if partial {
				p.output <- "pending stdout"
				p.error <- "pending stderr"
			}
			out, has, code, err := p.Read(context.Background(), WithTimeout(20))
			require.ErrorIs(t, err, ErrTimeout)
			assert.Equal(t, -1, code, "no exit status was received")
			assert.Equal(t, partial, has)
			if partial {
				assert.Contains(t, out, "pending stdout")
				assert.Contains(t, out, "pending stderr")
			}
		})
	}
}

func TestPipelineReadConfiguredTerminatorRemainsSuccessful(t *testing.T) {
	p := &Pipeline{options: NewOptions(nil), output: make(chan string, 1), error: make(chan string, 1)}
	p.output <- "ready>"
	out, has, code, err := p.Read(context.Background(), WithTimeout(20), WithTerminators([]string{"ready>"}))
	require.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "ready>")
}

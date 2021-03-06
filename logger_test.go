// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package zap

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/zap/spywrite"
)

func opts(opts ...Option) []Option {
	return opts
}

func withJSONLogger(t testing.TB, opts []Option, f func(*jsonLogger, func() []string)) {
	sink := newTestBuffer()
	errSink := newTestBuffer()

	allOpts := make([]Option, 0, 3+len(opts))
	allOpts = append(allOpts, All, Output(sink), ErrorOutput(errSink))
	allOpts = append(allOpts, opts...)
	jl := NewJSON(allOpts...)
	jl.StubTime()

	f(jl.(*jsonLogger), func() []string { return strings.Split(sink.String(), "\n") })
	assert.Empty(t, errSink.String(), "Expected error sink to be empty")
}

func assertMessage(t testing.TB, level, expectedMsg, actualLog string) {
	expectedLog := fmt.Sprintf(`{"msg":"%s","level":"%s","ts":0,"fields":{}}`, expectedMsg, level)
	assert.Equal(t, expectedLog, actualLog, "Unexpected log output.")
}

func assertFields(t testing.TB, jl Logger, getOutput func() []string, expectedFields ...string) {
	jl.Debug("")
	actualLogs := getOutput()
	for i, fields := range expectedFields {
		expectedLog := fmt.Sprintf(`{"msg":"","level":"debug","ts":0,"fields":%s}`, fields)
		assert.Equal(t, expectedLog, actualLogs[i], "Unexpected log output.")
	}
}

func TestJSONLoggerSetLevel(t *testing.T) {
	withJSONLogger(t, nil, func(jl *jsonLogger, _ func() []string) {
		assert.Equal(t, All, jl.Level(), "Unexpected initial level.")
		jl.SetLevel(Debug)
		assert.Equal(t, Debug, jl.Level(), "Unexpected level after SetLevel.")
	})
}

func TestJSONLoggerEnabled(t *testing.T) {
	withJSONLogger(t, opts(Info), func(jl *jsonLogger, _ func() []string) {
		assert.False(t, jl.Enabled(Debug), "Debug logs shouldn't be enabled at Info level.")
		assert.True(t, jl.Enabled(Info), "Info logs should be enabled at Info level.")
		assert.True(t, jl.Enabled(Warn), "Warn logs should be enabled at Info level.")
		assert.True(t, jl.Enabled(Error), "Error logs should be enabled at Info level.")
		assert.True(t, jl.Enabled(Panic), "Panic logs should be enabled at Info level.")
		assert.True(t, jl.Enabled(Fatal), "Fatal logs should be enabled at Info level.")

		for _, lvl := range []Level{Debug, Info, Warn, Error, Panic, Fatal} {
			jl.SetLevel(None)
			assert.False(t, jl.Enabled(lvl), "No logging should be enabled at None level.")
			jl.SetLevel(All)
			assert.True(t, jl.Enabled(lvl), "All logging should be enabled at All level.")
		}
	})
}

func TestJSONLoggerConcurrentLevelMutation(t *testing.T) {
	// Trigger races for non-atomic level mutations.
	proceed := make(chan struct{})
	jl := NewJSON()

	for i := 0; i < 50; i++ {
		go func(l Logger) {
			<-proceed
			jl.Level()
		}(jl)
		go func(l Logger) {
			<-proceed
			jl.SetLevel(Warn)
		}(jl)
	}
	close(proceed)
}

func TestJSONLoggerInitialFields(t *testing.T) {
	fieldOpts := opts(Fields(Int("foo", 42), String("bar", "baz")))
	withJSONLogger(t, fieldOpts, func(jl *jsonLogger, output func() []string) {
		assertFields(t, jl, output, `{"foo":42,"bar":"baz"}`)
	})
}

func TestJSONLoggerWith(t *testing.T) {
	fieldOpts := opts(Fields(Int("foo", 42)))
	withJSONLogger(t, fieldOpts, func(jl *jsonLogger, output func() []string) {
		// Child loggers should have copy-on-write semantics, so two children
		// shouldn't stomp on each other's fields or affect the parent's fields.
		jl.With().Debug("")
		jl.With(String("one", "two")).Debug("")
		jl.With(String("three", "four")).Debug("")
		assertFields(t, jl, output,
			`{"foo":42}`,
			`{"foo":42,"one":"two"}`,
			`{"foo":42,"three":"four"}`,
			`{"foo":42}`,
		)
	})
}

func TestJSONLoggerLog(t *testing.T) {
	withJSONLogger(t, nil, func(jl *jsonLogger, output func() []string) {
		jl.Log(Debug, "foo")
		assertMessage(t, "debug", "foo", output()[0])
	})
	withJSONLogger(t, nil, func(jl *jsonLogger, output func() []string) {
		assert.Panics(t, func() { jl.Log(Panic, "foo") }, "Expected logging at Panic level to panic.")
		assertMessage(t, "panic", "foo", output()[0])
	})
}

func TestJSONLoggerDebug(t *testing.T) {
	withJSONLogger(t, nil, func(jl *jsonLogger, output func() []string) {
		jl.Debug("foo")
		assertMessage(t, "debug", "foo", output()[0])
	})
}

func TestJSONLoggerInfo(t *testing.T) {
	withJSONLogger(t, nil, func(jl *jsonLogger, output func() []string) {
		jl.Info("foo")
		assertMessage(t, "info", "foo", output()[0])
	})
}

func TestJSONLoggerWarn(t *testing.T) {
	withJSONLogger(t, nil, func(jl *jsonLogger, output func() []string) {
		jl.Warn("foo")
		assertMessage(t, "warn", "foo", output()[0])
	})
}

func TestJSONLoggerError(t *testing.T) {
	withJSONLogger(t, nil, func(jl *jsonLogger, output func() []string) {
		jl.Error("foo")
		assertMessage(t, "error", "foo", output()[0])
	})
}

func TestJSONLoggerPanic(t *testing.T) {
	withJSONLogger(t, nil, func(jl *jsonLogger, output func() []string) {
		assert.Panics(t, func() {
			jl.Panic("foo")
		})
		assertMessage(t, "panic", "foo", output()[0])
	})
}

func TestJSONLoggerDFatal(t *testing.T) {
	withJSONLogger(t, nil, func(jl *jsonLogger, output func() []string) {
		jl.DFatal("foo")
		assertMessage(t, "error", "foo", output()[0])
	})
}

func TestJSONLoggerNoOpsDisabledLevels(t *testing.T) {
	withJSONLogger(t, nil, func(jl *jsonLogger, output func() []string) {
		jl.SetLevel(Warn)
		jl.Info("silence!")
		assert.Equal(t, []string{""}, output(), "Expected logging at a disabled level to produce no output.")
	})
}

func TestJSONLoggerInternalErrorHandling(t *testing.T) {
	buf := newTestBuffer()
	errBuf := newTestBuffer()

	jl := NewJSON(All, Output(buf), ErrorOutput(errBuf), Fields(Marshaler("user", fakeUser{"fail"})))
	jl.StubTime()
	output := func() []string { return strings.Split(buf.String(), "\n") }

	// Expect partial output, even if there's an error serializing
	// user-defined types.
	assertFields(t, jl, output, `{"user":{}}`)
	// Internal errors go to stderr.
	assert.Equal(t, "fail\n", errBuf.String(), "Expected internal errors to print to stderr.")
}

func TestJSONLoggerWriteMessageFailure(t *testing.T) {
	errBuf := &bytes.Buffer{}
	errSink := &spywrite.WriteSyncer{Writer: errBuf}
	logger := NewJSON(All, Output(AddSync(spywrite.FailWriter{})), ErrorOutput(errSink))

	logger.Info("foo")
	// Should log the error.
	assert.Equal(t, "failed\n", errBuf.String(), "Expected to log the error to the error output.")
	assert.True(t, errSink.Called(), "Expected logging an internal error to Sync error WriteSyncer.")
}

func TestJSONLoggerRuntimeLevelChange(t *testing.T) {
	// Test that changing a logger's level also changes the level of all
	// ancestors and descendants.
	grandparent := NewJSON(Fields(Int("generation", 1)))
	parent := grandparent.With(Int("generation", 2))
	child := parent.With(Int("generation", 3))

	all := []Logger{grandparent, parent, child}
	for _, logger := range all {
		assert.Equal(t, Info, logger.Level(), "Expected all loggers to start at Info level.")
	}

	parent.SetLevel(Debug)
	for _, logger := range all {
		assert.Equal(t, Debug, logger.Level(), "Expected all loggers to switch to Debug level.")
	}
}

func TestJSONLoggerSyncsOutput(t *testing.T) {
	sink := &spywrite.WriteSyncer{Writer: ioutil.Discard}
	logger := NewJSON(All, Output(sink))

	logger.Error("foo")
	assert.False(t, sink.Called(), "Didn't expect logging at error level to Sync underlying WriteCloser.")

	assert.Panics(t, func() { logger.Panic("foo") }, "Expected panic when logging at Panic level.")
	assert.True(t, sink.Called(), "Expected logging at panic level to Sync underlying WriteSyncer.")
}

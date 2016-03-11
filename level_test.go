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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevelString(t *testing.T) {
	tests := map[Level]string{
		All:        "all",
		Debug:      "debug",
		Info:       "info",
		Warn:       "warn",
		Error:      "error",
		Panic:      "panic",
		Fatal:      "fatal",
		None:       "none",
		Level(-42): "Level(-42)",
	}

	for lvl, stringLevel := range tests {
		assert.Equal(t, stringLevel, lvl.String())
	}
}

func TestLevelText(t *testing.T) {
	tests := []struct {
		text  string
		level Level
	}{
		{"all", All},
		{"debug", Debug},
		{"info", Info},
		{"warn", Warn},
		{"error", Error},
		{"panic", Panic},
		{"fatal", Fatal},
		{"none", None},
	}
	for _, tt := range tests {
		lvl := tt.level
		marshaled, err := lvl.MarshalText()
		assert.NoError(t, err, "Unexpected error marshaling level %v to text.", &lvl)
		assert.Equal(t, tt.text, string(marshaled), "Marshaling level %v to text yielded unexpected result.", &lvl)

		var unmarshaled Level
		err = unmarshaled.UnmarshalText([]byte(tt.text))
		assert.NoError(t, err, `Unexpected error unmarshaling text "%v" to level.`, tt.text)
		assert.Equal(t, tt.level, unmarshaled, `Text "%v" unmarshaled to an unexpected level.`, tt.text)
	}
}

func TestLevelNils(t *testing.T) {
	// Test all our nil-handling logic here.
	var l *Level
	assert.Equal(t, "Level(nil)", l.String(), "Unexpected result stringifying nil *Level.")

	_, err := l.MarshalText()
	assert.Equal(t, _errMarshalNilLevel, err, "Expected _errMarshalNilLevel.")

	assert.Panics(t, func() {
		var l *Level
		l.UnmarshalText([]byte("debug"))
	}, "Expected to panic when unmarshaling into a null pointer.")
}
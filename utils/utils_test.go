package utils

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/MaxIvanyshen/browser-engineering-go/engine"
)

func TestShow(t *testing.T) {
	tests := []struct {
		name     string
		resp     *engine.Response
		expected string
	}{
		{
			name: "ViewSource true",
			resp: &engine.Response{
				Body:       []byte("<html><body>Hello, World!</body></html>"),
				ViewSource: true,
			},
			expected: "<html><body>Hello, World!</body></html>\n",
		},
		{
			name: "ViewSource false",
			resp: &engine.Response{
				Body:       []byte("<html><body>Hello, World!</body></html>"),
				ViewSource: false,
			},
			expected: "Hello, World!",
		},
		{
			name: "No tags",
			resp: &engine.Response{
				Body:       []byte("Hello, World!"),
				ViewSource: false,
			},
			expected: "Hello, World!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call Show
			Show(tt.resp)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if output != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, output)
			}
		})
	}
}

func TestEntityDecoding(t *testing.T) {
	tests := []struct {
		name     string
		input    *engine.Response
		expected string
	}{
		{
			name: "Basic entities",
			input: &engine.Response{
				Body:       []byte("Hello &lt;World&gt; &amp; Everyone"),
				ViewSource: false,
			},
			expected: "Hello <World> & Everyone",
		},
		{
			name: "Quotes and apostrophes",
			input: &engine.Response{
				Body:       []byte("&quot;It's a test&quot;"),
				ViewSource: false,
			},
			expected: "\"It's a test\"",
		},
		{
			name: "No entities",
			input: &engine.Response{
				Body:       []byte("Just a normal string."),
				ViewSource: false,
			},
			expected: "Just a normal string.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call Show
			Show(tt.input)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if output != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, output)
			}
		})
	}
}

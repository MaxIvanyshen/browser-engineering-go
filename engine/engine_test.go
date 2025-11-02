package engine

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
)

func TestURL_Parse(t *testing.T) {
	tests := []struct {
		input       string
		expected    *URL
		expectError bool
	}{
		{
			input: "http://example.com/path",
			expected: &URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "/path",
			},
			expectError: false,
		},
		{
			input:       "ftp://example.com/path",
			expected:    nil,
			expectError: true,
		},
		{
			input:       "invalid-url",
			expected:    nil,
			expectError: true,
		},
	}
	for _, test := range tests {
		result, err := Parse(test.input)
		if test.expectError {
			if err == nil {
				t.Errorf("expected error for input %q, got nil", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %q: %v", test.input, err)
			}
			if result.Host != test.expected.Host || result.Scheme != test.expected.Scheme || result.Path != test.expected.Path {
				t.Errorf("for input %q, expected %+v, got %+v", test.input, test.expected, result)
			}
		}
	}
}

func TestRequest(t *testing.T) {
	bigStr := ""
	for range 2000 { // big string for testing, but not too big (to avoid chunked encoding)
		bigStr += "a"
	}

	go func() {
		http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello, World!")
		})
		http.HandleFunc("/index", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "<html><body>Index Page</body></html>")
		})
		http.HandleFunc("/big", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, bigStr)
		})
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	testCases := []struct {
		url      string
		expected string
	}{
		{"http://localhost:8080/test", "Hello, World!\n"},
		{"http://localhost:8080/index", "<html><body>Index Page</body></html>\n"},
		{"http://localhost:8080/big", bigStr + "\n"},
	}

	done := make(chan struct{})
	go addFileTestCases(&testCases, done)
	<-done            // wait for file test case to be added
	defer close(done) // close the channel to signal completion

	for _, tc := range testCases {
		url, err := Parse(tc.url)
		if err != nil {
			t.Fatalf("failed to parse URL %q: %v", tc.url, err)
		}
		log.Printf("Testing URL: %s", tc.url)
		response, err := Request(url, nil)
		if err != nil {
			t.Fatalf("request to %q failed: %v", tc.url, err)
		}
		if string(response) != tc.expected {
			t.Errorf("for URL %q, expected response %q, got %q", tc.url, tc.expected, string(response))
		}
	}
}

func addFileTestCases(testCases *[]struct {
	url      string
	expected string
}, done chan struct{}) {
	// Create a temporary file for testing file:// URLs
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		log.Fatalf("failed to create temp file: %v", err)
	}

	content := "This is a test file."
	if _, err := tmpFile.WriteString(content); err != nil {
		log.Fatalf("failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	*testCases = append(*testCases, struct {
		url      string
		expected string
	}{
		url:      "file://" + tmpFile.Name(),
		expected: content,
	})

	done <- struct{}{} // signal that the file test case is added

	<-done // wait for tests to finish
	os.Remove(tmpFile.Name())
}

func TestCustomHeaders(t *testing.T) {
	go func() {
		http.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
			customHeader := r.Header.Get("X-Custom-Header")
			userAgent := r.Header.Get("User-Agent")
			fmt.Fprintf(w, "%s\n%s", customHeader, userAgent)
		})
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	url, err := Parse("http://localhost:8081/headers")
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	headers := map[string]string{
		"X-Custom-Header": "CustomValue",
		"User-Agent":      "GoTestClient/1.0",
	}

	response, err := Request(url, headers)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	expected := "CustomValue\nGoTestClient/1.0"
	if string(response) != expected {
		t.Errorf("expected response %q, got %q", expected, string(response))
	}
}

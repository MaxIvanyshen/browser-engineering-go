package telnet

import (
	"fmt"
	"log"
	"net/http"
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
			} else if *result != *test.expected {
				t.Errorf("for input %q, expected %+v, got %+v", test.input, test.expected, result)
			}
		}
	}
}

func TestRequest(t *testing.T) {
	bigStr := ""
	for range 2000 { // big string for testing, but not too big (to aboid chunked encoding)
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
	for _, tc := range testCases {
		url, err := Parse(tc.url)
		if err != nil {
			t.Fatalf("failed to parse URL %q: %v", tc.url, err)
		}
		response, err := url.Request()
		if err != nil {
			t.Fatalf("request to %q failed: %v", tc.url, err)
		}
		if string(response) != tc.expected {
			t.Errorf("for URL %q, expected response %q, got %q", tc.url, tc.expected, string(response))
		}
	}
}

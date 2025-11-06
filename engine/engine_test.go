package engine

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
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
				scheme: "http",
				host:   "example.com",
				path:   "/path",
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
			if result.host != test.expected.host || result.scheme != test.expected.scheme || result.path != test.expected.path {
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

		// data URL test cases
		{"data:,Hello%2C%20World!", "Hello, World!"},
		{"data:text/plain;base64,SGVsbG8sIFdvcmxkIQ==", "Hello, World!"},
	}

	done := make(chan struct{})
	go addFileTestCases(&testCases, done)
	<-done            // wait for file test case to be added
	defer close(done) // close the channel to signal completion

	e := NewEngine()

	for _, tc := range testCases {
		url, err := Parse(tc.url)
		if err != nil {
			t.Fatalf("failed to parse URL %q: %v", tc.url, err)
		}
		log.Printf("Testing URL: %s", tc.url)
		response, err := e.Request(url, nil)
		if err != nil {
			t.Fatalf("request to %q failed: %v", tc.url, err)
		}
		if string(response.Body) != tc.expected {
			t.Errorf("for URL %q, expected response %q, got %q", tc.url, tc.expected, string(response.Body))
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

	e := NewEngine()

	response, err := e.Request(url, headers)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	expected := "CustomValue\nGoTestClient/1.0"
	if string(response.Body) != expected {
		t.Errorf("expected response %q, got %q", expected, string(response.Body))
	}
}

func TestConnectionKeepAlive(t *testing.T) {
	var connectionCount int
	go func() {
		http.HandleFunc("/keepalive", func(w http.ResponseWriter, r *http.Request) {
			connectionCount++
			fmt.Fprintf(w, "Connection count: %d", connectionCount)
		})
		log.Fatal(http.ListenAndServe(":8082", nil))
	}()

	url, err := Parse("http://localhost:8082/keepalive")
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	e := NewEngine()

	for i := 1; i <= 3; i++ {
		response, err := e.Request(url, nil)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		expected := fmt.Sprintf("Connection count: %d", i)
		if string(response.Body) != expected {
			t.Errorf("expected response %q, got %q", expected, string(response.Body))
		}
	}
}

func TestRedirectHandling(t *testing.T) {
	go func() {
		http.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/final", http.StatusFound)
		})
		http.HandleFunc("/final", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Final Destination")
		})
		log.Fatal(http.ListenAndServe(":8083", nil))
	}()

	testCases := []struct {
		url   string
		final string
	}{
		{"http://localhost:8083/redirect", "http://localhost:8083/final"},
		{"http://browser.engineering/redirect", "http://browser.engineering/http.html"},
		{"http://browser.engineering/redirect2", "http://browser.engineering/http.html"},
		{"http://browser.engineering/redirect3", "http://browser.engineering/http.html"},
	}

	e := NewEngine()

	for _, tc := range testCases {
		url, err := Parse(tc.url)
		if err != nil {
			t.Fatalf("failed to parse URL %q: %v", tc.url, err)
		}
		response, err := e.Request(url, nil)
		if err != nil {
			t.Fatalf("request to %q failed: %v", tc.url, err)
		}
		if response.URL != tc.final {
			t.Errorf("for URL %q, expected response %q, got %q", tc.url, tc.final, response.URL)
		}
	}
}

func TestCacheBehavior(t *testing.T) {
	var requestCount int
	go func() {
		http.HandleFunc("/cache", func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.Header().Set("Cache-Control", "max-age=2")
			fmt.Fprintln(w, "Cached Content")
		})
		log.Fatal(http.ListenAndServe(":8084", nil))
	}()

	url, err := Parse("http://localhost:8084/cache")
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}

	e := NewEngine()

	// First request should hit the server
	response, err := e.Request(url, nil)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if string(response.Body) != "Cached Content\n" {
		t.Errorf("expected 'Cached Content', got %q", string(response.Body))
	}
	if requestCount != 1 {
		t.Errorf("expected 1 request, got %d", requestCount)
	}

	// Second request should be served from cache
	response, err = e.Request(url, nil)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	if string(response.Body) != "Cached Content\n" {
		t.Errorf("expected 'Cached Content', got %q", string(response.Body))
	}
	if requestCount != 1 {
		t.Errorf("expected 1 request, got %d", requestCount)
	}

	// Wait for cache to expire
	time.Sleep(3 * time.Second)

	// Third request should hit the server again
	response, err = e.Request(url, nil)
	if err != nil {
		t.Fatalf("third request failed: %v", err)
	}
	if string(response.Body) != "Cached Content\n" {
		t.Errorf("expected 'Cached Content', got %q", string(response.Body))
	}
	if requestCount != 2 {
		t.Errorf("expected 2 requests, got %d", requestCount)
	}
}

func TestChunkedResponseHandling(t *testing.T) {
	go func() {
		http.HandleFunc("/chunked", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Transfer-Encoding", "chunked")
			w.Write([]byte("Hello, this is a chunked response."))
		})
		log.Fatal(http.ListenAndServe(":8085", nil))
	}()

	url, err := Parse("http://localhost:8085/chunked")
	if err != nil {
		t.Fatalf("failed to parse URL: %v", err)
	}
	e := NewEngine()
	response, err := e.Request(url, nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body := response.Body

	expected := "Hello, this is a chunked response."
	if string(body) != expected {
		t.Errorf("body mismatch\n got: %q\nwant: %q", body, expected)
	}
}

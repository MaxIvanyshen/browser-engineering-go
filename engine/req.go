package engine

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

var allowedSchemes = []string{"http", "https", "file"}

func Request(url *URL) ([]byte, error) {
	hostWithPort := url.Host
	if !strings.Contains(hostWithPort, ":") {
		hostWithPort = fmt.Sprintf("%s:%s", url.Host, url.Port)
	}

	var conn io.ReadWriteCloser
	var err error
	switch url.Scheme {
	case "http":
		conn, err = net.Dial("tcp", hostWithPort)
	case "https":
		conn, err = tls.Dial("tcp", hostWithPort, &tls.Config{})
	case "file":
		return os.ReadFile(url.Path)
	}
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", url.Path, url.Host)
	if _, err := conn.Write([]byte(req)); err != nil {
		return nil, err
	}

	var responseBuf bytes.Buffer
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil || n == 0 {
			break
		}
		responseBuf.Write(buf[:n])
	}

	responseData := responseBuf.Bytes()

	// Find the end of headers
	headerEnd := bytes.Index(responseData, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		return nil, fmt.Errorf("invalid HTTP response: no header end")
	}

	headerData := responseData[:headerEnd]
	bodyData := responseData[headerEnd+4:]

	headerLines := strings.Split(string(headerData), "\r\n")
	if len(headerLines) == 0 {
		return nil, fmt.Errorf("invalid HTTP response: no status line")
	}

	statusLine := headerLines[0]
	if !strings.HasPrefix(statusLine, "HTTP/1.1") {
		return nil, fmt.Errorf("unsupported HTTP version")
	}
	statusParts := strings.Split(statusLine, " ")
	if len(statusParts) < 2 {
		return nil, fmt.Errorf("invalid status line")
	}
	if _, err := strconv.Atoi(statusParts[1]); err != nil {
		return nil, fmt.Errorf("invalid status code: %s", statusParts[1])
	}

	headers := make(map[string]string)
	for _, line := range headerLines[1:] {
		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	if clStr, ok := headers["Content-Length"]; ok {
		if cl, err := strconv.Atoi(clStr); err == nil && len(bodyData) > cl {
			bodyData = bodyData[:cl]
		}
	}

	return bodyData, nil
}

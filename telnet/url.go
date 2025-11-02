package telnet

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type URL struct {
	Scheme string
	Host   string
	Path   string
}

func Parse(url string) (*URL, error) {
	parts := strings.Split(url, "://")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid URL format")
	}
	scheme := parts[0]
	if scheme != "http" {
		return nil, fmt.Errorf("unsupported scheme: %s", scheme)
	}
	if len(parts) > 1 {
		url = parts[1]
	} else {
		url = ""
	}
	parts = strings.SplitN(url, "/", 2)
	host := parts[0]

	if len(parts) > 1 {
		url = parts[1]
	} else {
		url = ""
	}

	return &URL{
		Scheme: scheme,
		Host:   host,
		Path:   "/" + url,
	}, nil
}

func (u *URL) String() string {
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path)
}

func (u *URL) Request() ([]byte, error) {
	hostWithPort := u.Host
	if !strings.Contains(hostWithPort, ":") {
		hostWithPort += ":80"
	}

	conn, err := net.Dial("tcp", hostWithPort)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", u.Path, u.Host)
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

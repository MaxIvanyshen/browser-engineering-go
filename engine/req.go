package engine

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

var allowedSchemes = []string{"http", "https", "file", "data"}

type Header struct {
	Key   string
	Value string
}

func (h *Header) String() string {
	return fmt.Sprintf("%s: %s", h.Key, h.Value)
}

func NewHeader(key, value string) *Header {
	return &Header{
		Key:   key,
		Value: value,
	}
}

type Response struct {
	Headers    map[string]string
	Body       []byte
	ViewSource bool
}

// urlUnescape decodes URL-encoded string
func urlUnescape(s string) (string, error) {
	var result []byte
	i := 0
	for i < len(s) {
		if s[i] == '%' {
			if i+2 >= len(s) {
				return "", fmt.Errorf("invalid percent encoding")
			}
			high := s[i+1]
			low := s[i+2]
			var val byte
			if high >= '0' && high <= '9' {
				val = (high - '0') << 4
			} else if high >= 'A' && high <= 'F' {
				val = (high - 'A' + 10) << 4
			} else if high >= 'a' && high <= 'f' {
				val = (high - 'a' + 10) << 4
			} else {
				return "", fmt.Errorf("invalid hex digit: %c", high)
			}
			if low >= '0' && low <= '9' {
				val |= low - '0'
			} else if low >= 'A' && low <= 'F' {
				val |= low - 'A' + 10
			} else if low >= 'a' && low <= 'f' {
				val |= low - 'a' + 10
			} else {
				return "", fmt.Errorf("invalid hex digit: %c", low)
			}
			result = append(result, val)
			i += 3
		} else {
			result = append(result, s[i])
			i++
		}
	}
	return string(result), nil
}

func ParseHeader(headerStr string) (*Header, error) {
	parts := strings.SplitN(headerStr, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid header format")
	}
	return &Header{
		Key:   strings.TrimSpace(parts[0]),
		Value: strings.TrimSpace(parts[1]),
	}, nil
}

func Request(url *URL, headers map[string]string) (*Response, error) {
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
		bytes, err := os.ReadFile(url.Path)
		if err != nil {
			return nil, err
		}
		return &Response{
			Headers: make(map[string]string),
			Body:    bytes,
		}, nil
	case "data":
		log.Println("data URL detected", url.Path)
		commaIndex := strings.Index(url.Path, ",")
		if commaIndex == -1 {
			return nil, fmt.Errorf("invalid data URL")
		}
		meta := url.Path[:commaIndex]
		data := url.Path[commaIndex+1:]
		isBase64 := strings.Contains(meta, ";base64")
		if isBase64 {
			decoded, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				return nil, err
			}
			return &Response{
				Headers: make(map[string]string),
				Body:    decoded,
			}, nil
		}
		unescaped, err := urlUnescape(data)
		if err != nil {
			return nil, err
		}
		return &Response{
			Headers: make(map[string]string),
			Body:    []byte(unescaped),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", url.Scheme)
	}
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if headers == nil {
		headers = make(map[string]string)
	}

	headers["Host"] = url.Host
	if _, ok := headers["Connection"]; !ok {
		headers["Connection"] = "close"
	}

	req := fmt.Sprintf("GET %s HTTP/1.1\r\n", url.Path)
	for k, v := range headers {
		req += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	req += "\r\n"
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

	respHeaders := make(map[string]string)
	for _, line := range headerLines[1:] {
		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			respHeaders[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	if clStr, ok := respHeaders["Content-Length"]; ok {
		if cl, err := strconv.Atoi(clStr); err == nil && len(bodyData) > cl {
			bodyData = bodyData[:cl]
		}
	}

	return &Response{
		Headers:    respHeaders,
		Body:       bodyData,
		ViewSource: url.ViewSource,
	}, nil
}

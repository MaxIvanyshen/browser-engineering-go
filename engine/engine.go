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

const MAX_REDIRECTS = 3

var allowedSchemes = []string{"http", "https", "file", "data"}

type Engine struct {
	connMap map[string]*io.ReadWriteCloser
	cache   map[string]*CacheValue[*Response]
}

func NewEngine() *Engine {
	return &Engine{
		connMap: make(map[string]*io.ReadWriteCloser),
		cache:   make(map[string]*CacheValue[*Response]),
	}
}

type Response struct {
	URL        string
	StatusCode int
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

func (e *Engine) Request(url *URL, headers map[string]string) (*Response, error) {
	if cacheValue, ok := e.cache[url.String()]; ok {
		if cacheValue.IsExpired() {
			delete(e.cache, url.String())
		} else {
			return cacheValue.Value, nil
		}
	}
	delete(e.cache, url.String())

	hostWithPort := url.host
	if !strings.Contains(hostWithPort, ":") {
		hostWithPort = fmt.Sprintf("%s:%s", url.host, url.port)
	}

	var conn io.ReadWriteCloser
	var err error
	if existingConn, ok := e.connMap[hostWithPort]; ok {
		conn = *existingConn
	} else {
		switch url.scheme {
		case "http":
			conn, err = net.Dial("tcp", hostWithPort)
		case "https":
			conn, err = tls.Dial("tcp", hostWithPort, &tls.Config{})
		case "file":
			bytes, err := os.ReadFile(url.path)
			if err != nil {
				return nil, err
			}
			return &Response{
				Headers: make(map[string]string),
				Body:    bytes,
			}, nil
		case "data":
			log.Println("data URL detected", url.path)
			commaIndex := strings.Index(url.path, ",")
			if commaIndex == -1 {
				return nil, fmt.Errorf("invalid data URL")
			}
			meta := url.path[:commaIndex]
			data := url.path[commaIndex+1:]
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
			return nil, fmt.Errorf("unsupported scheme: %s", url.scheme)
		}
		if err != nil {
			return nil, err
		}
	}

	if headers == nil {
		headers = make(map[string]string)
	}

	headers["Host"] = url.host
	if _, ok := headers["Connection"]; !ok {
		headers["Connection"] = "close"
	}

	req := fmt.Sprintf("GET %s HTTP/1.1\r\n", url.path)
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
		if err == io.EOF || n == 0 {
			break
		}
		if err != nil {
			return nil, err
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

	statusCode, err := strconv.Atoi(statusParts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid status code: %s", statusParts[1])
	}

	respHeaders := make(map[string]string)
	for _, line := range headerLines[1:] {
		if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
			respHeaders[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	if statusCode >= 300 && statusCode < 400 {
		if location, ok := respHeaders["Location"]; ok {
			if strings.HasPrefix(location, "/") {
				location = fmt.Sprintf("%s://%s%s", url.scheme, url.host, location)
			}
			newURL, err := Parse(location)
			if err != nil {
				return nil, err
			}
			newURL.redirectCount = url.redirectCount + 1
			if newURL.redirectCount > MAX_REDIRECTS {
				return nil, fmt.Errorf("maximum redirects exceeded")
			}
			return e.Request(newURL, headers)
		}
	}

	if clStr, ok := respHeaders["Content-Length"]; ok {
		if cl, err := strconv.Atoi(clStr); err == nil && len(bodyData) > cl {
			bodyData = bodyData[:cl]
		}
	}

	if connectionHeader, ok := respHeaders["Connection"]; ok {
		if strings.ToLower(connectionHeader) == "close" {
			conn.Close()
			delete(e.connMap, hostWithPort)
		} else if strings.ToLower(connectionHeader) == "keep-alive" {
			e.connMap[hostWithPort] = &conn
		}
	} else {
		conn.Close()
		delete(e.connMap, hostWithPort)
	}

	if transferEncoding, ok := respHeaders["Transfer-Encoding"]; ok && strings.ToLower(transferEncoding) == "chunked" {
		log.Println("Decoding chunked body")
		log.Printf("Raw body data: %q", bodyData)
		bodyData, err = decodeChunkedBody(bodyData)
		if err != nil {
			return nil, err
		}
	}

	r := &Response{
		URL:        url.String(),
		StatusCode: statusCode,
		Headers:    respHeaders,
		Body:       bodyData,
		ViewSource: url.ViewSource,
	}

	cacheControl, ok := respHeaders["Cache-Control"]
	if ok && strings.Contains(cacheControl, "max-age") {
		parts := strings.Split(cacheControl, "=")
		if len(parts) != 2 {
			return r, nil
		}
		maxAgeStr := strings.TrimSpace(parts[1])
		maxAge, err := strconv.Atoi(maxAgeStr)
		if err != nil || maxAge <= 0 {
			return r, nil
		}
		e.cache[url.String()] = NewCacheValue(r, int64(maxAge))
	} else {
		delete(e.cache, url.String())
	}

	return r, nil
}

func decodeChunkedBody(body []byte) ([]byte, error) {
	decoded := bytes.Buffer{}
	for i := 0; i < len(body); {
		j := i
		for j < len(body) && body[j] != '\r' {
			j++
		}
		if j >= len(body)-1 || body[j] != '\r' || body[j+1] != '\n' {
			return nil, fmt.Errorf("invalid chunked encoding")
		}
		sizeStr := string(body[i:j])
		size, err := strconv.ParseInt(sizeStr, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid chunk size: %v", err)
		}
		if size == 0 {
			break
		}
		i = j + 2
		if i+int(size) > len(body) {
			return nil, fmt.Errorf("chunk size exceeds body length")
		}
		decoded.Write(body[i : i+int(size)])
		i += int(size)
		if i+1 >= len(body) || body[i] != '\r' || body[i+1] != '\n' {
			return nil, fmt.Errorf("invalid chunked encoding after chunk data")
		}
		i += 2
	}
	return decoded.Bytes(), nil
}

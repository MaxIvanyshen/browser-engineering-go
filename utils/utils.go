package utils

import (
	"fmt"

	"github.com/MaxIvanyshen/browser-engineering-go/engine"
)

func Show(resp *engine.Response) {
	if resp.ViewSource {
		fmt.Println(string(resp.Body))
		return
	}
	inTag := false
	for _, b := range resp.Body {
		if b == '<' {
			inTag = true
		} else if b == '>' {
			inTag = false
		} else if !inTag {
			fmt.Printf("%c", b)
		}
	}
}

// urlUnescape decodes URL-encoded string
func UrlUnescape(s string) (string, error) {
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

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

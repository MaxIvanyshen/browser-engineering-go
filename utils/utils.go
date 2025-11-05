package utils

import (
	"fmt"
	"strings"

	"github.com/MaxIvanyshen/browser-engineering-go/engine"
)

var entityMap = map[string]string{
	"lt":   "<",
	"gt":   ">",
	"amp":  "&",
	"quot": "\"",
	"apos": "'",
}

func Show(resp *engine.Response) {
	if resp.ViewSource {
		fmt.Println(string(resp.Body))
		return
	}
	inTag := false
	for i := 0; i < len(resp.Body); i++ {
		b := resp.Body[i]
		if b == '<' {
			inTag = true
		} else if b == '>' {
			inTag = false
		} else if !inTag {
			if b == '&' {
				semiIndex := strings.IndexByte(string(resp.Body[i:]), ';')
				if semiIndex != -1 {
					semiColonIndex := i + semiIndex
					entity := string(resp.Body[i+1 : semiColonIndex])
					if val, ok := entityMap[entity]; ok {
						fmt.Printf("%s", val)
						i = semiColonIndex
					} else {
						fmt.Printf("%c", b)
					}
				} else {
					fmt.Printf("%c", b)
				}
			} else {
				fmt.Printf("%c", b)
			}
		}
	}
}

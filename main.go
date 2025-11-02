package main

import (
	"fmt"
	"os"

	"github.com/MaxIvanyshen/browser-engineering-go/telnet"
)

func main() {
	if len(os.Args) < 2 {
		println("Please provide a URL as an argument.")
		return
	}

	url, err := telnet.Parse(os.Args[1])
	if err != nil {
		panic(err)
	}

	content, err := url.Request()
	if err != nil {
		panic(err)
	}

	showContent(content)
}

func showContent(content []byte) {
	inTag := false
	for _, b := range content {
		if b == '<' {
			inTag = true
		} else if b == '>' {
			inTag = false
		} else if !inTag {
			fmt.Printf("%c", b)
		}
	}
}

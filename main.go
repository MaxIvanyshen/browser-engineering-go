package main

import (
	"fmt"
	"os"

	"github.com/MaxIvanyshen/browser-engineering-go/engine"
)

func main() {
	if len(os.Args) < 2 {
		println("Please provide a URL as an argument.")
		return
	}

	url, err := engine.Parse(os.Args[1])
	if err != nil {
		panic(err)
	}

	resp, err := engine.Request(url, nil)
	if err != nil {
		panic(err)
	}

	utils.Show(resp)
}

func showContent(resp *engine.Response) {
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

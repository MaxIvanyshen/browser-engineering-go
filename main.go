package main

import (
	"os"

	"github.com/MaxIvanyshen/browser-engineering-go/engine"
	"github.com/MaxIvanyshen/browser-engineering-go/utils"
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

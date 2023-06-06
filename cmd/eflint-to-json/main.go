package main

import (
	"fmt"
	"github.com/Olaf-Erkemeij/eflint-server/internal/parser"
	"os"
)

func main() {
	//fmt.Println(parser.String())
	filename := ""
	file := os.Stdin
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			panic(err)
		}
		defer f.Close()
		filename = os.Args[1]
		file = f
	}

	result, err := parser.ParseFile(filename, file)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(result))
}

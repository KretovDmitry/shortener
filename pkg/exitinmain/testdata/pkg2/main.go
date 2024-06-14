package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("vim-go")
	os.Exit(255) // want `os.Exit call inside main function`
}

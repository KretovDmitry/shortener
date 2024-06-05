package main

import "os"

func main() {
	f := func(code int) { os.Exit(code) } // want `os.Exit call inside main function`
	f(1)
}

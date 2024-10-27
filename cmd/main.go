package main

import (
	"os"

	"github.com/gkwa/polarpaw"
)

func main() {
	code := polarpaw.Execute()
	os.Exit(code)
}

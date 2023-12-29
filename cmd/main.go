package main

import (
	"os"

	"github.com/taylormonacelli/polarpaw"
)

func main() {
	code := polarpaw.Execute()
	os.Exit(code)
}

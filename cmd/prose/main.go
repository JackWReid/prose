package main

import (
	"fmt"
	"os"

	"github.com/JackWReid/prose/internal/editor"
)

var Version = "dev"

func main() {
	filenames := os.Args[1:]

	app := editor.NewApp(filenames)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "prose: %v\n", err)
		os.Exit(1)
	}
}

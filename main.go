package main

import (
	"fmt"
	"os"
)

const Version = "1.10.0"

func main() {
	filenames := os.Args[1:]

	app := NewApp(filenames)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "prose: %v\n", err)
		os.Exit(1)
	}
}

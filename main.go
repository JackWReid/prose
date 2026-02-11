package main

import (
	"fmt"
	"os"
)

func main() {
	var filename string
	if len(os.Args) > 1 {
		filename = os.Args[1]
	}

	app := NewApp(filename)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "prose: %v\n", err)
		os.Exit(1)
	}
}

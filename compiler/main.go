package main

import (
	"context"
	"fmt"
	"os"

	"github.com/s0cks/docker-chromium-screenshot/compiler/cmd"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx := context.Background()
	if err := cmd.Execute(ctx); err != nil {
		fmt.Printf("error: %v", err)
		return 1
	}

	return 0
}

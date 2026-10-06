package main

import (
	"fmt"
	"os"

	"github.com/s0cks/docker-chromium-screenshot/service/internal/cli"
)

// Set at build time: go build -ldflags "-X main.version=... -X main.commit=... -X main.branch=... -X main.buildTime=..."
func main() {
	if err := cli.RootCommand.Execute(); err != nil {
		fmt.Printf("failed to execute: %v", err)
		os.Exit(1)
	}
}

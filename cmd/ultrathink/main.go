package main

import (
	"fmt"
	"os"
)

var (
	// Version is set during build
	Version = "dev"
	// BuildTime is set during build
	BuildTime = "unknown"
)

func main() {
	fmt.Printf("Ultrathink v%s (built: %s)\n", Version, BuildTime)
	fmt.Println("AI-powered persistent memory system")
	fmt.Println()
	fmt.Println("Development in progress...")
	fmt.Println("See https://github.com/MycelicMemory/ultrathink for details")

	os.Exit(0)
}

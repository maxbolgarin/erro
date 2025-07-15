package main

import "fmt"

func main() {
	fmt.Println("This is the main entry point for the `erro` examples.")
	fmt.Println("Please run the individual examples in the subdirectories.")
	fmt.Println("\nAvailable examples:")
	fmt.Println("  - basic: Demonstrates basic error creation and wrapping.")
	fmt.Println("    Run with: go run ./examples/basic")
	fmt.Println("  - logging: Shows integration with structured logging.")
	fmt.Println("    Run with: go run ./examples/logging")
	fmt.Println("  - templates: Illustrates the use of error templates.")
	fmt.Println("    Run with: go run ./examples/templates")
	fmt.Println("  - collections: Covers the usage of error lists and sets.")
	fmt.Println("    Run with: go run ./examples/collections")
	fmt.Println("  - http: An example of an HTTP server that uses `erro` for error handling.")
	fmt.Println("    Run with: go run ./examples/http")
}
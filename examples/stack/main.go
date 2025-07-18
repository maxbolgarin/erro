//go:build go1.21

package main

import (
	"fmt"
	"log/slog"

	"github.com/maxbolgarin/erro"
)

func sensitiveFunction(cfg *erro.StackTraceConfig) erro.Error {
	return erro.New("sensitive error occurred", "user_id", "12345", "api_key", "secret_key_123", erro.StackTraceWithSkip(0, cfg))
}

func processPayment(cfg *erro.StackTraceConfig) erro.Error {
	return erro.Wrap(sensitiveFunction(cfg), "payment processing failed", "transaction_id", "tx_456")
}

func handleRequest(cfg *erro.StackTraceConfig) erro.Error {
	return erro.Wrap(processPayment(cfg), "request handling failed", "request_id", "req_789")
}

func main() {
	// Development mode (default) - shows all information
	fmt.Println("=== DEVELOPMENT MODE (Default) ===")
	err := handleRequest(erro.DevelopmentStackTraceConfig())
	fmt.Printf("%+v\n", err)

	// Production mode - hides sensitive information
	fmt.Println("\n=== PRODUCTION MODE ===")
	err = handleRequest(erro.ProductionStackTraceConfig())
	fmt.Printf("%+v\n", err)

	// Strict mode - minimal information
	fmt.Println("\n=== STRICT MODE ===")
	err = handleRequest(erro.StrictStackTraceConfig())
	fmt.Printf("%+v\n", err)

	// Custom configuration
	fmt.Println("\n=== CUSTOM CONFIGURATION ===")
	cfg := &erro.StackTraceConfig{
		ShowFileNames:     true,
		ShowFullPaths:     false,
		ShowFunctionNames: true, // Show function names but not paths
		ShowPackageNames:  false,
		ShowLineNumbers:   false,
		ShowAllCodeFrames: true,
		PathElements:      1, // Show only 1 path element (just parent directory + filename)
		FunctionRedacted:  "[FUNC]",
		MaxFrames:         3,
	}
	err = handleRequest(cfg)
	fmt.Printf("%+v\n", err)

	// Demonstrate secure formatting methods
	fmt.Println("\n=== SECURE FORMATTING METHODS ===")
	err = handleRequest(erro.ProductionStackTraceConfig())
	fmt.Printf("SecureString: %s\n", err.Stack().String())
	fmt.Printf("SecureFormatFull:\n%s\n", err.Stack().FormatFull())

	erro.LogError(err, func(message string, fields ...any) {
		slog.Error(message, fields...)
	}, erro.WithStackFormat(erro.StackFormatJSON))
}

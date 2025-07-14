package main

import (
	"fmt"
	"log/slog"

	"github.com/maxbolgarin/erro"
)

func sensitiveFunction() erro.Error {
	return erro.New("sensitive error occurred", "user_id", "12345", "api_key", "secret_key_123")
}

func processPayment() erro.Error {
	return erro.Wrap(sensitiveFunction(), "payment processing failed", "transaction_id", "tx_456")
}

func handleRequest() erro.Error {
	return erro.Wrap(processPayment(), "request handling failed", "request_id", "req_789")
}

func main() {
	// Development mode (default) - shows all information
	fmt.Println("=== DEVELOPMENT MODE (Default) ===")
	err := handleRequest()
	fmt.Printf("%+v\n", err)

	// Production mode - hides sensitive information
	fmt.Println("\n=== PRODUCTION MODE ===")
	erro.SetProductionStackTrace()
	err = handleRequest()
	fmt.Printf("%+v\n", err)

	// Strict mode - minimal information
	fmt.Println("\n=== STRICT MODE ===")
	erro.SetStrictStackTrace()
	err = handleRequest()
	fmt.Printf("%+v\n", err)

	// Completely disable stack traces
	fmt.Println("\n=== DISABLED STACK TRACES ===")
	erro.DisableStackTrace()
	err = handleRequest()
	fmt.Printf("%+v\n", err)

	// Custom configuration
	fmt.Println("\n=== CUSTOM CONFIGURATION ===")
	erro.SetDefaultStackTraceConfig(&erro.StackTraceConfig{
		Enabled:           true,
		ShowFullPaths:     false,
		ShowFunctionNames: true, // Show function names but not paths
		ShowPackageNames:  false,
		ShowLineNumbers:   false,
		ShowAllCodeFrames: true,
		PathElements:      1, // Show only 1 path element (just parent directory + filename)
		FunctionRedacted:  "[FUNC]",
		MaxFrames:         3,
	})
	err = handleRequest()
	fmt.Printf("%+v\n", err)

	// Demonstrate secure formatting methods
	fmt.Println("\n=== SECURE FORMATTING METHODS ===")
	erro.SetProductionStackTrace()
	err = handleRequest()
	fmt.Printf("SecureString: %s\n", err.Context().Stack().String())
	fmt.Printf("SecureFormatFull:\n%s\n", err.Context().Stack().FormatFull())

	erro.LogError(err, func(message string, fields ...any) {
		slog.Error(message, fields...)
	}, erro.WithStackFormat(erro.StackFormatJSON))
}

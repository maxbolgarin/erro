package main

import (
	"fmt"

	"github.com/maxbolgarin/erro"
)

func main() {
	// Basic error creation
	err1 := erro.New("this is a simple error")
	fmt.Printf("Basic error: %v\n\n", err1)

	// Error with fields
	err2 := erro.New("user not found", "user_id", 123, erro.StackTrace())
	fmt.Printf("Error with fields: %v\n\n", err2)

	// Wrapping an error
	err3 := erro.Wrap(err2, "failed to process request")
	fmt.Printf("Wrapped error: %v\n\n", err3)

	// Printing with stack trace
	fmt.Printf("Error with stack trace:\n%+v\n", err3)
}

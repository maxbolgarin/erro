package main

import (
	"fmt"
	"github.com/user/repo/erro"
)

// Define a custom error template
var ErrProductNotFound = erro.NewTemplate("product with id %d not found", erro.ClassNotFound)

func main() {
	err := ErrProductNotFound.New(999)
	fmt.Printf("Template error: %v\n", err)

	// You can also wrap an existing error with the template
	originalErr := erro.New("database timeout")
	wrappedErr := erro.DatabaseError.Wrap(originalErr, "could not fetch product")
	fmt.Printf("Wrapped template error: %v\n", wrappedErr)
}

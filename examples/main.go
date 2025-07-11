package main

import (
	"fmt"

	"github.com/maxbolgarin/erro"
)

func main() {
	err := erro.New("test", "test", "test").Fields("test23", "test12")
	wrapped := erro.Wrap(err, "wrapped", "wrap_field")
	wrapped2 := erro.Wrap(wrapped, "wrapped2", "wrap_field2")
	// fmt.Println(err.Error())
	// fmt.Println(wrapped2.Error())
	// fmt.Println(wrapped2.ErrorWithStack())

	fmt.Println(erro.LogFields(err))
	fmt.Println(erro.LogFields(wrapped))
	fmt.Println(erro.LogError(wrapped2))

	fmt.Println(err.StackFormat())

	fmt.Printf("Package: %s\n", erro.ExtractContext(err).Package)
	fmt.Printf("Function: %s\n", erro.ExtractContext(err).Function)
	fmt.Printf("File: %s\n", erro.ExtractContext(err).File)
	fmt.Printf("Line: %d\n", erro.ExtractContext(err).Line)
	fmt.Printf("Fields: %v\n", erro.ExtractContext(err).Fields)
	fmt.Printf("Code: %s\n", erro.ExtractContext(err).Code)
	fmt.Printf("Category: %s\n", erro.ExtractContext(err).Category)
	fmt.Printf("Severity: %s\n", erro.ExtractContext(err).Severity)
}

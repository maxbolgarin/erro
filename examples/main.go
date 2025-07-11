package main

import (
	"fmt"

	"github.com/maxbolgarin/erro"
)

func main() {
	err := erro.New("test", "test", "test").Fields("test23", "test12")
	wrapped := erro.Wrap(err, "wrapped", "wrap_field")
	wrapped2 := erro.Wrap(wrapped, "wrapped2", "wrap_field2").Code("wrapped2_code").Category("wrapped2_category").Severity("wrapped2_severity").Retryable(true).Tags("wrapped2_tag")
	// fmt.Println(err.Error())
	// fmt.Println(wrapped2.Error())
	// fmt.Println(wrapped2.ErrorWithStack())
	erro.LogError(wrapped2, func(message string, fields ...any) {
		fmt.Println(message)
		fmt.Println(fields)
	})

	fmt.Println(wrapped2.StackFormat())

	ctx := erro.ExtractContext(wrapped2)
	fmt.Printf("Package: %s\n", ctx.Package)
	fmt.Printf("Function: %s\n", ctx.Function)
	fmt.Printf("File: %s\n", ctx.File)
	fmt.Printf("Line: %d\n", ctx.Line)
	fmt.Printf("Fields: %v\n", ctx.Fields)
	fmt.Printf("Code: %s\n", ctx.Code)
	fmt.Printf("Category: %s\n", ctx.Category)
	fmt.Printf("Severity: %s\n", ctx.Severity)
}

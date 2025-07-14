//go:build go1.21

package main

import (
	"errors"
	"fmt"

	"github.com/maxbolgarin/erro"
)

func test3() erro.Error {
	return erro.New("test3", "f", "v").WithFields("f3", "v3").WithSeverity(erro.SeverityCritical)
}

func test2() erro.Error {
	return test3()
}

func test() erro.Error {
	return erro.Wrap(test2(), "wrapped", "wrapped_field").WithFields("f3", "v3").WithSeverity(erro.SeverityInfo)
}

func main() {

	//erro.SetGlobalFormatter(erro.GetFormatErrorWithFullContextBase(erro.WithStackFormat(erro.StackFormatList)))

	// err := test().WithCategory(erro.CategoryDatabase)
	// fmt.Printf("%+v\n", err)
	// err2 := erro.NewLight("test2").WithClass(erro.ClassValidation).WithCategory(erro.CategoryAPI).WithSeverity(erro.SeverityHigh)
	// erro.LogError(err2, func(message string, fields ...any) {
	// 	slog.Info(message, fields...)
	// }, erro.WithStackFormat(erro.StackFormatList))

	// erro.SetGlobalFormatter(erro.FormatErrorSimple)

	// errLight := erro.NewLight("test", "f", "v").WithFields("f2", "v2").WithCategory(erro.CategoryDatabase).WithClass(erro.ClassValidation)
	// fmt.Println(errLight)
	// for i := 0; i < 10; i++ {
	// 	errLight = erro.WrapLight(errLight, "wrapped", "wrapped_field").WithSeverity(erro.SeverityCritical)
	// }
	// fmt.Println(errLight)

	errBase := errors.New("test")
	err1 := erro.Wrap(errBase, "wrapped", "wrapped_field").WithCategory(erro.CategoryDatabase).WithClass(erro.ClassValidation).WithRetryable(true)
	fmt.Println(erro.Wrap(err1, "wrapped2", "wrapped_field2").WithCategory(erro.CategoryAPI).WithClass(erro.ClassValidation).WithRetryable(true))

	// fmt.Printf("%+v\n", erro.NewBuilderWithError(err, "test", "f", "v").
	// 	WithCategory(erro.CategoryDatabase).
	// 	WithClass(erro.ClassValidation).
	// 	WithSeverity(erro.SeverityHigh).
	// 	WithID("ID_123").
	// 	WithRetryable(true).
	// 	WithSpan(nil).
	// 	Build())

	// fmt.Printf("%+v\n", erro.APIError.New("test", "f", "v"))
}

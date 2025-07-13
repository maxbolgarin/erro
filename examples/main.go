//go:build go1.21

package main

import (
	"fmt"
	"log/slog"

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
	//err := test().Category(erro.CategoryDatabase)
	err2 := erro.NewLight("test2").WithClass(erro.ClassValidation).WithCategory(erro.CategoryAPI).WithSeverity(erro.SeverityHigh)
	erro.LogError(err2, func(message string, fields ...any) {
		slog.Info(message, fields...)
	}, erro.WithStackFormat(erro.StackFormatList))

	fmt.Printf("%+v\n", erro.NewLight("test").WithFields("f", "v"))
}

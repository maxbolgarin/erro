//go:build go1.21

package main

import (
	"log/slog"

	"github.com/maxbolgarin/erro"
)

func test3() erro.Error {
	return erro.New("test3", "f", "v").Fields("f3", "v3").Severity(erro.Critical)
}

func test2() erro.Error {
	return test3()
}

func test() erro.Error {
	return erro.Wrap(test2(), "wrapped", "wrapped_field").Fields("f3", "v3").Severity(erro.Info)
}

func main() {
	err := test()
	erro.LogError(err, func(message string, fields ...any) {
		slog.Info(message, fields...)
	}, erro.WithStackFormat(erro.StackFormatList))
}

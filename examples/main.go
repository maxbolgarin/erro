//go:build go1.21

package main

import (
	"log/slog"

	"github.com/maxbolgarin/erro"
)

func main() {
	err := erro.New("test", "test", "test").Fields("test23", "test12")
	wrapped := erro.Wrap(err, "wrapped", "wrap_field")
	//wrapped2 := erro.Wrap(wrapped, "wrapped2", "wrap_field2").Code("wrapped2_code").Category("wrapped2_category").Severity("wrapped2_severity").Retryable(true).Tags("wrapped2_tag")
	// fmt.Println(err.Error())
	// fmt.Println(wrapped2.Error())
	// fmt.Println(wrapped2.ErrorWithStack())
	erro.LogError(wrapped, func(message string, fields ...any) {
		slog.Info(message, fields...)
	}, erro.MinimalLogOpts()...)

}

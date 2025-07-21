//go:build go1.21

package main

import (
	"log/slog"
	"os"

	"github.com/maxbolgarin/erro"
)

func main() {
	// Using slog for structured logging
	handler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(handler)

	err := erro.New("payment failed",
		"order_id", 456,
		"customer_id", 789,
		erro.CategoryPayment,
		erro.SeverityCritical,
	)
	err = erro.Wrap(err, "wrapped error", "wrap_field", "wrap_value")

	// got fields in log
	logger.Error("got fields in log", erro.LogFields(err)...)

	// send error in log (that is slog behaviour with errors)
	logger.Error("got JSON in log", "error", err)

	// got error in log with fields
	logger.Error("got error message with fields", "error", err.Error())
}

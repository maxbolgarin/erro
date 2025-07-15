package main

import (
	"github.com/user/repo/erro"
	"log/slog"
	"os"
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

	logger.Error("a critical payment error occurred", erro.LogFields(err)...)
}

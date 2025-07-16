package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/maxbolgarin/erro"
)

func handleRequest(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		err := erro.New("user_id is required", erro.ClassValidation)
		http.Error(w, err.Error(), erro.HTTPCode(err))
		return
	}

	if userID == "123" {
		err := erro.New("user is not authorized", erro.ClassPermissionDenied)
		http.Error(w, err.Error(), erro.HTTPCode(err))
		return
	}

	fmt.Fprintln(w, "Success!")
}

func main() {
	http.HandleFunc("/", handleRequest)
	fmt.Println("Starting server on :8080")
	fmt.Println("Try visiting:")
	fmt.Println("  http://localhost:8080")
	fmt.Println("  http://localhost:8080?user_id=1")
	fmt.Println("  http://localhost:8080?user_id=123")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

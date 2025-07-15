package main

import (
	"fmt"
	"github.com/user/repo/erro"
)

func main() {
	// --- Error List ---
	fmt.Println("--- Error List ---")
	list := erro.NewList()
	list.Add(erro.New("first error"))
	list.Add(erro.New("second error"))
	list.Add(erro.New("third error"))

	if err := list.Err(); err != nil {
		fmt.Printf("Collected errors:\n%v\n\n", err)
	}

	// --- Error Set (for deduplication) ---
	fmt.Println("--- Error Set ---")
	set := erro.NewSet()
	set.Add(erro.New("this error is unique"))
	set.Add(erro.New("this error will appear twice"))
	set.Add(erro.New("this error will appear twice"))

	if err := set.Err(); err != nil {
		fmt.Printf("Unique errors:\n%v\n", err)
	}
}

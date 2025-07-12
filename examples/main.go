//go:build go1.21

package main

import (
	"fmt"

	"github.com/maxbolgarin/erro"
)

func test2() error {
	return erro.New("test2", "f", "v").Fields("f2", "v2").Severity(erro.Critical)
}

func test() error {
	return erro.Wrap(test2(), "wrapped", "wrapped_field").Fields("f3", "v3").Severity(erro.Critical)
}

func main() {
	err := test()
	fmt.Printf("%+v\n", err)
}

package main

import (
	"fmt"

	"github.com/maxbolgarin/erro"
)

func main() {
	err := erro.New("test", "test", "test").Fields("test23", "test12")
	wrapped := erro.Wrap(err, "wrapped", "wrap_field")
	wrapped2 := erro.Wrap(wrapped, "wrapped2", "wrap_field2")
	// fmt.Println(err.Error())
	// fmt.Println(wrapped2.Error())
	// fmt.Println(wrapped2.ErrorWithStack())

	fmt.Println(erro.LogFields(err))
	fmt.Println(erro.LogFields(wrapped))
	fmt.Println(erro.LogError(wrapped2))

}

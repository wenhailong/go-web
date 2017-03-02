package main

import (
	"errors"
	"fmt"
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func NewError(format string, a ...interface{}) error {
	s := fmt.Sprintf(format, a)
	return errors.New(s)
}

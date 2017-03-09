package main

import (
	"fmt"
)

const (
	ERROR_INTERNAL          = 0x10000001
	ERROR_DB_OPERATE_FAIELD = 0x10000002
	ERROR_URL_PARAM_INVALID = 0x10000003
	ERROR_NO_BUYER          = 0x10000004
	ERROR_USER_NOT_FOUND    = 0x10000005
)

type FollowerError struct {
	Code int    `json:"code"`
	Msg  string `json:"errMsg"`
}

func (p FollowerError) Error() string {
	return p.Msg
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

func NewError(code int, format string, a ...interface{}) error {
	var s string
	if len(a) != 0 {
		s = fmt.Sprintf(format, a)
	} else {
		s = format
	}

	return FollowerError{code, s}
}

package errortypeassertion

import (
	"errors"
	"fmt"
	"os"
)

func GoodErrorsAs(err error) {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		fmt.Println(pathErr.Path)
	}
}

func GoodInterfaceAssertion(err error) {
	_, _ = err.(interface{ Timeout() bool })
}

func GoodTypeSwitch(err error) {
	switch e := err.(type) {
	case interface{ Temporary() bool }:
		fmt.Println(e.Temporary())
	}
}

func BadTypeSwitch(err error) {
	switch e := err.(type) {
	case *os.PathError: // want `type assertion on error to \*os\.PathError bypasses wrapped errors; use errors\.As instead`
		fmt.Println(e.Path)
	case interface{ Temporary() bool }:
		fmt.Println(e.Temporary())
	}
}

func BadSingleValue(err error) {
	_ = err.(*os.PathError) // want `type assertion on error to \*os\.PathError bypasses wrapped errors; use errors\.As instead`
}

func BadTwoValue(err error) {
	if pathErr, ok := err.(*os.PathError); ok { // want `type assertion on error to \*os\.PathError bypasses wrapped errors; use errors\.As instead`
		fmt.Println(pathErr.Path)
	}
}

type errorAlias = error

func BadAlias(err errorAlias) {
	_ = err.(*os.PathError) // want `type assertion on error to \*os\.PathError bypasses wrapped errors; use errors\.As instead`
}

func SuppressedPreviousLine(err error) {
	//nolint:errortypeassertion
	_ = err.(*os.PathError)
}

func SuppressedSameLine(err error) {
	_ = err.(*os.PathError) //nolint:errortypeassertion
}

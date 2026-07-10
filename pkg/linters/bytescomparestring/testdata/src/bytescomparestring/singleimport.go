package bytescomparestring

import "fmt"

func singleImportEqual(a, b []byte) bool {
	fmt.Println("compare")
	return string(a) == string(b) // want `string\(a\) == string\(b\) is a \[\]byte comparison written the long way; use bytes\.Equal\(a, b\) for clearer intent`
}

func singleImportNotEqual(a, b []byte) bool {
	fmt.Println("compare")
	return string(a) != string(b) // want `string\(a\) != string\(b\) is a \[\]byte comparison written the long way; use !bytes\.Equal\(a, b\) for clearer intent`
}

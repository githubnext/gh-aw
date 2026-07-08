package bytescomparestring

func noImportEqual(a, b []byte) bool {
	return string(a) == string(b) // want `string\(a\) == string\(b\) allocates; use bytes\.Equal\(a, b\) instead`
}

func noImportNotEqual(a, b []byte) bool {
	return string(a) != string(b) // want `string\(a\) != string\(b\) allocates; use !bytes\.Equal\(a, b\) instead`
}

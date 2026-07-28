package bytescomparezero

import bx "bytes"

func badAliasedEqual(a, b []byte) bool {
	return bx.Compare(a, b) == 0 // want `use bytes\.Equal\(a, b\) instead of bytes\.Compare comparison with 0`
}

func badAliasedNotEqual(a, b []byte) bool {
	return bx.Compare(a, b) != 0 // want `use !bytes\.Equal\(a, b\) instead of bytes\.Compare comparison with 0`
}

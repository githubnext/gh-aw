package lenstringzero

func isEmpty(s string) bool {
return len(s) == 0 // want `use s == "" to check for empty string instead of len\(s\) == 0`
}

func isNotEmpty(s string) bool {
return len(s) != 0 // want `use s != "" to check for non-empty string instead of len\(s\) != 0`
}

func flippedEmpty(s string) bool {
return 0 == len(s) // want `use s == "" to check for empty string instead of len\(s\) == 0`
}

func flippedNotEmpty(s string) bool {
return 0 != len(s) // want `use s != "" to check for non-empty string instead of len\(s\) != 0`
}

func alreadyGoodEmpty(s string) bool {
return s == ""
}

func alreadyGoodNotEmpty(s string) bool {
return s != ""
}

func sliceNotFlagged(s []byte) bool {
return len(s) == 0
}

func arrayNotFlagged(s [1]byte) bool {
return len(s) != 0
}

func lenNotZeroOp(s string) bool {
return len(s) > 0
}

func lenNotComparedToZero(s string) bool {
return len(s) == 1
}

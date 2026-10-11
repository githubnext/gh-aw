package a

import r "reflect"

func testImportAlias(a, b interface{}) bool {
	equal := r.DeepEqual
	return equal(a, b) // want `reflect.DeepEqual\(\) is inefficient`
}

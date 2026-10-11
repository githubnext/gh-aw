package a

import . "reflect"

func testDotImport(a, b interface{}) bool {
	equal := DeepEqual
	return equal(a, b) || DeepEqual(a, b) // want `reflect.DeepEqual\(\) is inefficient` `reflect.DeepEqual\(\) is inefficient`
}

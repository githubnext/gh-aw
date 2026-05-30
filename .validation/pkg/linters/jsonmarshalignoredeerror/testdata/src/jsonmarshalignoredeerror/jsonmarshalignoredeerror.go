package jsonmarshalignoredeerror

import "encoding/json"

type Foo struct{ X int }

func Bad() {
	f := Foo{X: 1}
	val, _ := json.Marshal(f) // want `error return from json\.Marshal is discarded`
	_ = val

	var f2 Foo
	_ = json.Unmarshal([]byte(`{}`), &f2) // want `error return from json\.Unmarshal is discarded`
}

func Good() error {
	f := Foo{X: 1}
	val, err := json.Marshal(f)
	if err != nil {
		return err
	}
	_ = val

	var f2 Foo
	if err := json.Unmarshal([]byte(`{}`), &f2); err != nil {
		return err
	}
	return nil
}

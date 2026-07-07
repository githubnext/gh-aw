package writebytestring

import (
	"bytes"
	"os"
)

type customWriter struct{}

func (c *customWriter) Write(p []byte) (int, error) { return len(p), nil }

func bad() {
	var buf bytes.Buffer
	s := "hello"
	buf.Write([]byte(s)) // want `buf\.Write\(\[\]byte\(s\)\) can be replaced with io\.WriteString\(buf, s\) to avoid a \[\]byte allocation`

	buf.Write([]byte("world")) // want `buf\.Write\(\[\]byte\("world"\)\) can be replaced with io\.WriteString\(buf, "world"\) to avoid a \[\]byte allocation`
}

type myString string

func badNamedString() {
	var buf bytes.Buffer
	s := myString("hello")
	buf.Write([]byte(s)) // want `buf\.Write\(\[\]byte\(s\)\) can be replaced with io\.WriteString\(buf, s\) to avoid a \[\]byte allocation`
}

func badFile() {
	f, err := os.Create("/tmp/test.txt")
	if err != nil {
		return
	}
	defer f.Close()
	msg := "hello"
	f.Write([]byte(msg)) // want `f\.Write\(\[\]byte\(msg\)\) can be replaced with io\.WriteString\(f, msg\) to avoid a \[\]byte allocation`
}

func badCustomWriter() {
	w := &customWriter{}
	s := "hello"
	w.Write([]byte(s)) // want `w\.Write\(\[\]byte\(s\)\) can be replaced with io\.WriteString\(w, s\) to avoid a \[\]byte allocation`
}

func goodBytes() {
	var buf bytes.Buffer
	b := []byte("hello")
	buf.Write(b) // already []byte — no conversion
}

func goodWriteString() {
	// Using io.WriteString is the idiomatic form.
	var buf bytes.Buffer
	s := "hello"
	buf.WriteString(s)
}

func goodNotString() {
	var buf bytes.Buffer
	n := 42
	buf.Write([]byte{byte(n)})
}

func suppressed() {
	var buf bytes.Buffer
	s := "hello"
	//nolint:writebytestring
	buf.Write([]byte(s))
}

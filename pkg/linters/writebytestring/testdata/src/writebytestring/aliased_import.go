package writebytestring

import (
	"bytes"
	ioutil "io"
)

// badAliasedIO has "io" imported as "ioutil"; the fix must emit
// ioutil.WriteString, not io.WriteString.
func badAliasedIO(w ioutil.Writer) {
	var buf bytes.Buffer
	s := "hello"
	buf.Write([]byte(s)) // want `buf\.Write\(\[\]byte\(s\)\) can be replaced with io\.WriteString\(&buf, s\) to potentially avoid a \[\]byte allocation if the writer implements io\.StringWriter`
	_ = w
}

package writebytestring

import "bytes"

func needsImport(s string) {
	var buf bytes.Buffer
	buf.Write([]byte(s))       // want `buf\.Write\(\[\]byte\(s\)\) can be replaced with io\.WriteString\(&buf, s\) to potentially avoid a \[\]byte allocation if the writer implements io\.StringWriter`
	buf.Write([]byte("hello")) // want `buf\.Write\(\[\]byte\("hello"\)\) can be replaced with io\.WriteString\(&buf, "hello"\) to potentially avoid a \[\]byte allocation if the writer implements io\.StringWriter`
}

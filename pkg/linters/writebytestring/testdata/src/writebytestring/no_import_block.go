package writebytestring

func noImportsNeedsImport(s string) {
	w := &customWriter{}
	w.Write([]byte(s)) // want `w\.Write\(\[\]byte\(s\)\) can be replaced with io\.WriteString\(w, s\) to potentially avoid a \[\]byte allocation if the writer implements io\.StringWriter`
}

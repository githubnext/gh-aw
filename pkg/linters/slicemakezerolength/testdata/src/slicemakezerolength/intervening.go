package slicemakezerolength

func unrelatedStatement(items []string) []string {
	s := make([]string, 0) // want `before this range loop can use capacity len\(items\)`
	unrelated := 1
	_ = unrelated
	for _, item := range items {
		s = append(s, item)
	}
	return s
}

func unrelatedDeclaration(items []string) []string {
	var s = make([]string, 0) // want `before this range loop can use capacity len\(items\)`
	println("collecting")
	for _, item := range items {
		s = append(s, item)
	}
	return s
}

func targetMutated(items []string) []string {
	s := make([]string, 0)
	s = append(s, "prefix")
	for _, item := range items {
		s = append(s, item)
	}
	return s
}

func inputMutated(items []string) []string {
	s := make([]string, 0)
	items = append(items, "extra")
	for _, item := range items {
		s = append(s, item)
	}
	return s
}

func inputDeclaredLater() []string {
	s := make([]string, 0)
	items := []string{"item"}
	for _, item := range items {
		s = append(s, item)
	}
	return s
}

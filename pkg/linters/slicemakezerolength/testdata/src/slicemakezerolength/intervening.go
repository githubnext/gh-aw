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

func unrelatedRange(items []string, other []int) []string {
	s := make([]string, 0) // want `before this range loop can use capacity len\(items\)`
	for _, n := range other {
		println(n)
	}
	for _, item := range items {
		s = append(s, item)
	}
	return s
}

func unrelatedUnknownSizeRange(items []string, other <-chan int) []string {
	s := make([]string, 0) // want `before this range loop can use capacity len\(items\)`
	for n := range other {
		println(n)
	}
	for _, item := range items {
		s = append(s, item)
	}
	return s
}

func inputUsedInEarlierRange(items []string) []string {
	s := make([]string, 0)
	for _, item := range items {
		println(item)
	}
	for _, item := range items {
		s = append(s, item)
	}
	return s
}

func guardedPointer(p *[]int, skip bool) []int {
	s := make([]int, 0)
	if skip {
		return nil
	}
	for _, v := range *p {
		s = append(s, v)
	}
	return s
}

func branchBeforePointer(p *[]int, skip bool) []int {
	s := make([]int, 0)
	if skip {
		goto done
	}
	for _, v := range *p {
		s = append(s, v)
	}
done:
	return s
}

func earlyReturnInUnrelatedRange(p *[]int, other []bool) []int {
	s := make([]int, 0)
	for _, skip := range other {
		if skip {
			return nil
		}
	}
	for _, v := range *p {
		s = append(s, v)
	}
	return s
}

func unrelatedClosure(items []string) []string {
	s := make([]string, 0) // want `before this range loop can use capacity len\(items\)`
	noop := func() { return }
	noop()
	for _, item := range items {
		s = append(s, item)
	}
	return s
}

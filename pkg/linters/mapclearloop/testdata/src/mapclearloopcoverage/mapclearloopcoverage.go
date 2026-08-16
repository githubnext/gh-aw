package mapclearloopcoverage

func coldMap() {
	m := map[string]int{}
	for k := range m {
		delete(m, k)
	}
}

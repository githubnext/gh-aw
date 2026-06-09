package sortslice

type customSort struct{}

func (customSort) Slice(_ []string, _ func(i, j int) bool)       {}
func (customSort) SliceStable(_ []string, _ func(i, j int) bool) {}

func GoodShadowedSortIdentifier(items []string) {
	sort := customSort{}
	sort.Slice(items, func(i, j int) bool { return items[i] < items[j] })
	sort.SliceStable(items, func(i, j int) bool { return items[i] < items[j] })
}

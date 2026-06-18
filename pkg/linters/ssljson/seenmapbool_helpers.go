package ssljson

func hasStringKey(set map[string]struct{}, key string) bool {
	_, ok := set[key]
	return ok
}

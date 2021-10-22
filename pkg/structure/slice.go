package structure

// ContainsStrings searches a slice of strings for a case-sensitive match
func ContainsStrings(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

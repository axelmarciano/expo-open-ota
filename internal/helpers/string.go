package helpers

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func StringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

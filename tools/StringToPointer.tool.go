package tools

func StringP(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

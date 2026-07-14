package rgw

func boolFloat(value bool) float64 {
	if value {
		return 1
	}

	return 0
}

package apipkg

// combineStringsAsY combines two string slices that both derive from string and
// return the combined form as type []Y.
func combineStringsAsY[X ~string, Y ~string](xx []X, yy []Y) []Y {
	xxAsY := make([]Y, len(xx))
	for i, x := range xx {
		xxAsY[i] = Y(x)
	}
	return append(yy, xxAsY...)
}

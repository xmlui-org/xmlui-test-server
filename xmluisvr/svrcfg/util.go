package cfgldr

func cloneSlice[V any](in []V) (out []V) {
	out = make([]V, len(in))
	copy(out, in)
	return out
}

func cloneMap[K comparable, V any](src map[K]V) map[K]V {
	dst := make(map[K]V, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

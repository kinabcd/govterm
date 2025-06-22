package govterm

func Pop[T int | Savepoint](slice *[]T) (T, bool) {
	s := *slice
	if len(s) == 0 {
		var zeroVal T
		return zeroVal, false
	}
	val := s[len(s)-1]
	*slice = s[:len(s)-1]
	return val, true
}

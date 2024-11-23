package capigraph

type Int16Set map[int16]any

func (s *Int16Set) add(n int16) {
	(*s)[n] = struct{}{}
}

func (s *Int16Set) del(n int16) {
	delete(*s, n)
}

func stringSliceToSet(intSlice []int16) *Int16Set {
	s := Int16Set{}
	for _, n := range intSlice {
		s[n] = struct{}{}
	}
	return &s
}

func (s *Int16Set) subtract(setToSubtract *Int16Set) {
	for n := range *setToSubtract {
		delete(*s, n)
	}
}

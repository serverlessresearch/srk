package cfbench

type void struct{}
var member void

type stringSet map[string]void

func (s *stringSet) add(key string) {
	(*s)[key] = member
}

func (s *stringSet) remove(key string) {
	delete(*s, key)
}

func (s *stringSet) contains(key string) bool {
	_, exists := (*s)[key]
	return exists
}

func (s *stringSet) size() int {
	return len(*s)
}

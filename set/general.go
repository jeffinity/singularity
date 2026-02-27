package set

type set[T comparable] struct {
	hash map[T]nothing
}

// New Create a new set
func New[T comparable](initial ...T) Set[T] {
	s := &set[T]{make(map[T]nothing)}

	for _, v := range initial {
		s.Insert(v)
	}
	return s
}

func (s *set[T]) Difference(other Set[T]) Set[T] {
	n := make(map[T]nothing)

	for k := range s.hash {
		if !other.Has(k) {
			n[k] = nothing{}
		}
	}
	return &set[T]{hash: n}
}

func (s *set[T]) All() []T {
	if s == nil || len(s.hash) == 0 {
		return []T{}
	}

	all := make([]T, 0, len(s.hash))
	for i := range s.hash {
		all = append(all, i)
	}
	return all
}

func (s *set[T]) Do(f func(T)) {
	for k := range s.hash {
		f(k)
	}
}

func (s *set[T]) DoE(f func(T) error) error {
	for k := range s.hash {
		err := f(k)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *set[T]) Has(element T) bool {
	_, exists := s.hash[element]
	return exists
}

func (s *set[T]) Insert(elements ...T) {
	for _, e := range elements {
		s.hash[e] = nothing{}
	}
}

func (s *set[T]) Intersection(other Set[T]) Set[T] {
	n := make(map[T]nothing)

	for k := range s.hash {
		if other.Has(k) {
			n[k] = nothing{}
		}
	}
	return &set[T]{n}
}

func (s *set[T]) Len() int {
	return len(s.hash)
}

func (s *set[T]) ProperSubsetOf(other Set[T]) bool {
	return s.SubsetOf(other) && s.Len() < other.Len()
}

// Remove an element from the set
func (s *set[T]) Remove(element T) {
	delete(s.hash, element)
}

// SubsetOf Test whether this set is a subset of "set"
func (s *set[T]) SubsetOf(other Set[T]) bool {
	if s.Len() > other.Len() {
		return false
	}
	for k := range s.hash {
		if !other.Has(k) {
			return false
		}
	}
	return true
}

// Union Find the union of two sets
func (s *set[T]) Union(other Set[T]) Set[T] {
	n := make(map[T]nothing)

	for k := range s.hash {
		n[k] = nothing{}
	}

	for _, k := range other.All() {
		n[k] = nothing{}
	}

	return &set[T]{hash: n}
}

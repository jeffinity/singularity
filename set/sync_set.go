package set

import "sync"

//https://github.com/ImSingee/go-ex/blob/master/aset/set.go

type (
	SyncSet[T comparable] struct {
		hash map[T]nothing
		lock sync.RWMutex
	}
)

// NewSyncSet Create a new synchronized set
func NewSyncSet[T comparable](initial ...T) Set[T] {
	s := &SyncSet[T]{hash: make(map[T]nothing)}

	for _, v := range initial {
		s.Insert(v)
	}
	return s
}

// Difference Find the difference between two sets
// 返回的是自己存在而传入的 set 不存在的元素集合
func (s *SyncSet[T]) Difference(set Set[T]) Set[T] {

	s.lock.RLock()
	defer s.lock.RUnlock()

	n := make(map[T]nothing)
	for k := range s.hash {
		if !set.Has(k) {
			n[k] = nothing{}
		}
	}
	return &SyncSet[T]{hash: n}
}

func (s *SyncSet[T]) All() []T {

	if s == nil {
		return []T{}
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	if len(s.hash) == 0 {
		return []T{}
	}

	all := make([]T, 0, len(s.hash))
	for i := range s.hash {
		all = append(all, i)
	}
	return all
}

// Do Call f for each item in the set
func (s *SyncSet[T]) Do(f func(T)) {

	s.lock.RLock()
	defer s.lock.RUnlock()

	for k := range s.hash {
		f(k)
	}
}

// DoE Call f for each item in the set
func (s *SyncSet[T]) DoE(f func(T) error) error {

	s.lock.RLock()
	defer s.lock.RUnlock()

	for k := range s.hash {
		err := f(k)
		if err != nil {
			return err
		}
	}

	return nil
}

// Has Test to see whether the element is in the set
func (s *SyncSet[T]) Has(element T) bool {

	s.lock.RLock()
	defer s.lock.RUnlock()

	_, exists := s.hash[element]
	return exists
}

// Insert Add element(s) to the set
func (s *SyncSet[T]) Insert(elements ...T) {

	s.lock.Lock()
	defer s.lock.Unlock()

	for _, e := range elements {
		s.hash[e] = nothing{}
	}
}

// Intersection Find the intersection of two sets
func (s *SyncSet[T]) Intersection(set Set[T]) Set[T] {

	s.lock.RLock()
	defer s.lock.RUnlock()

	n := make(map[T]nothing)

	for k := range s.hash {
		if set.Has(k) {
			n[k] = nothing{}
		}
	}
	return &SyncSet[T]{hash: n}
}

// Len Return the number of items in the set
func (s *SyncSet[T]) Len() int {

	s.lock.RLock()
	defer s.lock.RUnlock()

	return len(s.hash)
}

// ProperSubsetOf Test whether this set is a proper subset of "set"
func (s *SyncSet[T]) ProperSubsetOf(set Set[T]) bool {

	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.SubsetOf(set) && s.Len() < set.Len()
}

// Remove an element from the set
func (s *SyncSet[T]) Remove(element T) {

	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.hash, element)
}

// SubsetOf Test whether this set is a subset of "set"
func (s *SyncSet[T]) SubsetOf(set Set[T]) bool {

	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.Len() > set.Len() {
		return false
	}
	for k := range s.hash {
		if !set.Has(k) {
			return false
		}
	}
	return true
}

// Union Find the union of two sets
func (s *SyncSet[T]) Union(set Set[T]) Set[T] {

	s.lock.RLock()
	defer s.lock.RUnlock()

	n := make(map[T]nothing)

	for k := range s.hash {
		n[k] = nothing{}
	}
	for _, k := range set.All() {
		n[k] = nothing{}
	}

	return &SyncSet[T]{hash: n}
}

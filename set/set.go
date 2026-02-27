package set

//https://github.com/ImSingee/go-ex/blob/master/aset/set.go

type nothing struct{}

type Set[T comparable] interface {

	// Len Return the number of items in the set
	Len() int

	// Has Test to see whether the element is in the set
	Has(element T) bool

	// Insert Add element(s) to the set
	Insert(elements ...T)

	// Remove an element from the set
	Remove(element T)

	// Difference Find the difference between two sets
	// 返回的是自己存在而传入的 set 不存在的元素集合
	Difference(set Set[T]) Set[T]

	All() []T

	// Do Call f for each item in the set
	Do(f func(T))

	// DoE Call f for each item in the set
	DoE(f func(T) error) error

	// Intersection Find the intersection of two sets
	Intersection(set Set[T]) Set[T]

	// ProperSubsetOf Test whether this set is a proper subset of "set"
	ProperSubsetOf(set Set[T]) bool

	// SubsetOf Test whether this set is a subset of "set"
	SubsetOf(set Set[T]) bool

	// Union Find the union of two sets
	Union(set Set[T]) Set[T]
}

// Intersection 返回若干个 set 的交集
func Intersection[T comparable](sets ...Set[T]) Set[T] {
	if len(sets) == 0 {
		return New[T]()
	}

	b := sets[0]
	for _, s := range sets[1:] {
		b = b.Intersection(s)
	}
	return b
}

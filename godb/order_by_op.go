package godb

import (
	"fmt"
	"sort"
)

// TODO: some code goes here
type OrderBy struct {
	orderBy []Expr // OrderBy should include these two fields (used by parser)
	child   Operator
	//add additional fields here
	ascendings []bool
}

// Order by constructor -- should save the list of field, child, and ascending
// values for use in the Iterator() method. Here, orderByFields is a list of
// expressions that can be extacted from the child operator's tuples, and the
// ascending bitmap indicates whether the ith field in the orderByFields
// list should be in ascending (true) or descending (false) order.
func NewOrderBy(orderByFields []Expr, child Operator, ascending []bool) (*OrderBy, error) {
	if len(ascending) != len(orderByFields) {
		return nil, fmt.Errorf("list ascending should be equal list orderByFields")
	}
	return &OrderBy{
		orderBy:    orderByFields,
		child:      child,
		ascendings: ascending,
	}, nil
}

func (o *OrderBy) Descriptor() *TupleDesc {
	// TODO should we need use copy() method
	return o.child.Descriptor().copy()
}

// Return a function that iterators through the results of the child iterator in
// ascending/descending order, as specified in the construtor.  This sort is
// "blocking" -- it should first construct an in-memory sorted list of results
// to return, and then iterate through them one by one on each subsequent
// invocation of the iterator function.
//
// Although you are free to implement your own sorting logic, you may wish to
// leverage the go sort pacakge and the [sort.Sort] method for this purpose.  To
// use this you will need to implement three methods:  Len, Swap, and Less that
// the sort algorithm will invoke to preduce a sorted list. See the first
// example, example of SortMultiKeys, and documentation at: https://pkg.go.dev/sort
func (o *OrderBy) Iterator(tid TransactionID) (func() (*Tuple, error), error) {
	// get all tuple of child, sort here, in function, just return tuple from mem.
	tuples := make([]*Tuple, 0) // []*Tuple or []Tuple ???
	// use []*Tuple -> tuplesPointer -> change value when the first one change
	// use []Tuple -> tuplesCopy -> don't change value if the first one change
	iter, _ := o.child.Iterator(tid)
	tuple, _ := iter()
	for tuple != nil {
		tuples = append(tuples, tuple)
		tuple, _ = iter()
	}

	// create lessFunc
	lessFuncs := make([]lessFunc, 0)

	// OrderedLessThan    0
	// OrderedEqual       1
	// OrderedGreaterThan 2

	// eg in library
	// increasingLines := func(c1, c2 *Change) bool {
	// 	return c1.lines < c2.lines => ascending
	// }
	// decreasingLines := func(c1, c2 *Change) bool {
	// 	return c1.lines > c2.lines // Note: > orders downwards.
	// }
	for idx, expr := range o.orderBy {
		// I fail here many times, because I don't know it need temp
		tempIdx := idx
		tempExpr := expr
		lessFuncs = append(lessFuncs, func(p1, p2 *Tuple) bool {
			order, _ := p1.compareField(p2, tempExpr)
			if o.ascendings[tempIdx] {
				return order < OrderedEqual
			}
			// we want descending
			// if p1 > p2 -> return true
			// if p1 <= p2 -> return false
			return order > OrderedEqual
		})
	}

	OrderedBy(lessFuncs...).Sort(tuples)

	counter := 0

	return func() (*Tuple, error) {
		// eg counter = 0, len(tuples) = 1 -> serve for 0
		// eg counter = 1, len(tuples) = 1 -> error
		if counter < len(tuples) {
			counter += 1
			return tuples[counter-1], nil
		}

		return nil, nil
	}, nil //replace me
}

type lessFunc func(p1, p2 *Tuple) bool

// multiSorter implements the Sort interface, sorting the changes within.
type multiSorter struct {
	tuples []*Tuple
	less   []lessFunc
}

func (ms *multiSorter) Sort(tuples []*Tuple) {
	ms.tuples = tuples
	sort.Sort(ms)
}

// OrderedBy returns a Sorter that sorts using the less functions, in order.
// Call its Sort method to sort the data.
func OrderedBy(less ...lessFunc) *multiSorter {
	return &multiSorter{
		less: less,
	}
}

// Len is part of sort.Interface.
func (ms *multiSorter) Len() int {
	return len(ms.tuples)
}

// Swap is part of sort.Interface.
func (ms *multiSorter) Swap(i, j int) {
	ms.tuples[i], ms.tuples[j] = ms.tuples[j], ms.tuples[i]
}

// Less is part of sort.Interface. It is implemented by looping along the
// less functions until it finds a comparison that discriminates between
// the two items (one is less than the other). Note that it can call the
// less functions twice per call. We could change the functions to return
// -1, 0, 1 and reduce the number of calls for greater efficiency: an
// exercise for the reader.
func (ms *multiSorter) Less(i, j int) bool {
	p, q := ms.tuples[i], ms.tuples[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		switch {
		case less(p, q):
			// p < q, so we have a decision.
			return true
		case less(q, p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	return ms.less[k](p, q)
}

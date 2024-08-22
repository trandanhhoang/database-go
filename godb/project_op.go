package godb

import "fmt"

type Project struct {
	selectFields []Expr // required fields for parser
	outputNames  []string
	child        Operator
	//add additional fields here
	isDistinct bool
}

// Project constructor -- should save the list of selected field, child, and the child op.
// Here, selectFields is a list of expressions that represents the fields to be selected,
// outputNames are names by which the selected fields are named (should be same length as
// selectFields; throws error if not), distinct is for noting whether the projection reports
// only distinct results, and child is the child operator.
func NewProjectOp(selectFields []Expr, outputNames []string, distinct bool, child Operator) (Operator, error) {
	if len(outputNames) != len(selectFields) {
		return nil, fmt.Errorf("outputNames should be same length as selected field")
	}

	return &Project{
		selectFields: selectFields,
		outputNames:  outputNames,
		child:        child,
		isDistinct:   distinct,
	}, nil
}

// Return a TupleDescriptor for this projection. The returned descriptor should contain
// fields for each field in the constructor selectFields list with outputNames
// as specified in the constructor.
// HINT: you can use expr.GetExprType() to get the field type
func (p *Project) Descriptor() *TupleDesc {
	res := &TupleDesc{}

	for idx, selectedField := range p.selectFields {
		res.Fields = append(res.Fields, selectedField.GetExprType())
		res.Fields[idx].Fname = p.outputNames[idx]
	}

	return res
}

// Project operator implementation.  This function should iterate over the
// results of the child iterator, projecting out the fields from each tuple. In
// the case of distinct projection, duplicate tuples should be removed.
// To implement this you will need to record in some data structure with the
// distinct tuples seen so far.  Note that support for the distinct keyword is
// optional as specified in the lab 2 assignment.
func (p *Project) IteratorWithoutDistinct(tid TransactionID) (func() (*Tuple, error), error) {
	ite, _ := p.child.Iterator(tid)

	return func() (*Tuple, error) {
		tuple, _ := ite()
		if tuple == nil {
			return nil, nil
		}
		fieldTypes := make([]FieldType, len(p.selectFields))
		for idx, expr := range p.selectFields {
			fieldTypes[idx] = expr.GetExprType()
		}

		tuple, _ = tuple.project(fieldTypes)

		return &Tuple{
			Desc:   *p.Descriptor(),
			Fields: tuple.Fields,
		}, nil

	}, nil
}

func (p *Project) Iterator(tid TransactionID) (func() (*Tuple, error), error) {
	ite, _ := p.child.Iterator(tid)

	// create a set here
	distinctMap := make(map[any]int)

	return func() (*Tuple, error) {
		tuple, _ := ite()
		for tuple != nil {
			fieldTypes := make([]FieldType, len(p.selectFields))
			for idx, expr := range p.selectFields {
				fieldTypes[idx] = expr.GetExprType()
			}
			tuple, _ = tuple.project(fieldTypes)
			// check in set
			if p.isDistinct && distinctMap[tuple.tupleKey()] == 1 {
				// try to get a new one
				tuple, _ = ite()
				continue
			}

			// set a new one
			if p.isDistinct {
				distinctMap[tuple.tupleKey()] = 1
			}

			return &Tuple{
				Desc:   *p.Descriptor(),
				Fields: tuple.Fields,
			}, nil
		}
		return nil, nil
	}, nil
}

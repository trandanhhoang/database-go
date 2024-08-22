package godb

type DeleteOp struct {
	file  DBFile
	child Operator
}

// Construtor.  The delete operator deletes the records in the child
// Operator from the specified DBFile.
func NewDeleteOp(deleteFile DBFile, child Operator) *DeleteOp {
	return &DeleteOp{
		file:  deleteFile,
		child: child,
	}
}

// The delete TupleDesc is a one column descriptor with an integer field named "count"
func (i *DeleteOp) Descriptor() *TupleDesc {
	return i.child.Descriptor()
}

// Return an iterator function that deletes all of the tuples from the child
// iterator from the DBFile passed to the constuctor and then returns a
// one-field tuple with a "count" field indicating the number of tuples that
// were deleted.  Tuples should be deleted using the [DBFile.deleteTuple]
// method.
func (dop *DeleteOp) Iterator(tid TransactionID) (func() (*Tuple, error), error) {
	ite, _ := dop.child.Iterator(tid)
	fields := []FieldType{
		{"count", "", IntType},
	}
	res := &Tuple{
		Desc:   TupleDesc{fields},
		Fields: []DBValue{IntField{0}},
	}
	counter := int64(0)
	return func() (*Tuple, error) {
		tuple, _ := ite()
		for tuple != nil {

			dop.file.deleteTuple(tuple, tid)

			counter++
			res.Fields[0] = IntField{counter}

			tuple, _ = ite()
		}
		return res, nil
	}, nil
}

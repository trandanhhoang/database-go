package godb

// TODO: some code goes here
type InsertOp struct {
	file      DBFile
	child     Operator
	tupleDesc *TupleDesc
}

// Construtor.  The insert operator insert the records in the child
// Operator into the specified DBFile.
func NewInsertOp(insertFile DBFile, child Operator) *InsertOp {
	fields := []FieldType{
		{"count", "", IntType},
	}

	return &InsertOp{
		file:      insertFile,
		child:     child,
		tupleDesc: &TupleDesc{fields},
	}
}

// The insert TupleDesc is a one column descriptor with an integer field named "count"
func (i *InsertOp) Descriptor() *TupleDesc {
	return i.tupleDesc
}

// Return an iterator function that inserts all of the tuples from the child
// iterator into the DBFile passed to the constuctor and then returns a
// one-field tuple with a "count" field indicating the number of tuples that
// were inserted.  Tuples should be inserted using the [DBFile.insertTuple]
// method.
func (iop *InsertOp) Iterator(tid TransactionID) (func() (*Tuple, error), error) {
	ite, _ := iop.child.Iterator(tid)
	counter := int64(0)
	return func() (*Tuple, error) {
		tuple, _ := ite()
		for tuple != nil {

			err := iop.file.insertTuple(tuple, tid)
			if err != nil {
				return nil, err
			}
			counter++
			tuple, _ = ite()
		}
		return &Tuple{
			Desc:   *iop.tupleDesc,
			Fields: []DBValue{IntField{counter}},
		}, nil
	}, nil
}

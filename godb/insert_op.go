package godb

// TODO: some code goes here
type InsertOp struct {
	file  DBFile
	child Operator
}

// Construtor.  The insert operator insert the records in the child
// Operator into the specified DBFile.
func NewInsertOp(insertFile DBFile, child Operator) *InsertOp {
	return &InsertOp{
		file:  insertFile,
		child: child,
	}
}

// The insert TupleDesc is a one column descriptor with an integer field named "count"
func (i *InsertOp) Descriptor() *TupleDesc {
	return i.child.Descriptor()
}

// Return an iterator function that inserts all of the tuples from the child
// iterator into the DBFile passed to the constuctor and then returns a
// one-field tuple with a "count" field indicating the number of tuples that
// were inserted.  Tuples should be inserted using the [DBFile.insertTuple]
// method.
func (iop *InsertOp) Iterator(tid TransactionID) (func() (*Tuple, error), error) {
	ite, _ := iop.child.Iterator(tid)
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

			iop.file.insertTuple(tuple, tid)

			counter++
			res.Fields[0] = IntField{counter}

			tuple, _ = ite()
		}
		return res, nil
	}, nil
}

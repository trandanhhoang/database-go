package godb

import (
	"log"
	"os"
)

// This function should load the csv file in fileName into a heap file (see
// [HeapFile.LoadFromCSV]) and then compute the sum of the integer field in
// string and return its value as an int The supplied csv file is comma
// delimited and has a header If the file doesn't exist or can't be opened, or
// the field doesn't exist, or the field is not and integer, should return an
// err. Note that when you create a HeapFile, you will need to supply a file
// name;  you can supply a non-existant file, in which case it will be created.
// However, subsequent invocations of this method will result in tuples being
// reinserted into this file unless you delete (e.g., with [os.Remove] it before
// calling NewHeapFile.
func computeFieldSum(fileName string, td TupleDesc, sumField string) (int, error) {
	bp := NewBufferPool(5)
	newFileName := "test-csv.dat"
	if _, err := os.Stat(newFileName); err == nil {
		os.Remove(newFileName)
	}
	heapFile, err := NewHeapFile(newFileName, &td, bp)
	if err != nil {
		return 0, err
	}

	f, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("Couldn't open test_heap_file.csv")
		return 0, err
	}

	err = heapFile.LoadFromCSV(f, true, ",", false)
	if err != nil {
		log.Fatalf("Load failed, %s", err)
	}

	iter, _ := heapFile.Iterator(NewTID())
	counter := 0
	tuple, err := iter()
	fieldType := FieldType{
		Fname: sumField,
		Ftype: IntType,
	}
	for tuple != nil {
		idx, _ := findFieldInTd(fieldType, heapFile.tupleDesc)
		counter += int(tuple.Fields[idx].(IntField).Value)
		tuple, err = iter()
	}

	return counter, nil // replace me
}

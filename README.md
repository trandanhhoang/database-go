# Database from scratch written in Go

- Implement GoDB, a basic database management system.

## Introduction

- Base on labs of [course 6.5830/6.5831](http://dsg.csail.mit.edu/6.5830/).

- GoDB consists of:
  - Structures that represent fields, tuples, and tuple schemas;
  - Methods that apply predicates and conditions to tuples;
  - One or more access methods (e.g., heap files) that store relations on disk and provide a way to iterate through tuples of those relations;
  - A collection of operator classes (e.g., select, join, insert, delete, etc.) that process tuples;
  - A buffer pool that caches active tuples and pages in memory and handles concurrency control and transactions (neither of which you need to worry about for this lab); and,
  - A catalog that stores information about available tables and their schemas.

## Labs 1

- The main function of lab1 in this experiment is to access data stored on the disk.

- Let thing DB in OOP
  - We need to know about class (components of it)
    - In class, we have
      - field: How they interact with each other
      - method:
        - method name -> What the purpose of it ?
        - method implement -> How they work ?

### Tuple (Record, Row, what ever, ...) (Implement here)

- First, we will observation, try to pass all test in tuple_test.

- Defination: The Tuple struct in GoDB is used to store the in-memory value of a database tuple.

  - We just support 2 type here: String and Int.
  - Tuple objects are created by the underlying access methods (e.g., heap files, or B-trees). We just work with heapfiles in this lab.
  - Support:
    - Projection: simply mean selecting from rows from table.

- Like I said above, let look Tuple in OOP.

```go
type Tuple struct {
	Desc   TupleDesc // type of columns
	Fields []DBValue // value of columns
	Rid    recordID // track the page and position **this page** was read from. It's not the ID autogenerate in create-table.
}

type recordID interface {
}

type RecordID struct {
	PageNo int
	SlotNo int
} // Like comment above, track page and position
```

- Now you will ask, what is SlotNo ?

  - We use **access method** `heapfiles` () in this lab
    - `Access methods`: provide a way to read or write data from disk that is arranged in a specific way. Common access methods include heap files (unsorted files of tuples) and B-trees (range lookup), hash index for equality lookup. multi-dimension index such as R-tree.
  - Heapfile is a file, it have pages (each page have 4096 bytes)
    - Page have header (to keep number of max slot, used slot), the rest payload used to put tuples.
    - You can think slot as tuple
    - How we can calcualte slot ?
      - eg: use 8 byte for max slot, 8 bytes for used slot
        - Tuple desc: 2 column, String (32 byte) and int(8 bytes).
        - max slot = (4096-8) / (32+8) = 102 slots.

- Continue observation method of Tuple class

  - writeTo(\*bytes.Buffer): `Everything is a file`, so all record is saved in file.

  - readTupleFrom(\*bytes.Buffer, TupleDesc): This method used for reading tuple from file to memory. But Tuple have 3 field (desc, tuplesValue, Rid). We have desc, tuplesValue. Where RID ?, keep this question, and find the answer later.

  - equals(Tuple) bool
  - joinTuples(t1 \*Tuple, t2 \*Tuple) \*Tuple
  - compareField(Tuple, Expr) orderByState
  - project([]FieldType) \*Tuple
  - tupleKey() any

### BufferPool (Just terminology here)

- BP: in simple term, is used to read pages from disk and write them back to disk. The following diagram explains the working process of the buffer pool.

![alt text](bufferpool.png)

- It wasn't simple. Many required thing need to implement.

- The buffer pool (class BufferPool in GoDB) is responsible for caching pages in memory that have been recently read from disk. All operators read and write pages from various files on disk through the buffer pool. It consists of a fixed number of pages, defined by the numPages parameter to the BufferPool constructor NewBufferPool.
- For this lab, you only need to implement the constructor and the BufferPool.getPage() method used by the HeapFile iterator. The buffer pool stores structs that implement the Page interface; these pages can be read from underlying database files (such as a heap file) which implement the DBFile interface using the readPage method. The BufferPool should store up to numPages pages. If more than numPages requests are made for different pages, you should evict one of them according to an eviction policy of your choice. Note that you should not evict dirty pages (pages where the Page method isDirty() returns true), for reasons we will explain when we discuss transactions later in the class. You don't need to worry about locking in lab 1.

### HeapFile (access method - datastructure for organizing and accessing data on disk.) (Just terminology here)

- A `HeapFile` object is arranged into a set of pages, each of which consists of a fixed number of bytes for storing tuples, (defined by the constant PageSize), including a header
  - 1 HeapFile Object for each table, each page hold a set of tuples.
- A `Page` are type **HeapPage** have implement the **Page** interface
  - Page are fixed size and Tuple are fixed size, so all pages hold the same number of tuples
- GoDB store heap file on disks as pages of data arrangement consecutively on disk. On disk, each page have header, follow by **PageSize**. Header consist 32 bit integer with the number of tuples, and second 32 bit integer with the number of used tuples.

### Heap page. (Implement here)

- I haved explained it above, try to find with **Page have header**

- Let look HeapPage in OOP.

```go
type heapPage struct {
	tupleDesc *TupleDesc
	file      *HeapFile
	pageNo    int
	// For Header
	numSlots  int32
	usedSlots int32
	// For the rest
	tuples []*Tuple
  // mark dirty
	dirty  bool
}
```

- Note that to process deletions you will likely delete tuples at a specific position (slot) in the heap page. This means that after a page is read from disk, tuples should retain the same slot number. Because GoDB will never evict a dirty page, it's OK if tuples are renumbered when they are written back to disk.

- It still hard with the support of chatGPT.
- Now try to pass all test in heap_page_test.go
- Some trick lor:
  - initFromBuffer need append RID after use readTupleFrom() method.

### HeapFile (Implement here)

- Let look HeapFile in OOP.

```go
type HeapFile struct {
	bufPool *BufferPool
	sync.Mutex
	// From constructor
	tupleDesc *TupleDesc
	fromFile  string
	// user define
	numberOfPage int
	sizeOfTuple  int
	Pages        []*heapPage
}
```

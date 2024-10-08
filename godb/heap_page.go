package godb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unsafe"
)

/* HeapPage implements the Page interface for pages of HeapFiles. We have
provided our interface to HeapPage below for you to fill in, but you are not
required to implement these methods except for the three methods that the Page
interface requires.  You will want to use an interface like what we provide to
implement the methods of [HeapFile] that insert, delete, and iterate through
tuples.

In GoDB all tuples are fixed length, which means that given a TupleDesc it is
possible to figure out how many tuple "slots" fit on a given page.

In addition, all pages are PageSize bytes.  They begin with a header with a 32
bit integer with the number of slots (tuples), and a second 32 bit integer with
the number of used slots.

Each tuple occupies the same number of bytes.  You can use the go function
unsafe.Sizeof() to determine the size in bytes of an object.  So, a GoDB integer
(represented as an int64) requires unsafe.Sizeof(int64(0)) bytes.  For strings,
we encode them as byte arrays of StringLength, so they are size
((int)(unsafe.Sizeof(byte('a')))) * StringLength bytes.  The size in bytes  of a
tuple is just the sum of the size in bytes of its fields.

Once you have figured out how big a record is, you can determine the number of
slots on on the page as:

remPageSize = PageSize - 8 // bytes after header
numSlots = remPageSize / bytesPerTuple //integer division will round down

To serialize a page to a buffer, you can then:

write the number of slots as an int32
write the number of used slots as an int32
write the tuples themselves to the buffer

You will follow the inverse process to read pages from a buffer.

Note that to process deletions you will likely delete tuples at a specific
position (slot) in the heap page.  This means that after a page is read from
disk, tuples should retain the same slot number. Because GoDB will never evict a
dirty page, it's OK if tuples are renumbered when they are written back to disk.

*/

type heapPage struct {
	tupleDesc *TupleDesc
	file      *HeapFile
	pageNo    int
	// user define
	// For Header
	totalSlots int32
	usedSlots  int32
	// For the rest
	tuples []*Tuple
	dirty  bool
	offset int // for flush page, But if have pageNo*4096 = offset ??
}

// Construct a new heap page
func newHeapPage(desc *TupleDesc, pageNo int, f *HeapFile) *heapPage {
	headerSizeAsByte := 8
	// Khởi tạo số lượng slots dựa trên kích thước của một trang và kích thước của một tuple
	// giả sử rằng PageSize và kích thước của mỗi Tuple đều đã được xác định trước
	// Ta sẽ tính số lượng slots dựa trên công thức:
	// numSlots = (PageSize - headerSize) / tupleSize
	tupleSize := calculateTupleSize(desc)
	numSlots := (PageSize - headerSizeAsByte) / tupleSize

	return &heapPage{
		totalSlots: int32(numSlots),
		usedSlots:  0, // Khi mới tạo thì chưa có tuple nào được sử dụng
		tupleDesc:  desc,
		tuples:     make([]*Tuple, numSlots),
		file:       f,
		pageNo:     pageNo,
	}
}

func calculateTupleSize(desc *TupleDesc) int {
	tupleSize := 0
	for _, field := range desc.Fields {
		switch field.Ftype {
		case IntType:
			tupleSize += int(unsafe.Sizeof(int64(0))) // Giả sử kích thước của int là 8 byte
		case StringType:
			tupleSize += StringLength // Kích thước của chuỗi cố định
		}
	}
	return tupleSize
}

func (h *heapPage) getNumSlots() int {
	return int(h.totalSlots - h.usedSlots)
}

var counter1 = 0

// Insert the tuple into a free slot on the page, or return an error if there are
// no free slots.  Set the tuples rid and return it.
func (h *heapPage) insertTuple(t *Tuple) (recordID, error) {
	rid := RecordID{PageNo: h.pageNo, SlotNo: 0}
	// Check if there's space for more tuples
	if h.usedSlots >= h.totalSlots {
		return nil, fmt.Errorf("no free slots available")
	}

	// Find a free slot
	for i, tuple := range h.tuples {
		if tuple == nil {
			h.tuples[i] = t
			rid.SlotNo = i
			h.tuples[i].Rid = rid
			h.usedSlots += 1
			h.setDirty(true)
			return rid, nil
		}
	}

	return nil, fmt.Errorf("no free slots available")
}

// Delete the tuple in the specified slot number, or return an error if
// the slot is invalid
func (h *heapPage) deleteTuple(rid recordID) error {
	for i, tuple := range h.tuples {
		if tuple != nil && tuple.Rid == rid {
			h.tuples[i] = nil
			h.usedSlots -= 1
			h.setDirty(true)
			return nil
		}
	}
	return fmt.Errorf("slot number %v is invalid", rid)
}

// Page method - return whether or not the page is dirty
func (h *heapPage) isDirty() bool {
	return h.dirty
}

// Page method - mark the page as dirty
func (h *heapPage) setDirty(dirty bool) {
	h.dirty = dirty
}

// Page method - return the corresponding HeapFile
// for this page.
func (p *heapPage) getFile() *DBFile {
	var dbFile DBFile = p.file
	return &dbFile
}

// Allocate a new bytes.Buffer and write the heap page to it. Returns an error
// if the write to the the buffer fails. You will likely want to call this from
// your [HeapFile.flushPage] method.  You should write the page header, using
// the binary.Write method in LittleEndian order, followed by the tuples of the
// page, written using the Tuple.writeTo method.
func (h *heapPage) toBuffer() (*bytes.Buffer, error) {
	bytesBuffer := new(bytes.Buffer)
	err := binary.Write(bytesBuffer, binary.LittleEndian, h.totalSlots)
	if err != nil {
		return nil, err
	}
	err = binary.Write(bytesBuffer, binary.LittleEndian, h.usedSlots)
	if err != nil {
		return nil, err
	}
	for _, tuple := range h.tuples {
		if tuple != nil {
			err := tuple.writeTo(bytesBuffer)
			if err != nil {
				return nil, err
			}
		}
	}

	// This code need for NumPages() work
	paddingSize := 4096 - bytesBuffer.Len()
	paddingBytes := make([]byte, paddingSize)
	_, err = bytesBuffer.Write(paddingBytes)
	if err != nil {
		return nil, err
	}

	return bytesBuffer, nil

}

// Read the contents of the HeapPage from the supplied buffer.
func (h *heapPage) initFromBuffer(buf *bytes.Buffer) error {
	// Đọc số lượng slots max từ header
	err := binary.Read(buf, binary.LittleEndian, &h.totalSlots)
	if err != nil {
		return fmt.Errorf("could not read numSlots: %w", err)
	}

	// Đọc số lượng slots đã sử dụng từ header
	err = binary.Read(buf, binary.LittleEndian, &h.usedSlots)
	if err != nil {
		return fmt.Errorf("could not read usedSlots: %w", err)
	}

	// Đọc từng tuple từ buffer
	for i := 0; i < int(h.usedSlots); i++ {
		tuple, err := readTupleFrom(buf, h.tupleDesc)
		if err != nil {
			return fmt.Errorf("could not read tuple %d: %w", i, err)
		}
		h.tuples[i] = tuple
		// lack of RID
		h.tuples[i].Rid = RecordID{PageNo: h.pageNo, SlotNo: i}
	}

	return nil
}

// Return a function that iterates through the tuples of the heap page.  Be sure
// to set the rid of the tuple to the rid struct of your choosing beforing
// return it. Return nil, nil when the last tuple is reached.

// used slot maybe not dense
func (p *heapPage) tupleIter() func() (*Tuple, error) {
	i := 0
	return func() (*Tuple, error) {
		for i < int(p.totalSlots) && p.tuples[i] == nil {
			i++
		}
		if i <= int(p.totalSlots) && i < len(p.tuples) {
			if p.tuples[i] != nil {
				tuple := p.tuples[i]
				tuple.Rid = p.tuples[i].Rid
				i++
				return tuple, nil
			}
		}
		return nil, nil
	}
}

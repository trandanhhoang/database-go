package godb

import (
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"
)

//BufferPool provides methods to cache pages that have been read from disk.
//It has a fixed capacity to limit the total amount of memory used by GoDB.
//It is also the primary way in which transactions are enforced, by using page
//level locking (you will not need to worry about this until lab3).

// Permissions used to when reading / locking pages
type RWPerm int

const (
	ReadPerm  RWPerm = iota
	WritePerm RWPerm = iota
)

type TaskDo string

const (
	ReadTask   TaskDo = "READ TASK"
	InsertTask TaskDo = "INSERT TASK"
	DeleteTask TaskDo = "DELETE TASK"
)

type BufferPool struct {
	// max page can saved
	numPages int
	// Giả sử chỉ có 1 bufferpool duy nhất cho DB.
	pages map[any]*Page // DBFile -> pageNo -> Page

	// Think about what happens if two threads simultaneously try to read or evict a page
	mu                sync.Mutex
	mapPageLocksByTid map[TransactionID]map[any]*PageLock          // DBFile -> pageNo -> Page
	waitTidLocks      map[TransactionID]map[TransactionID]struct{} // deadlock detection and prevention
}

type PageLock struct {
	page   *Page
	pageNo int
	perm   RWPerm
	key    any
}

// Create a new BufferPool with the specified number of pages
func NewBufferPool(numPages int) *BufferPool {
	return &BufferPool{
		numPages:          numPages,
		pages:             make(map[any]*Page),
		mu:                sync.Mutex{},
		mapPageLocksByTid: map[TransactionID]map[any]*PageLock{},
		waitTidLocks:      map[TransactionID]map[TransactionID]struct{}{},
	}
}

// Testing method -- iterate through all pages in the buffer pool
// and flush them using [DBFile.flushPage]. Does not need to be thread/transaction safe
func (bp *BufferPool) FlushAllPages() {
	for _, page := range bp.pages {
		p := *page
		file := *p.getFile()
		file.flushPage(&p)
	}
}

// Abort the transaction, releasing locks. Because GoDB is FORCE/NO STEAL, none
// of the pages tid has dirtired will be on disk so it is sufficient to just
// release locks to abort. You do not need to implement this for lab 1.
func (bp *BufferPool) AbortTransaction(tid TransactionID) {
	log.Printf("abort %v", *tid)
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.clearMap(tid)
}

// Commit the transaction, releasing locks. Because GoDB is FORCE/NO STEAL, none
// of the pages tid has dirtied will be on disk, so prior to releasing locks you
// should iterate through pages and write them to disk.  In GoDB lab3 we assume
// that the system will not crash while doing this, allowing us to avoid using a
// WAL. You do not need to implement this for lab 1.
func (bp *BufferPool) CommitTransaction(tid TransactionID) {
	log.Println("commit ", *tid)
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// loop through all page
	for _, pgLock := range bp.mapPageLocksByTid[tid] {
		hp := (*pgLock.page).(*heapPage)
		if !hp.isDirty() { // why dirty continue, where dirty is put here ?
			// no dirty here
			continue
		}
		hp.file.mu.Lock()
		if err := hp.file.flushPage(pgLock.page); err != nil {
			panic(err)
		}
		hp.file.mu.Unlock()
	}
	// release lock
	delete(bp.mapPageLocksByTid, tid)
	bp.deleteWaitTidLocks(tid)
}

func (bp *BufferPool) BeginTransaction(tid TransactionID) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return nil
}

// Retrieve the specified page from the specified DBFile (e.g., a HeapFile), on
// behalf of the specified transaction. If a page is not cached in the buffer pool,
// you can read it from disk uing [DBFile.readPage]. If the buffer pool is full (i.e.,
// already stores numPages pages), a page should be evicted.  Should not evict
// pages that are dirty, as this would violate NO STEAL. If the buffer pool is
// full of dirty pages, you should return an error. For lab 1, you do not need to
// implement locking or deadlock detection. [For future labs, before returning the page,
// attempt to lock it with the specified permission. If the lock is
// unavailable, should block until the lock is free. If a deadlock occurs, abort
// one of the transactions in the deadlock]. You will likely want to store a list
// of pages in the BufferPool in a map keyed by the [DBFile.pageKey].
func (bp *BufferPool) GetPage(file DBFile, pageNo int, tid TransactionID, perm RWPerm, task TaskDo) (*Page, error) {
	log.Printf("getpage by tid %v for task %v pageNo %v", *tid, task, pageNo)
	bp.mu.Lock()
	defer bp.mu.Unlock()

	key := file.pageKey(pageNo)
	// Nếu trang đã có trong bộ nhớ đệm, trả về trang đó
	if page, ok := bp.pages[key]; ok {
		if err := bp.handleTransactionInGetPage(pageNo, key, page, tid, perm, task); err != nil {
			return nil, err
		}
		return page, nil
	}

	// evict page if bp full
	if len(bp.pages) >= bp.numPages {
		hasDelete := false
		for _, otherPage := range bp.pages {
			if !(*otherPage).isDirty() {
				hpPage := (*otherPage).(*heapPage)
				delete(bp.pages, hpPage.file.pageKey(hpPage.pageNo).(heapHash))
				hasDelete = true
				break
			}
		}
		if !hasDelete {
			return nil, errors.New("buffer pool is full of dirty pages")
		}
	}

	// Page not in cached, load from disk
	newPage, err := file.readPage(pageNo)
	if err != nil {
		return nil, err
	}

	// Thêm trang vào bộ nhớ đệm
	bp.pages[key] = newPage

	if err := bp.handleTransactionInGetPage(pageNo, key, newPage, tid, perm, task); err != nil {
		return nil, err
	}

	return newPage, nil
}

func (bp *BufferPool) handleTransactionInGetPage(pageNo int, key any, page *Page, tid TransactionID, perm RWPerm, task TaskDo) error {
	// this can loop forever
	// why, because after the first write.
	for bp.isConflicted(pageNo, tid, perm) {
		err := bp.waitWhenConflict(pageNo, key, page, tid, perm, task)
		if err != nil {
			return err
		}
	}
	// No conflict, save map or upgrade lock exclusive
	bp.saveMapAndUpgradeLock(pageNo, key, page, tid, perm)

	// delete wait tid lock
	bp.deleteWaitTidLocks(tid)
	return nil
}

func (bp *BufferPool) saveMapAndUpgradeLock(pageNo int, key any, page *Page, tid TransactionID, perm RWPerm) {
	log.Println("saveMapAndUpgradeLock ", *tid)
	isNeedUpradeLock := perm == WritePerm
	if _, ok := bp.mapPageLocksByTid[tid][key]; !ok || isNeedUpradeLock {
		if bp.mapPageLocksByTid[tid] == nil {
			bp.mapPageLocksByTid[tid] = map[any]*PageLock{}
		}
		bp.mapPageLocksByTid[tid][key] = &PageLock{page: page, pageNo: pageNo, perm: perm, key: key}
	}
}

func (bp *BufferPool) clearMap(tid TransactionID) {
	for _, pgLock := range bp.mapPageLocksByTid[tid] {
		// delete page with write perm
		if pgLock.perm == WritePerm {
			delete(bp.pages, pgLock.key)
		}
	}
	// release lock
	delete(bp.mapPageLocksByTid, tid)
	bp.deleteWaitTidLocks(tid)
}

func (bp *BufferPool) waitWhenConflict(pageNo int, key any, page *Page, tid TransactionID, perm RWPerm, task TaskDo) error {
	log.Println("waitWhenConflict ", *tid)
	if bp.deadLockPrevent(tid, bp.waitTidLocks[tid], map[TransactionID]bool{}, 0) {
		// bp.abortTransactionWithoutLock(tid)
		log.Printf("Abort transaction tid %v, perm %v, task %v", *tid, perm, task)
		// delete after abort transaction
		bp.clearMap(tid)
		time.Sleep(time.Duration(10+rand.Intn(10)) * time.Millisecond)
		return errors.New("transaction is aborted")
	}
	// You will need to release the mutex before blocking (to allow another thread/transaction to attempt to acquire the lock)
	// unlock for another trans and wait
	bp.mu.Unlock()
	time.Sleep(time.Duration(10+rand.Intn(10)) * time.Millisecond)
	bp.mu.Lock()
	return nil
}

// func (bp *BufferPool) abortTransactionWithoutLock(tid TransactionID) {
// 	for _, pgLock := range bp.mapPageLocksByTid[tid] {
// 		if pgLock.perm == WritePerm {
// 			//(*pgLock.page).setDirty(false)
// 			delete(bp.pages, pgLock.pageKey)
// 		}
// 	}
// 	delete(bp.mapPageLocksByTid, tid)
// 	bp.deleteWaitTidLocks(tid)
// }

// tid 1 -> page 1 (read), page 2(read)
// tid 2 -> page 2 (write)
func (bp *BufferPool) isConflicted(pageNo int, tid TransactionID, perm RWPerm) bool {
	log.Printf("check conflict tid %v, perm %v", *tid, perm)
	// bp.printMapTidLockPages()
	isConflicted := false
	// loop through other id to check
	for t, pageLocks := range bp.mapPageLocksByTid {
		if t == tid { // ignore cur tid
			continue
		}
		for _, otherPageLock := range pageLocks {
			if otherPageLock.pageNo == pageNo && (otherPageLock.perm == WritePerm || perm == WritePerm) {
				// if conflict, we must wait.
				if bp.waitTidLocks[tid] == nil {
					bp.waitTidLocks[tid] = map[TransactionID]struct{}{}
				}
				bp.waitTidLocks[tid][t] = struct{}{}
				isConflicted = true
			}
		}
	}
	return isConflicted
}

// DFS, visited
// 1 <- 2
// |  /^
// | /
// v
// 3
func (bp *BufferPool) deadLockPrevent(root TransactionID, tidMap map[TransactionID]struct{}, visitedMap map[TransactionID]bool, counter int) bool {
	// log.Printf("deadLockPrevent tid %v,lenTid %v, tidMap %v cnt %v", *root, len(tidMap), tidMap, counter)
	if _, ok := tidMap[root]; ok { // cycle detected
		// bp.printWaitTidLock()
		log.Printf("cycle tid %v, cnt %v", *root, counter)
		return true
	}
	for tid := range tidMap {
		// neighbour
		if visitedMap[tid] {
			continue
		}
		visitedMap[tid] = true
		if bp.deadLockPrevent(root, bp.waitTidLocks[tid], visitedMap, counter+1) {
			// bp.printWaitTidLock()
			log.Printf("cycle tid %v, cnt %v", *root, counter)
			return true
		}
	}
	return false
}

func (bp *BufferPool) deleteWaitTidLocks(tid TransactionID) {
	log.Println("deleteWaitTidLocks ", *tid)
	for key, tidMap := range bp.waitTidLocks {
		if tid == key {
			continue
		}
		delete(tidMap, tid)
		if len(tidMap) == 0 {
			delete(bp.waitTidLocks, key)
		}
	}
	delete(bp.waitTidLocks, tid)
}

func (bp *BufferPool) printMapTidLockPages() {
	//for each to print it
	log.Println("len", len(bp.mapPageLocksByTid))
	for tid, pageLocks := range bp.mapPageLocksByTid {
		for _, pageLock := range pageLocks {
			log.Printf("printMapTidLockPages tid %v, pageNo %v, perm %v, key %v", *tid, pageLock.pageNo, pageLock.perm, pageLock.key)
		}
	}
}

func (bp *BufferPool) printWaitTidLock() {
	//for each to print it
	log.Println("len", len(bp.waitTidLocks))
	for tid, mapTids := range bp.waitTidLocks {
		for key, value := range mapTids {
			log.Printf("printWaitTidLock tid %v, tid2 %v, value %v", *tid, *key, value)
		}
	}
}

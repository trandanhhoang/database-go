# README - Labs 3

- The main function of lab3 is to support: locking.
  - Locking and transactions is tricky, that interesting.

## Minimize terminology that course provide (you can directly go to part "Implement time" if you have understand)

### Transaction Locks vs Mutex

- We using 2 phase locking protocol to lock a part of database (e.g., a page)
- We use mutex to prevent 2 threads from concurrently executing a piece of code.
  - https://pkg.go.dev/sync#Mutex
  - mutex can be locked or unlocked
- concurrent maps may be useful in your buffer pool implementation.
  - https://pkg.go.dev/sync

### Transactions, Locking, and Concurrency Control

- you should make sure you understand what a transaction is and how strict two-phase locking (which you will use to ensure isolation and atomicity of your transactions) works.

#### Transactions

- A transaction is a group of database actions (e.g., inserts, deletes, and reads) that are executed atomically; that is, either all of the actions complete or none of them do
- Each transaction runs in a separate thread, Multiple threads attempting to invoke methods on the database. You will need to use mutexes or other synchronization primitives to prevent race conditions (or indeterminate behavior).

#### ACID properties

- Atomicity: Strict two-phase locking and careful buffer management ensure atomicity.
- Consistency: The database is transaction consistent by virtue of atomicity. Other consistency issues (e.g., key constraints) are not addressed in GoDB.
- Isolation: Strict two-phase locking provides isolation.
- Durability: A FORCE buffer management policy ensures durability (see Section 2.3 below).

#### Recovery and Buffer management

- Just implement a NO STEAL/FORCE buffer management policy, it means.

  - You shouldn't evict dirty (updated) pages from the buffer pool if they are locked by an uncommitted transaction (this is NO STEAL). Your buffer pool implementation already does this.
  - On transaction commit, you should force dirty pages to disk (e.g., write the pages out) (this is FORCE).

- `Note`:
  - Assume that GoDB will not crash while processing a CommitTransaction or AbortTransaction command. That mean we don't need implement log-based recovery, since you will never need to `undo` any work (you never evict dirty pages), and you will never need to `redo` any work (you force updates on commit and will not crash during commit processing).

#### Granting lock

- Allow a caller to request or release a (shared or exclusive) lock on a specific object on behalf of a specific transaction.
- We just do page leve locking here.
- Create data structures that keep track of which locks each transaction holds and check to see if a lock should be granted to a transaction when it is requested
- You will need to implement shared and exclusive locks; recall that these work as follows:
  - Before a transaction can read an object, it must have a shared lock on it.
  - Before a transaction can write an object, it must have an exclusive lock on it.
  - Multiple transactions can have a shared lock on an object.
  - Only one transaction may have an exclusive lock on an object.
  - If transaction t is the only transaction holding a shared lock on an object o, t may upgrade its lock on o to an exclusive lock.
- If a transaction requests a lock that cannot be immediately granted, your code should block, waiting for that lock to become available (i.e., be released by another transaction running in a different thread). Recall that multiple threads will be running concurrently. Be careful about race conditions in your lock implementation --- think about how concurrent invocations to your lock may affect the behavior. To block a thread, you can simply call time.Sleep for a few milliseconds.

#### Lock Lifetime

- Implement strict two-phase locking.
  - This means that transactions should acquire the appropriate type of lock on any object before accessing that object and shouldn't release any locks until after the transaction is committed.
  - Lock on BufferPool.GetPage(), and release in CommitTransaction() and AbortTransaction().

## Implement time

- Base on exercise 1, we need:
  - create Data structures that keep track of the shared and exclusive locks each transaction is currently holding.
  - There will be multiple threads concurrently calling GetPage() during this test. Use sync.Mutex or the sync.Map construct to prevent race conditions.
- The simplest approach is: Associate a Mutex with your buffer pool

  - Acquire this mutex before you access any of the data structures you used to keep track of which pages are locked; this will ensure only one thread is trying to acquire a page lock at a time.
  - If you successfully acquire the page lock, you should release the mutex after lock acquisition.
  - If you fail to acquire the lock, you will block.
  - You will need to release the mutex before blocking (to allow another thread/transaction to attempt to acquire the lock)
  - Attempt to re-acquire the mutex before trying to re-acquire the lock.

- `I have no idea what I need to do`
  - let see how many test file we have NOT pass yet
    - deadlock_test.go, locking_test.go, transactions_test.go
    - lab2_extra_test.go (I fix some trivial method and it pass)
- Let observation 3 test above.

  - deadlock_test.go
    - It's about 150 line with 3 method TestUpgradeWriteDeadlock, TestWriteWriteDeadlock, TestReadWriteDeadlock
    - Because exercise 1 not talk about some test (not-unit-test), let ignore it now.
  - locking_test.go
    - 149 line, 16 method (8 method utils, 8 method test)
    - Just read the last one and make decision.
  - transactions_test.go
    - 400 line, ...23 method, about 12 method test, 11 method util.

- Observation exercise 2. After implement BeginX(), CommitX(), AbortX(). We should pass the locking_test.go
  - TestAttemptTransactionTwice, and TestTransaction{Commit, Abort} unit tests and the TestAbortEviction system test will pass.
- `Okay, we have know what test should be passed first, let see what locking_test.go do`

```go
func TestAcquireReadLocksOnSamePage(t *testing.T) {
	bp, hf, tid1, tid2 := lockingTestSetUp(t)
	metaLockTester(t, bp,
		tid1, hf, 0, ReadPerm,
		tid2, hf, 0, ReadPerm,
		true)
}
```

- lockingTestSetUp(t) create 2 tid and BeginTransaction with this.

```go
tid1 := NewTID()
bp.BeginTransaction(tid1)
tid2 := NewTID()
bp.BeginTransaction(tid2)
```

- In metaLockTester(t, bp,tid1, hf, 0, ReadPerm,tid2, hf, 0, ReadPerm,true)

```go
bp.GetPage(file1, pgNo1, tid1, perm1) // we getPage 0 with tid 1
grabLock(t, bp, tid2, file2, pgNo2, perm2, expected) // we getPage 1 with tid 2
```

- We need see what grabLock() do here

```go
func grabLock(t *testing.T,
	bp *BufferPool, tid TransactionID, file DBFile, pgNo int, perm RWPerm,
	expected bool) {

	lg := startGrabber(bp, tid, file, pgNo, perm)

	time.Sleep(100 * time.Millisecond)

	var acquired bool = lg.acquired()
	if expected != acquired {
		t.Errorf("Expected %t, found %t", expected, acquired)
	}

	// TODO how to kill stalling lg?
}

func startGrabber(bp *BufferPool, tid TransactionID, file DBFile, pgNo int, perm RWPerm) *LockGrabber {
	lg := NewLockGrabber(bp, tid, file, pgNo, perm)
	go lg.run()
	return lg
}

func NewLockGrabber(bp *BufferPool, tid TransactionID, file DBFile, pgNo int, perm RWPerm) *LockGrabber {
	return &LockGrabber{bp, tid, file, pgNo, perm,
		false, nil, sync.Mutex{}, sync.Mutex{}}
}

func (lg *LockGrabber) run() {
	// Try to get the page from the buffer pool.
	_, err := lg.bp.GetPage(lg.file, lg.pgNo, lg.tid, lg.perm)
	if err == nil {
		lg.alock.Lock()
		lg.acq = true
		lg.alock.Unlock()
	} else {
		lg.elock.Lock()
		lg.err = err
		lg.elock.Unlock()

		lg.bp.AbortTransaction(lg.tid)
	}
}
```

- We have not implement anything, let try to run the first test with 2 `Readperm`
  - ez pass, because grabLock expected = True
- The second test with `ReadPerm for tid 1` and `WritePerm for tid 2`
  - We fail here, because grabLock expected = False. Why ?
  - Now, we need learn 2PL (2 phase locking protocol in README.md)

### After few days to learn about concurency, 2PL

- I can answer the question above, just Read-Read have permission. All another pair (RW, WR, RR), the later TID will be blocked.

### Handle concurrency

- Vì chúng ta khoá theo page, nên hãy tạo 1 map[any]\*PageLock (với key any giống như pages ở trên, là 1 hash của file+pageNo). và value là loại khoá (share/read hay exclude/write)
  - with pagelock is

```go
type PageLock struct {
	page   *Page
	pageNo int
	perm   RWPerm
}
```

- Nếu 2 transactions cùng đọc vào 1 page thì sao?, chúng ta nên dùng map[tid]map[any]\*PageLock
- with any is key (heaphash of fromFile and pageNo)

```go
func (f *HeapFile) pageKey(pgNo int) any {
	hH := heapHash{FileName: f.fromFile, PageNo: pgNo}
	return hH
}
```

#### implement get page

#### debug time

```bash
go test -v -timeout 30s -run ^TestAcquireWriteReadLocksOnSamePage$ github.com/srmadden/godb > test_output.log 2>&1
```

#### If we don’t implement buffer_pool.CommitTransaction(), what happen ???

- problem 1. test read-write will pass coincident, because the write later can’t access the lock because conflicted (map have 300 tids with pageLock)

  - context: 1 page chứa được tối đa 102 record, có 300 record từ file CSV.
  - Step
    - step 1. load from cvs, ghi 300 record, chúng ta sẽ gặp
      - 294 lần lỗi readPage = 102 (lỗi đọc page 0 full) + 96\*2 (lỗi đọc page 0 và page 1 full)
      - Sau đó trong Map sẽ có 300 key của tid từ 1 → 300
        - 102 key là list 1 phần tử
        - 102 key là list 2 phần tử.
        - 96 key là list 3 phần tử.
        - 102+102*2+96*3 = **594 vòng lặp = 601-8+1(kiểm chứng bằng log)**
    - Step 2:
      - Read với tid 301 ở page 0, check xem trong map trên, cần 2 điều kiện để bị **conflict:** 1. tid đang nắm page 0 + 2. write perm.
      - Bởi vì loadfromcsv toàn là read perm, nên chúng ta bỏ qua trương hợp conflict ở đây (nếu conflict, phải dùng graph để tránh bị deadlock)
    - Step 3:
      - Write với tid 302 ở page 1, check xem trong map trên, cần 2 điều kiện để bị **conflict:** 1. tid đang nắm page 0 + 2. write perm. => Đã bị conflict.
      ```go
      for bp.isConflicted(pageNo, tid, perm) {
      		log.Println("do I reach here ??? ", *tid)
      		err := bp.waitWhenConflict(pageNo, key, page, tid, perm)
      		if err != nil {
      			return err
      		}
      	}
      ```
      - Lúc này thì vòng lặp này sẽ diễn ra vô tận, cho tới khi test timeout → pass nhưng ko chuẩn.

- problem 2. test write-tead, write hang forever, timeout after 30s.
  - step 1: like step 1 above
  - step 2: like step 3 above, hang forever when write.
    ```go
    func metaLockTester(t *testing.T, bp *BufferPool,
    	tid1 TransactionID, file1 DBFile, pgNo1 int, perm1 RWPerm,
    	tid2 TransactionID, file2 DBFile, pgNo2 int, perm2 RWPerm,
    	expected bool) {
    	// without remove in commit
    	// read tid 1, write tid 2, will can pass
    	// write tid 1, read tid 2, we can't reach the grabLock() below
    	bp.GetPage(file1, pgNo1, tid1, perm1)
    	grabLock(t, bp, tid2, file2, pgNo2, perm2, expected)
    }
    ```
    - We can’t access method grabLock(t, bp, tid2, file2, pgNo2, perm2, expected), because bp.GetPage(…) hang forever.

#### So we must implement commit to remove the mapPageLocksByTid map[TransactionID]map[any]\*PageLock // DBFile -> pageNo -> Page

```go
func (bp *BufferPool) CommitTransaction(tid TransactionID) {
	log.Println("commit ", &tid)
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// loop through all page of the tid, flush if dirty, ignore if no change.
	for _, pgLock := range bp.mapPageLocksByTid[tid] {
		hp := (*pgLock.page).(*heapPage)
		if !hp.isDirty() { // why dirty continue, where dirty is put here ?
			// no dirty here
			continue
		}
		hp.file.Lock()
		if err := hp.file.flushPage(pgLock.page); err != nil {
			panic(err)
		}
		hp.file.Unlock()
	}
	// release lock of that tid
	delete(bp.mapPageLocksByTid, tid)
	bp.deleteWaitTidLocks(tid)
}
```

- Okay, we can think like this
  - mu.lock → get page → map save PageLock with READ/WRITE PERM → mu.unlock
  - commit → mu.lock → flush page→ mu.unlock.
- So,

  - Map for check conflicted.
  - Mutex for ?
    - let think what happen if mutex don’t join here (for 2 tid with same page)
      - tid 1 for page 0 → get page → map save PageLock with READ PERM
      - tid 2 for page 0 → get page → map save PageLock with WRITE PERM
        - Ngay lúc này, cả 2 đều có thể pass method isConflicted() và được ghi vào map với 2 tid khác nhau cho cùng 1 page.
          - Lỗi đã xảy ra.
    - hệ quả có thể xảy ra là gì ?
      - Lý thuyết, với n transaction, có 2^N khả năng tương tác( xung đột) giữa các transaction → bài toán NP-hard
        - không có xung đột, toàn bộ xung đột vs nhau, 1 xung đột vs 2, … (3 transaction thì sẽ có 8 khả năng tương tác).
        - **đọc ghi → đọc không chính xác**
        - **ghi ghi → mất dữ liệu, trạng thái không nhất quán**
        - **đọc đọc → thường ko có vấn đề, nhưng sẽ có vấn đề nếu đọc 1 transaction ko ổn định (cần xem lại)**
      - **cách 1: Xử lý bằng 2PL → đảm bảo atomic và isolation → biến thể (strict và rigougous)**
        - Cách làm:
          - 1 pha đầu chỉ có thể thêm khoá, ko được giải phóng khoá.
          - 1 pha đầu chỉ có thể giải phóng khoá, ko được thêm khoá
        - Nhược điểm:
          - deadlock → phải resolve, resolve như thế nào ?
          - starve → hiệu suất kém
        - Tại sao 2pl giải quyết được:
          - dùng lock để các giao dịch ghi không bị can thiệp bởi tid khác, hay không được can thiệp vào các tid khác.
        - Đối với DB của chúng ta, chúng ta yêu cầu đúng khoá, và không giải phóng khóa cho tới khi commit, là đã hoàn thành
          - cần phải suy nghĩ sâu hơn, để biết để hiện thực 1 cách hoàn chỉnh, thì phải làm như thế nào, tại sao bài toán đơn giản mình đang giải ko cần quá phức tạp trong cách hiện thực.
        - Ví dụ thực tế trong database của chúng ta
          - bufferpool begin transaction với tid 1,
            - heapfile insert records vào line cuối cùng của page 0
              - yêu cầu bufferpool lấy page 0, permission write → EZ insert
            - **heapfile** insert tiếp 1 record nữa.
              - yêu cầu bufferpool lấy page 1, permission write. vậy là tid 1 đã nắm 2 page.
          - Phải có tới tận lúc bp.CommitTransaction(tid) thì 2 page trên sẽ được flush → flush xong, mới được giải phóng khoá.
        - Phân tích các biến thể.
      - **cách 2: xử lý bằng order time**
        - đọc, ghi: T1 ghi < T2 đọc → t1 phải bị rollback.
        - ghi, ghi: T1 ghi < T2 ghi → t1 phải bị rollback.
      - **cách 3: Optimistic concurency control (giả định ko có xung đột, ko cần khoá), gồm 2 phase → khá giống compare and swap.**
        - (validate phase) khi kết thúc, kiểm tra có xung đột không → có thì huỷ bỏ, và thực hiện lại.
        - (commit phase) ko có xung đột, flush.
      - **Cách 4: MVCC (multiversion concurency control), lưu trữ nhiều phiên bản của dữ liệu để đọc và ghi mà ko bị lock lẫn nhau**
        - phiên bản cũ: nếu ghi → tạo ra phiên bản mới. các tid khác có thể đọc phiên bản cũ này cho tới khi phiên bản mới ghi xong.
        - phiên bản mới: Sau khi transaction hoàn thành, phiên bản mới sẽ trở thành phiên bản "hiện tại" mà các transaction mới sẽ đọc.
        - MVCC giúp giảm thiểu các xung đột đọc-ghi và tăng hiệu suất trong các hệ thống có nhiều transaction đồng thời. PostgreSQL và MySQL InnoDB là hai ví dụ điển hình của DBMS sử dụng MVCC.
      - **Cách 5: Serialize Snapshot Isolation, ignore.**
    - Vậy DB chúng ta dùng cách 1, 2PL, chuyện gì sẽ xảy ra.
      - **Ví dụ 1: 2 TID với WRITE PERMISSION.** 2 record mới được insert cho mỗi tid, ví dụ ban đầu page có record 1 duy nhất.
        - Sau khi flush, 1 trong 2 record sẽ được thêm vào (chính xác phải là 3 record)
      - **Ví dụ 2: TID 1 READ - TID 2 WRITE.**
        - TID 1 đọc x = 100
        - TID 2 xử lý với x = 150
        - Lỗi có thể là gì ?
          - dirty read, tính sai.+ inconsistency
      - **Ví dụ 3: TID 1 WRITE - TID 2 READ.**
        - TID 1 viết x.= 150
        - TID 2 đọc với giá trị mới hoặc cũ.
        - lỗi có thể là gì ?
          - dirty read: TID xử lý trên giá trị mới mà TID 1 viết, rồi TID 1 rollback.
          - lost update: TID 2 xử lý trên data cũ nếu đọc giá trị cũ

- Chúng ta đã pass hết tất cả test trong locking_test.go

- Exercise 2
  - want us pass TestAttemptTransactionTwice, and TestTransaction{Commit, Abort}. DONE
  - TestAbortEviction -> we fail
- Let find the reason, we have method AbortTransaction like below

  -

  ```go
  func (bp *BufferPool) AbortTransaction(tid TransactionID) {
    bp.mu.Lock()
    defer bp.mu.Unlock()
    for _, pgLock := range bp.mapPageLocksByTid[tid] {
      if pgLock.perm == WritePerm {
        delete(bp.pages, pgLock.key)
      }
    }
    // release lock
    delete(bp.mapPageLocksByTid, tid)
    bp.deleteWaitTidLocks(tid)
  }
  ```

  - In buffer_pool.AbortTransaction() We have delete(bp.pages, pgLock.key), but change (insert records) still here. I think the perm is not WRITE

    - After debug, the perm is not WRITE. But in heap_file.insertTuple(), I see the perm is WRITE.

    ```go
    func (f *HeapFile) insertTuple(t *Tuple, tid TransactionID) error {
      numPages := f.NumPages()

      for i := 0; i < numPages; i++ {
        page, err := f.bufPool.GetPage(f, i, tid, WritePerm)
        if err != nil {
          return err
        }
        // ...
      }}
    ```

    ```go
      //Create HeapPage, insert tuple, flush page
      heapPage := newHeapPage(f.tupleDesc, f.NumPages(), f)
      heapPage.insertTuple(t)
      var page Page = heapPage
      err := f.flushPage(&page)
      if err != nil {
        return err
      }
      return nil
    ```

    - Because it falldown to the later handle in insertTuple(), the new page is created and I insertTuple without WRITE permission.
      - I fix the method in commit "fix: fix inserTuple method". and pass the test.

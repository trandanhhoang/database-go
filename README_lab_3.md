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

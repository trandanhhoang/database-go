# Database written in Go

- Implement GoDB, a basic database management system.
- Learn how database work internally by implementing it.

## Introduction

- Base on [course 6.5830/6.5831](http://dsg.csail.mit.edu/6.5830/).
- Git repository: https://github.com/MIT-DB-Class/go-db-hw-2023

- GoDB consists of:

  - In lab 1
    - Structures that represent fields, tuples, and tuple schemas
    - One or more access methods (e.g., heap files) that store relations on disk and provide a way to iterate through tuples of those relations
    - A buffer pool that caches active tuples and pages in memory
  - In lab 2

    - A collection of operator classes (e.g., select, join, insert, delete, etc.) that process tuples

  - In lab 3
    - Buffer pool need to handles concurrency control and transactions
    - Support Locking

- `Lack of`: I will talk about all of the lack in README_lack_of.md
  - Not support log-based recovery
    - by assume GoDB will not crash while processing a CommitTransaction or AbortTransaction => no need undo or redo any work

## For detail, you can read README_lab\_[1-3].md for more information.

## This segment here for extend learning, you can ignore it.

ref:

- Concurrency Control and Recovery: https://dsg.csail.mit.edu/6.5830/2023/lectures/franklin97concurrency.pdf
- https://dsg.csail.mit.edu/6.5830/2023/lectures/lec15.pdf

### Transaction schedule

| Transfer                | ReportSum                                   |
| ----------------------- | ------------------------------------------- |
| 01 A bal := Read(A)     |                                             |
| 02 A bal := A bal - $50 |                                             |
| 03 Write(A,A bal)       |                                             |
|                         | 01 A bal := Read(A) /_ value is $250 _/     |
|                         | 02 B bal := Read(B) /_ value is $200 _/     |
|                         | 03 Print(A bal + B bal) /_ result = $450 _/ |
| 04 B bal := Read(B)     |                                             |
| 05 B bal := B bal + $50 |                                             |
| 06 Write(B,B bal)       |                                             |

- Two operations are said to `conflict` if they both operate on the same data item and at least one of them is a write().

- We have 3 way to test conflict (how about resolution):

  - Operations (O1 in T1, O2 in T2), O1 alway precedes O2 or O2 alway precedes O1
  - Swap non-conflicting operations to get serial schedule
  - Builde precedence graph, check for cycles

- Write 1 -> Read 0 will make conflict.

### Buffer management Issues

- After crash (disk persist, mem is reset), we must
  - UNDO: remove effect of incomplete or aborted tranx "loser" that had not commited for preserving atomicity -> Logical
  - REDO: re-instating the effect of commited tranx "winner" for durability -> Physical, đảm bảo commited tranx được thực hiện trên disk
- STEAL: allow update made by an uncommitted tranx to overwrite the most recent commited value of data in non-volatile storage (MEM)
- NO-STEAL: you shouldn't evict dirty (updated) pages from the buffer pool if they are locked by an uncommitted transaction.
- FORCE: ensure that all update made by a tranx are reflected on RAM before tranx is allowed to commit.
  - another explain: On transaction commit, you should force dirty pages to disk
- NO-FORCE:

=> nếu sử dụng No-Steal, Force, chúng ta sẽ ít yêu cầu UNDO, REDO. but they restrict the flexibility of the buffer manager
=> Many buffer managers support STEAL, No-force, but we do the ez here, in lab 3.

### Two phase locking protocol

- Why we call it 2 phase locking

  - A transaction cannot release any locks until it has acquired all of its locks

- Conditions:

  - Before every read, acquire a shared lock
  - Before every write, acquire an exclusive lock (or upgrade a shared)
  - Release lock only after last lock has been acquire, all operaions on an object are finished (2pl.png + Correctness Intuition giải thích cho ý này)

- Correctness Intuition:

  - khi 1 transaction gặp lock point
    - 1. Các transaction khác không cầm được lock -> không thể gây conflict action cho tới khi T release lock.
    - 2. Transaction nào mà cầm được lock rồi, thì chỉ release lock cho tới khi hoàn thành các hành động gây conflict

- Problems:
  - Deadlocks: complex deadlocks are `POSSIBLE`
    - Solution: abort 1 of the transaction.
  - Cascading Aborts
    - Because the solution above, lead to another problem(2pl-cascade.png)

### Strict Two phase locking protocol

- Conditions:
  - Before every read, acquire a shared lock
  - Before every write, acquire an exclusive lock (or upgrade a shared)
  - Release Elock only after last lock has been acquire, all operaions on an object are finished
  - Release Xlock only after the transaction commits -> avoid cascading
    - another trans never read other transaction's uncommitted data
  - Commit order = serialization order.

### Rigorous Two phase locking protocol

- With Strict 2Pl

  - How does DBMS know a transaction no longer needs a lock?
  - Difficult, since transactions can be issued interactively

- Conditions:
  - Before every read, acquire a shared lock
  - Before every write, acquire an exclusive lock (or upgrade a shared)
  - Release all locks only after the transaction commits

### Phantom problems

- If we just lock the records -> we meet this problem for range query.
- Example:
  - tid 1 select > 100 (you get 1 record).
  - tid 2 create a record with val = 101.
  - => Now you wrong

#### Solving phantom, need a way to lock range

- next key locking. It just work for table that index on ID. If not, we can `lock page`.

### Optimistic Concurrency Control

- ALternative to locking for isolation
- Approach:
  - Store writes in a per-tranx buffer
  - Track read and write sets
  - At commit, check if transaction conflicted with earlier (concurrent) tranx
  - Abort tranx that conflict
  - Install writes at end of tranx
- “Optimistic” in that it does not block, hopes to “get lucky” arrive in serial interleaving

#### Tradeoff

- PROS:
  - No need to wait lock, no deadlock
- CONS:
  - Tranx conflict often restarted
  - Tranx can starve (never making progress)

#### Recent work

- Focus on OCC, because it have high throughput (>10M tranx / sec)
- E.g., https://people.eecs.berkeley.edu/~wzheng/silo.pdf

#### DEMO

- READ UNCOMMITTED doesn’t have conflicts between counts and
  updates
  – SERIALIZABLE xactions may have to block/abort on update due to
  concurrent readers

### Recovery recap

ref: https://dsg.csail.mit.edu/6.5830/2023/lectures/lec15.pdf

#### Write ahead logging

- ARIES normal operation with 2 key DataStructure

  - transaction table:
    - lastLSN: most recent log record writtenby that transaction
    - TID
  - dirty page table:
    - pgNo
    - recLSN: log record that first dirties the page

- You can make an example about how ARIES work with WAL in this ref above.

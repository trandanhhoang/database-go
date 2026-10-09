# Review GoDB: những chỗ sai ảnh hưởng tới việc hiểu Database

File này chỉ tập trung vào các lỗi **liên quan tới khái niệm DB** (buffer pool, lock, recovery, record id, query execution).
Các vấn đề kiểu log quá nhiều, nuốt lỗi `x, _ :=`... được bỏ qua vì không ảnh hưởng tới việc hiểu DB.

Các lỗi được đánh dấu ✅ đã được **chạy thực nghiệm** để xác nhận (bằng test tạm, không commit), output thật được dán kèm.

| # | Chủ đề DB | Chỗ sai | Xác nhận |
|---|-----------|---------|----------|
| 1 | Recovery: FORCE / STEAL | README ghi ngược với code | đọc code |
| 2 | Lock table, định danh page, deadlock | `isConflicted` chỉ so `pageNo`, bỏ qua file | ✅ |
| 3 | Halloween problem | `insert into t select * from t` chạy không dừng | ✅ |
| 4 | RecordID (địa chỉ vật lý của tuple) | Insert dùng chung con trỏ tuple → Rid của bảng gốc bị đổi | ✅ |
| 5 | Schema / TupleDesc | Tuple đọc từ heap file thiếu `TableQualifier` → test `limit` fail | ✅ |
| 6 | Thuật toán join | Nested loop join quá chậm → `TestBigJoinOptional` timeout | ✅ (test fail) |
| 7 | Lock vs Latch | Giữ mutex toàn cục trong lúc đọc disk | đọc code |
| 8 | Abort | Abort không được báo ra ngoài | đọc code |

---

## 1. FORCE / NO-STEAL — README đang ghi ngược

`README.md` ghi: *"NO-FORCE, STEAL"*. Nhưng code làm **FORCE + NO-STEAL**:

- **FORCE**: `CommitTransaction` ghi mọi page dirty ra disk *trước khi* commit xong (`buffer_pool.go`, vòng `flushPage`).
- **NO-STEAL**: khi buffer pool đầy, `GetPage` **chỉ evict page sạch**, gặp toàn page dirty thì trả lỗi `buffer pool is full of dirty pages`.

Đây là 2 câu hỏi quyết định DB có cần log để recovery hay không:

```
                     STEAL (được ghi page chưa commit ra disk?)
                       Không (NO-STEAL)        Có (STEAL)
                  ┌───────────────────────┬───────────────────────┐
 FORCE            │  Không cần UNDO       │  Cần UNDO             │
 (commit thì phải │  Không cần REDO       │  Không cần REDO       │
  ghi page ra?)   │  ◀── GoDB đang ở đây  │                       │
                  ├───────────────────────┼───────────────────────┤
 NO-FORCE         │  Không cần UNDO       │  Cần UNDO + REDO      │
                  │  Cần REDO             │  (Postgres, MySQL...) │
                  └───────────────────────┴───────────────────────┘
```

Giải thích:

- **Vì sao NO-STEAL thì không cần UNDO?** Page của txn chưa commit không bao giờ nằm trên disk.
  Nếu crash hoặc abort, chỉ cần bỏ page trong RAM đi. Đó chính là việc `clearMap()` làm khi abort: `delete(bp.pages, key)`.
- **Vì sao FORCE thì không cần REDO?** Commit xong thì dữ liệu chắc chắn đã nằm trên disk, nên crash sau commit không mất gì.
  *(Giả định của lab: không crash **giữa lúc** đang flush.)*
- **Cái giá phải trả:**
  - NO-STEAL: txn lớn sửa nhiều page hơn kích thước buffer pool là không chạy được (xem phần 3, lỗi `buffer pool is full of dirty pages`).
  - FORCE: mỗi commit phải ghi ngẫu nhiên nhiều page ra disk, rất chậm.

  DB thật vì vậy chọn **STEAL + NO-FORCE** và dùng **WAL (write-ahead log)** để có UNDO/REDO.

👉 Nên sửa README thành "FORCE, NO-STEAL", và ghi chú rằng STEAL/NO-FORCE là thứ DB thật làm nhờ WAL.

---

## 2. Lock table: page phải được định danh bằng (file, pageNo), không chỉ pageNo ✅

### Code hiện tại

```go
// buffer_pool.go — isConflicted
if otherPageLock.pageNo == pageNo && (otherPageLock.perm == WritePerm || perm == WritePerm) {
```

Chỉ so `pageNo`. Nhưng **page 0 của bảng `a` và page 0 của bảng `b` là 2 page khác nhau**.
Buffer pool đã dùng đúng `heapHash{FileName, PageNo}` làm key cho cache, nhưng lock table lại không dùng key này.

### Sơ đồ

```
 Đúng:                                   Code hiện tại:
 ┌─────────── a.dat ──────────┐          lock table chỉ thấy "page 0"
 │ page0 [X-lock: T1]  page1  │
 └────────────────────────────┘          T1: X-lock page 0 ─┐
 ┌─────────── b.dat ──────────┐                              ├─ "trùng page 0" → T2 phải chờ ❌
 │ page0 [S-lock: T2]  page1  │          T2: S-lock page 0 ─┘
 └────────────────────────────┘
 → không xung đột, T2 chạy ngay
```

### Thực nghiệm

T1 lấy `WritePerm` trên `a.dat` page 0, T2 đọc `b.dat` page 0:

```
A: T2 doc b.dat page0 BI CHAN boi lock cua T1 tren a.dat page0
```

### Hệ quả với deadlock detection

Code dựng **waits-for graph** (`waitTidLocks`) và tìm chu trình bằng DFS. Ý tưởng này đúng. Nhưng xung đột giả sẽ sinh **cạnh giả**,
dẫn tới **deadlock giả** và abort oan:

```
 T1 giữ X(a.p0)        T2 giữ X(b.p0)
 T1 muốn đọc c.p0  → bị coi là chờ T2   (vì T2 giữ "page 0")
 T2 muốn đọc d.p0  → bị coi là chờ T1   (vì T1 giữ "page 0")

        T1 ──chờ──▶ T2
         ▲          │
         └───chờ────┘      → "deadlock!" → abort, dù 4 page hoàn toàn khác nhau
```

### Cách sửa

So sánh bằng key (đã có sẵn trong `PageLock.key`):

```go
func (bp *BufferPool) isConflicted(key any, tid TransactionID, perm RWPerm) bool {
	...
	if otherPageLock.key == key && (otherPageLock.perm == WritePerm || perm == WritePerm) {
```

### Ôn lại: ma trận tương thích lock (Strict 2PL)

```
              đang giữ S    đang giữ X
 xin S          ✅ OK         ❌ chờ
 xin X          ❌ chờ        ❌ chờ
```

Nếu cùng một txn xin X trên page mình đang giữ S thì đó là **lock upgrade**. Code xử lý đúng bằng cách bỏ qua lock của chính mình (`if t == tid { continue }`).
Lưu ý: 2 txn cùng giữ S rồi cùng xin upgrade lên X là ca deadlock kinh điển, và waits-for graph sẽ bắt được.

---

## 3. Halloween problem ✅

### Thực nghiệm

Bảng `h` có **3 dòng**. Chạy `insert into h select * from h`, kết quả mong đợi là 6 dòng:

```
B: insert into h select * from h (3 dong ban dau): res=<nil> err=buffer pool is full of dirty pages numPages=31
```

Thay vì dừng ở 6 dòng, câu lệnh insert liên tục cho tới khi buffer pool đầy page dirty và chết.

### Vì sao

`HeapFile.Iterator` tính lại `f.NumPages()` **ở mỗi lần gọi**, và quét qua mọi slot. Dòng vừa insert sẽ được chính scan đó đọc lại:

```
 scan con trỏ ▼
 page0: [r1][r2][r3][  ][  ]...
        đọc r1 → insert r1' vào slot trống
 page0: [r1][r2][r3][r1'][ ]...
             ▼
        đọc r2 → insert r2'
 page0: [r1][r2][r3][r1'][r2']
                     ▼
        đọc r1' (dòng mới!) → insert r1'' ...  ♾️  vòng lặp không dừng
```

Tên gọi "Halloween problem" đến từ IBM System R năm 1976: câu lệnh *"tăng lương 10% cho ai lương < 25k"* chạy qua index trên cột lương.
Người được tăng lương bị dời về phía sau index, rồi lại được scan và tăng tiếp, cho tới khi ai cũng ≥ 25k.

### Cách DB thật xử lý

1. **Materialize trước**: đọc hết kết quả của child vào bộ nhớ hoặc file tạm, sau đó mới insert. Cách này đơn giản nhất cho GoDB:
   ```go
   // trong InsertOp.Iterator
   var buf []*Tuple
   for t, _ := ite(); t != nil; t, _ = ite() { buf = append(buf, t) }
   for _, t := range buf { iop.file.insertTuple(t, tid) }
   ```
2. **Snapshot / MVCC** (Postgres): mỗi tuple có `xmin`, tức txn đã tạo ra nó. Scan bỏ qua dòng do chính câu lệnh hiện tại tạo ra.

*Chỉ chụp `NumPages()` một lần lúc tạo iterator là **chưa đủ**: dòng mới vẫn có thể rơi vào slot trống ở page mà scan chưa đi qua.*

### Quan sát thêm: file phình ra dù txn thất bại

`numPages=31` dù txn đã lỗi. `insertTuple` ghi page rỗng mới **thẳng ra disk** (`flushPage`) bên ngoài transaction, nên abort không thu hồi được.
Hệ quả không sai dữ liệu (page rỗng), nhưng đây là ví dụ cho việc "thay đổi cấu trúc file" cũng cần được tính vào transaction/recovery.
DB thật cũng ghi log cho thao tác cấp phát page.

---

## 4. RecordID và chuyện dùng chung con trỏ tuple ✅

### Khái niệm

`RecordID = (pageNo, slotNo)` là **địa chỉ vật lý** của một dòng. Delete và index đều dựa vào nó để tìm đúng dòng.

### Code hiện tại

```go
// heap_page.go — insertTuple
h.tuples[i] = t          // lưu thẳng con trỏ của caller
h.tuples[i].Rid = rid    // và ghi đè Rid lên chính object đó
```

`InsertOp` truyền vào **chính con trỏ tuple đang nằm trong page cache của bảng nguồn**. Kết quả là một object tuple nằm ở 2 page, và Rid của bảng nguồn bị ghi đè.

### Thực nghiệm

Đọc 1 tuple từ bảng nguồn (`a.dat` trong output), insert nó sang bảng đích (`b.dat`, đã có 5 dòng), rồi xem lại tuple gốc trong bảng nguồn:

```
C: Rid cua tuple trong a.dat truoc={0 0}, sau khi insert sang b.dat={0 5}
```

### Sơ đồ

```
 Trước:                                   Sau insert:
 a.dat page0                              a.dat page0
 slot0 ──▶ Tuple{x,1, Rid=(0,0)}          slot0 ──┐
                                                  ├──▶ Tuple{x,1, Rid=(0,5)}  ← cùng 1 object!
 b.dat page0                              b.dat page0
 slot5  (trống)                           slot5 ──┘
```

### Hậu quả cụ thể

Chạy `insert into t select * from t`, rồi `delete from t where ...` với scan qua page 0:

- Tuple ở page 0 slot 0 giờ mang `Rid=(1,k)`.
- `deleteTuple` dùng Rid nên đi tới page 1 slot k và xoá **bản copy**. Bản gốc ở page 0 vẫn còn.
- Một bảng có thể **xoá nhầm dòng** chỉ vì một câu insert trước đó.

### Cách sửa

Copy tuple trước khi lưu vào page:

```go
nt := &Tuple{Desc: t.Desc, Fields: append([]DBValue(nil), t.Fields...)}
nt.Rid = rid
h.tuples[i] = nt
```

### Lưu ý thêm: slot bị đánh số lại khi ghi ra disk

`toBuffer` chỉ ghi các slot khác `nil` (dồn lại), còn `initFromBuffer` đọc vào slot `0..usedSlots-1`:

```
 RAM:  [A][nil][C]        →  disk: [A][C]  →  đọc lại: slot0=A, slot1=C
 Rid của C: (p,2)                                     Rid của C: (p,1)  ← đổi!
```

Spec của lab cho phép điều này, vì NO-STEAL nên page đang bị sửa không bao giờ bị đọc lại giữa chừng.
Nhưng nếu sau này bạn làm **index** (index lưu Rid) thì Rid phải **ổn định**. DB thật giữ nguyên slot (có bitmap hoặc slot directory), hoặc cập nhật index khi dời dòng.

---

## 5. TupleDesc của tuple không khớp với Descriptor của bảng ✅ (lý do test `limit` fail)

`TestParseEasy` fail ở query `select * from t limit 1+2` dù 3 dòng in ra **giống hệt** kết quả mong đợi. In ra descriptor để so:

```
PLAN {Fields:[{Fname:name TableQualifier:t ...} {Fname:age TableQualifier:t ...}]}
TUP  {Fields:[{Fname:name TableQualifier:  ...} {Fname:age TableQualifier:  ...}]}
```

- Parser gắn alias `t` vào descriptor của bảng (`setTableAlias`).
- Nhưng tuple mà `HeapFile.Iterator` trả ra vẫn mang descriptor cũ, không có qualifier.
- Các query khác pass vì `Project` hoặc `Agg` tạo tuple mới với descriptor đúng. `Limit` thì trả thẳng tuple của child nên lộ ra lỗi.

Khái niệm: **mỗi operator trong cây plan cam kết một schema đầu ra** (`Descriptor()`), và tuple nó trả ra phải đúng schema đó.
Cách sửa đơn giản: trong `HeapFile.Iterator`, gán `tuple.Desc = *f.Descriptor()` trước khi trả về.

---

## 6. Nested loop join vs Hash join (lý do `TestBigJoinOptional` timeout)

Code hiện tại là **nested loop join**: với mỗi dòng bên trái thì quét lại toàn bộ bảng phải.

```
 Nested loop: O(|L| × |R|)                Hash join: O(|L| + |R|)

 for l in L:                              Build:  for r in R: H[r.key] += r
   for r in R:          ← quét lại R            ┌────────────────┐
     if l.k == r.k:        |L| lần              │ key → [r, r..] │  (hash table)
       emit(l,r)                                └────────────────┘
                                          Probe:  for l in L:
                                                    for r in H[l.key]: emit(l,r)
```

Với `|L| = |R| = 100.000`: nested loop cần khoảng **10 tỷ** lần so sánh, hash join chỉ khoảng **200 nghìn**.

Đề yêu cầu không dùng quá `maxBufferSize` dòng trong bộ nhớ. Cách làm: build hash table theo **từng khối** `maxBufferSize` dòng của bên trái, mỗi khối quét bên phải một lần.
Đây là *block hash join*, chi phí `O(|L|/B × |R|)`, đúng như comment ở cuối `join_op.go`.

Bug nhỏ đi kèm: khi bảng trái rỗng, `Iterator` trả về `nil, nil`, tức **hàm iterator là `nil`**, và nơi gọi sẽ panic.
Nên trả về một hàm luôn trả `nil, nil`.

---

## 7. Lock vs Latch

Có 2 loại "khoá" khác nhau trong DB:

|            | **Lock** | **Latch** |
|------------|----------|-----------|
| Bảo vệ     | dữ liệu logic (page/tuple) cho **transaction** | cấu trúc dữ liệu trong RAM cho **thread** |
| Giữ bao lâu | tới khi commit/abort (Strict 2PL) | vài micro giây, chỉ trong 1 thao tác |
| Trong GoDB | `mapPageLocksByTid` | `bp.mu`, `f.mu` |
| Deadlock   | phát hiện bằng waits-for graph, abort txn | phải **tránh** bằng thứ tự lấy latch |

Chỗ chưa đúng tinh thần latch: `GetPage` giữ `bp.mu` (latch toàn cục) **trong lúc đọc disk** (`file.readPage`).
Một thread chờ I/O thì mọi thread khác cũng phải chờ, kể cả thread chỉ cần page đã có sẵn trong cache.
DB thật nhả latch trước khi làm I/O (đánh dấu page "đang load"), hoặc chia buffer pool thành nhiều partition, mỗi partition một latch.

Phần chờ lock thì code làm đúng: `waitWhenConflict` **nhả** `bp.mu` trước khi sleep. Nếu không làm vậy, txn đang giữ lock sẽ không bao giờ lấy được latch để commit và nhả lock.

---

## 8. Abort phải được báo ra ngoài

Phần lớn chỗ nuốt lỗi có thể bỏ qua. Riêng lỗi **abort** thì liên quan trực tiếp tới khái niệm transaction:

```go
// insert_op.go
tuple, _ := ite()      // nếu child bị abort, lỗi bị bỏ, tuple = nil
for tuple != nil { ... }
return count            // → báo "đã insert N dòng"
```

Khi bị chọn làm nạn nhân deadlock, `clearMap()` đã bỏ hết page dirty và nhả lock, tức txn **đã bị rollback**.
Nhưng operator vẫn trả về "thành công, N dòng". Phía client sẽ tưởng dữ liệu đã được ghi.

Nguyên tắc: khi DB abort một txn, **client phải biết** (Postgres trả về `ERROR: deadlock detected`) để còn retry.
Atomicity vẫn đúng (không có dữ liệu nửa vời), nhưng **thông báo kết quả thì sai**.

---

## Gợi ý thứ tự sửa nếu muốn

1. Phần 5: 1 dòng, làm `TestParseEasy` xanh lại.
2. Phần 2: so sánh bằng `key`, sửa xung đột giả và deadlock giả.
3. Phần 4: copy tuple khi insert.
4. Phần 3: materialize trong `InsertOp`.
5. Phần 1: sửa README.
6. Phần 6: hash join (bài tập optional, nhưng học được nhiều nhất).

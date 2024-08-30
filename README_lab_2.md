# README - Labs 2

- The main function of lab2 is to support: insert, delete record; filter, join, aggregate.

## Filter and Join (Implement)

- Filter: this operator is return tuples that satisfy a predicate.
- Let look Filter in OOP.

  - Fields:
    - op BoolOp
    - left Expr
    - right Expr
    - child Operator
    - getter func(DBValue) T
  - Let make question, why we need Operator interface, why File, ~~Page~~, Filter, Join, Aggre, ... need implement this interface ???
    - Because, from a tuple, we want loop through each operation.
    - eg:
      ```sql
      SELECT column_name, SUM(column_to_sum) AS total
      FROM table_name
      WHERE filter_column = filter_value
      GROUP BY column_name;
      ```
    - We can think as: Aggre(Filter(File))

- Join: this opeator join tuples for its children that match equality predicate.
  - Implemented by nested loop join
  - Can be optimize with `sort merge join` or `hash join` -> pass TestBigJoinOptional
  - Learn about joins: https://dsg.csail.mit.edu/6.5830/2023/lectures/lec11.pdf

## Aggregates (Implement)

- Aggregate: sum, count, min, max, avg
- GroupBy: empty, []

- eg: we have 4 tuple

#### Table user

| name  | age |
| ----- | --- |
| sam   | 25  |
| hoang | 999 |
| hoang | 999 |
| hoang | 999 |

####

```sql
select count(*) from user;
```

- Aggregate = count; GroupBy = empty
  - count = 4

####

```sql
select name,count(*) from user
group by name;
```

- Aggregate = count; GroupBy = `[name]`
  - sam, count = 1
  - hoang, count = 3

#### Implement

- Let look Aggregates in OOP

- fields:

  - groupByFields []Expr // empty; or list of group by
  - newAggState []AggState // sum, count, min, max, avg
  - child Operator // file that can iter() all tuples

- method:
  - Iterator(tid TransactionID) (func() (\*Tuple, error), error)
    - This method need hour to implement, you need to read test, write draft and handle carefully
    - Let make an example for this case:
      - select name,count(\*) from user
        group by name; with table above.
    - We need some internal field below, i will explain one by one:
      - groupByList = []Tuple
      - mapAggStates map[any]\*[]AggState
    - Let walk through each record
      - The first record (sam, 25)
        - groupByList = [{name,"sam"}]
        - mapAggStates = {key:"name+sam", value:AggState = [CountAgg(1)]}
      - The second record (hoang, 999)
        - groupByList = [{name,"sam"},{name,"hoang"}]
        - mapAggStates = [{key:"name+sam", value:AggState = [CountAgg(1)]}, {key:"name+hoang", value:AggState = [CountAgg(1)]}]
      - The third record (hoang, 999)
        - groupByList = [{name,"sam"},{name,"hoang"}]
        - mapAggStates = [{key:"name+sam", value:AggState = [CountAgg(1)]}, {key:"name+hoang", value:AggState = [CountAgg(2)]}]
      - The fourth record (hoang, 999)
        - groupByList = [{name,"sam"},{name,"hoang"}]
        - mapAggStates = [{key:"name+sam", value:AggState = [CountAgg(1)]}, {key:"name+hoang", value:AggState = [CountAgg(3)]}]
    - Finalize: we want 2 tuple like below
      - [{name,"sam"},{count,1}]
      - [{name,"hoang"},{count,3}]
  - Write new test TestGbyNameWithCountAggAndSumAgg
    - You can think it as select name,count(name), sum(age) from user group by name;
    - Finalize: we want 3 tuple like below
      - [{name,"sam"},{count,1},{sum,25}]
      - [{name,"hoang"},{count,3},{sum,2997}]
  - Write new test TestGbyNameAndAddressWithCountAggAndSumAgg
    - This test need a new table

####

| name  | age | address |
| ----- | --- | ------- |
| sam   | 25  | mit     |
| hoang | 999 | kbang   |
| hoang | 999 | saigon  |
| hoang | 999 | kbang   |

####

    - You can think it as select name,count(name), sum(age) from user group by name,address;
    - Finalize: we want 3 tuple like below
      - [{name,"sam"},{address,"mit"},{count,1},{sum,25}]
      - [{name,"hoang"},{address,"saigon"},{count,1},{sum,999}]
      - [{name,"hoang"},{address,"kbang"},{count,2},{sum,1998}]

## Insert and Delete (Implement)

- If you have done 2 task above, this one is trivial.

## Projection (Implement)

- Project iterates through its child, selects some of each tuple's fields, and returns them. Optionally, you will need to support the DISTINCT keyword, meaning that identical tuples should be returned only once. For example, given a dataset like:
- It support for

  - "SELECT name FROM table"
  - "SELECT DISTINCE name FROM table"

- Until now, I know the reason, why we need method `Descriptor()` for interface `Operator`, this method use for describe what is the tupleDesc of the `tuple` that return by method Iterator(), for example:

  - join_op.op merge tuple left and right with each other, so Descriptor return leftOp.Descriptor().merge(rightOp.Descriptor())
  - filter_op.go just return the child, because Iterator() don't change anything in tuple
  - agg_op.go, return depend on groupByFields (to show value after groupBy), and newAggState (value of the Agg - sum, count, avg, ...)

- If you have done 3 task above, this one is trivial.

## Order by

- It needs to support ordering by more than one field, with each field in either ascending or descending order.
- eg:

```sql
SELECT name, age, salary
  FROM table
  ORDER BY name ASC, age DESC
```

- `Now, I have a question, if heapfile have pages, each page have tuples. For example, with billion tuples here, how can we create a memory to sort all tuples in OrderBy() here ???`

  - I have found this video valuable here
    - https://www.youtube.com/watch?v=F9XmmS8rL4c
  - We can use K-Way datastructure
    - It kind like merge sort, each time sort, we save the result in disk.

- But we just implement the easy one here, put all data in mem.

  - I fail many time because syntax of Go with closure method

  ```go
  for idx, expr := range o.orderBy {
  	tempIdx := idx
  	tempExpr := expr
  	lessFuncs = append(lessFuncs, func(p1, p2 *Tuple) bool {
  		order, _ := p1.compareField(p2, tempExpr)
  		if o.ascendings[tempIdx] {
  			return order < OrderedEqual
  		}
  		// we want descending
  		// if p1 > p2 -> return true
  		// if p1 <= p2 -> return false
  		return order > OrderedEqual
  	})
  }
  ```

  - If you don't capture idx & expr here. It will use the last one for all closure method
    - Because closure just capture the reference, not the value.

## Limit (Implement - easy one)

## Run a simple query test

- SELECT COUNT(\*) FROM t, t2 WHERE t.name = t2.name AND t.age > 30.

- I stuck when work with file, bp save file with name "t" and "t2", but with catalog, file name become ".//t" and ".//t2". So I need an hour to debug and fixed it.
- After fixing bug above, I passed this test.

## Query Parser

- select name,age,getsubstr(epochtodatetimestring(epoch() - age*365*24*60*60),24,4) birthyear from t"
- The first test. I failed, let find the reason.

  - I can't get the birthday, we need fix project_op.go
  - It still error, because this lab is created from 2023, so the result is not match
  - After update the result file. New error come from tuple.go
    - Tuple equal(Tuple) method no need to compare 2 RID
  - DONE

- The second test. Now I know why Descriptor is important.

  - Query select sum(age + 10) , sum(age) from t
    - In project: outputNames is sum(age), sum(age) -> first error
    - In tuple after agg: Fname = sum(t.age)0, Fname = sum(t.age)1
  - How can I resolve this error ?
    - It depend on the parser of this lab, I will try to read some code, if it take time. I will ignore it
    - I check at file q2-easy-result.csv -> the columns actually: sum(age),sum(age). HF
      - outputNames is sum(age), sum(age) -> Not error, we must adapt with it.
    - Fix in project_op, I passed this test

- The third test, I think I will pass in the first try.

  - Sadly, I fail, but I know the reason.

  ```go
  func (f *FuncExpr) GetExprType() FieldType {
    fType, exists := funcs[f.op]
    //todo return err
    if !exists {
      return FieldType{f.op, "", IntType}
    }
    ft := FieldType{f.op, "", IntType}
    for _, fe := range f.args {
      fieldExpr, ok := (*fe).(*FieldExpr)
      if ok {
        ft = fieldExpr.GetExprType()
      }
    }
  return FieldType{ft.Fname, ft.TableQualifier, fType.outType}
  }
  ```

  - This method is a reason. It want (min + max), but the result is just max. So I can just return the max -> FAIL
  - I fix the project_op.go. I passed test 3 now, but i need to confirm the test at project_op_test and test 1,2 don't fail again.

- Great, everything pass.
- The fourth test, I fail because TupleDesc with TableQualifier is not "t"
  - I will end this lab from here. I don't want to deal with this anymore. I want to learn about transaction, lock, .... So I will move to lab 3. If lab 3 still stuck by this error, I will resolve it then.

## SQL Query Optimization. Why Is It So Hard To Get Right?

- https://dsg.csail.mit.edu/6.5830/2023/lectures/lec10.pdf
- Ví dụ, bạn có 1 câu query như sau,
- SELECT Average(Rating) FROM reviews WHERE mid = 932 and Rating > 9
- 1. Enumerate logically equivalent plans
  - scan + filter | index mid | index rating
  - bạn có thể filter có 2 cách để xử lý, SCAN + FILTER qua tất cả các record, hoặc dùng index ở cac column khac nhau.
- 2. Enumerate all alternative physical query plans
  - nghĩa là thứ tự select, join có thể hoán vị cho nhau
- 3. Estimate the cost of each of the alternative query plan
  - => 9 logic \* 36 physic = 324 alternative plans. We use `dynamic programming` to enumerating the entire plan.
- 4. Run the plan with lowest estimated overall cost
  - In general, there is not enough space in the catalogs to store summary statistics for each distinct attribute value
  - Solution: histogram. thực sự thì mình chưa đọc đoạn này.

## Estimating Costs

- I/O times: cost of reading pages from mass storage
- CPU time: cost of applying predicate and operating on tuple in memory
- For a parallel database, the cost of redistributing/ shuffling rows must also be considered (EXTENDS - can ignore now)

## Real example - which plan is cheaper.

- Pre-condition:

  - 1. Table `REVIEW` have field mid, date (order by date)
  - 2. scan 100MB/second
  - 3. CPU time = 0.1 μs (10^-6s)/row for `FILTER predicate`
  - 4. CPU time = 0.1 μs (10^-6s)/row for `AVG predicate`
  - 5. time Random disk I/O = 0.003 second per diskI/O

- Query: SELECT Average(Rating) FROM reviews WHERE mid = 932

  - scan + filter

    - 100k pages. each pages have 100 records => 8 second
    - filter is applied to 10M rows
      - 100 rows matched => 1 second
    - AVG apply for 100 rows -> ignore
    - => 9 seconds

  - index MID
    - 100 rows are retrieved using `MID index` => .3 seconds
    - Why: Khi dữ liệu không được sắp xếp theo MID, các bản ghi có mid = 932 có thể nằm rải rác ở nhiều nơi khác nhau trên đĩa.
    - giả sử chúng ta cần đọc tối đa 100 page, vậy thì cần 0.3s (c4 - condition 4)

- What happen if not 100 row, it becomse 10.000 rows
  - now index need 30 second (check image query-predicate.png)

## So sanh' cac loai. JOIN.

- sort-merge join, index-nested join, nested loop join.

### Examplle

- SELECT \* FROM Reviews WHERE 7/1 < date < 7/31 AND rating > 9

#### Precondition

- Reviews: 1M records
- 7/1 < date < 7/31: selectivity factor = 0.1 => 100.000 records
- rating > 9: selectivity factor = 0.01 => 10.000 records

#### tuỳ thuộc vào selectivity factor, mà các loại join có sự phù hợp khác nhau

- If 2 predicate above are not correlated
  - row matched = .1\*.01\*1M = 1000 rows
- If 2 predicate above are correlated

  - row matched = .1\*1M = 100000 rows

- INL(0,0001 trở xuống)
- NL (0,0001 - 0.001)
- SM (từ 0.001 trở lên)

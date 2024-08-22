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

## Limit

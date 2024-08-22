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

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

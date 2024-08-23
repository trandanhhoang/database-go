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

## For detail, you can read README_lab\_[1-3].md for more information.

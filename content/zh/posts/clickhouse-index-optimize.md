---
title: "ClickHouse稀疏索引原理解析"
date: 2022-06-13T16:23:14+08:00
draft: true
Tags: [ClickHouse]
---



问个问题，如何优化一条SQL语句？我们首先想到的肯定是建索引。对于ClickHouse也不例外，尤其是稀疏索引（类似传统数据库中的主键索引）对性能的影响非常大。在下文中，我将结合例子对稀疏索引进行详细分析。

> 注：本文内容大量参考[官方文档](https://clickhouse.com/docs/en/guides/improving-query-performance)，如果有余力，强烈建议先行阅读

## 数据准备

这里，我将就我比较熟悉的时序数据进行距离，首先通过如下SQL建表：

```
-- create dabase
create database test;
use test;
-- create table
CREATE TABLE cpu
(
    `hostname`     String,
    `reginon`      String,
    `datacenter`   String,
    `timestamp`    DateTime64(9,'Asia/Shanghai') CODEC (DoubleDelta),
    `usage_user`   Float64,
    `usage_system` Float64
)
    ENGINE = MergeTree() PRIMARY KEY tuple();

optimize table cpu_ts final ;
```

并将本地的样本数据导入：

```shell
cat example/output.csv |clickhouse-client -d test -q 'INSERT into cpu FORMAT CSV'
```

查询行数：`select count(*) from cpu;`，一共1209600行

> [optimize table](https://clickhouse.com/docs/en/sql-reference/statements/optimize/) 会强制进行merge之类的操作，使其达到最终状态

### 查询某段时间范围内的CPU使用率

SQL如下:

```sql
select ts, avg(usage_user) from cpu where timestamp > '2022-01-15 06:59:59.000000000' and timestamp < '2022-01-15 07:59:59.000000000' group by toStartOfMinute(timestamp) as ts order by ts;
```

统计结果如下：

```shell
60 rows in set. Elapsed: 0.025 sec. Processed 1.21 million rows, 9.72 MB (48.68 million rows/s., 391.19 MB/s.)
```

> 注意：clickhouse客户端对每次查询会给出简要的性能数据，便于用户进行简单分析

可以看到，尽管我们只查询了一个小时范围内的数据，但是依然扫描121万行数据，也就是进行了一次**全表扫描！**

显然，这样的是不够的，我们需要建立合理的稀疏索引，而它将显著提高查询性能。

## 稀疏索引

稀疏索引(Sparse Primary Indexes)，功能上类似于MySQL中的主键索引，不过实现原理是截然不同的。长话短说，如若建立类似于B+树那种面向具体行的索引，索引将占用大量内存和磁盘并且影响写入性能。而且，也无需精确定位到行粒度。

因此，其总体设计上，将数据块按组（**粒度**）划分，并通过索引（**mark**）标记，这种设计使得索引的内存占用足够小。同时仍能显著提高查询性能，尤其是OLAP场景下的大数据量的范围查询和数据分析。

### 配置Primary Key

稀疏索引可通过[primary key](https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree/#mergetree-query-clauses)指定，让我们通过primary key来优化示例中的cpu表:

```
-- create table with timestamp as primary key
CREATE TABLE cpu_ts
(
    `hostname`     String,
    `reginon`      String,
    `datacenter`   String,
    `timestamp`    DateTime64(9,'Asia/Shanghai') CODEC (DoubleDelta),
    `usage_user`   Float64,
    `usage_system` Float64
)
    ENGINE = MergeTree() PRIMARY KEY (timestamp);

-- insert data from cpu
insert into cpu_ts select * from cpu;

optimize table cpu_ts final ;
```

执行相同的分析SQL：

```sql
select ts, avg(usage_user) from cpu_ts where timestamp > '2022-01-15 06:59:59.000000000' and timestamp < '2022-01-15 07:59:59.000000000' group by toStartOfMinute(timestamp) as ts order by ts;
```

输出如下：

```shell
60 rows in set. Elapsed: 0.023 sec. Processed 5.38 thousand rows, 86.02 KB (236.19 thousand rows/s., 3.78 MB/s.)
```

可以看到同样的结果，**仅扫描5千多行!** 下面，让我们通过EXPALIN来分析：

```sql
EXPLAIN indexes = 1 select ts, avg(usage_user) from cpu_ts where timestamp > '2022-01-15 06:59:59.000000000' and timestamp < '2022-01-15 07:59:59.000000000' group by toStartOfMinute(timestamp) as ts order by ts;
```

输出如下：

```
┌─explain────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ Expression (Projection)                                                                                                            │
│   MergingSorted (Merge sorted streams for ORDER BY)                                                                                │
│     MergeSorting (Merge sorted blocks for ORDER BY)                                                                                │
│       PartialSorting (Sort each block for ORDER BY)                                                                                │
│         Expression (Before ORDER BY)                                                                                               │
│           Aggregating                                                                                                              │
│             Expression (Before GROUP BY)                                                                                           │
│               Filter (WHERE)                                                                                                       │
│                 SettingQuotaAndLimits (Set limits and quota after reading from storage)                                            │
│                   ReadFromMergeTree                                                                                                │
│                   Indexes:                                                                                                         │
│                     PrimaryKey                                                                                                     │
│                       Keys:                                                                                                        │
│                         timestamp                                                                                                  │
│                       Condition: and((timestamp in (-inf, '1642204799.000000000')), (timestamp in ('1642201199.000000000', +inf))) │
│                       Parts: 1/1                                                                                                   │
│                       Granules: 1/148                                                                                              │
└────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

17 rows in set. Elapsed: 0.003 sec. 
```



可以看到，借助与稀疏索引，我们仅仅扫描了一个粒度的数据，**优化效果十分明显**。接下来，我们将结合示例来介绍稀疏索引的底层实现原理

### 实现原理



## 参考

- https://github.com/timescale/tsbs
- https://clickhouse.com/docs/en/guides/improving-query-performance

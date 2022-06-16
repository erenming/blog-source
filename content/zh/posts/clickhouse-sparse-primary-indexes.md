---
title: "ClickHouse系数索引介绍"
date: 2022-06-13T16:23:14+08:00
draft: true
Tags: [ClickHouse]
---

> 本文已作为ClickHouse中文版本文档贡献，[点击查看详情]()

我们将通过本指南深入理解ChlickHouse索引。我们将举例并讨论如下细节：

- [ClickHouse的索引与传统关系型数据库管理系统的关系]()
- ClickHouse是如何构建与使用表级的稀疏索引
- ClickHouse索引的最佳实践

!!! info "Header"

​    本指南主要讨论ClickHouse的稀疏索引。

​    关于ClickHouse的[二级跳数索引]()，可查阅[教程]()

## 数据集

在整个指南中，我们将使用一份匿名网站流量的数据集作为数据样本

- 我们将从数据样本集中选择一份887万行子集。
- 887万条事件，未压缩的情况下大约为700MB。存入ClickHouse并压缩后的大小为200mb。
- 在我们的子集中，每行都包括3列表示网站用户（**UserID**列）在某个特定时间点（**EventTime**列）访问了某个地址（**URL**列）

通过这些列，我们已经可以假想一些典型的网站分析查询了，例如：

- “某个用户访问次数最多的10个地址是哪些？”
- “某个地址被哪10个用户频繁地访问？”
- “某个用户最喜欢在哪个时间段(比如，一周中的哪几天)访问某个地址”

# 测试机器

文档中给出的所有运行数字，均由运行在一台 MacBook Pro with the Apple M1 Pro chip and 16GB of RAM 上的Clickhouse 22.2.1实例给出

# 一次全表扫描

为了能了解一条不带有稀疏索引的查询是如何基于我们的数据集执行的，我们通过执行如下SQL DDL语句创建一张表（以MergeTree作为表引擎）：

```sql
CREATE TABLE hits_NoPrimaryKey
(
    `UserID` UInt32,
    `URL` String,
    `EventTime` DateTime
)
ENGINE = MergeTree
PRIMARY KEY tuple();
```

接下来通过下述SQL将我们的数据集插入到表中。为了能加载存储在clickhouse.com的数据集，需要使用[URL 表函数]() 并与 [模式推断]() 结合：

```sql
INSERT INTO hits_NoPrimaryKey SELECT
   intHash32(c11::UInt64) AS UserID,
   c15 AS URL,
   c5 AS EventTime
FROM url('https://datasets.clickhouse.com/hits/tsv/hits_v1.tsv.xz')
WHERE URL != '';
```

执行返回如下：

```
Ok.

0 rows in set. Elapsed: 145.993 sec. Processed 8.87 million rows, 18.40 GB (60.78 thousand rows/s., 126.06 MB/s.)
```

ClickHouse客户端的输出向我们表名，上述语句一共往表中插入了887万行。

最后，为简化后续讨论并使得图表和结果保持一致性，我们通过使用FINAL关键字[优化]()数据表：

```sql
OPTIMIZE TABLE hits_NoPrimaryKey FINAL;
```





# 参考

- https://clickhouse.com/docs/en/guides/improving-query-performance/sparse-primary-indexes/
- https://clickhouse.com/docs/en/guides/improving-query-performance/skipping-indexes

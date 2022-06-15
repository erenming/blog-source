---
title: "ClickHouse索引介绍与优化建议"
date: 2022-06-13T16:23:14+08:00
draft: true
Tags: [ClickHouse]
---

总所周知，ClickHouse最值得人称赞的是写入性能，然而其查询性能其实也可以非常好，不过得针对不同的Schema合理建立不同的索引，尤其是稀疏索引

# 稀疏索引

英文名：Sparse Primary Indexes，概念上可以类比于MySQL中的主键索引，不过实际上是两种完全不同的东西。

> 样本数据：来自官方文档的样本数据
>
> 软硬件：MBP， 2.3 GHz Dual-Core Intel Core i5，16 GB 2133 MHz LPDDR3，ClickHouse 22.2.1	q
>
> 

## 介绍



## Generic exclusion search algorithm





# 参考

- https://clickhouse.com/docs/en/guides/improving-query-performance/sparse-primary-indexes/
- https://clickhouse.com/docs/en/guides/improving-query-performance/skipping-indexes

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

> - output.csv是时序数据样本，时间间隔为1秒，包含了从2022-01-01 08:00:00到2022-01-15 07:59:59一共1209600条记录
>
> - [optimize table](https://clickhouse.com/docs/en/sql-reference/statements/optimize/) 会强制进行merge之类的操作，使其达到最终状态

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

## 稀疏主键索引

稀疏主键索引(Sparse Primary Indexes)，以下简称主键索引。其功能上类似于MySQL中的主键索引，不过实现原理是截然不同的。长话短说，如若建立类似于B+树那种面向具体行的索引，索引将占用大量内存和磁盘并且影响写入性能。而且，也无需精确定位到行粒度。

因此，其总体设计上，将数据块按组（**粒度**）划分，并通过索引（**mark**）标记，这种设计使得索引的内存占用足够小。同时仍能显著提高查询性能，尤其是OLAP场景下的大数据量的范围查询和数据分析。

### 配置Primary Key

索引可通过[primary key](https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree/#mergetree-query-clauses)指定，让我们通过primary key来优化示例中的cpu表:

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
ENGINE = MergeTree()
PRIMARY KEY (timestamp)
ORDER BY (timestamp, hostname);

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
60 rows in set. Elapsed: 0.004 sec. Processed 12.29 thousand rows, 196.58 KB (3.21 million rows/s., 51.31 MB/s.)
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



可以看到，借助主键索引，我们仅仅扫描了一个粒度的数据，**优化效果十分明显**。

### 实现原理

#### 数据排布

ClickHouse底层数据是按照列存储，先让我们来看下cpu_ts表的某个part`all_1_1_0/`在磁盘上的存储结构：

> all_1_1_0：{分区}\_{最小block数}\_{最大block数}\_{表示经历了几次merge}

```shell
-rw-r----- 1 clickhouse clickhouse  589 Jun 26 05:42 checksums.txt
-rw-r----- 1 clickhouse clickhouse  179 Jun 26 05:42 columns.txt
-rw-r----- 1 clickhouse clickhouse    7 Jun 26 05:42 count.txt
-rw-r----- 1 clickhouse clickhouse  55K Jun 26 05:42 datacenter.bin
-rw-r----- 1 clickhouse clickhouse 3.4K Jun 26 05:42 datacenter.mrk2
-rw-r----- 1 clickhouse clickhouse   10 Jun 26 05:42 default_compression_codec.txt
-rw-r----- 1 clickhouse clickhouse  34K Jun 26 05:42 hostname.bin
-rw-r----- 1 clickhouse clickhouse 3.4K Jun 26 05:42 hostname.mrk2
-rw-r----- 1 clickhouse clickhouse 1.2K Jun 26 05:42 primary.idx
-rw-r----- 1 clickhouse clickhouse  51K Jun 26 05:42 reginon.bin
-rw-r----- 1 clickhouse clickhouse 3.4K Jun 26 05:42 reginon.mrk2
-rw-r----- 1 clickhouse clickhouse 147K Jun 26 05:42 timestamp.bin
-rw-r----- 1 clickhouse clickhouse 3.4K Jun 26 05:42 timestamp.mrk2
-rw-r----- 1 clickhouse clickhouse 2.5M Jun 26 05:42 usage_system.bin
-rw-r----- 1 clickhouse clickhouse 3.4K Jun 26 05:42 usage_system.mrk2
-rw-r----- 1 clickhouse clickhouse 2.5M Jun 26 05:42 usage_user.bin
-rw-r----- 1 clickhouse clickhouse 3.4K Jun 26 05:42 usage_user.mrk2
```

其中主要文件如下:

- *.bin：每个表中的列上的值压缩后的数据文件
- column.txt：此表中所有列以及每列的类型
- default_compression_codec.txt：数据文件默认的压缩算法
- primary.idx：数据索引，后文详解
- *.mrk2：数据标记，后文详解
- count.txt：此part中数据的行数

此外通过指定的PRIMARY KEY，其数据行会以timestamp升序排列并生成主键索引。同时根据ORDER BY的定义，在按PRIMARY KEY指定的前缀排序，数据继续按后缀的字段顺序排列(注意，PRIMARY KEY必须是ORDER BY的前缀)。

#### 索引建立

在ClickHouse中，列中的值是按照组(粒度，默认8192行)进行划分的。**粒度**，换句话说，就是ClickHouse中的一组不可划分的最小数据单元。

当需要分析数据时，ClickHouse会读取整个粒度内的所有数据，而不是一行一行地读取（批处理的思想无处不在哈）。因此对于cpu_ts表内的数据，会被划分为148个组(ceil(1210000/8192=148)，其整体**数据排布**如下所示：



从上图可以看出，表内数据被按照每8192行划分为148个粒度，每个组内取PRIMARY KEY中指定的最小数据作为标记（标记可以理解为组的编号），存储在`primary.idx`中。我们使用od命令将primary.idx打印出来：

```shell
0000000  1640995200000000000  1641003392000000000
0000020  1641011584000000000  1641019776000000000
# 粒度数组：[1640995200000000000, 1641003392000000000, 1641011584000000000, 1641019776000000000]
```

> (1642151554000000000-1642143362000000000)/1000000000 = 8192

当我们根据timestamp字段进行过滤时，ClickHouse会通过二分搜索算法对primary.txt进行搜索，找到对应的组标记，从而实现快速定位。对于示例查询语句，ClickHouse定位到的标记是148，那么如果通过标记反向找到实际的数据块呢？

#### 数据定位

ClickHouse引入了`{column_name}.mrk`文件，用来存储粒度编号与压缩前后数据的位置(offset)，如下图所示：



```shell
0000000                    0                    0
0000020                 8192                 1071
0000040                    0                 8192

# (0, 0, 8192)为一组，表示压缩后偏移量为0，压缩前偏移量为0，粒度内一共8192行
# (1071, 0, 8192)为一组，表示压缩后偏移量为1071， 压缩前偏移量为0，粒度内一共8192行
```

前文中，我们通过primary.idx已经拿到了具体的粒度编号，接着我们通过编号在{column}.mrk中定位到数据压缩前后的偏移量，

然后通过**压缩后的**偏移量将数据文件(*.bin)中的相关数据块加载到内存，然后再根据**压缩前的**偏移量将相关的数据发送到分析引擎。图示如下：





## 参考

- https://github.com/timescale/tsbs
- https://clickhouse.com/docs/en/guides/improving-query-performance
- https://zhuanlan.zhihu.com/p/397411559
- https://stackoverflow.com/questions/65198241/whats-the-process-of-clickhouse-primary-index

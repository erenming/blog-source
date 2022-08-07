---
title: "Clickhouse MergeTree解读"
date: 2022-08-01T23:21:05+08:00
draft: true
---

在ClickHouse的表引擎种，当属**MergeTree（合并树）**最为常用也最为完备，实际使用中绝大部分场景都会使用它。因此要真正的深入理解ClickHouse的底层实现，那么最重要的就是吃透**MergeTree**。

> MergeTree写入一批数据，数据以数据片段的形式落盘，并由后台线程在某个时刻进行合并，合并后的片段还能继续合并，如此往复，因而由此得名合并树

# 存储结构

MergeTree在磁盘上是以分区的形式排布，即结构如下所示

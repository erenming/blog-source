---
title: "浅析Go内存分配器的实现"
date: 2021-06-15T21:08:09+08:00
draft: false
tags: ["Go"]
---

# 为什么需要内存分配器？

总说周知，内存作为一种相对稀缺的资源，在操作系统中以*虚拟内存*的形式来作为一种内存抽象提供给进程，这里可以简单地把它看做一个连续的地址集合`{0, 1, 2, ..., M}`，由栈空间、堆空间、代码片、数据片等地址空间段组合而成，如下图所示(出自[CS:APP3e, Bryant and O'Hallaron](https://csapp.cs.cmu.edu/)的第9章第9节)

<img src="https://raw.githubusercontent.com/erenming/image-pool/master/blog/image-20210525231530967.png" alt="image-20210525231530967" style="zoom:50%;" />



这里我们重点关注Heap（堆），堆是一块动态的虚拟内存地址空间。在C语言中，我们通常使用`malloc`来申请内存以及使用`free`来释放内存，也许你想问，这样不就足够了吗？但是，这种手动的内存管理会带来很多问题，比如：

1. 给程序员带来额外的心智负担，必须得及时释放掉不再使用的内存空间，否则就很容易出现内存泄露
2. 随着内存的不断申请与释放，会产生大量的[内存碎片](https://en.wikipedia.org/wiki/Fragmentation_(computing))，这将大大降低内存的利用率

因此，正确高效地管理内存空间是非常有必要的，常见的技术实现有Sequential allocation, Free-List allocation等。那么，在Go中，内存是如何被管理的呢？

> 注：此为Go1.13.6的实现逻辑，随版本更替某些细节会有些许不同

# 实现原理

Go的内存分配器是基于[TCMalloc](https://google.github.io/tcmalloc/design.html#spans)设计的，因此我建议你先行查阅，这将有利于理解接下来的内容。

大量工程经验证明，程序中的小对象占了绝大部分，且生命周期都较为短暂。因此，Go将内存划分为各种类别(Class)，并各自形成Free-List。相较于单一的Free-List分配器，分类后主要有以下优点：

- 其一方面减少不必要的搜索时间，因为对象只需要在其所属类别的空闲链表中搜索即可

- 另一方面减少了内存碎片化，同一类别的空闲链表，每个对象分配的空间都是一样大小(不足则补齐)，因此该链表除非无空闲空间，否则总能分配空间，避免了内存碎片

那么，Go内存分配器具体是如何实现的呢？接下来，我将以自顶向下的方式，从宏观到微观，层层拨开她的神秘面纱。

# 数据结构

首先，介绍Go内存分配中相关的数据结构。其总体概览图如下所示：

![go-mem-alloctor](https://raw.githubusercontent.com/erenming/image-pool/master/blog/go-mem-alloctor.png)

### heapArena

在操作系统中，我们一般把堆看做是一块连续的虚拟内存空间。

Go将其划分为数个相同大小的连续空间块，称之`arena`，其中，heapArena则作为arena空间的管理单元，其结构如下所示：

```go
type heapArena struct {
  bitmap [heapArenaBitmapBytes]byte
  spans [pagesPerArena]*mspan
  ...
}
```

- bitmap: 表示arena区域中的哪些地址保存了对象，哪些地址保存了指针
- spans: 表示arena区域中的哪些操作系统页(8K)属于哪些mspan

### mheap

然后，则是核心角色mheap了，它是Go内存管理中的核心数据结构，作为全局唯一变量，其结构如下所示：

```go
type mheap struct {
	free      mTreap
  ...
  allspans []*mspan
  ...
  arenas [1 << arenaL1Bits]*[1 << arenaL2Bits]*heapArena
  ...
  central [numSpanClasses]struct {
		mcentral mcentral
		pad      [cpu.CacheLinePadSize - unsafe.Sizeof(mcentral{})%cpu.CacheLinePadSize]byte
	}
}
```

- free: 使用树堆的结构来保存各种类别的空闲mspan
- allspans: 用以记录了分配过了的mspan
- arenas: 表示其覆盖的所有arena区域，通过虚拟内存地址计算得到下标索引
- central: 表示其覆盖的所有mcentral，一共134个，对应67个类别

### mcentral

而`mcentral`充当`mspan`的中心管理员，负责管理某一类别的mspan，其结构如下：

```go
type mcentral struct {
	lock      mutex
	spanclass spanClass
	nonempty  mSpanList
	empty     mSpanList
}
```

- lock: 全局互斥锁，因为多个线程会并发请求
- spanclass：mspan类别
- nonempty：mspan的双端链表，且其中至少有一个mspan包含空闲对象
- empty：mspan的双端链表，但不确定其中的mspan是否包含空闲对象

### mcache

`mcache`充当mspan的线程本地缓存角色，其与线程处理器(P)一一绑定。

这样呢，当mcache有空闲mspan时，则无需向mcentral申请，因此可以避免诸多不必要的锁消耗。结构如下所示：

```go
type mcache struct {
  ...
  alloc [numSpanClasses]*mspan
  ...
}
```

- alloc: 表示各个类别的mspan

### mspan

`mspan`作为虚拟内存的实际管理单元，管理着一片内存空间(npages个页)，其结构如下所示：

```go
type mspan struct {
	next *mspan     // 指向下一个mspan
	prev *mspan     // 指向前一个mspan
  ...
	npages    uintptr
  freeindex uintptr
  nelems    uintptr // 总对象个数
  ...
  allocBits  *gcBits
	gcmarkBits *gcBits
}
```

- next指针指向下一个mspan，prev指针指向前一个mspan，因此各个mspan彼此之间形成一个双端链表，并被runtime.mSpanList作为链表头。
- npages：mspan所管理的页的数量
- freeindex：空闲对象的起始位置，如果freeindex等于nelems时，则代表此mspan无空闲对象可分配了
- allocBits：标记哪些元素已分配，哪些未分配。与freeindex结合，可跳过已分配的对象
- gcmarkBits：标记哪些对象存活，每次GC结束时，将其设为allocBits

通过上述对Go内存管理中各个关键数据结构的介绍，想必现在，我们已经对其有了一个大概的轮廓。接下来，让我们继续探究，看看Go具体是如何利用这些数据结构来实现高效的内存分配算法

# 算法

## 分配内存

内存分配算法，其主要函数为`runtime.mallocgc`，其基本步骤简述如下：

- 判断待分配对象的大小
- 若对象小于maxTinySize（16B），且不为指针，则执行微对象分配算法
- 若对象小于maxSmallSize（32KB），则执行小对象分配算法
- 否则，则执行大对象分配算法

在微对象以及小对象分配过程中，如果span中找不到足够的空闲空间，Go会触发层级的内存分配申请策略。其基本步骤如下：

- 先从mcache寻找对应类别的span，若有空闲对象，则成功返回
- 若无，则向mcentral申请，分别从nonempty和empty中寻找匹配的span，若找到，则成功返回
- 若还未找到，则继续向mheap申请，从mheap.free中寻找，若找到，则成功返回
- 若未找到，则需扩容，从关联的arena中申请，若关联的arena中空间也不足，则向OS申请额外的arena
- 扩容完毕后，继续从mheap.free中寻找，若仍未找到，则抛出错误

# 学到了什么

- 本地线程缓存，提高性能：通过mcache缓存小对象的span，并优先在mcache中分配，降低锁竞争
- 无处不在的[BitMap](https://www.jianshu.com/p/6082a2f7df8e)应用场景：通过二进制位来映射对象，例如mspan.allocBits用以表示对象是否分配
- 多级分配策略：自底向上，性能损耗：低->高，频率：高->低，能有效提高性能，思想上类似CPU中的多级缓存

# 总结

本文主要介绍了Go内存分配中的一些重要组件以及分配算法。可以看到，其主要思想还是基于TCMalloc的策略，将对象根据大小分类，并使用不同的分配策略。此外，还采用逐层的内存申请策略，大大提高内存分配的性能。

# 参考

- https://google.github.io/tcmalloc/
- http://goog-perftools.sourceforge.net/doc/tcmalloc.html
- https://medium.com/@ankur_anand/a-visual-guide-to-golang-memory-allocator-from-ground-up-e132258453ed
- https://www.cnblogs.com/zkweb/p/7880099.html
- https://www.cnblogs.com/luozhiyun/p/14349331.html
- https://draveness.me/golang/docs/part3-runtime/ch07-memory/golang-memory-allocator/

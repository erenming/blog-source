---
title: Golang内存优化实践指南
date: 2021-01-09 19:52:52
tags: ["Go"]
---

最近做了许多有关Go内存优化的工作，总结了一些定位、调优方面的套路和经验，于是，想通过这篇文章与大家分享讨论。

## 发现问题

性能优化领域有一条总所周知的铁律，即：**不要过早地优化**。编写一个程序，首先应该保证其功能的正确性，以及诸如设计是否合理、需求等是否满足，过早地优化只会引入不必要的复杂度以及设计不合理等各种问题。

那么何时才能开始优化呢？一句话，*问题出现时*。诸如程序出现频繁OOM，CPU使用率异常偏高等情况。如今，在这微服务盛行的时代，公司内部都会拥有一套或简单或复杂的监控系统，当系统给你发出相关告警时，你就要开始重视起来了。

## 问题定位

### 1. 查看内存曲线

首先，当程序发生OOM时，首先应该查看程序的内存使用量曲线，可以通过现有监控系统查看，或者prometheus之类的开源工具。

曲线一般都是呈上升趋势，比如goroutine泄露的曲线一般是使用量缓慢上升直至OOM，而内存分配不合理往往时在高负载时快速攀升以致OOM。

### 2. 问题复现

这块是可选项，但是最好能保证复现。如果能在本地或debug环境复现问题，这将非常有利于我们反复进行测试和验证。

### 3. 使用pprof定位

Go官方工具提供了pporf来专门用以性能问题定位，首先得在程序中开启pprof收集功能，这里假定问题程序已开启pprof。(对这块不够了解的同学，建议通过这两篇文章([1](https://golang.org/pkg/net/http/pprof/), [2](https://www.freecodecamp.org/news/how-i-investigated-memory-leaks-in-go-using-pprof-on-a-large-codebase-4bec4325e192/))学习下pprof工具的基本用法)

接下来，我们复现问题场景，并及时获取heap和groutine的采样信息。

- 获取heap信息: curl http://loalhost:6060/debug/pprof/heap -o h1.out
- 获取groutine信息：curl http://loalhost:6060/debug/pprof/goroutine -o g1.out

这里你可能想问，这样就够了吗？

当然不是，只获取一份样本信息是不够的。内存使用量是不断变化的(通常是上升)，因此我们需要的也是期间heap、gourtine信息的变化信息，而非瞬时值。一般来说，我们需要一份正常情况下的样本信息，一份或多份内存升高期间的样本信息。

数据收集完毕后，我们按照如下3个方面来排查定位。

#### 排查goroutine泄露

使用命令`go tool pprof --base g1.out g2.out` ，比较goroutine信息来判断是否有goroutine激增的情况。

进入交互界面后，输入`top`命令，查看期间goroutine的变化。

同时可执行`go tool pprof --base g2.out g3.out`来验证。我之前写了的一篇[实战文章](https://www.cnblogs.com/erenming/p/13198605.html)，记录了goroutine泄露的排查过程。

#### 排查内存使用量

使用命令`go tool pprof --base h1.out h2.out`，比较当前堆内存的使用量信息来判断内存使用量。

进入交互界面后，输入`top`命令，查看期间堆内存使用量的变化。

#### 排查内存分配量

当上述排查方向都没发现问题时，那就要查看期间是否有大量的内存申请了，以至于GC都来不及回收。使用命令`go tool pprof --alloc_space --base h1.out h2.out`，通过比较前后内存分配量来判断是否有分配不合理的现象。

进入交互界面后，输入`top`命令，查看期间堆内存分配量的变化。

一般来说，通过上述3个方面的排查，我们基本就能定位出究竟是哪方面的问题导致内存激增了。我们可以通过`web`命令，更为直观地查看问题函数(方法)的完整调用链。

## 问题优化

定位到问题根因后，接下来就是优化阶段了。这个阶段需要对Go本身足够熟悉，还得对问题程序的业务逻辑有所了解。

我梳理了一些常见的优化手段，仅供参考。实际场景还是得实际分析。

#### goroutine泄露

这种问题还是比较好修复的，需要显式地保证goroutine能正确退出，而非以一些自以为的假设来保证。例如，通过传递`context.Context`对象来显式退出

```go
go func(ctx context.Context) {
  for {
    select {
      case <-ctx.Done():
    default:
      }
    ...
  }
}(ctx)
```

#### 对象复用

在一些热点代码处，我们应该避免每次调用都申请新的内存，因为在极端情况下，内存分配速度可能会超过GC的速度，从而导致内存激增。这种情况下，我们可以采取复用对象的方式，例如我们可以使用`sync.Pool`来复用对象

```go
var pool = sync.Pool{New: func() interface{} { return make([]byte, 4096) }}

func fn() {
	buf := pool.Get().([]byte) // takes from pool or calls New
	// do work
	pool.Put(buf) // returns buf to the pool
}
```

#### 避免[]byte和string转换

在Go中，使用`string()`或`[]byte()`来实现[]byte和string的类型转换，会额外申请一块内存来复制。我们可以通过一些技巧来避免复制，例如`*(*[]byte)(unsafe.Pointer(&s))`来实现string转[]byte

除此之外，还有很多优化方法，可以看看dave cheney大神的[这篇文章](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html#memory-and-gc)，真得写得非常好。

## 优化验证

最后一步，我们需要验证优化的结果，毕竟你至少得说服自己，*你的优化是的确有成效的*。

除了通过复现测试来验证有效性外的，还可以编写Benchmark测试用例来比较优化前后的内存分配情况（在Benchmark测试用例中加入一行`b.ReportAllocs()`，即可得到内存分配量信息）

## 总结

性能调优是一项必备但是较为困难的技能，不仅需要熟悉语言、操作系统等基本知识，还需要一定的经验积累。

本文介绍了针对Go程序内存问题的发现、定位、优化以及验证，希望能对你排查内存问题有所帮助（还有某些情况未能没考虑到，欢迎评论区参与讨论）。

## 参考

- https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html
- https://golang.org/pkg/net/http/pprof/
- https://www.freecodecamp.org/news/how-i-investigated-memory-leaks-in-go-using-pprof-on-a-large-codebase-4bec4325e192/




---
title: Golang中的string实现
tags: [Golang]
date: 2019-12-11 21:31:47
---

说到`string`类型，我们往往都能很熟练地对它进行各种处理，包括迭代、随机访问和匹配等等操作。然而在工作中，我发现迭代一个字符串产生的字符的类型与随机访问一个字符的类型却并不相同，为什么会这么奇怪呢？于是我决定一探究竟
<!--more-->
## string 简析

在Golang中，字符串本质上看一看做一个**只读的字节切片**(仅比切片少了一个Cap属性)。它的底层结构我们可以查看`reflect.StringHeader`得到:

```golang
type StringHeader struct {
  Data uintptr
  Len  int
}
```
例如针对字符串`"你好"`，其在内存中的表示如下图所示：
![nihao](https://blog-1300816757.cos.ap-shanghai.myqcloud.com/string-in-golang/nihao.png)
Go的源文件默认使用UTF-8编码，所有的字符串字面量一般也是UTF-8编码的，故这里的`你`编码为`\xe4\xbd\xa0`，`好`编码为`\xe5\xa5\xbd`。UTF-8编码不是我们讨论的重点，具体可参考[这篇博客]([https://chorer.github.io/2019/09/16/CB-%E6%B7%B1%E5%85%A5%E7%90%86%E8%A7%A3%E8%AE%A1%E7%AE%97%E6%9C%BA%E7%B3%BB%E7%BB%9Fcp1/](https://chorer.github.io/2019/09/16/CB-深入理解计算机系统cp1/))。

这里我们运行下述代码

```golang
	s := []byte{0xe4, 0xbd, 0xa0}
	fmt.Printf("char is %s", string(s))
```

得到运行结果`char is 你`。

虽然字符串并非切片，但是支持切片操作。对于同一字面量，不同的字符串变量指向相同的底层数组，这是因为字符串是只读的，为了节省内存，相同字面量的字符串通常对应于同一字符串常量。例如：

```go
	s := "hello, world"
	s1 := "hello, world"
	s2 := "hello, world"[7:]
	fmt.Printf("%d \n", (*reflect.StringHeader)(unsafe.Pointer(&s)).Data) // 17598361
	fmt.Printf("%d \n", (*reflect.StringHeader)(unsafe.Pointer(&s1)).Data) // 17598361
	fmt.Printf("%d \n", (*reflect.StringHeader)(unsafe.Pointer(&s2)).Data) // 17598368
```

可以看到，三者均指向同一个底层数组。对于s1, s2由于是同一字符串常量`hello, world`，故指向一个底层数组，以`h`为起始位置；而s2是字面量`hello, world`的切片操作后生成的字符串，也指向同一字节底层数组，不过是以`w`为起始位置。

## 迭代字符串

当我们使用`for range`迭代字符串时，每次迭代Go都会用UTF-8解码出一个`rune`类型的字符，且索引为当前`rune`的起始位置(以字节为最下单位)。

```go
	for index, char := range "你好" {
		fmt.Printf("start at %d, Unicode = %U, char = %c\n", index, char, char)
	}
```

得到运行结果

```go
start at 0, Unicode = U+4F60, char = 你
start at 3, Unicode = U+597D, char = 好
```

## 随机访问字符串

当我们用下标访问字符串时，返回的值为单个字节，而我们直觉中，应该返回一个字符才合理。这还是因为`string`的后端数组是一个字节切片而非一个字符切片

```go
	s := "你好"
	fmt.Printf("s[%d] = %q, hex = %x, Unicode = %U", 1, s[1], s[1], s[1])
```

得到运行结果

```go
s[1] = '½', hex = bd, Unicode = U+00BD
```

这里我们打印出来索引位置为1的字节，为`0xbd`，其Unicode为`U+00BD`, 代表的字符为`½`。(你可以通过[这里](https://unicode.yunser.com/unicode)查询)

## 到底什么是rune、字符和字节

字节：即byte，它由8个位组成，即1byte = 8bit，是计算机中的基本计量单位，在Go中为`byte`类型，其实际上为`uint8`的别名

字符：字符的概念比较模糊，在Unicode中通常用[code point](https://en.wikipedia.org/wiki/Code_point)来表示。在我的理解里，是一种信息单元(例如一个符号、字母等)

rune：其实际上是`int32`的别名，但是为了方便将字符与整数值区分开，从而新增了rune类型代表一个字符。

## 总结

- 通过下标访问字符串时，返回的是一个字节，这往往与我们的直觉相背。所以，如果你一定要通过下标访问字符串，可以先将其转换为`[]rune`类型
- 字符串可以看做是一个**只读字节切片**, 支持切片操作。

*本人才疏学浅，文章难免有些不足之处，非常欢迎大家评论指出。*


### 参考

- https://chorer.github.io/2019/09/16/CB-%E6%B7%B1%E5%85%A5%E7%90%86%E8%A7%A3%E8%AE%A1%E7%AE%97%E6%9C%BA%E7%B3%BB%E7%BB%9Fcp1/
- https://draveness.me/golang/datastructure/golang-string.html
- https://berryjam.github.io/2018/03/%E4%BB%8Egolang%E5%AD%97%E7%AC%A6%E4%B8%B2string%E9%81%8D%E5%8E%86%E8%AF%B4%E8%B5%B7/
- https://en.wikipedia.org/wiki/Code_point
- https://github.com/chai2010/advanced-go-programming-book/blob/master/ch1-basic/ch1-03-array-string-and-slice.md
---
title: "在Go种如何访问和修改私有对象"
date: 2024-08-12T21:49:04+08:00
draft: false
Tags: [Go, 2024]
---
我们都知道，基本上所有主流编程语言都支持对变量、类型以及函数方法等设置私有或公开，从而帮助程序员设计出封装优秀的模块。然而，实际开发中，难免需要使用第三方包的私有函数或方法，或修改其中的私有变量、熟悉等。在可观测性数据采集器开发中，由于继承了很多采集插件，经常需要魔改其中的代码，因此我对Go语言中如何修改这些私有对象的方式做了一个总结，以供后续参考。



# 方式方法

## 修改指针

指针本质上就是一个内存地址，这种方式下，我们通过对指针的计算（如果你有C/C++的经验，想必对指针运算一定有所耳闻），从而找到目标对象的内存地址，进而可以获取并修改指针所指向对象的值。

`Examples`：

```go
// pa/a.go
package pa

type ExportedType struct {
	intField    int
	stringField string
	flag        bool
}

func (t *ExportedType) String() string {
	return fmt.Sprintf("ExportedType{flag: %v}", t.flag)
}

// main/main.go
func main() {
	et := &pa.ExportedType{}
	fmt.Printf("before edit: %s\n", et)
	ptr := unsafe.Pointer(et)                                                      // line 1
	flagPtr := unsafe.Pointer(uintptr(ptr) + unsafe.Sizeof(0) + unsafe.Sizeof("")) // line 2
	flagField := (*bool)(flagPtr)                                                  // line 3
	*flagField = true                                                              // line 4
	fmt.Printf("after edit: %s\n", et)
}
```

`Outputs`：

```text
before edit: ExportedType{flag: false}                                                                 after edit: ExportedType{flag: true} 
```

`Explains`:

- `line 1`：将指针`et`转换为`unsafe.Pointer`类型，方便后续计算
- `line 2`：通过指针计算得到字段`flag`地址
- `line 3`：类型转换，将指针类似转换为`(*bool)`，因为此时`flagPtr`的值就是flag字段的内存位置，因此转换时并不会失败
- `line 4`：通过`*flagFiled`得到实际的值，并进行修改

# reflect

reflect包本质是上也是获取字段的指针并修改，只不过借助reflect包，我们的代码可以写的更为清晰易读。

`Examples`：

```go
// pa/a.go
package pa

type ExportedType struct {
	intField    int
	stringField string
	flag        bool
}

func (t *ExportedType) String() string {
	return fmt.Sprintf("ExportedType{flag: %v}", t.flag)
}

// main/main.go
func main() {
	et := &pa.ExportedType{}
	fmt.Printf("before edit: %s\n", et)
	val := reflect.ValueOf(et).Elem()                  // line 1
	flagPtr := val.FieldByName("flag").UnsafePointer() // line 2
	flagField := (*bool)(flagPtr)
	*flagField = true
	fmt.Printf("after edit: %s\n", et)
}

```
`Outputs`：

```text
before edit: ExportedType{flag: false}                                                                 after edit: ExportedType{flag: true} 
```
`Explains`:

- `line 1`：通过reflect包获取反射对象
- `line 2`：通过`FieldByName`找到flag字段并获取其指针，从而无需手动计算获取flag字段的指针

# go:linkname

`//go:linkname`是Go编译器的一种机制，在编译的时候，编译器识别到`//go:linkname`后，会将当前函数或变量链接到源函数（个人感觉可以简单理解成Linux中的软链接）。

用法：

```go
//go:linkname <定义别名> <定义所在包路径>.<定义名>
```

`Examples`:

- `a.go`

```go
package pa

import "fmt"

var globalFlag bool

func PrintGlobalFlag() {
	fmt.Println("[var] globalFlag: ", globalFlag)
}

func printGlobalFlag() {
	fmt.Println("[func] globalFlag: ", globalFlag)
}

func (t *ExportedType) printGlobalFlag() {
	fmt.Println("[method] globalFlag: ", globalFlag)
}

type ExportedType struct {
	intField    int
	stringField string
	flag        bool
}

func (t *ExportedType) String() string {
	return fmt.Sprintf("ExportedType{flag: %v}", t.flag)
}
```

- `main.go`

```go
package main

import (
	"github.com/erenming/blog-source/codes/get-set-private-object-in-go/pa"
	_ "unsafe"
)

//go:linkname myGlobalFlag github.com/erenming/blog-source/codes/get-set-private-object-in-go/pa.globalFlag
var myGlobalFlag bool

//go:linkname printGlobalFlag github.com/erenming/blog-source/codes/get-set-private-object-in-go/pa.printGlobalFlag
func printGlobalFlag()

//go:linkname methodPrintGlobalFlag github.com/erenming/blog-source/codes/get-set-private-object-in-go/pa.(*ExportedType).printGlobalFlag
func methodPrintGlobalFlag(t *pa.ExportedType)

func main() {
	// var
	pa.PrintGlobalFlag()
	myGlobalFlag = true
	pa.PrintGlobalFlag()
	// func
	printGlobalFlag()
	myGlobalFlag = false
	printGlobalFlag()
	// method
	t := &pa.ExportedType{}
	methodPrintGlobalFlag(t)
	myGlobalFlag = true
	methodPrintGlobalFlag(t)
}

```

需要注意的是，当要对方法试用`//go:linkname`时，不能直接复制黏贴。首先，需转换方法的表现形式（即转换为函数的形式），由`func (t *ExportedType) printGlobalFlag()`转换为`printGlobalFlag(t *pa.ExportedType)`；然后，定义包名定义哪里的形式改为`<定义所在包路径>.(<方法所属的结构体类型>).<定义名>`

# 应用场景总结

| 场景             | 推荐方法          | 注意点                         |
| ---------------- | ----------------- | ------------------------------ |
| 使用私有全局变量 | go:linkname       | -                              |
| 使用私有函数     | go:linkname       | 当函数中包含私有类型时无法使用 |
| 使用私有方法     | go:linkname       | 当函数中包含私有类型时无法使用 |
| 使用私有字段     | reflect，修改指针 | -                              |

## 特殊场景

当方法所属的结构体类型私有时，这种场景下，我们需要做一些额外tricks。

`Examples`:

```go
// pa/a.go
package pa

import "fmt"

var globalFlag bool

func (t *privateType) printGlobalFlag() {
	fmt.Println("[method.privateType] globalFlag: ", globalFlag)
}

type privateType struct {
	intField    int
	stringField string
	flag        bool
}

func GetPrivateType() *privateType {
	return &privateType{}
}

// main.go
package main

import (
	"github.com/erenming/blog-source/codes/get-set-private-object-in-go/preceiver/pa"
	"unsafe"
	_ "unsafe"
)

//go:linkname methodPrintGlobalFlag github.com/erenming/blog-source/codes/get-set-private-object-in-go/preceiver/pa.(*privateType).printGlobalFlag
func methodPrintGlobalFlag(t *main_privateType)

type main_privateType struct {
	intField    int
	stringField string
	flag        bool
}

func main() {
	// method
	t := pa.GetPrivateType()
	convertedPT := *(*main_privateType)(unsafe.Pointer(t))
	methodPrintGlobalFlag(&convertedPT)
}
```

`Explains`:

- 首先，我们先在复制私有类型并黏贴到`main.go`里，并对其重命名以便于区分（也可以同名，但这样不易读）。

- 然后，我们通过`GetPrivateType`得到私有类型的对象指针

- 最后我们将其该指针的类型转换为`main_privateType`，并作为参数传递给方法

  > 至于这样做的原理，我猜测是因为main_privateType和privateType由于类型定义完全一样，其在内存上的分布也一模一样，所以可以如此转换



# 参考

- https://medium.com/@yardenlaif/accessing-private-functions-methods-types-and-variables-in-go-951acccc05a6
- https://stackoverflow.com/questions/17981651/is-there-any-way-to-access-private-fields-of-a-struct-from-another-package
---
title: 用Golang实现并理解Web中间件
date: 2019-12-20 14:39:21
tags: [Go, Backend]
---

在编写web应用中，我们常常会遇到这样的需求，比如，我们需要上报每个API的运行时间到运维监控系统。这时候你可以像下述代码一样将统计的逻辑写到每个路由函数中。
<!--more-->
```go
func someApi(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// your logic
	metrics.Upload(time.Since(start))
}
```

然而，这显然有悖`DRY`原则，我们需要将这些非业务逻辑剥离出来以实现解耦。这时候，中间件就能派上用场了，为了简单起见，我们这里将采用标准库`net/http`来实现。

## 准备工作

```go
func hello(w http.ResponseWriter, r *http.Request) {
	log.Println("execute hello func")
	w.Write([]byte("hello, world"))
}

func main() {
	http.Handle("/", http.HandlerFunc(hello))
	http.ListenAndServe(":3000", nil)
}
```

这里，我们创建了一个hello函数并将其转换成一个`Handler`用以处理HTTP请求。

## 中间件的实现

```go
func middlewareOne(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("middleware one start") // before logic
		next.ServeHTTP(w, r)
		log.Println("middleware one end") // after logic
	})
}
```

这里，我们实现了一个中间件函数`middlewareOne`，它接收一个`next`的参数其类型为`http.Handler`并返回一个新的`http.Handler`。而`next.ServeHTTP(w, r)`会回调`next`函数自身，即`next(w, r)`。看到这里，你可能会有点懵，我们需要先复习一下`Handler`，`HandlerFunc`，`ServeHTTP`三者的关系。

下面是三者的定义：

```go
// A Handler responds to an HTTP request.
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}

type HandlerFunc func(ResponseWriter, *Request)

// ServeHTTP calls f(w, r).
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	f(w, r)
}

```

`Handler`是一个接口，用以处理HTTP请求，换句话说，一个函数若想成为一个路由函数必须得是一个`Handler`接口。

`HandlerFunc`是一个签名为`func(ResponseWriter, *Request)`的函数类型，其实现了`ServerHTTP`方法且签名相同，故`HandleFunc`是一个`Handler`，其`ServerHTTP`方法调用`HandlerFunc`自身。

三者关系如下图所示：

![relation](https://blog-1300816757.cos.ap-shanghai.myqcloud.com/middleware/req-flow.jpg)

好了，接下来我们将中间件函数应用到我们的`Handler`上。

```go
func main() {
	http.Handle("/", middlewareOne(http.HandlerFunc(hello)))
	http.ListenAndServe(":3000", nil)
}
```

运行程序，然后访问`http://127.0.0.1:3000/`，控制台将输出如下结果。

```go
2019/12/middleware one start
2019/12/execute hello func
2019/12/middleware one end
```

当然，如果你想应用多个中间件，你只需要再套上一层，例如下述代码：

```go
func middlewareTwo(next http.Handler) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		log.Println("middleware two start")
		next.ServeHTTP(w, r)
		log.Println("middleware two end")
	})
}

func main() {
	http.Handle("/", middlewareTwo(middlewareOne(http.HandlerFunc(hello))))
	http.ListenAndServe(":3000", nil)
}
```

我画出了该函数链的执行流程，如下图所示：

![func-chain](https://blog-1300816757.cos.ap-shanghai.myqcloud.com/middleware/func-chain.jpg)

可以看到，如果我们把路由函数`hello`看做为汉堡里的肉饼，中间件函数看做成面包。那么，`middlewareOne`包住了肉饼`hello`，而`middlewareTwo`又包住了`middlewareTwo`。

## 总结

1. 中间件函数与Python中的装饰器函数十分类似，都是在原函数逻辑不变的条件下进行扩展。
2. 对于Web框架而言，类似于`Flask`里面的`before_request`和`after_request`钩子函数，`next.ServeHTTP`之前逻辑的等同于`before_request`里的逻辑、之后的等同于于`after_request`里的逻辑。
3. 在业务开发中，我们应尽量将非业务逻辑抽象到中间件中，实现代码的松耦合与易扩展。

*本人才疏学浅，文章难免有些不足之处，非常欢迎大家评论指出。*


## 参考

- https://github.com/chai2010/advanced-go-programming-book/blob/master/ch5-web/ch5-03-middleware.md
- https://www.alexedwards.net/blog/making-and-using-middleware
- https://golang.org/doc/effective_go.html#interface_methods
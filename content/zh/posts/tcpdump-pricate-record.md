---
title: 一次抓包排查实战记录
date: 2020-08-02 11:00:43
tags: [Linux]
---

## 问题的发现

周五，本是一个风清气爽，令人愉悦的日子。我本还在美滋滋地等待着下班，然而天有不测，有用户反应容器日志看不到了，根据经验我知道，日志采集&收集链路上很可能又发生了阻塞。

登录目标容器所在机器找到日志采集容器，并娴熟地敲下`docker logs --tail 200 -f <container-id>`命令，发现确实阻塞了，阻塞原因是上报日志的请求500了，从而不断重试导致日志采集阻塞。

随后，我找到收集端的容器，查看日志，发现确实有请求报500了，并且抛出了`Unknown value type`的错误，查看相关代码。

业务代码：

```go
if _, err := jsonparser.ArrayEach(body, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
    ...
}); err != nil {
    return err // 错误抛出点
}
```

jsonparser包中代码：

![image-20200802113337369](https://raw.githubusercontent.com/erenming/image-pool/master/blogimage-20200802113337369.png)

显然问题出在了对body的解析上，究竟是什么样的body导致了解析错误呢？接下来，就是tcpdump和wireshark上场的时候了。

## 使用Tcpdump抓包

首先，我们通过tcpdump抓到相关的请求。由于日志采集端会不断重试，因此最简单的方法便是登录采集端所在机器，并敲下如下命令`tcpdump -i tunl0 dst port 7777 -w td.out` ，并等待10-20秒。

熟悉tcpdump的小伙伴，对这条命令显然已经心领神会了。尽管如此，这里我还是稍微解释下。

- `-i tunl0`：-i 参数用来指定网卡，由于采集器并没有通过`eth0`。因此，实战中，有时发现命令正确缺抓不到包的情况时，不妨指定下别的网卡。网络错综复杂，不一定都会通过`eth0`网卡。
- `dst port 777`： 指定了目标端口作为过滤参数，收集端程序的端口号是7777
- `-w td.out`:  表明将抓包记录保存在td.out文件中，这是因为json body是用base64编码并使用gzip加密后传输的，因此我得使用wireshark来抽离出来。（主要还是wireshark太香了:)，界面友好，操作简单，功能强大）

接着，我用`scp`命令将`td.out`文件拷到本地。并使用wireshar打开它

## 使用Wireshark分析

![](https://raw.githubusercontent.com/erenming/image-pool/master/blogimage-20200802120035766.png)

打开后，首先映入眼帘的则是上图内容，看起来很多？不要慌，由于我们排查的是http请求，在过滤栏里输入HTTP，过滤掉非HTTP协议的记录。

![image-20200802120641740](https://raw.githubusercontent.com/erenming/image-pool/master/blog/image-20200802120641740.png)

我们可以很清楚地发现，所有的HTTP都是发往一个IP的，且长度都是59，显然这些请求都是日志采集端程序不断重试的请求。接下来，我们只需要将某个请求里的body提取出来查看即可。

![image-20200802120944352](https://raw.githubusercontent.com/erenming/image-pool/master/blog/image-20200802120944352.png)

很幸运，wireshark提供了这种功能，如上图所示，我们成功提取出来body内容。为`bnVsbA==`，使用base64解码后为`null`。

## 解决问题

既然body的内容为null，那么调用`jsonparser.ArrayEach`报错也是意料之中的了，body内容必须得是一个JsonArray。

然而，采集端为何会发送body为null的请求呢，深入源码，发现了如下一段逻辑。

```go
func (e *jsonEncoder) encode(obj []publisher.Event) (*bytes.Buffer, error) {
	var events []map[string]interface{}
	for _, o := range obj {
		m, err := transformMap(o)
		if err != nil {
			logp.Err("Fail to transform map with err: %s", err)
			continue
		}
		events = append(events, m)
	}
	data, err := json.Marshal(events)
	if err != nil {
		return nil, errors.Wrap(err, "fail to json marshal events")
	}
  ...
}
```

由于`transforMap`函数回对obj中的元素进行转换，成功后添加到events中。

但是，由于使用的是`var events []map[string]interface{}`这种声明方式，在Golang中，slice的零值为nil，因此events此时的值为nil。而当obj中所有的对象，被`transforMap`失败时，events使用json序列化后则为null了。

这里我们需改变evetns的声明方式，使用`events := make([]map[string]interface{}, 0)`或者`events := []map[string]interface{}{}`的方式替代，此时events被初始化了，并指向的是一个cap为0的slice对象，其序列化后为`[]`。

这样即使没有对象添加到events中，上报的也是一个空数组。



## 参考

- https://www.cnblogs.com/ggjucheng/archive/2012/01/14/2322659.html
- https://www.k0rz3n.com/2017/04/17/wireshark/
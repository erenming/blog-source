---
title: "聊一聊opentelemetry-ollector的设计与实现"
date: 2022-08-07T23:22:34+08:00
draft: true
---

在一个典型的可观测系统中，所有数据都至少需要经历`采集->传输->处理`三个步骤，才能写入到数据库里。其中`采集`步骤一般是由各式各样的代理(Agent)完成的（例如指标由promethes采集，日志由fluentBit采集等）。然而Agent千千万万，意味着数据协议也是千千万万种，因此我们需要一个组件作为统一的入口将各式各样的数据协议转为内部的数据协议，也就是其中的`处理`步骤。

那么很显然，既然OpenTelementry的愿景是在可观测性领域中，建立完全开源且不绑定任何商业厂商的数据模型&协议标准，那么它同样也需要一个组件来出来千千万万种的数据协议，而这个组件就是**opentelemetry-ollector**

# 架构设计

# 模块设计

## Pipeline

## Receiver

## Processor

## Exporter

# 一些思考

# 参考

- [Observability Engineering](https://www.oreilly.com/library/view/observability-engineering/9781492076438/)
- 

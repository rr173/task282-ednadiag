# ednadiag — 环境 DNA 采样空白传播诊断服务

环境 DNA  metabarcoding 实验链中，提取空白与现场样本同批次共现时可能产生假阳性信号。本服务追溯空白传播路径、评分归类并发布可信度快照。

## 本地运行

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./cmd/ednadiag --addr :8080 --db ednadiag.db
go run ./cmd/ednadiag --smoke-test
```

## 目录结构

```
cmd/ednadiag/          HTTP 入口与 smoke-test
internal/httpapi/      /api 路由
internal/service/      编排与链级串行锁
internal/chain/        实验链状态机
internal/propagation/  空白传播分析
internal/diagnosis/    路径评分与裁决
internal/snapshot/     可信度快照
internal/store/        SQLite 持久化
internal/model/        领域模型与错误
```

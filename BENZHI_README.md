# BENZHI 评测说明

基于 Go 实现的环境 DNA 采样空白传播诊断后端服务，一款后端服务，完成样本/空白/批次录入、实验链追溯、空白传播路径分析、污染评分与可信度快照发布。

## 启动

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go run ./cmd/ednadiag --addr :8080 --db ednadiag.db
```

## 自检（不启动长驻服务）

```bash
go run ./cmd/ednadiag --smoke-test
```

`--smoke-test` 会真实创建批次与空白污染场景、运行追溯与快照发布，关闭并重新打开数据库验证持久化与重启恢复，最后以 0 退出码结束。

## 构建门禁

```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go vet   ./...
CGO_ENABLED=0 GOTOOLCHAIN=local go test  ./...
go run ./cmd/ednadiag --smoke-test
```

## HTTP API（前缀 /api）

样本/空白/批次：`POST/GET /api/samples`、`/api/blanks`、`/api/batches` 及 `/{id}`
特征：`POST/GET /api/features`、`/api/features/{id}`
实验链：`POST/GET /api/chains`、`/api/chains/{id}/steps`、`POST /api/chains/{id}/trace|publish|seal`
路径与诊断：`GET /api/chains/{id}/paths`、`POST /api/paths/{id}/confirm|reject`、`POST /api/chains/{id}/isolate`
快照：`POST /api/snapshots`、`POST /api/snapshots/{id}/publish`、`GET /api/snapshots/{id}`
自检：`GET /api/stats`、`GET /api/selfcheck`

## 持久化

SQLite（modernc.org/sqlite，CGO 无关）。建表：samples、blanks、batches、features、chains、chain_steps、contam_paths、snapshots。封存链拒绝写入；已发布快照载荷在发布时固化。

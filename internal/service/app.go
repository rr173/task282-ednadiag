// Package service 装配各业务包，提供高层编排接口与“同一实验链裁决串行”的并发控制。
package service

import (
	"sync"

	"task282-ednadiag/internal/chain"
	"task282-ednadiag/internal/diagnosis"
	"task282-ednadiag/internal/propagation"
	"task282-ednadiag/internal/sample"
	"task282-ednadiag/internal/snapshot"
	"task282-ednadiag/internal/store"
)

// App 应用编排根。
type App struct {
	Store  *store.Store
	Sample *sample.Service
	Chain  *chain.Service
	Prop   *propagation.Service
	Diag   *diagnosis.Service
	Snap   *snapshot.Service

	locks sync.Map // chainID -> *sync.Mutex
}

// NewApp 装配应用（打开数据库并构造全部子服务）。
func NewApp(dbPath string) (*App, error) {
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, err
	}
	return NewAppWithStore(st), nil
}

// NewAppWithStore 基于已有 Store 装配（供测试/自检复用）。
func NewAppWithStore(st *store.Store) *App {
	prop := propagation.NewService(st)
	return &App{
		Store:  st,
		Sample: sample.NewService(st),
		Chain:  chain.NewService(st),
		Prop:   prop,
		Diag:   diagnosis.NewService(st, prop),
		Snap:   snapshot.NewService(st),
	}
}

// chainMu 返回某实验链的串行裁决锁（首次访问时创建）。
func (a *App) chainMu(id string) *sync.Mutex {
	m, _ := a.locks.LoadOrStore(id, &sync.Mutex{})
	return m.(*sync.Mutex)
}

package pool

import (
	"context"
	"errors"
	"log"
	"runtime/debug"
	"sync"

	"github.com/panjf2000/ants/v2"
)

const (
	defaultQueueMultiplier = 10
)

var (
	jobPool          *ants.Pool
	jobPoolLock      sync.RWMutex
	handlerMu        sync.RWMutex // guards userPanicHandler and logger
	userPanicHandler func(ctx context.Context, r interface{}, stack []byte)
)

// Logger 接口，用于解耦日志，可选注入
type Logger interface {
	Error(msg string, fields ...any)
	Info(msg string, fields ...any)
}

// 默认 Logger，使用标准库 log 打印
type defaultLogger struct{}

func (d defaultLogger) Error(msg string, fields ...any) {
	log.Println("[ERROR]", msg, fields)
}

func (d defaultLogger) Info(msg string, fields ...any) {
	log.Println("[INFO]", msg, fields)
}

// logger 实例，默认使用 defaultLogger
var logger Logger = defaultLogger{}

// SetLogger 允许用户注入自己的 Logger（并发安全）
func SetLogger(l Logger) {
	if l != nil {
		handlerMu.Lock()
		logger = l
		handlerMu.Unlock()
	}
}

// JobPool 暴露全局 Goroutine 池实例（只读）
var JobPool *ants.Pool

// New 初始化 Goroutine 池（线程安全）
// size: Goroutine 数量, queueSize: 最大阻塞任务数
func New(size, queueSize int) error {
	if size <= 0 {
		return errors.New("pool size must be positive")
	}

	jobPoolLock.Lock()
	defer jobPoolLock.Unlock()

	if jobPool != nil {
		return errors.New("pool already initialized")
	}

	if queueSize < size {
		queueSize = size * defaultQueueMultiplier
	}

	pool, err := ants.NewPool(
		size,
		ants.WithPreAlloc(true),
		ants.WithNonblocking(false),
		ants.WithMaxBlockingTasks(queueSize),
	)
	if err != nil {
		return errors.New("failed to create pool: " + err.Error())
	}

	jobPool = pool
	JobPool = pool
	return nil
}

// Submit 提交任务到 Goroutine 池（线程安全）
func Submit(task func()) error {
	jobPoolLock.RLock()
	defer jobPoolLock.RUnlock()

	if jobPool == nil {
		return errors.New("pool not initialized")
	}
	return jobPool.Submit(task)
}

// SubmitWithCtx 提交带上下文的任务，自动捕获 panic
func SubmitWithCtx(ctx context.Context, task func(ctx context.Context)) error {
	wrappedTask := func() {
		defer handlePanic(ctx)
		task(ctx)
	}

	jobPoolLock.RLock()
	defer jobPoolLock.RUnlock()

	if jobPool == nil {
		return errors.New("pool not initialized")
	}
	return jobPool.Submit(wrappedTask)
}

// Release 释放资源（线程安全）
func Release() {
	jobPoolLock.Lock()
	defer jobPoolLock.Unlock()

	if jobPool != nil {
		jobPool.Release()
		jobPool = nil
		JobPool = nil
	}
}

// OnPanic 设置用户自定义 panic 处理函数（可用于上报监控、报警等，并发安全）
func OnPanic(handler func(ctx context.Context, r interface{}, stack []byte)) {
	handlerMu.Lock()
	userPanicHandler = handler
	handlerMu.Unlock()
}

// handlePanic 捕获 panic 并记录日志，调用用户自定义处理器（并发安全）
func handlePanic(ctx context.Context) {
	if r := recover(); r != nil {
		stack := debug.Stack()

		handlerMu.RLock()
		l := logger
		h := userPanicHandler
		handlerMu.RUnlock()

		l.Error("goroutine panic recovered", r, stack)

		if h != nil {
			h(ctx, r, stack)
		}
	}
}

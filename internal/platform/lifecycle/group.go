package lifecycle

import (
	"context"
	"fmt"
	"sync"
)

// Group 监督一组共享 Context 的关键后台任务。
type Group struct {
	ctx    context.Context
	cancel context.CancelFunc
	errors chan error
	tasks  sync.WaitGroup
}

// NewGroup 创建最多缓存 taskCount 个并发错误的任务组。
func NewGroup(parent context.Context, taskCount int) *Group {
	if taskCount < 1 {
		taskCount = 1
	}
	ctx, cancel := context.WithCancel(parent)
	return &Group{
		ctx:    ctx,
		cancel: cancel,
		errors: make(chan error, taskCount),
	}
}

// Go 启动一个关键任务，并在其意外退出时报告带名称的错误。
func (g *Group) Go(name string, run func(context.Context) error) {
	g.tasks.Add(1)
	go func() {
		defer g.tasks.Done()
		err := run(g.ctx)
		if g.ctx.Err() != nil {
			return
		}
		if err == nil {
			err = fmt.Errorf("%s stopped unexpectedly", name)
		} else {
			err = fmt.Errorf("%s: %w", name, err)
		}
		select {
		case g.errors <- err:
		case <-g.ctx.Done():
		}
	}()
}

// WaitForStop 等待父 Context 取消或任一关键任务意外退出。
func (g *Group) WaitForStop() error {
	select {
	case <-g.ctx.Done():
		return nil
	case err := <-g.errors:
		return err
	}
}

// Stop 向任务组内的所有任务广播停止信号。
func (g *Group) Stop() {
	g.cancel()
}

// Wait 在 Context 到期前等待全部任务退出。
func (g *Group) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		g.tasks.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for services to stop: %w", ctx.Err())
	}
}

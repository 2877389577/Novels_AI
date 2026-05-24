// Package event 提供一个轻量级的进程内事件发布订阅系统。
//
// 使用场景：
//  1. 当一个业务动作完成后，需要通知其他模块做后续处理，但不希望两个模块直接互相依赖。
//     例如：角色创建成功后发布 character.created 事件，后续可以由统计模块、缓存模块或 AI 分析模块订阅。
//  2. 当功能仍在同一个 Go 进程内运行，不需要跨进程投递、不需要消息持久化、不需要失败重试队列。
//  3. 当事件处理失败需要立刻反馈给发布方，发布方可以根据返回错误决定是否记录日志或中断流程。
//
// 基本用法：
//
//	bus := event.NewBus()
//
//	unsubscribe := bus.Subscribe("character.created", func(ctx context.Context, evt event.Event) error {
//		characterID, ok := evt.Payload.(int64)
//		if !ok {
//			return fmt.Errorf("事件载荷类型错误")
//		}
//		fmt.Println("新角色 ID:", characterID)
//		return nil
//	})
//	defer unsubscribe()
//
//	err := bus.Publish(ctx, event.New("character.created", int64(1001)))
//	if err != nil {
//		// 只要任意订阅者返回错误，Publish 就会返回合并后的错误。
//		slog.ErrorContext(ctx, "处理角色创建事件失败", "err", err)
//	}
//
// 设计约定：
//  1. Publish 是同步调用：发布方会等待当前事件名下的所有处理器执行结束。
//  2. 处理器执行顺序与订阅顺序一致，便于排查问题；但业务逻辑不应该依赖该顺序。
//  3. Subscribe 返回的函数用于取消订阅，重复调用是安全的。
//  4. 处理器列表会在 Publish 开始时复制一份，因此处理器内部新增或取消订阅不会影响当前这次发布。
//  5. Event.Payload 是 any，调用方应在处理器里自行做类型断言；如果需要强约束，可以在业务层封装专用事件结构体。
//  6. 这是内存事件总线，服务重启后订阅关系和未处理事件都不会保留；需要跨进程或持久化时应接入消息队列。
package event

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Handler 是事件处理函数。
//
// ctx 来自 Publish 调用方，处理器应该把它继续传给数据库、HTTP 或 AI 调用，
// 这样请求取消、超时和 request_id 等上下文信息可以沿用原调用链。
type Handler func(ctx context.Context, evt Event) error

// Event 是事件总线中传递的标准事件结构。
type Event struct {
	// Name 是事件名称，建议使用“领域.动作”的小写形式，例如 character.created、novel.deleted。
	Name string
	// Payload 是事件载荷，保存本次事件需要传递给订阅者的业务数据。
	Payload any
	// Metadata 保存事件的补充信息，例如操作者 ID、来源模块、追踪标识等。
	Metadata map[string]string
	// OccurredAt 是事件发生时间，New 会自动填充当前时间。
	OccurredAt time.Time
}

// New 创建一个事件。
//
// payload 可以是 ID、DTO、模型快照或专用事件结构体。为了降低模块耦合，推荐优先传递
// “订阅者真正需要的最小数据”，不要把整个业务上下文都塞进 Payload。
func New(name string, payload any) Event {
	return Event{
		Name:       name,
		Payload:    payload,
		Metadata:   map[string]string{},
		OccurredAt: time.Now(),
	}
}

// Bus 是进程内事件总线。
//
// Bus 可以作为依赖注入到 biz 或 service 中，也可以在 bootstrap 里创建一个全局实例。
// 它本身是并发安全的：多个 goroutine 可以同时 Subscribe、Unsubscribe 和 Publish。
type Bus struct {
	mu            sync.RWMutex
	nextID        uint64
	subscriptions map[string][]subscription
}

type subscription struct {
	id      uint64
	handler Handler
}

// NewBus 创建一个空事件总线。
func NewBus() *Bus {
	return &Bus{
		subscriptions: make(map[string][]subscription),
	}
}

// Subscribe 订阅指定事件名。
//
// 参数说明：
//  1. name：事件名称，必须和发布时的 Event.Name 完全一致。
//  2. handler：事件处理函数，不能为 nil。
//
// 返回值：
//  1. unsubscribe：取消订阅函数。业务模块退出、测试结束或临时订阅不再需要时应调用它。
//
// 示例：
//
//	unsubscribe := bus.Subscribe("novel.updated", func(ctx context.Context, evt event.Event) error {
//		novelID := evt.Payload.(uint)
//		return refreshNovelCache(ctx, novelID)
//	})
//	defer unsubscribe()
//
// 注意：
//  1. 如果 name 为空或 handler 为 nil，会返回一个空取消函数，不会注册订阅。
//  2. 重复调用 unsubscribe 不会 panic，也不会影响其他订阅。
func (b *Bus) Subscribe(name string, handler Handler) func() {
	if name == "" || handler == nil {
		return func() {}
	}

	b.mu.Lock()
	b.nextID++
	id := b.nextID
	b.subscriptions[name] = append(b.subscriptions[name], subscription{
		id:      id,
		handler: handler,
	})
	b.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			b.unsubscribe(name, id)
		})
	}
}

// Publish 发布事件，并同步执行当前事件名下的所有订阅者。
//
// 行为说明：
//  1. 如果 evt.Name 为空，Publish 不执行任何处理器并返回 nil。
//  2. 如果没有订阅者，Publish 返回 nil，发布方不需要为“无人订阅”做特殊处理。
//  3. 如果多个订阅者返回错误，Publish 会使用 errors.Join 合并错误后返回。
//  4. 即使某个订阅者返回错误，后续订阅者仍会继续执行，避免一个订阅者阻断其他订阅者。
//
// 示例：
//
//	err := bus.Publish(ctx, event.Event{
//		Name:    "chapter.deleted",
//		Payload: chapterID,
//		Metadata: map[string]string{
//			"source": "chapter_service",
//		},
//		OccurredAt: time.Now(),
//	})
func (b *Bus) Publish(ctx context.Context, evt Event) error {
	if evt.Name == "" {
		return nil
	}
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now()
	}
	if evt.Metadata == nil {
		evt.Metadata = map[string]string{}
	}

	handlers := b.handlers(evt.Name)
	var errs []error
	for _, handler := range handlers {
		if err := handler(ctx, evt); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (b *Bus) handlers(name string) []Handler {
	b.mu.RLock()
	defer b.mu.RUnlock()

	items := b.subscriptions[name]
	handlers := make([]Handler, 0, len(items))
	for _, item := range items {
		handlers = append(handlers, item.handler)
	}

	return handlers
}

func (b *Bus) unsubscribe(name string, id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	items := b.subscriptions[name]
	for index, item := range items {
		if item.id != id {
			continue
		}

		items = append(items[:index], items[index+1:]...)
		if len(items) == 0 {
			delete(b.subscriptions, name)
			return
		}
		b.subscriptions[name] = items
		return
	}
}

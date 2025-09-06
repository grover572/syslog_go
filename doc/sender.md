# 发送器实现说明

## 核心组件

### Sender（pkg/sender/sender.go）

发送器的主要实现，负责消息的生成和发送。

```go
type Sender struct {
    config      *config.Config
    connPool    *ConnectionPool
    rateLimiter *RateLimiter
    stats       *Statistics
    ctx         context.Context
    cancel      context.CancelFunc
    wg          sync.WaitGroup
    templateEngine *template.Engine
}
```

主要方法：
- `NewSender(cfg *config.Config)`: 创建新的发送器实例
- `Start()`: 启动发送器
- `Stop()`: 停止发送器并关闭连接池
- `GetStats()`: 获取统计信息

## 实现流程

### 1. 发送器初始化

```go
// 创建发送器
func NewSender(cfg *config.Config) (*Sender, error) {
    ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)

    s := &Sender{
        config: cfg,
        ctx:    ctx,
        cancel: cancel,
        stats:  &Statistics{StartTime: time.Now()},
    }

    // 初始化连接池
    if err := s.initConnectionPool(); err != nil {
        return nil, fmt.Errorf("初始化连接池失败: %w", err)
    }

    // 初始化速率限制器
    s.rateLimiter = NewRateLimiter(cfg.EPS)

    return s, nil
}
```

### 2. 消息发送流程

```go
// 发送工作协程
func (s *Sender) sendWorker(workerID int) {
    defer s.wg.Done()

    for {
        select {
        case <-s.ctx.Done():
            return
        default:
            // 等待直到允许发送
            s.rateLimiter.Wait()

            // 生成消息
            message, err := s.generateMessage()
            if err != nil {
                atomic.AddInt64(&s.stats.Failed, 1)
                continue
            }

            // 发送消息
            if err = s.sendMessage(message); err != nil {
                atomic.AddInt64(&s.stats.Failed, 1)
            } else {
                atomic.AddInt64(&s.stats.Sent, 1)
            }
        }
    }
}
```

## 性能优化

### 1. 并发处理

- 使用多个goroutine并发发送消息
- 使用WaitGroup确保所有goroutine正确退出
- 使用context控制生命周期

### 2. 速率控制

- 使用RateLimiter控制发送速率
- 支持配置每秒事件数(EPS)
- 避免发送过快导致目标服务器过载

### 3. 连接池管理

- 复用TCP/UDP连接
- 支持配置并发连接数
- 自动处理连接的获取和释放

## 错误处理

### 1. 连接错误

```go
// 发送消息
func (s *Sender) sendMessage(msg *syslog.Message) error {
    conn, err := s.connPool.Get()
    if err != nil {
        return fmt.Errorf("获取连接失败: %w", err)
    }
    defer s.connPool.Put(conn)

    // 发送消息
    data := msg.Bytes()
    _, err = conn.Write(data)
    if err != nil {
        return fmt.Errorf("写入数据失败: %w", err)
    }

    return nil
}
```

### 2. 消息生成错误

- 处理模板变量解析错误
- 处理数据文件读取错误
- 记录错误统计信息

## 监控统计

### 1. 统计信息

```go
type Statistics struct {
    Sent      int64     // 已发送消息数
    Failed    int64     // 发送失败数
    StartTime time.Time // 开始时间
    EndTime   time.Time // 结束时间
    mutex     sync.RWMutex
}
```

### 2. 性能监控

- 实时统计发送成功和失败数
- 计算发送速率
- 支持定期打印统计信息
- 在发送完成时输出最终统计
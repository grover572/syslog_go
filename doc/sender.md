# 发送器实现说明

## 核心组件

### Sender（sender/sender.go）

发送器的主要实现，负责消息的生成和发送。

```go
type Sender struct {
    config     *config.Config
    engine     *template.Engine
    pool       *ConnectionPool
    statistics *Statistics
}
```

主要方法：
- `NewSender(config *config.Config)`: 创建新的发送器实例
- `Start()`: 启动发送器
- `Stop()`: 停止发送器
- `SendMessage(message string)`: 发送单条消息

### ConnectionPool（sender/connection.go）

连接池管理，处理TCP/UDP连接的创建、复用和关闭。

```go
type ConnectionPool struct {
    protocol    string
    connections chan net.Conn
    target      string
    sourceIP    string
}
```

主要方法：
- `NewConnectionPool(config *config.Config)`: 创建连接池
- `GetConnection()`: 获取一个可用连接
- `ReleaseConnection(conn net.Conn)`: 释放连接回池
- `Close()`: 关闭所有连接

## 实现流程

### 1. 发送器初始化

```go
// 创建发送器
func NewSender(config *config.Config) (*Sender, error) {
    // 初始化模板引擎
    engine := template.NewEngine(config.TemplatePath, config.Verbose)
    
    // 创建连接池
    pool := NewConnectionPool(config)
    
    // 初始化统计信息
    stats := NewStatistics()
    
    return &Sender{
        config:     config,
        engine:     engine,
        pool:       pool,
        statistics: stats,
    }, nil
}
```

### 2. 消息发送流程

```go
// 发送消息
func (s *Sender) SendMessage(message string) error {
    // 获取连接
    conn := s.pool.GetConnection()
    defer s.pool.ReleaseConnection(conn)
    
    // 生成消息内容
    content := s.engine.GenerateMessage(message)
    
    // 格式化Syslog消息
    syslogMsg := syslog.Format(content, s.config)
    
    // 发送消息
    _, err := conn.Write([]byte(syslogMsg))
    if err != nil {
        s.statistics.IncrementFailures()
        return err
    }
    
    s.statistics.IncrementSuccess()
    return nil
}
```

### 3. 速率控制

```go
// 速率限制器
type RateLimiter struct {
    rate      int
    bucket    chan struct{}
    closeOnce sync.Once
    done      chan struct{}
}

// 控制发送速率
func (s *Sender) controlRate() {
    ticker := time.NewTicker(time.Second / time.Duration(s.config.EPS))
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            s.sendChan <- struct{}{}
        case <-s.done:
            return
        }
    }
}
```

## 性能优化

### 1. 连接池优化

```go
// 预创建连接
func (p *ConnectionPool) initConnections() {
    for i := 0; i < p.size; i++ {
        conn, err := p.createConnection()
        if err != nil {
            continue
        }
        p.connections <- conn
    }
}

// 动态扩缩容
func (p *ConnectionPool) adjustSize() {
    currentSize := len(p.connections)
    if currentSize < p.minSize {
        p.grow(p.minSize - currentSize)
    } else if currentSize > p.maxSize {
        p.shrink(currentSize - p.maxSize)
    }
}
```

### 2. 内存优化

```go
// 使用对象池复用消息对象
var messagePool = sync.Pool{
    New: func() interface{} {
        return &Message{}
    },
}

// 获取消息对象
func getMessageFromPool() *Message {
    return messagePool.Get().(*Message)
}

// 释放消息对象
func releaseMessageToPool(msg *Message) {
    msg.Reset()
    messagePool.Put(msg)
}
```

### 3. 并发优化

```go
// 工作协程池
type WorkerPool struct {
    workers chan struct{}
    tasks   chan func()
}

// 并发发送消息
func (s *Sender) sendConcurrently(messages []string) {
    var wg sync.WaitGroup
    for _, msg := range messages {
        wg.Add(1)
        s.workerPool.tasks <- func() {
            defer wg.Done()
            s.SendMessage(msg)
        }
    }
    wg.Wait()
}
```

## 错误处理

### 1. 连接错误处理

```go
// 连接重试机制
func (p *ConnectionPool) getConnectionWithRetry() (net.Conn, error) {
    for i := 0; i < p.retryCount; i++ {
        conn, err := p.GetConnection()
        if err == nil {
            return conn, nil
        }
        time.Sleep(p.retryInterval)
    }
    return nil, errors.New("max retry count exceeded")
}
```

### 2. 发送错误处理

```go
// 处理发送错误
func (s *Sender) handleSendError(err error) {
    if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
        // 处理超时错误
        s.statistics.IncrementTimeouts()
    } else if opErr, ok := err.(*net.OpError); ok {
        // 处理网络操作错误
        s.handleNetworkError(opErr)
    } else {
        // 处理其他错误
        s.statistics.IncrementFailures()
    }
}
```

## 监控统计

### 1. 性能指标收集

```go
type Statistics struct {
    TotalSent    uint64
    TotalSuccess uint64
    TotalFailures uint64
    TotalTimeouts uint64
    AverageLatency float64
    mutex         sync.RWMutex
}

// 更新统计信息
func (s *Statistics) Update(success bool, latency time.Duration) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    s.TotalSent++
    if success {
        s.TotalSuccess++
        s.updateLatency(latency)
    } else {
        s.TotalFailures++
    }
}
```

### 2. 状态报告

```go
// 生成状态报告
func (s *Sender) GenerateReport() *Report {
    stats := s.statistics.GetSnapshot()
    return &Report{
        Duration:       time.Since(s.startTime),
        MessagesPerSec: float64(stats.TotalSent) / time.Since(s.startTime).Seconds(),
        SuccessRate:    float64(stats.TotalSuccess) / float64(stats.TotalSent),
        AverageLatency: stats.AverageLatency,
        ActiveConns:    s.pool.ActiveConnections(),
    }
}
```
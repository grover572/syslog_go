# Mock实现说明

## 核心组件

### MockServer（mock/server.go）

模拟Syslog服务器的主要实现，负责接收和验证消息。

```go
type MockServer struct {
    config     *config.Config
    listener   net.Listener
    udpConn    *net.UDPConn
    messages   chan string
    done       chan struct{}
    statistics *Statistics
}
```

主要方法：
- `NewMockServer(config *config.Config)`: 创建新的Mock服务器实例
- `Start()`: 启动服务器
- `Stop()`: 停止服务器
- `GetStatistics()`: 获取统计信息

### MessageValidator（mock/validator.go）

消息验证器，负责验证接收到的消息格式和内容。

```go
type MessageValidator struct {
    engine *template.Engine
    rules  []ValidationRule
}
```

主要方法：
- `NewMessageValidator(engine *template.Engine)`: 创建消息验证器
- `AddRule(rule ValidationRule)`: 添加验证规则
- `Validate(message string)`: 验证消息

## 实现流程

### 1. 服务器初始化

```go
// 创建Mock服务器
func NewMockServer(config *config.Config) (*MockServer, error) {
    // 初始化模板引擎
    engine := template.NewEngine(config.TemplatePath, config.Verbose)
    
    // 创建消息验证器
    validator := NewMessageValidator(engine)
    
    // 初始化统计信息
    stats := NewStatistics()
    
    return &MockServer{
        config:     config,
        messages:   make(chan string, 1000),
        done:       make(chan struct{}),
        statistics: stats,
        validator:  validator,
    }, nil
}
```

### 2. 消息接收流程

```go
// TCP消息处理
func (s *MockServer) handleTCPConnection(conn net.Conn) {
    defer conn.Close()
    reader := bufio.NewReader(conn)
    
    for {
        // 读取消息
        message, err := reader.ReadString('\n')
        if err != nil {
            return
        }
        
        // 验证消息
        if err := s.validator.Validate(message); err != nil {
            s.statistics.IncrementInvalid()
            continue
        }
        
        s.messages <- message
        s.statistics.IncrementValid()
    }
}

// UDP消息处理
func (s *MockServer) handleUDPMessages() {
    buffer := make([]byte, 65535)
    for {
        n, _, err := s.udpConn.ReadFromUDP(buffer)
        if err != nil {
            continue
        }
        
        message := string(buffer[:n])
        if err := s.validator.Validate(message); err != nil {
            s.statistics.IncrementInvalid()
            continue
        }
        
        s.messages <- message
        s.statistics.IncrementValid()
    }
}
```

### 3. 消息验证

```go
// 验证规则接口
type ValidationRule interface {
    Validate(message string) error
}

// Syslog格式验证
type SyslogFormatRule struct{}

func (r *SyslogFormatRule) Validate(message string) error {
    if !syslog.IsValidFormat(message) {
        return errors.New("invalid syslog format")
    }
    return nil
}

// 模板变量验证
type TemplateVariableRule struct {
    engine *template.Engine
}

func (r *TemplateVariableRule) Validate(message string) error {
    return r.engine.ValidateVariables(message)
}
```

## 性能优化

### 1. 并发处理

```go
// 工作协程池
type WorkerPool struct {
    workers chan struct{}
    tasks   chan func()
}

// 并发处理消息
func (s *MockServer) processMessages() {
    pool := NewWorkerPool(s.config.Workers)
    for message := range s.messages {
        msg := message // 创建副本
        pool.Submit(func() {
            s.processMessage(msg)
        })
    }
}
```

### 2. 缓冲优化

```go
// 消息缓冲池
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 0, 4096)
    },
}

// 获取缓冲区
func getBuffer() []byte {
    return bufferPool.Get().([]byte)
}

// 释放缓冲区
func releaseBuffer(buf []byte) {
    buf = buf[:0]
    bufferPool.Put(buf)
}
```

## 错误处理

### 1. 网络错误处理

```go
// 处理监听错误
func (s *MockServer) handleListenError(err error) {
    if opErr, ok := err.(*net.OpError); ok {
        if opErr.Op == "listen" {
            // 处理端口占用错误
            s.handlePortInUse(opErr)
        } else {
            // 处理其他网络错误
            s.handleNetworkError(opErr)
        }
    }
}
```

### 2. 验证错误处理

```go
// 处理验证错误
func (s *MockServer) handleValidationError(err error) {
    switch err := err.(type) {
    case *template.VariableError:
        // 处理变量错误
        s.statistics.IncrementVariableErrors()
    case *syslog.FormatError:
        // 处理格式错误
        s.statistics.IncrementFormatErrors()
    default:
        // 处理其他错误
        s.statistics.IncrementOtherErrors()
    }
}
```

## 监控统计

### 1. 性能指标收集

```go
type Statistics struct {
    TotalReceived      uint64
    ValidMessages      uint64
    InvalidMessages    uint64
    VariableErrors     uint64
    FormatErrors       uint64
    OtherErrors        uint64
    AverageProcessTime float64
    mutex             sync.RWMutex
}

// 更新统计信息
func (s *Statistics) Update(valid bool, processTime time.Duration) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    s.TotalReceived++
    if valid {
        s.ValidMessages++
        s.updateProcessTime(processTime)
    } else {
        s.InvalidMessages++
    }
}
```

### 2. 状态报告

```go
// 生成状态报告
func (s *MockServer) GenerateReport() *Report {
    stats := s.statistics.GetSnapshot()
    return &Report{
        Duration:          time.Since(s.startTime),
        MessagesPerSec:    float64(stats.TotalReceived) / time.Since(s.startTime).Seconds(),
        ValidationRate:    float64(stats.ValidMessages) / float64(stats.TotalReceived),
        AverageProcessTime: stats.AverageProcessTime,
        ErrorBreakdown:    s.getErrorBreakdown(),
    }
}
```
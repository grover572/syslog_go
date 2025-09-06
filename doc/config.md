# 配置实现说明

## 核心组件

### Config（pkg/config/config.go）

配置管理的主要实现，负责配置的加载、验证和访问。

```go
type Config struct {
    // 基础配置
    Target   string `mapstructure:"target" yaml:"target"`       // 目标服务器地址
    SourceIP string `mapstructure:"source_ip" yaml:"source_ip"` // 源IP地址
    Protocol string `mapstructure:"protocol" yaml:"protocol"`   // 传输协议

    // Syslog配置
    Format   string `mapstructure:"format" yaml:"format"`     // Syslog格式
    Facility int    `mapstructure:"facility" yaml:"facility"` // Facility值
    Severity int    `mapstructure:"severity" yaml:"severity"` // Severity值

    // 发送控制
    EPS      int           `mapstructure:"eps" yaml:"eps"`           // 每秒事件数
    Duration time.Duration `mapstructure:"duration" yaml:"duration"` // 发送持续时间

    // 数据源配置
    TemplateDir  string `mapstructure:"template_dir" yaml:"template_dir"`   // 模板目录
    TemplateFile string `mapstructure:"template_file" yaml:"template_file"` // 指定模板文件
    DataFile     string `mapstructure:"data_file" yaml:"data_file"`         // 数据文件
    Message      string `mapstructure:"message" yaml:"message"`             // 消息内容

    // 高级配置
    Concurrency int           `mapstructure:"concurrency" yaml:"concurrency"` // 并发连接数
    RetryCount  int           `mapstructure:"retry_count" yaml:"retry_count"` // 重试次数
    Timeout     time.Duration `mapstructure:"timeout" yaml:"timeout"`         // 连接超时
    BufferSize  int           `mapstructure:"buffer_size" yaml:"buffer_size"` // 缓冲区大小

    // 监控配置
    EnableStats   bool          `mapstructure:"enable_stats" yaml:"enable_stats"`     // 启用统计
    StatsInterval time.Duration `mapstructure:"stats_interval" yaml:"stats_interval"` // 统计间隔
    Verbose       bool          `mapstructure:"verbose" yaml:"verbose"`               // 详细输出
}
```

主要方法：
- `DefaultConfig()`: 返回默认配置
- `LoadConfig(configFile string)`: 从文件加载配置
- `Validate()`: 验证配置有效性
- `GetPriority()`: 计算Syslog优先级

## 实现流程

### 1. 配置加载

```go
// 加载配置文件
func LoadConfig(configFile string) (*Config, error) {
    cfg := DefaultConfig()

    // 如果指定了配置文件，尝试读取
    if configFile != "" {
        viper.SetConfigFile(configFile)
        if err := viper.ReadInConfig(); err != nil {
            return nil, fmt.Errorf("读取配置文件失败: %w", err)
        }
    }

    // 将viper配置解析到结构体
    if err := viper.Unmarshal(cfg); err != nil {
        return nil, fmt.Errorf("配置解析失败: %w", err)
    }

    // 验证配置
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("配置验证失败: %w", err)
    }

    return cfg, nil
}
```

### 2. 配置验证

```go
// 验证配置的有效性
func (c *Config) Validate() error {
    if c.Target == "" {
        return fmt.Errorf("目标服务器地址不能为空")
    }

    if c.Protocol != "udp" && c.Protocol != "tcp" {
        return fmt.Errorf("协议必须是 udp 或 tcp")
    }

    if c.Format != "rfc3164" && c.Format != "rfc5424" {
        return fmt.Errorf("格式必须是 rfc3164 或 rfc5424")
    }

    if c.Facility < 0 || c.Facility > 23 {
        return fmt.Errorf("Facility必须在0-23范围内")
    }

    if c.Severity < 0 || c.Severity > 7 {
        return fmt.Errorf("Severity必须在0-7范围内")
    }

    if c.EPS <= 0 {
        return fmt.Errorf("EPS必须大于0")
    }

    if c.Duration <= 0 {
        return fmt.Errorf("持续时间必须大于0")
    }

    if c.Concurrency <= 0 {
        return fmt.Errorf("并发数必须大于0")
    }

    return nil
}
```

### 3. 默认配置

```go
// 返回默认配置
func DefaultConfig() *Config {
    return &Config{
        Target:        "localhost:514",
        Protocol:      "udp",
        Format:        "rfc3164",
        Facility:      16, // local0
        Severity:      6,  // info
        EPS:           10,
        Duration:      60 * time.Second,
        TemplateDir:   "./data/templates",
        Concurrency:   1,
        RetryCount:    3,
        Timeout:       5 * time.Second,
        BufferSize:    1000,
        EnableStats:   true,
        StatsInterval: 5 * time.Second,
        Verbose:       false,
    }
}
```

## 性能优化

### 1. 配置缓存

- 使用viper缓存配置文件内容
- 避免频繁读取配置文件

### 2. 参数验证

- 提前验证配置参数有效性
- 避免运行时错误

## 错误处理

### 1. 配置错误

- 文件读取错误
- 格式解析错误
- 参数验证错误

### 2. 运行时错误

- 网络连接错误
- 资源分配错误
- 超时错误
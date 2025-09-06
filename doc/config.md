# 配置实现说明

## 核心组件

### Config（config/config.go）

配置管理的主要实现，负责配置的加载、验证和访问。

```go
type Config struct {
    // 基本配置
    Protocol    string `yaml:"protocol"`    // 协议类型：tcp/udp
    Host        string `yaml:"host"`        // 目标主机
    Port        int    `yaml:"port"`        // 目标端口
    SourceIP    string `yaml:"source_ip"`   // 源IP地址
    
    // 性能配置
    EPS         int    `yaml:"eps"`         // 每秒发送消息数
    Workers     int    `yaml:"workers"`     // 工作协程数
    BufferSize  int    `yaml:"buffer_size"` // 缓冲区大小
    
    // 模板配置
    TemplatePath string `yaml:"template_path"` // 模板文件路径
    Variables    map[string]Variable `yaml:"variables"` // 自定义变量
    
    // 调试配置
    Verbose     bool   `yaml:"verbose"`     // 是否输出详细日志
    LogLevel    string `yaml:"log_level"`   // 日志级别
}
```

主要方法：
- `LoadConfig(path string)`: 加载配置文件
- `Validate()`: 验证配置有效性
- `GetValue(key string)`: 获取配置值

### Variable（config/variable.go）

变量配置的定义和处理。

```go
type Variable struct {
    Type    string   `yaml:"type"`    // 变量类型
    Values  []string `yaml:"values"`  // 可选值列表
    Min     int      `yaml:"min"`     // 最小值
    Max     int      `yaml:"max"`     // 最大值
    Length  int      `yaml:"length"`  // 字符串长度
    Pattern string   `yaml:"pattern"` // 正则表达式模式
}
```

## 实现流程

### 1. 配置加载

```go
// 加载配置文件
func LoadConfig(path string) (*Config, error) {
    // 读取配置文件
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("read config file: %v", err)
    }
    
    // 解析YAML
    config := &Config{}
    if err := yaml.Unmarshal(data, config); err != nil {
        return nil, fmt.Errorf("parse config: %v", err)
    }
    
    // 设置默认值
    config.setDefaults()
    
    // 验证配置
    if err := config.Validate(); err != nil {
        return nil, fmt.Errorf("validate config: %v", err)
    }
    
    return config, nil
}
```

### 2. 配置验证

```go
// 验证配置
func (c *Config) Validate() error {
    // 验证协议
    if !isValidProtocol(c.Protocol) {
        return errors.New("invalid protocol")
    }
    
    // 验证主机和端口
    if err := validateAddress(c.Host, c.Port); err != nil {
        return err
    }
    
    // 验证性能参数
    if err := c.validatePerformance(); err != nil {
        return err
    }
    
    // 验证模板配置
    if err := c.validateTemplate(); err != nil {
        return err
    }
    
    return nil
}

// 验证性能参数
func (c *Config) validatePerformance() error {
    if c.EPS <= 0 {
        return errors.New("eps must be positive")
    }
    
    if c.Workers <= 0 {
        return errors.New("workers must be positive")
    }
    
    if c.BufferSize <= 0 {
        return errors.New("buffer_size must be positive")
    }
    
    return nil
}
```

### 3. 变量配置处理

```go
// 验证变量配置
func (v *Variable) Validate() error {
    switch v.Type {
    case "random_choice":
        if len(v.Values) == 0 {
            return errors.New("random_choice requires non-empty values")
        }
    case "random_int":
        if v.Min >= v.Max {
            return errors.New("random_int requires min < max")
        }
    case "random_string":
        if v.Length <= 0 {
            return errors.New("random_string requires positive length")
        }
    case "pattern":
        if _, err := regexp.Compile(v.Pattern); err != nil {
            return fmt.Errorf("invalid pattern: %v", err)
        }
    default:
        return fmt.Errorf("unknown variable type: %s", v.Type)
    }
    return nil
}
```

## 性能优化

### 1. 配置缓存

```go
// 配置缓存
type ConfigCache struct {
    config     atomic.Value
    updateTime time.Time
    mutex      sync.RWMutex
}

// 获取配置（带缓存）
func (c *ConfigCache) GetConfig() *Config {
    if config := c.config.Load(); config != nil {
        return config.(*Config)
    }
    return nil
}

// 更新配置缓存
func (c *ConfigCache) UpdateConfig(config *Config) {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    c.config.Store(config)
    c.updateTime = time.Now()
}
```

### 2. 变量值缓存

```go
// 变量值缓存
type VariableCache struct {
    values map[string]interface{}
    mutex  sync.RWMutex
}

// 获取变量值（带缓存）
func (c *VariableCache) GetValue(name string) (interface{}, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    
    value, ok := c.values[name]
    return value, ok
}
```

## 错误处理

### 1. 配置错误

```go
// 配置错误类型
type ConfigError struct {
    Field   string
    Message string
}

func (e *ConfigError) Error() string {
    return fmt.Sprintf("config error: %s: %s", e.Field, e.Message)
}

// 处理配置错误
func handleConfigError(err error) error {
    switch err := err.(type) {
    case *yaml.TypeError:
        return &ConfigError{"yaml", err.Error()}
    case *ConfigError:
        return err
    default:
        return &ConfigError{"unknown", err.Error()}
    }
}
```

### 2. 变量错误

```go
// 变量错误类型
type VariableError struct {
    Name    string
    Message string
}

func (e *VariableError) Error() string {
    return fmt.Sprintf("variable error: %s: %s", e.Name, e.Message)
}

// 处理变量错误
func handleVariableError(name string, err error) error {
    return &VariableError{name, err.Error()}
}
```

## 扩展性设计

### 1. 配置接口

```go
// 配置提供者接口
type ConfigProvider interface {
    Load() (*Config, error)
    Save(*Config) error
    Watch(chan<- *Config)
}

// 文件配置提供者
type FileConfigProvider struct {
    path string
}

// 远程配置提供者
type RemoteConfigProvider struct {
    endpoint string
    client   *http.Client
}
```

### 2. 变量类型扩展

```go
// 变量生成器接口
type VariableGenerator interface {
    Generate() interface{}
    Validate() error
}

// 注册变量类型
func RegisterVariableType(name string, generator VariableGenerator) {
    variableGenerators[name] = generator
}
```
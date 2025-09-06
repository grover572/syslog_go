# 模板引擎实现说明

## 核心组件

### Engine（template/engine.go）

模板引擎的主要实现，负责模板的加载、解析和渲染。

```go
type Engine struct {
    verbose bool
    parser  *VariableParser
}
```

主要方法：
- `NewEngine(configPath string, verbose bool)`: 创建新的模板引擎实例
- `LoadConfig(configPath string)`: 加载模板配置文件
- `GenerateMessage(template string)`: 生成消息内容

### VariableParser（template/parser.go）

变量解析器，处理模板中的变量替换。

```go
type VariableParser struct {
    verbose bool
    customVariables map[string]Variable
}
```

主要方法：
- `NewVariableParser(verbose bool)`: 创建变量解析器实例
- `RegisterCustomVariable(name string, v Variable)`: 注册自定义变量
- `Parse(template string)`: 解析模板中的变量

## 变量类型

### 内置变量

1. IP地址相关
   - `RANDOM_IP`: 生成随机IP地址
   - `RANGE_IP`: 在指定范围内生成IP地址
   - 实现：使用net包进行IP地址操作

2. 随机数据
   - `RANDOM_INT`: 生成指定范围内的随机整数
   - `RANDOM_STRING`: 生成指定长度的随机字符串
   - 实现：使用math/rand包生成随机数

3. 网络相关
   - `MAC`: 生成随机MAC地址
   - `RANDOM_PORT`: 生成随机端口号
   - 实现：使用自定义算法生成符合格式的数据

### 自定义变量

通过YAML配置文件定义，支持以下类型：

1. random_choice
```yaml
CUSTOM_STATUS:
  type: "random_choice"
  values:
    - "正常"
    - "警告"
    - "错误"
```

2. random_int
```yaml
CUSTOM_SCORE:
  type: "random_int"
  min: 0
  max: 100
```

## 实现流程

1. 初始化
```go
// 创建模板引擎
engine := NewEngine(configPath, verbose)

// 创建变量解析器
parser := NewVariableParser(verbose)
```

2. 配置加载
```go
// 加载配置文件
config := LoadConfig(configPath)

// 注册自定义变量
for name, v := range config.Variables {
    parser.RegisterCustomVariable(name, v)
}
```

3. 模板解析
```go
// 解析模板
result := parser.Parse(template)

// 替换变量
for _, v := range result.Variables {
    value := generateValue(v)
    result.Content = strings.Replace(result.Content, v.Raw, value, 1)
}
```

## 性能优化

1. 正则表达式缓存
```go
// 预编译正则表达式
var variableRegex = regexp.MustCompile(`{{([^}]+)}})`)
```

2. 变量缓存
```go
// 缓存已注册的变量
type VariableParser struct {
    customVariables map[string]Variable
    // ...
}
```

3. 字符串处理优化
```go
// 使用strings.Builder进行字符串拼接
var builder strings.Builder
for _, part := range parts {
    builder.WriteString(part)
}
```

## 错误处理

1. 配置验证
```go
// 验证变量配置
func validateVariable(v Variable) error {
    switch v.Type {
    case "random_choice":
        if len(v.Values) == 0 {
            return errors.New("random_choice requires non-empty values")
        }
    case "random_int":
        if v.Min >= v.Max {
            return errors.New("random_int requires min < max")
        }
    }
    return nil
}
```

2. 变量解析错误处理
```go
// 处理变量解析错误
func (p *VariableParser) Parse(template string) (*Result, error) {
    matches := variableRegex.FindAllStringSubmatch(template, -1)
    if matches == nil {
        return nil, errors.New("no variables found in template")
    }
    // ...
}
```

## 扩展性设计

1. 变量接口
```go
// 变量生成器接口
type VariableGenerator interface {
    Generate() string
}
```

2. 自定义变量类型
```go
// 注册新的变量类型
func (p *VariableParser) RegisterVariableType(name string, generator VariableGenerator) {
    p.generators[name] = generator
}
```

3. 配置扩展
```go
// 支持多种配置格式
type Config struct {
    Variables map[string]Variable `yaml:"variables" json:"variables"`
}
```
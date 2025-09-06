# 模板引擎实现说明

## 核心组件

### Engine（pkg/template/engine.go）

模板引擎的主要实现，负责模板的加载、解析和渲染。

```go
type Engine struct {
    templateCache map[string]string
    parser       *VariableParser
    configPath   string        // 自定义变量配置文件路径
    verbose     bool          // 是否显示详细日志
}
```

主要方法：
- `NewEngine(configPath string, verbose bool)`: 创建新的模板引擎实例
- `LoadTemplate(name, content string)`: 加载模板到缓存
- `GenerateMessage(templateName string)`: 生成消息内容
- `SetVariableParser(parser *VariableParser)`: 设置变量解析器

### VariableParser（pkg/template/parser.go）

变量解析器，处理模板中的变量替换。

```go
type VariableParser struct {
    random          *rand.Rand
    customVariables map[string]CustomVariable
    verbose         bool
}
```

## 实现流程

### 1. 引擎初始化

```go
// 创建模板引擎
func NewEngine(configPath string, verbose bool) *Engine {
    // 创建变量解析器
    parser := NewVariableParser(verbose)

    e := &Engine{
        templateCache: make(map[string]string),
        parser:       parser,
        configPath:   configPath,
        verbose:     verbose,
    }
    
    // 如果提供了配置文件路径，尝试加载自定义变量
    if configPath != "" {
        if err := e.loadCustomVariables(configPath); err != nil {
            if e.verbose {
                fmt.Printf("警告: 加载自定义变量配置失败: %v\n", err)
            }
        }
    }
    
    return e
}
```

### 2. 模板处理

```go
// 处理模板
func (e *Engine) processTemplate(template string) (string, error) {
    // 匹配变量表达式 {{变量名:参数}}
    varRegex := regexp.MustCompile(`\{\{\s*([^{}]+?)\s*\}\}`)

    // 替换所有变量
    var lastErr error
    result := varRegex.ReplaceAllStringFunc(template, func(match string) string {
        // 提取变量表达式
        expr := varRegex.FindStringSubmatch(match)[1]

        // 使用当前的变量解析器
        value, err := e.parser.Parse(expr)
        if err != nil {
            // 记录错误
            lastErr = fmt.Errorf("解析变量 %s 失败: %w", expr, err)
            // 如果解析失败，保留原始表达式
            return match
        }

        return value
    })

    if lastErr != nil {
        return "", lastErr
    }

    return strings.TrimSpace(result), nil
}
```

## 变量类型

### 内置变量

1. IP地址相关
   - `RANDOM_IP`: 生成随机IP地址
   - `RANGE_IP`: 在指定范围内生成IP地址
   - `RANDOM_IPV6`: 生成随机IPv6地址

2. 网络相关
   - `MAC`: 生成随机MAC地址
   - `RANDOM_PORT`: 生成随机端口号
   - `PROTOCOL`: 生成网络协议名称

3. 随机数据
   - `RANDOM_INT`: 生成指定范围内的随机整数
   - `RANDOM_STRING`: 生成指定长度的随机字符串
   - `EMAIL`: 生成随机邮箱地址

### 自定义变量

通过YAML配置文件定义，支持以下类型：

```yaml
# 随机选择类型变量示例
CUSTOM_STATUS:
  type: random_choice
  values:
    - "正常"
    - "警告"
    - "错误"
    - "严重"

# 随机整数类型变量示例
CUSTOM_SCORE:
  type: random_int
  min: 0
  max: 100

# 随机字符串类型变量示例
CUSTOM_ID:
  type: random_string
  length: 8
```

## 性能优化

### 1. 模板缓存

- 使用`templateCache`缓存已加载的模板
- 避免重复解析相同的模板内容

### 2. 正则优化

- 预编译变量匹配的正则表达式
- 使用高效的正则表达式模式

### 3. 内存优化

- 使用`strings.Builder`进行字符串拼接
- 复用变量解析器实例

## 错误处理

### 1. 配置错误

- 验证配置文件格式
- 检查必要的配置项
- 处理配置加载失败

### 2. 变量错误

- 处理变量解析失败
- 验证变量参数
- 记录详细的错误信息
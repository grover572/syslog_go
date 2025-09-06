# Mock实现说明

## 核心组件

### Mock命令（cmd/root.go）

Mock命令用于生成模拟数据，支持模板变量替换。

```go
var mockCmd = &cobra.Command{
    Use:   "mock",
    Short: "生成模拟数据",
    Long:  `生成模拟数据...
}
```

主要参数：
- `message`: 指定消息模板
- `output`: 输出文件路径
- `count`: 生成消息的数量
- `append`: 追加到输出文件

## 实现流程

### 1. 模板引擎初始化

```go
// 创建模板引擎
configPath := "template.yml"
if _, err := os.Stat(configPath); os.IsNotExist(err) {
    configPath = "" // 如果文件不存在，使用空字符串
}
verbose := viper.GetBool("verbose")
engine := template.NewEngine(configPath, verbose)
```

### 2. 消息生成流程

```go
// 加载消息模板
engine.LoadTemplate("message", mockMessage)

// 生成指定数量的消息
var messages []string
for i := 0; i < mockCount; i++ {
    msg, err := engine.GenerateMessage("message")
    if err != nil {
        fmt.Fprintf(os.Stderr, "生成第 %d 条消息时出错: %v\n", i+1, err)
        os.Exit(1)
    }
    messages = append(messages, msg)
}
```

## 变量解析

### 1. 变量类型

- 网络变量：RANDOM_IP、RANGE_IP、MAC等
- 时间变量：TIMESTAMP
- 随机数据：RANDOM_INT、RANDOM_STRING

### 2. 解析实现

```go
// processTemplate 处理模板
func (e *Engine) processTemplate(template string) (string, error) {
    // 匹配变量表达式 {{变量名:参数}}
    varRegex := regexp.MustCompile(`\{\{\s*([^{}]+?)\s*\}\}`)

    // 替换所有变量
    result := varRegex.ReplaceAllStringFunc(template, func(match string) string {
        // 提取变量表达式
        expr := varRegex.FindStringSubmatch(match)[1]

        // 使用变量解析器
        value, err := e.parser.Parse(expr)
        if err != nil {
            return match
        }
        return value
    })

    return strings.TrimSpace(result), nil
}
```

## 错误处理

### 1. 模板错误

- 验证模板格式
- 检查变量语法
- 处理变量解析错误

### 2. 输出错误

- 文件写入错误处理
- 追加模式错误处理

## 输出处理

### 1. 文件输出

```go
if mockOutput != "" {
    if mockAppend {
        // 追加模式
        f, err := os.OpenFile(mockOutput, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err != nil {
            fmt.Fprintf(os.Stderr, "打开输出文件失败: %v\n", err)
            os.Exit(1)
        }
        defer f.Close()
        
        _, err = f.WriteString(output)
        if err != nil {
            fmt.Fprintf(os.Stderr, "写入输出文件失败: %v\n", err)
            os.Exit(1)
        }
    } else {
        // 覆盖模式
        err = os.WriteFile(mockOutput, []byte(output), 0644)
        if err != nil {
            fmt.Fprintf(os.Stderr, "写入输出文件失败: %v\n", err)
            os.Exit(1)
        }
    }
} else {
    fmt.Print(output)
}
```
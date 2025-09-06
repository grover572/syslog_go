# Syslog发送工具

一个功能强大的Syslog消息发送工具，支持高性能批量发送、模板化数据生成和交互式配置。

## 功能特性

### 核心功能
- 🚀 **高性能发送**: 支持TCP/UDP协议，可配置并发连接和发送速率
- 📝 **模板系统**: 支持动态变量替换的模板系统
- 🎯 **多种数据源**: 支持模板文件和数据文件
- 📊 **实时监控**: 实时统计发送状态和性能指标
- 🌐 **协议支持**: 完整支持RFC3164和RFC5424格式

### 高级特性
- ⚡ **连接池管理**: 智能连接复用和管理
- 🎛️ **速率控制**: 精确的EPS（每秒事件数）控制
- 🔄 **自动重试**: 可配置的重试机制

## 快速开始

### 安装

```bash
# 克隆项目
git clone <repository-url>
cd syslog_go

# 构建（需要Go 1.21+）
go build
```

### 基本使用

```bash
# 使用默认配置发送
go run . send -m "Hello World"

# 指定目标地址和协议
go run . send -m "Test Message" -t 192.168.1.100:514 -p tcp

# 使用模板变量
go run . send -m "源IP: {{RANDOM_IP}}, 目标IP: {{RANDOM_IP}}" -e 10

# 使用mock命令测试模板
go run . mock -m "源IP: {{RANDOM_IP}}, 目标IP: {{RANDOM_IP}}" -n 5
```

## 命令行参数

### Send命令
```
使用方法:
  syslog_go send [flags]

常用标志:
  -m, --message string       消息内容或模板
  -t, --target string        目标服务器地址 (默认 "localhost:514")
  -e, --eps int              每秒事件数 (默认 10)
  -d, --duration string      发送持续时间 (默认 "60s")
  -p, --protocol string      传输协议 tcp/udp (默认 "udp")
  -f, --format string        Syslog格式 rfc3164/rfc5424 (默认 "rfc3164")
  -s, --source-ip string     源IP地址
  -L, --facility int         Facility值 (默认 16)
  -S, --severity int         Severity值 (默认 6)
  -v, --verbose              显示详细信息
```

### Mock命令
```
使用方法:
  syslog_go mock [flags]

常用标志:
  -m, --message string       消息内容或模板
  -n, --number int           生成消息的数量 (默认 1)
  -v, --verbose              显示详细信息
```

## 模板变量

### 内置变量

#### 网络变量
- `{{RANDOM_IP}}` - 随机IP地址
- `{{RANGE_IP:192.168.1.1/24}}` - 指定范围内的IP地址
- `{{RANDOM_PORT}}` - 随机端口
- `{{MAC}}` - 随机MAC地址

#### 时间变量
- `{{TIMESTAMP}}` - 当前时间戳

#### 随机数据
- `{{RANDOM_INT:1-100}}` - 指定范围内的随机整数
- `{{RANDOM_STRING:10}}` - 指定长度的随机字符串

### 自定义变量

在`template.yml`中定义：

```yaml
variables:
  CUSTOM_STATUS:
    type: "random_choice"
    values:
      - "正常"
      - "警告"
      - "错误"
  
  CUSTOM_SCORE:
    type: "random_int"
    min: 0
    max: 100
```

使用示例：
```bash
go run . mock -m "状态: {{CUSTOM_STATUS}}, 分数: {{CUSTOM_SCORE}}" -n 5
```

## 许可证

MIT License
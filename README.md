# Syslog发送工具

一个功能强大的Syslog消息发送工具，支持高性能批量发送、模板化数据生成和交互式配置。

## 功能特性

### 核心功能
- 🚀 **高性能发送**: 支持TCP/UDP协议，可配置并发连接和发送速率
- 📝 **模板系统**: 基于文件的模板系统，支持动态变量替换
- 🎯 **多种数据源**: 支持模板文件、数据文件和混合模式
- 📊 **实时监控**: 实时统计发送状态和性能指标
- 🔧 **交互式配置**: 命令行交互模式，便于快速配置
- 🌐 **协议支持**: 完整支持RFC3164和RFC5424格式

### 高级特性
- ⚡ **连接池管理**: 智能连接复用和管理
- 🎛️ **速率控制**: 精确的EPS（每秒事件数）控制
- 🔄 **自动重试**: 可配置的重试机制
- 📈 **负载测试**: 支持高并发压力测试
- 🎨 **丰富模板**: 内置安全、系统、网络、应用等多种日志模板

## 快速开始

### 安装

```bash
# 克隆项目
git clone <repository-url>
cd syslog_sender

# 构建（需要Go 1.21+）
go build -o syslog_sender ./cmd/syslog_sender

# 或使用构建脚本
./scripts/build.ps1
```

### 基本使用

```bash
# 使用默认配置发送
./syslog_sender

# 指定配置文件
./syslog_sender --config config.yaml

# 交互式模式
./syslog_sender --interactive

# 快速发送测试
./syslog_sender --target localhost:514 --eps 100 --duration 30s
```

### 配置示例

```yaml
# 基础配置
target: "localhost:514"
protocol: "udp"
format: "rfc3164"

# 发送控制
eps: 10
duration: "60s"
concurrency: 1

# 数据源
template_dir: "./data/templates"
```

## 使用场景

### 1. 安全设备测试
```bash
# 模拟SSH登录日志
./syslog_sender --template-file ./data/templates/security/ssh_login.log --eps 50

# 模拟防火墙日志
./syslog_sender --template-file ./data/templates/security/firewall.log --eps 100
```

### 2. 系统日志模拟
```bash
# 模拟内核日志
./syslog_sender --template-file ./data/templates/system/kernel.log --eps 20

# 模拟应用日志
./syslog_sender --template-dir ./data/templates/application --eps 200
```

### 3. 压力测试
```bash
# 高并发测试
./syslog_sender --eps 1000 --concurrency 10 --duration 5m

# 长时间稳定性测试
./syslog_sender --eps 50 --duration 24h
```

### 4. SIEM测试
```bash
# 混合日志类型
./syslog_sender --template-dir ./data/templates --eps 500 --duration 1h
```

## 模板系统

### 模板文件格式

模板文件支持动态变量替换：

```
{{timestamp}} {{hostname}} sshd[{{pid}}]: Accepted password for {{username}} from {{random_ip}} port {{random_port}} ssh2
{{timestamp}} {{hostname}} sshd[{{pid}}]: Failed password for {{username}} from {{random_ip}} port {{random_port}} ssh2
```

### 支持的变量类型

#### 网络变量
- `{{random_ip}}` - 随机IP地址
- `{{internal_ip}}` - 内网IP地址
- `{{external_ip}}` - 外网IP地址
- `{{random_port}}` - 随机端口
- `{{random_mac}}` - 随机MAC地址

#### 时间变量
- `{{timestamp}}` - 标准时间戳
- `{{timestamp_apache}}` - Apache格式时间
- `{{timestamp_iso}}` - ISO格式时间
- `{{timestamp_unix}}` - Unix时间戳

#### 用户变量
- `{{username}}` - 用户名
- `{{hostname}}` - 主机名

#### 系统变量
- `{{pid}}` - 进程ID
- `{{process}}` - 进程名
- `{{hex_id}}` - 十六进制ID
- `{{session_id}}` - 会话ID

#### HTTP变量
- `{{http_method}}` - HTTP方法
- `{{http_status}}` - HTTP状态码
- `{{url_path}}` - URL路径
- `{{user_agent}}` - 用户代理

#### 数据变量
- `{{response_size}}` - 响应大小
- `{{bytes}}` - 字节数
- `{{duration}}` - 持续时间

### 自定义变量

在 `data/variables/placeholders.yaml` 中定义自定义变量：

```yaml
custom_variables:
  my_custom_ip:
    type: "random_choice"
    values:
      - "192.168.1.100"
      - "192.168.1.101"
      - "192.168.1.102"
```

## 命令行参数

```
使用方法:
  syslog_sender [flags]

标志:
  -c, --config string          配置文件路径 (默认 "config.yaml")
  -t, --target string          目标服务器地址 (默认 "localhost:514")
  -p, --protocol string        传输协议 tcp/udp (默认 "udp")
  -f, --format string          Syslog格式 rfc3164/rfc5424 (默认 "rfc3164")
  -e, --eps int                每秒事件数 (默认 10)
  -d, --duration string        发送持续时间 (默认 "60s")
      --concurrency int        并发连接数 (默认 1)
      --template-dir string    模板目录路径
      --template-file string   模板文件路径
      --data-file string       数据文件路径
  -i, --interactive           交互式模式
      --source-ip string       源IP地址
      --facility int           Facility值 (默认 16)
      --severity int           Severity值 (默认 6)
  -h, --help                   帮助信息
  -v, --version               版本信息
```

## 配置文件

完整的配置文件示例：

```yaml
# 目标配置
target: "localhost:514"
source_ip: ""
protocol: "udp"

# Syslog配置
format: "rfc3164"
facility: 16
severity: 6

# 发送控制
eps: 10
duration: "60s"
concurrency: 1
retry_count: 3
timeout: "5s"
buffer_size: 1000

# 数据源
template_dir: "./data/templates"
template_file: ""
data_file: ""

# 监控
enable_stats: true
stats_interval: "5s"
```

## 性能优化

### 高性能配置

```yaml
# 高并发配置
eps: 1000
concurrency: 10
buffer_size: 5000
timeout: "1s"

# 网络优化
protocol: "udp"  # UDP比TCP更快
```

### 内存优化

```yaml
# 内存优化配置
buffer_size: 1000  # 适中的缓冲区
concurrency: 2     # 较少的并发数
```

## 监控和统计

工具提供实时统计信息：

- 发送速率 (EPS)
- 成功/失败计数
- 网络延迟
- 连接状态
- 内存使用

## 故障排除

### 常见问题

1. **连接被拒绝**
   - 检查目标服务器地址和端口
   - 确认防火墙设置
   - 验证Syslog服务是否运行

2. **发送速率不达预期**
   - 增加并发连接数
   - 检查网络带宽
   - 调整缓冲区大小

3. **模板变量不生效**
   - 检查模板文件格式
   - 验证变量配置文件
   - 确认变量名称正确

### 调试模式

```bash
# 启用详细日志
./syslog_sender --config config.yaml --verbose

# 测试模式（不实际发送）
./syslog_sender --dry-run
```

## 开发

### 项目结构

```
syslog_sender/
├── cmd/syslog_sender/     # 主程序入口
├── pkg/                   # 核心包
│   ├── config/           # 配置管理
│   ├── syslog/           # Syslog协议
│   ├── sender/           # 发送核心
│   ├── template/         # 模板引擎
│   └── ui/               # 交互界面
├── data/                 # 数据文件
│   ├── templates/        # 模板文件
│   ├── variables/        # 变量配置
│   └── samples/          # 示例数据
├── scripts/              # 构建脚本
└── docs/                 # 文档
```

### 构建

```bash
# 开发构建
go build -o syslog_sender ./cmd/syslog_sender

# 发布构建
./scripts/build.ps1 -Release

# 跨平台构建
./scripts/build.ps1 -Target linux -Arch amd64
```

### 测试

```bash
# 运行测试
go test ./...

# 性能测试
go test -bench=. ./pkg/sender

# 覆盖率测试
go test -cover ./...
```

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request！

## 更新日志

### v1.0.0
- 初始版本发布
- 支持基本Syslog发送功能
- 模板系统实现
- 交互式配置界面
- 性能监控和统计
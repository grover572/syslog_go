# Syslog Sender 使用指南

## 目录
1. [简介](#简介)
2. [安装配置](#安装配置)
3. [功能说明](#功能说明)
4. [使用方法](#使用方法)
5. [高级特性](#高级特性)

## 简介
Syslog Sender是一个高性能的Syslog消息发送工具，支持多种传输协议和消息格式，可用于系统日志测试、性能评估和模拟真实环境下的日志发送场景。

## 安装配置

### 系统要求
- 支持Windows和Linux操作系统
- 无特殊依赖要求

### 配置说明

#### 基础配置
```yaml
target: "localhost:514"    # 目标服务器地址
source_ip: ""             # 源IP地址
protocol: "udp"           # 传输协议(udp/tcp)
format: "rfc3164"         # Syslog格式(rfc3164/rfc5424)
facility: 16              # Facility值(0-23)
severity: 6               # Severity值(0-7)
```

#### 发送控制
```yaml
eps: 10                   # 每秒事件数
duration: 60s             # 发送持续时间
concurrency: 1            # 并发连接数
retry_count: 3            # 重试次数
timeout: 5s               # 连接超时
buffer_size: 1000         # 缓冲区大小
```

## 功能说明

### 核心功能
1. 支持UDP和TCP传输协议
2. 支持RFC3164和RFC5424格式
3. 支持模板化消息生成
4. 支持多并发连接
5. 内置性能统计和监控

### 消息模板
- 支持自定义模板目录
- 支持数据文件驱动
- 支持动态变量替换

## 使用方法

### 基本命令
```bash
# 使用默认配置发送消息
syslog_go send

# 指定配置文件
syslog_go send -c config.yaml

# 指定目标服务器和协议
syslog_go send --target 192.168.1.100:514 --protocol tcp
```

### 高级用法
```bash
# 使用模板发送消息
syslog_go send --template-file custom.tmpl --data-file data.json

# 设置发送速率和持续时间
syslog_go send --eps 100 --duration 5m

# 启用详细输出
syslog_go send --verbose
```

## 高级特性

### 性能优化
- 连接池复用
- 消息缓冲
- 并发控制

### 错误处理
- 自动重试机制
- 超时控制
- 错误统计

### 监控统计
- 实时吞吐量
- 延迟统计
- 错误计数

## 常见问题

1. 连接失败
   - 检查目标服务器地址和端口
   - 确认网络连接正常
   - 验证防火墙设置

2. 性能问题
   - 调整并发连接数
   - 优化缓冲区大小
   - 检查网络带宽

3. 消息格式错误
   - 确认Syslog格式设置
   - 检查模板语法
   - 验证数据文件格式

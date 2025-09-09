# 源IP地址模拟功能

本文档介绍如何使用syslog_go的源IP地址模拟功能。

## 功能概述

syslog_go现在支持两种源IP地址设置模式：

1. **本机IP地址模式**：使用本机已有的IP地址作为源地址
2. **原始套接字模式**：使用原始套接字技术模拟任意源IP地址

## 使用方法

### 基本语法

```bash
./syslog_go send -s <源IP地址> -t <目标地址> -p <协议> [其他参数]
```

### 示例

#### 使用本机IP地址
```bash
# 使用本机IP 192.168.1.100
./syslog_go send -s 192.168.1.100 -t 192.168.1.1:514 -p udp -e 10
```

#### 模拟任意源IP地址
```bash
# 模拟源IP 1.1.1.1（需要特殊权限）
./syslog_go send -s 1.1.1.1 -t 192.168.1.1:514 -p tcp -e 5
```

## 系统要求和权限

### Linux系统

在Linux系统上使用原始套接字模拟源IP地址需要以下条件之一：

1. **Root权限**：
   ```bash
   sudo ./syslog_go send -s 1.1.1.1 -t target:514 -p tcp
   ```

2. **CAP_NET_RAW能力**：
   ```bash
   # 给程序添加CAP_NET_RAW能力
   sudo setcap cap_net_raw+ep ./syslog_go
   
   # 然后可以以普通用户运行
   ./syslog_go send -s 1.1.1.1 -t target:514 -p tcp
   ```

### Windows系统

在Windows系统上使用原始套接字需要：

1. **管理员权限**：
   - 以管理员身份运行PowerShell或命令提示符
   - 然后执行syslog_go命令

2. **注意事项**：
   - Windows对原始套接字有更严格的限制
   - 某些Windows版本可能完全禁用原始套接字
   - 如果原始套接字创建失败，程序会自动回退到标准连接

### macOS系统

类似于Linux，需要root权限或适当的系统配置。

## 工作原理

### 本机IP地址模式

当指定的源IP地址是本机已有的IP地址时：

1. 程序检测IP地址是否为本机IP
2. 使用标准的`net.Dialer`设置`LocalAddr`
3. 创建正常的TCP/UDP连接

### 原始套接字模式

当指定的源IP地址不是本机IP时：

1. 程序尝试创建原始套接字（SOCK_RAW）
2. 手动构造IP头和传输层头（TCP/UDP）
3. 在IP头中设置指定的源IP地址
4. 直接发送原始数据包

## 技术细节

### IP包构造

原始套接字模式下，程序会构造完整的IP数据包：

- **IP头**：20字节，包含源IP、目标IP、协议类型等
- **传输层头**：TCP头（20字节）或UDP头（8字节）
- **数据载荷**：Syslog消息内容

### 原始套接字实现
- 使用 `SOCK_RAW` 套接字类型
- 设置 `IP_HDRINCL` 选项自行构造IP头
- 手动构造TCP/UDP头部和数据包
- 计算IP、TCP、UDP校验和
- TCP使用SYN标志建立连接（适合单向发送）

### 平台差异
- **Windows**: IP_HDRINCL = 2
- **Linux**: IP_HDRINCL = 1
- 不同的套接字句柄类型和系统调用

### 校验和计算

程序会自动计算并设置：
- IP头校验和
- TCP/UDP校验和（包括伪头）

## 限制和注意事项

### 网络限制

1. **路由器/防火墙过滤**：
   - 许多网络设备会过滤或丢弃源IP地址不匹配的数据包
   - ISP可能实施源地址验证（BCP 38）

2. **网络拓扑**：
   - 模拟的源IP必须在网络路由表中有返回路径
   - 否则可能无法收到响应（对于TCP连接）

### 安全考虑

1. **合法使用**：
   - 仅在授权的网络环境中使用
   - 不要用于恶意攻击或欺骗

2. **测试环境**：
   - 建议在隔离的测试网络中验证功能
   - 避免在生产网络中进行未授权的测试

### 功能限制

1. **协议支持**：
   - 目前支持TCP和UDP协议
   - 不支持其他传输层协议

2. **IPv6支持**：
   - 当前实现主要针对IPv4
   - IPv6支持可能需要额外开发

## 故障排除

### 常见错误

1. **权限不足**：
   ```
   创建原始套接字失败: Operation not permitted
   ```
   **解决方案**：使用root权限或设置CAP_NET_RAW能力

2. **Windows权限错误**：
   ```
   An attempt was made to access a socket in a way forbidden by its access permissions
   ```
   **解决方案**：以管理员身份运行

3. **网络不可达**：
   ```
   发送数据包失败: Network is unreachable
   ```
   **解决方案**：检查路由配置和网络连通性

### 调试建议

1. **先测试本机IP**：
   ```bash
   # 获取本机IP
   ip addr show  # Linux
   ipconfig      # Windows
   
   # 使用本机IP测试
   ./syslog_go send -s <本机IP> -t target:514 -p udp
   ```

2. **检查网络抓包**：
   ```bash
   # 使用tcpdump或Wireshark查看数据包
   sudo tcpdump -i any host <目标IP>
   ```

3. **逐步测试**：
   - 先在本地环回接口测试
   - 再测试局域网内的目标
   - 最后测试跨网段的目标

## 示例脚本

### Linux测试脚本

```bash
#!/bin/bash

# 设置权限
sudo setcap cap_net_raw+ep ./syslog_go

# 测试本机IP
echo "测试本机IP..."
./syslog_go send -s $(hostname -I | awk '{print $1}') -t 127.0.0.1:514 -p udp -e 1

# 测试模拟IP
echo "测试模拟IP..."
./syslog_go send -s 10.0.0.100 -t 127.0.0.1:514 -p udp -e 1
```

### Windows测试脚本

```powershell
# 以管理员身份运行PowerShell

# 获取本机IP
$localIP = (Get-NetIPAddress -AddressFamily IPv4 | Where-Object {$_.IPAddress -ne "127.0.0.1"} | Select-Object -First 1).IPAddress

# 测试本机IP
Write-Host "测试本机IP: $localIP"
.\syslog_go.exe send -s $localIP -t "127.0.0.1:514" -p udp -e 1

# 测试模拟IP
Write-Host "测试模拟IP: 192.168.100.100"
.\syslog_go.exe send -s "192.168.100.100" -t "127.0.0.1:514" -p udp -e 1
```

## 总结

源IP地址模拟功能为syslog测试提供了更大的灵活性，但需要注意权限要求和网络限制。在使用前请确保：

1. 具有适当的系统权限
2. 在授权的网络环境中使用
3. 了解相关的技术限制
4. 遵守网络安全最佳实践

如有问题，请参考故障排除部分或联系技术支持。
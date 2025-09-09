#!/bin/bash

# 测试TCP源地址模拟功能
# 需要root权限运行

echo "=== TCP源地址模拟测试 ==="
echo

# 检查是否有root权限
if [ "$EUID" -ne 0 ]; then
    echo "警告: 建议使用root权限运行此脚本以测试原始套接字功能"
    echo "使用命令: sudo $0"
    echo
fi

# 获取本机IP地址
LOCAL_IP=$(ip route get 8.8.8.8 | awk '{print $7; exit}')
echo "检测到本机IP: $LOCAL_IP"
echo

# 启动网络抓包 (后台运行)
echo "启动网络抓包..."
tcpdump -i any -n "port 514" > /tmp/syslog_capture.txt 2>&1 &
TCPDUMP_PID=$!
echo "抓包进程ID: $TCPDUMP_PID"
sleep 2
echo

# 测试1: 使用本机IP发送TCP消息
echo "测试1: 使用本机IP发送TCP消息"
echo "命令: ./syslog_go_linux_x64 send -e 1 -p tcp -t localhost:514 -m 'Test with local IP'"
./syslog_go_linux_x64 send -e 1 -p tcp -t localhost:514 -m "Test with local IP"
echo
sleep 2

# 测试2: 使用非本机IP发送TCP消息 (需要root权限)
echo "测试2: 使用非本机IP发送TCP消息 (需要root权限)"
echo "命令: ./syslog_go_linux_x64 send -s 192.168.1.100 -e 1 -p tcp -t localhost:514 -m 'Test with spoofed IP'"
./syslog_go_linux_x64 send -s 192.168.1.100 -e 1 -p tcp -t localhost:514 -m "Test with spoofed IP"
echo
sleep 2

# 测试3: UDP对比测试
echo "测试3: UDP对比测试"
echo "命令: ./syslog_go_linux_x64 send -s 192.168.1.100 -e 1 -p udp -t localhost:514 -m 'Test UDP with spoofed IP'"
./syslog_go_linux_x64 send -s 192.168.1.100 -e 1 -p udp -t localhost:514 -m "Test UDP with spoofed IP"
echo
sleep 2

# 停止抓包
echo "停止网络抓包..."
kill $TCPDUMP_PID 2>/dev/null
sleep 1

# 显示抓包结果
echo "=== 网络抓包结果 ==="
if [ -f /tmp/syslog_capture.txt ]; then
    cat /tmp/syslog_capture.txt
    echo
else
    echo "未找到抓包文件"
fi

echo "=== 测试完成 ==="
echo "注意事项:"
echo "1. 如果看到权限错误，请使用 sudo 运行此脚本"
echo "2. 检查抓包结果中的源IP地址是否正确"
echo "3. 如果没有抓到包，可能是防火墙或路由问题"
echo "4. 可以手动运行: tcpdump -i any -n 'port 514' 来监控网络流量"
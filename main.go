package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"syslog_go/cmd"
)

func main() {
	// 创建一个带缓冲的信号通道，用于接收操作系统的中断信号
	c := make(chan os.Signal, 1)
	// 注册要监听的信号：SIGINT(Ctrl+C)和SIGTERM(终止信号)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// 启动一个goroutine来处理信号
	go func() {
		// 阻塞等待信号
		<-c
		// 收到信号后打印关闭提示
		fmt.Println("\n正在关闭...")
		// 正常退出程序（退出码为0）
		os.Exit(0)
	}()

	// 执行cobra的根命令
	if err := cmd.Execute(); err != nil {
		// 如果执行出错，打印错误信息到标准错误输出
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		// 异常退出程序（退出码为1）
		os.Exit(1)
	}
}
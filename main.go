// Package main 是应用程序的入口包
package main

import (
	"fmt"
	"os"
	"os/signal" // 提供信号处理功能
	"syscall"    // 系统调用包，用于定义系统信号常量

	"syslog_go/cmd" // 导入命令行处理包
)

// main 是应用程序的入口函数
// 它负责：
// 1. 设置信号处理，优雅处理SIGINT和SIGTERM信号
// 2. 启动cobra命令行处理
func main() {
	// 创建一个带缓冲的信号通道，用于接收操作系统的中断信号
	// 缓冲大小为1，避免信号处理协程阻塞
	c := make(chan os.Signal, 1)

	// 注册要监听的信号：
	// SIGINT(Ctrl+C)和SIGTERM(终止信号)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// 启动一个goroutine来处理信号
	// 当收到终止信号时，优雅退出程序
	go func() {
		// 阻塞等待信号
		<-c
		// 收到信号后打印关闭提示
		fmt.Println("\n正在关闭...")
		// 正常退出程序（退出码为0）
		os.Exit(0)
	}()

	// 执行cobra的根命令
	// 这将启动命令行界面，处理用户输入的命令
	if err := cmd.Execute(); err != nil {
		// 如果执行出错，打印错误信息到标准错误输出
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		// 异常退出程序（退出码为1）
		os.Exit(1)
	}
}
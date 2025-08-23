package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"syslog_sender/cmd"
)

func main() {
	// 设置信号处理
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\n正在优雅关闭...")
		os.Exit(0)
	}()

	// 执行根命令
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
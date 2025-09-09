// Package cmd 提供命令行功能的实现
package cmd

import (
	"fmt"
	"os"
	"os/signal" // 提供信号处理功能
	"syscall"    // 系统调用包

	"github.com/spf13/cobra" // 命令行框架
	"syslog_go/pkg/server"  // Syslog服务器实现
)

// 命令行参数
var (
	serverHost string // 服务器监听的主机地址
	serverPort int    // 服务器监听的端口号
)

// serverCmd 表示服务器命令
// 它实现了一个可以同时监听UDP和TCP的Syslog服务器
var serverCmd = &cobra.Command{
	// 命令名称
	Use:   "server",
	// 简短描述
	Short: "启动Syslog测试服务器",
	// 详细描述和使用示例
	Long: `启动一个用于测试的Syslog服务器

主要功能:
✓ 支持UDP/TCP协议
✓ 兼容RFC3164/5424格式
✓ 自动解析消息格式
✓ 实时显示接收日志

示例:
  # 在所有网卡上监听514端口（需要root权限）
  syslog_go server -H 0.0.0.0 -p 514

  # 仅本地监听1514端口
  syslog_go server -H 127.0.0.1 -p 1514`,
	// 命令执行函数
	Run: func(cmd *cobra.Command, args []string) {
		// 创建服务器实例
		// NewServer函数接收主机地址和端口参数
		srv := server.NewServer(serverHost, serverPort)

		// 启动服务器
		// Start方法会初始化并启动UDP和TCP监听器
		if err := srv.Start(); err != nil {
			fmt.Printf("启动服务器失败: %v\n", err)
			os.Exit(1) // 发生错误时退出程序
		}

		// 创建信号通道并等待中断信号
		// 这允许服务器在收到Ctrl+C或终止信号时优雅关闭
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan // 阻塞等待信号

		// 优雅关闭服务器
		// Stop方法会关闭所有监听器
		fmt.Println("正在关闭服务器...")
		srv.Stop()
	},
}

// init 初始化服务器命令
// 它将服务器命令添加到根命令，并设置命令行参数
func init() {
	// 将server命令添加到根命令
	rootCmd.AddCommand(serverCmd)

	// 添加命令行参数
	// -H, --host: 指定服务器监听的主机地址，默认为127.0.0.1
	serverCmd.Flags().StringVarP(&serverHost, "host", "H", "127.0.0.1", "监听地址")
	// -p, --port: 指定服务器监听的端口，默认为514
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 514, "监听端口")
}
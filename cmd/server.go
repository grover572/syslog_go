package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"syslog_go/pkg/server"
)

var (
	serverHost string
	serverPort int
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "启动Syslog测试服务器",
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
	Run: func(cmd *cobra.Command, args []string) {
		// 创建服务器实例
		srv := server.NewServer(serverHost, serverPort)

		// 启动服务器
		if err := srv.Start(); err != nil {
			fmt.Printf("启动服务器失败: %v\n", err)
			os.Exit(1)
		}

		// 等待中断信号
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		// 优雅关闭服务器
		fmt.Println("正在关闭服务器...")
		srv.Stop()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// 添加命令行参数
	serverCmd.Flags().StringVarP(&serverHost, "host", "H", "0.0.0.0", "监听地址")
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 514, "监听端口")
}
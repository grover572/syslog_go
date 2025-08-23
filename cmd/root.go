package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"syslog_sender/pkg/config"
	"syslog_sender/pkg/sender"
	"syslog_sender/pkg/ui"
)

var (
	cfgFile string
	interactive bool
	cfg *config.Config
)

// rootCmd 代表没有调用子命令时的基础命令
var rootCmd = &cobra.Command{
	Use:   "syslog_sender",
	Short: "一个强大的Syslog发送工具",
	Long: `Syslog Sender是一个用Go语言编写的高性能Syslog发送工具。

功能特性:
- 支持RFC3164和RFC5424协议
- 可模拟源IP地址
- 支持速率控制(EPS)
- 模板化日志生成
- 交互式配置界面
- 实时统计监控`,
	Run: func(cmd *cobra.Command, args []string) {
		if interactive {
			// 启动交互式模式
			ui.StartInteractiveMode()
			return
		}

		// 加载配置
		var err error
		cfg, err = config.LoadConfig(cfgFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "配置加载失败: %v\n", err)
			os.Exit(1)
		}

		// 创建并启动发送器
		s, err := sender.NewSender(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "发送器创建失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("开始发送Syslog消息到 %s\n", cfg.Target)
		fmt.Printf("发送速率: %d EPS, 持续时间: %v\n", cfg.EPS, cfg.Duration)

		if err := s.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "发送失败: %v\n", err)
			os.Exit(1)
		}
	},
}

// Execute 添加所有子命令到根命令并设置标志。
// 这由main.main()调用。它只需要对rootCmd调用一次。
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// 全局标志
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件路径 (默认为 ./config.yaml)")
	rootCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "启动交互式模式")

	// 发送配置标志
	rootCmd.Flags().StringP("target", "t", "localhost:514", "目标Syslog服务器地址")
	rootCmd.Flags().StringP("source-ip", "s", "", "源IP地址(用于IP伪造)")
	rootCmd.Flags().StringP("protocol", "p", "udp", "传输协议 (udp/tcp)")
	rootCmd.Flags().IntP("eps", "e", 10, "每秒发送事件数")
	rootCmd.Flags().DurationP("duration", "d", 60*time.Second, "发送持续时间")
	rootCmd.Flags().StringP("format", "f", "rfc3164", "Syslog格式 (rfc3164/rfc5424)")
	rootCmd.Flags().StringP("template-dir", "", "./data/templates", "模板目录路径")
	rootCmd.Flags().StringP("template-file", "", "", "指定模板文件")
	rootCmd.Flags().StringP("data-file", "", "", "数据文件路径")
	rootCmd.Flags().IntP("facility", "", 16, "Syslog Facility (0-23)")
	rootCmd.Flags().IntP("severity", "", 6, "Syslog Severity (0-7)")

	// 绑定标志到viper
	viper.BindPFlag("target", rootCmd.Flags().Lookup("target"))
	viper.BindPFlag("source_ip", rootCmd.Flags().Lookup("source-ip"))
	viper.BindPFlag("protocol", rootCmd.Flags().Lookup("protocol"))
	viper.BindPFlag("eps", rootCmd.Flags().Lookup("eps"))
	viper.BindPFlag("duration", rootCmd.Flags().Lookup("duration"))
	viper.BindPFlag("format", rootCmd.Flags().Lookup("format"))
	viper.BindPFlag("template_dir", rootCmd.Flags().Lookup("template-dir"))
	viper.BindPFlag("template_file", rootCmd.Flags().Lookup("template-file"))
	viper.BindPFlag("data_file", rootCmd.Flags().Lookup("data-file"))
	viper.BindPFlag("facility", rootCmd.Flags().Lookup("facility"))
	viper.BindPFlag("severity", rootCmd.Flags().Lookup("severity"))
}

// initConfig 读取配置文件和环境变量
func initConfig() {
	if cfgFile != "" {
		// 使用指定的配置文件
		viper.SetConfigFile(cfgFile)
	} else {
		// 在当前目录查找配置文件
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // 读取匹配的环境变量

	// 如果找到配置文件，则读取它
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "使用配置文件:", viper.ConfigFileUsed())
	}
}
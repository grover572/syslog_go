package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"syslog_go/pkg/config"
	"syslog_go/pkg/sender"
	"syslog_go/pkg/template"
)

var (
	mockMessage string
	mockOutput string
	mockCount  int
	mockAppend bool
)

// mockCmd 生成模拟数据
var mockCmd = &cobra.Command{
	Use:   "mock",
	Short: "生成模拟数据",
	Long: `生成模拟数据

支持的模板变量:
1. {{RANDOM_STRING:选项1,选项2,...}} - 从给定选项中随机选择，支持权重
2. {{RANDOM_INT:最小值-最大值}} - 生成指定范围内的随机整数
3. {{ENUM:选项1,选项2,...}} - 从选项列表中随机选择一个
4. {{MAC}} - 生成随机MAC地址
5. {{RANDOM_IP}} 或 {{RANDOM_IPV4}} - 生成随机IPv4地址
   {{RANDOM_IP:internal}} - 生成内网IPv4地址
   {{RANDOM_IP:external}} - 生成外网IPv4地址
6. {{RANGE_IP:192.168.1.1-192.168.1.100}} - 生成指定范围内的IPv4地址
   {{RANGE_IP:192.168.1.0/24}} - 生成指定CIDR范围内的IPv4地址
   {{RANGE_IP:2001:db8::/32}} - 生成指定CIDR范围内的IPv6地址
7. {{RANDOM_IPV6}} - 生成标准格式的IPv6地址
   {{RANDOM_IPV6:internal}} - 生成内网IPv6地址 (fd00::/8)
   {{RANDOM_IPV6:external}} - 生成外网IPv6地址 (2000::/3)
   {{RANDOM_IPV6:compressed}} - 生成压缩格式的IPv6地址（包含::）`,
	Run: func(cmd *cobra.Command, args []string) {
		// 如果没有提供任何参数，显示帮助信息
		if len(args) == 0 && mockMessage == "" && mockOutput == "" && mockCount == 1 && !mockAppend {
			cmd.Help()
			return
		}

		// 如果提供了位置参数，将其作为输出文件
		if len(args) > 0 {
			mockOutput = args[0]
			if len(args) > 1 {
				fmt.Fprintln(os.Stderr, "警告: 只使用第一个参数作为输出文件")
			}
		}

		if mockMessage == "" {
			fmt.Fprintln(os.Stderr, "错误: 必须使用 -m/--message 指定消息模板")
			os.Exit(1)
		}

		// 创建模板引擎
		// 检查当前目录下是否存在template.yml
		configPath := "template.yml"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			configPath = "" // 如果文件不存在，使用空字符串
		}
		verbose := viper.GetBool("verbose")
		engine := template.NewEngine(configPath, verbose)

		// 加载消息模板
		engine.LoadTemplate("message", mockMessage)

		// 生成指定数量的消息
		var messages []string
		for i := 0; i < mockCount; i++ {
			msg, err := engine.GenerateMessage("message")
			if err != nil {
				fmt.Fprintf(os.Stderr, "生成第 %d 条消息时出错: %v\n", i+1, err)
				os.Exit(1)
			}
			messages = append(messages, msg)
		}

		// 将结果写入文件或输出到标准输出
		output := strings.Join(messages, "\n") + "\n"
		if mockOutput != "" {
			var err error
			if mockAppend {
				// 追加模式
				f, err := os.OpenFile(mockOutput, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "打开输出文件失败: %v\n", err)
					os.Exit(1)
				}
				defer f.Close()
				
				_, err = f.WriteString(output)
				if err != nil {
					fmt.Fprintf(os.Stderr, "写入输出文件失败: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("已追加 %d 条消息到 %s\n", mockCount, mockOutput)
			} else {
				// 覆盖模式
				err = os.WriteFile(mockOutput, []byte(output), 0644)
				if err != nil {
					fmt.Fprintf(os.Stderr, "写入输出文件失败: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("已生成 %d 条消息并写入到 %s\n", mockCount, mockOutput)
			}
		} else {
			fmt.Print(output)
		}
	},
}

var (
	message string
	cfg     *config.Config
)

// rootCmd 代表发送命令
var rootCmd = &cobra.Command{
	Use:   "syslog_go",
	Short: "高性能Syslog测试工具",
	Long: `Syslog Go - 专业的Syslog日志测试工具

可用命令:
  send     发送Syslog消息（默认）
  server   启动Syslog测试服务器
  mock     生成模拟数据

发送功能:
✓ 支持UDP/TCP协议
✓ 兼容RFC3164/5424格式
✓ 可配置发送速率(EPS)
✓ 支持模板化消息生成
✓ 内置多种变量函数
✓ 实时监控统计`,
	Run: func(cmd *cobra.Command, args []string) {
		// 显示帮助信息
		cmd.Help()
	},
}

// Execute 添加所有子命令到根命令并设置标志。
// 这由main.main()调用。它只需要对rootCmd调用一次。
func Execute() error {
	return rootCmd.Execute()
}

// sendCmd 发送Syslog消息
var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "发送Syslog消息",
	Long: `发送Syslog消息

主要功能:
✓ 支持UDP/TCP协议
✓ 兼容RFC3164/5424格式
✓ 可配置发送速率(EPS)
✓ 支持模板化消息生成
✓ 内置多种变量函数
✓ 实时监控统计`,
	Run: func(cmd *cobra.Command, args []string) {
		// 使用默认配置
		cfg = config.DefaultConfig()

		// 从命令行参数更新配置
		cfg.Target = viper.GetString("target")
		cfg.SourceIP = viper.GetString("source_ip")
		cfg.Protocol = viper.GetString("protocol")
		cfg.EPS = viper.GetInt("eps")
		cfg.Duration = viper.GetDuration("duration")
		cfg.Format = viper.GetString("format")
		cfg.DataFile = viper.GetString("data_file")
		cfg.Facility = viper.GetInt("facility")
		cfg.Severity = viper.GetInt("severity")
		cfg.Verbose = viper.GetBool("verbose")

		// 如果指定了消息内容，直接设置到配置中
		if message != "" {
			cfg.Message = message
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

func init() {
	// 隐藏completion命令
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	// 添加子命令
	rootCmd.AddCommand(mockCmd)
	rootCmd.AddCommand(sendCmd)

	// mock命令标志
	mockCmd.Flags().StringVarP(&mockMessage, "message", "m", "", "指定消息模板 (支持模板变量，使用 {{变量名:参数}} 格式)")
	mockCmd.Flags().StringVarP(&mockOutput, "output", "o", "", "输出文件路径 (默认输出到标准输出)")
	mockCmd.Flags().IntVarP(&mockCount, "count", "n", 1, "生成消息的数量")
	mockCmd.Flags().BoolVarP(&mockAppend, "append", "a", false, "追加到输出文件 (默认覆盖文件)")
	mockCmd.Flags().BoolP("verbose", "v", false, "显示详细信息")
	viper.BindPFlag("verbose", mockCmd.Flags().Lookup("verbose"))

	// 发送命令标志
	sendCmd.Flags().StringVarP(&message, "message", "m", "", "指定消息内容 (支持模板变量，使用 {{变量名:参数}} 格式，详见mock命令)")
	sendCmd.Flags().StringP("target", "t", "localhost:514", "目标服务器地址")
	sendCmd.Flags().StringP("source-ip", "s", "", "源IP地址")
	sendCmd.Flags().StringP("protocol", "p", "udp", "传输协议 (udp/tcp)")
	sendCmd.Flags().IntP("eps", "e", 10, "每秒事件数")
	sendCmd.Flags().DurationP("duration", "d", 60*time.Second, "发送持续时间")
	sendCmd.Flags().StringP("format", "f", "rfc3164", "日志格式 (rfc3164/rfc5424)")
	sendCmd.Flags().StringP("data-file", "D", "", "数据文件")
	sendCmd.Flags().IntP("facility", "L", 16, "Syslog Facility (0-23)")
	sendCmd.Flags().IntP("severity", "S", 6, "Syslog Severity (0-7)")
	sendCmd.Flags().BoolP("verbose", "v", false, "显示详细信息")

	// 绑定标志到viper
	viper.BindPFlag("target", sendCmd.Flags().Lookup("target"))
	viper.BindPFlag("source_ip", sendCmd.Flags().Lookup("source-ip"))
	viper.BindPFlag("protocol", sendCmd.Flags().Lookup("protocol"))
	viper.BindPFlag("eps", sendCmd.Flags().Lookup("eps"))
	viper.BindPFlag("duration", sendCmd.Flags().Lookup("duration"))
	viper.BindPFlag("format", sendCmd.Flags().Lookup("format"))
	viper.BindPFlag("data_file", sendCmd.Flags().Lookup("data-file"))
	viper.BindPFlag("facility", sendCmd.Flags().Lookup("facility"))
	viper.BindPFlag("severity", sendCmd.Flags().Lookup("severity"))
	viper.BindPFlag("verbose", sendCmd.Flags().Lookup("verbose"))
	viper.BindPFlag("message", sendCmd.Flags().Lookup("message"))
}

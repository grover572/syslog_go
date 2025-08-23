package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"syslog_sender/pkg/config"
	"syslog_sender/pkg/sender"
)

// InteractiveUI 交互式用户界面
type InteractiveUI struct {
	config *config.Config
	reader *bufio.Reader
}

// NewInteractiveUI 创建新的交互式界面
func NewInteractiveUI() *InteractiveUI {
	return &InteractiveUI{
		config: config.DefaultConfig(),
		reader: bufio.NewReader(os.Stdin),
	}
}

// StartInteractiveMode 启动交互式模式
func StartInteractiveMode() {
	ui := NewInteractiveUI()
	ui.showWelcome()
	ui.mainMenu()
}

// showWelcome 显示欢迎信息
func (ui *InteractiveUI) showWelcome() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("           Syslog发送工具 - 交互式模式")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("功能特性:")
	fmt.Println("  • 支持RFC3164和RFC5424协议")
	fmt.Println("  • 可配置发送速率(EPS)")
	fmt.Println("  • 模板化日志生成")
	fmt.Println("  • 实时统计监控")
	fmt.Println("  • 支持TCP/UDP传输")
	fmt.Println(strings.Repeat("=", 60) + "\n")
}

// mainMenu 主菜单
func (ui *InteractiveUI) mainMenu() {
	for {
		fmt.Println("\n=== 主菜单 ===")
		fmt.Println("1. 基础配置")
		fmt.Println("2. 发送控制")
		fmt.Println("3. 数据源配置")
		fmt.Println("4. 查看当前配置")
		fmt.Println("5. 开始发送")
		fmt.Println("6. 退出")
		fmt.Print("\n请选择 (1-6): ")

		choice := ui.readInput()
		switch choice {
		case "1":
			ui.basicConfigMenu()
		case "2":
			ui.sendControlMenu()
		case "3":
			ui.dataSourceMenu()
		case "4":
			ui.showCurrentConfig()
		case "5":
			ui.startSending()
		case "6":
			fmt.Println("\n感谢使用！")
			return
		default:
			fmt.Println("无效选择，请重新输入")
		}
	}
}

// basicConfigMenu 基础配置菜单
func (ui *InteractiveUI) basicConfigMenu() {
	for {
		fmt.Println("\n=== 基础配置 ===")
		fmt.Printf("1. 目标服务器 (当前: %s)\n", ui.config.Target)
		fmt.Printf("2. 源IP地址 (当前: %s)\n", ui.getDisplayValue(ui.config.SourceIP, "自动"))
		fmt.Printf("3. 传输协议 (当前: %s)\n", ui.config.Protocol)
		fmt.Printf("4. Syslog格式 (当前: %s)\n", ui.config.Format)
		fmt.Printf("5. Facility (当前: %d - %s)\n", ui.config.Facility, getFacilityName(ui.config.Facility))
		fmt.Printf("6. Severity (当前: %d - %s)\n", ui.config.Severity, getSeverityName(ui.config.Severity))
		fmt.Println("7. 返回主菜单")
		fmt.Print("\n请选择 (1-7): ")

		choice := ui.readInput()
		switch choice {
		case "1":
			ui.configTarget()
		case "2":
			ui.configSourceIP()
		case "3":
			ui.configProtocol()
		case "4":
			ui.configFormat()
		case "5":
			ui.configFacility()
		case "6":
			ui.configSeverity()
		case "7":
			return
		default:
			fmt.Println("无效选择，请重新输入")
		}
	}
}

// sendControlMenu 发送控制菜单
func (ui *InteractiveUI) sendControlMenu() {
	for {
		fmt.Println("\n=== 发送控制 ===")
		fmt.Printf("1. 发送速率 (当前: %d EPS)\n", ui.config.EPS)
		fmt.Printf("2. 持续时间 (当前: %v)\n", ui.config.Duration)
		fmt.Printf("3. 并发连接数 (当前: %d)\n", ui.config.Concurrency)
		fmt.Printf("4. 连接超时 (当前: %v)\n", ui.config.Timeout)
		fmt.Println("5. 返回主菜单")
		fmt.Print("\n请选择 (1-5): ")

		choice := ui.readInput()
		switch choice {
		case "1":
			ui.configEPS()
		case "2":
			ui.configDuration()
		case "3":
			ui.configConcurrency()
		case "4":
			ui.configTimeout()
		case "5":
			return
		default:
			fmt.Println("无效选择，请重新输入")
		}
	}
}

// dataSourceMenu 数据源配置菜单
func (ui *InteractiveUI) dataSourceMenu() {
	for {
		fmt.Println("\n=== 数据源配置 ===")
		fmt.Printf("1. 模板目录 (当前: %s)\n", ui.getDisplayValue(ui.config.TemplateDir, "未设置"))
		fmt.Printf("2. 指定模板文件 (当前: %s)\n", ui.getDisplayValue(ui.config.TemplateFile, "未设置"))
		fmt.Printf("3. 数据文件 (当前: %s)\n", ui.getDisplayValue(ui.config.DataFile, "未设置"))
		fmt.Println("4. 返回主菜单")
		fmt.Print("\n请选择 (1-4): ")

		choice := ui.readInput()
		switch choice {
		case "1":
			ui.configTemplateDir()
		case "2":
			ui.configTemplateFile()
		case "3":
			ui.configDataFile()
		case "4":
			return
		default:
			fmt.Println("无效选择，请重新输入")
		}
	}
}

// 配置方法实现
func (ui *InteractiveUI) configTarget() {
	fmt.Printf("\n当前目标服务器: %s\n", ui.config.Target)
	fmt.Print("请输入新的目标服务器地址 (格式: IP:端口): ")
	input := ui.readInput()
	if input != "" {
		ui.config.Target = input
		fmt.Println("目标服务器已更新")
	}
}

func (ui *InteractiveUI) configSourceIP() {
	fmt.Printf("\n当前源IP: %s\n", ui.getDisplayValue(ui.config.SourceIP, "自动"))
	fmt.Print("请输入源IP地址 (留空表示自动): ")
	input := ui.readInput()
	ui.config.SourceIP = input
	fmt.Println("源IP已更新")
}

func (ui *InteractiveUI) configProtocol() {
	fmt.Println("\n选择传输协议:")
	fmt.Println("1. UDP (推荐)")
	fmt.Println("2. TCP")
	fmt.Print("请选择 (1-2): ")

	choice := ui.readInput()
	switch choice {
	case "1":
		ui.config.Protocol = "udp"
	case "2":
		ui.config.Protocol = "tcp"
	default:
		fmt.Println("无效选择，保持当前设置")
		return
	}
	fmt.Printf("协议已设置为: %s\n", ui.config.Protocol)
}

func (ui *InteractiveUI) configFormat() {
	fmt.Println("\n选择Syslog格式:")
	fmt.Println("1. RFC3164 (传统格式)")
	fmt.Println("2. RFC5424 (新格式)")
	fmt.Print("请选择 (1-2): ")

	choice := ui.readInput()
	switch choice {
	case "1":
		ui.config.Format = "rfc3164"
	case "2":
		ui.config.Format = "rfc5424"
	default:
		fmt.Println("无效选择，保持当前设置")
		return
	}
	fmt.Printf("格式已设置为: %s\n", ui.config.Format)
}

func (ui *InteractiveUI) configFacility() {
	fmt.Println("\n常用Facility值:")
	fmt.Println("0=kernel, 1=user, 4=auth, 16=local0, 17=local1, 18=local2")
	fmt.Printf("当前值: %d\n", ui.config.Facility)
	fmt.Print("请输入新的Facility值 (0-23): ")

	input := ui.readInput()
	if value, err := strconv.Atoi(input); err == nil && value >= 0 && value <= 23 {
		ui.config.Facility = value
		fmt.Printf("Facility已设置为: %d\n", value)
	} else {
		fmt.Println("无效输入，请输入0-23之间的数字")
	}
}

func (ui *InteractiveUI) configSeverity() {
	fmt.Println("\nSeverity级别:")
	fmt.Println("0=emerg, 1=alert, 2=crit, 3=err, 4=warning, 5=notice, 6=info, 7=debug")
	fmt.Printf("当前值: %d\n", ui.config.Severity)
	fmt.Print("请输入新的Severity值 (0-7): ")

	input := ui.readInput()
	if value, err := strconv.Atoi(input); err == nil && value >= 0 && value <= 7 {
		ui.config.Severity = value
		fmt.Printf("Severity已设置为: %d\n", value)
	} else {
		fmt.Println("无效输入，请输入0-7之间的数字")
	}
}

func (ui *InteractiveUI) configEPS() {
	fmt.Printf("\n当前EPS: %d\n", ui.config.EPS)
	fmt.Print("请输入新的EPS值 (1-10000): ")

	input := ui.readInput()
	if value, err := strconv.Atoi(input); err == nil && value > 0 && value <= 10000 {
		ui.config.EPS = value
		fmt.Printf("EPS已设置为: %d\n", value)
	} else {
		fmt.Println("无效输入，请输入1-10000之间的数字")
	}
}

func (ui *InteractiveUI) configDuration() {
	fmt.Printf("\n当前持续时间: %v\n", ui.config.Duration)
	fmt.Print("请输入新的持续时间 (如: 60s, 5m, 1h): ")

	input := ui.readInput()
	if duration, err := time.ParseDuration(input); err == nil {
		ui.config.Duration = duration
		fmt.Printf("持续时间已设置为: %v\n", duration)
	} else {
		fmt.Println("无效格式，请使用如 60s, 5m, 1h 的格式")
	}
}

func (ui *InteractiveUI) configConcurrency() {
	fmt.Printf("\n当前并发数: %d\n", ui.config.Concurrency)
	fmt.Print("请输入新的并发连接数 (1-100): ")

	input := ui.readInput()
	if value, err := strconv.Atoi(input); err == nil && value > 0 && value <= 100 {
		ui.config.Concurrency = value
		fmt.Printf("并发数已设置为: %d\n", value)
	} else {
		fmt.Println("无效输入，请输入1-100之间的数字")
	}
}

func (ui *InteractiveUI) configTimeout() {
	fmt.Printf("\n当前超时时间: %v\n", ui.config.Timeout)
	fmt.Print("请输入新的连接超时时间 (如: 5s, 10s): ")

	input := ui.readInput()
	if timeout, err := time.ParseDuration(input); err == nil {
		ui.config.Timeout = timeout
		fmt.Printf("超时时间已设置为: %v\n", timeout)
	} else {
		fmt.Println("无效格式，请使用如 5s, 10s 的格式")
	}
}

func (ui *InteractiveUI) configTemplateDir() {
	fmt.Printf("\n当前模板目录: %s\n", ui.getDisplayValue(ui.config.TemplateDir, "未设置"))
	fmt.Print("请输入模板目录路径: ")
	input := ui.readInput()
	if input != "" {
		ui.config.TemplateDir = input
		fmt.Println("模板目录已更新")
	}
}

func (ui *InteractiveUI) configTemplateFile() {
	fmt.Printf("\n当前模板文件: %s\n", ui.getDisplayValue(ui.config.TemplateFile, "未设置"))
	fmt.Print("请输入模板文件路径: ")
	input := ui.readInput()
	ui.config.TemplateFile = input
	fmt.Println("模板文件已更新")
}

func (ui *InteractiveUI) configDataFile() {
	fmt.Printf("\n当前数据文件: %s\n", ui.getDisplayValue(ui.config.DataFile, "未设置"))
	fmt.Print("请输入数据文件路径: ")
	input := ui.readInput()
	ui.config.DataFile = input
	fmt.Println("数据文件已更新")
}

// showCurrentConfig 显示当前配置
func (ui *InteractiveUI) showCurrentConfig() {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("                当前配置")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("目标服务器:     %s\n", ui.config.Target)
	fmt.Printf("源IP地址:       %s\n", ui.getDisplayValue(ui.config.SourceIP, "自动"))
	fmt.Printf("传输协议:       %s\n", ui.config.Protocol)
	fmt.Printf("Syslog格式:     %s\n", ui.config.Format)
	fmt.Printf("Facility:       %d (%s)\n", ui.config.Facility, getFacilityName(ui.config.Facility))
	fmt.Printf("Severity:       %d (%s)\n", ui.config.Severity, getSeverityName(ui.config.Severity))
	fmt.Printf("发送速率:       %d EPS\n", ui.config.EPS)
	fmt.Printf("持续时间:       %v\n", ui.config.Duration)
	fmt.Printf("并发连接数:     %d\n", ui.config.Concurrency)
	fmt.Printf("连接超时:       %v\n", ui.config.Timeout)
	fmt.Printf("模板目录:       %s\n", ui.getDisplayValue(ui.config.TemplateDir, "未设置"))
	fmt.Printf("模板文件:       %s\n", ui.getDisplayValue(ui.config.TemplateFile, "未设置"))
	fmt.Printf("数据文件:       %s\n", ui.getDisplayValue(ui.config.DataFile, "未设置"))
	fmt.Println(strings.Repeat("=", 50))
}

// startSending 开始发送
func (ui *InteractiveUI) startSending() {
	fmt.Println("\n准备开始发送...")
	ui.showCurrentConfig()
	fmt.Print("\n确认开始发送? (y/N): ")

	confirm := ui.readInput()
	if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
		fmt.Println("已取消发送")
		return
	}

	// 验证配置
	if err := ui.config.Validate(); err != nil {
		fmt.Printf("配置验证失败: %v\n", err)
		return
	}

	// 创建发送器
	s, err := sender.NewSender(ui.config)
	if err != nil {
		fmt.Printf("创建发送器失败: %v\n", err)
		return
	}

	// 开始发送
	fmt.Println("\n开始发送，按 Ctrl+C 停止...")
	if err := s.Start(); err != nil {
		fmt.Printf("发送失败: %v\n", err)
	}
}

// 辅助方法
func (ui *InteractiveUI) readInput() string {
	input, _ := ui.reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func (ui *InteractiveUI) getDisplayValue(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func getFacilityName(facility int) string {
	facilities := map[int]string{
		0: "kernel", 1: "user", 2: "mail", 3: "daemon", 4: "auth",
		5: "syslog", 6: "lpr", 7: "news", 8: "uucp", 9: "cron",
		10: "authpriv", 11: "ftp", 16: "local0", 17: "local1",
		18: "local2", 19: "local3", 20: "local4", 21: "local5",
		22: "local6", 23: "local7",
	}
	if name, ok := facilities[facility]; ok {
		return name
	}
	return "unknown"
}

func getSeverityName(severity int) string {
	severities := map[int]string{
		0: "emerg", 1: "alert", 2: "crit", 3: "err",
		4: "warning", 5: "notice", 6: "info", 7: "debug",
	}
	if name, ok := severities[severity]; ok {
		return name
	}
	return "unknown"
}
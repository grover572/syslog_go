#!/usr/bin/env pwsh
# Syslog发送工具测试脚本

Write-Host "=== Syslog发送工具项目结构检查 ===" -ForegroundColor Green

# 检查项目结构
$RequiredDirs = @(
    "cmd\syslog_sender",
    "pkg\config",
    "pkg\syslog", 
    "pkg\sender",
    "pkg\template",
    "pkg\ui",
    "data\templates\security",
    "data\templates\system",
    "data\templates\network",
    "data\templates\application",
    "data\variables",
    "data\samples",
    "scripts"
)

$RequiredFiles = @(
    "go.mod",
    "main.go",
    "cmd\root.go",
    "pkg\config\config.go",
    "pkg\syslog\protocol.go",
    "pkg\sender\sender.go",
    "pkg\sender\connection.go",
    "pkg\template\engine.go",
    "pkg\template\variables.go",
    "pkg\ui\interactive.go",
    "config.yaml",
    "README.md"
)

Write-Host "`n检查目录结构..." -ForegroundColor Yellow
$MissingDirs = @()
foreach ($Dir in $RequiredDirs) {
    if (Test-Path $Dir) {
        Write-Host "✓ $Dir" -ForegroundColor Green
    } else {
        Write-Host "✗ $Dir" -ForegroundColor Red
        $MissingDirs += $Dir
    }
}

Write-Host "`n检查必需文件..." -ForegroundColor Yellow
$MissingFiles = @()
foreach ($File in $RequiredFiles) {
    if (Test-Path $File) {
        Write-Host "✓ $File" -ForegroundColor Green
    } else {
        Write-Host "✗ $File" -ForegroundColor Red
        $MissingFiles += $File
    }
}

# 检查模板文件
Write-Host "`n检查模板文件..." -ForegroundColor Yellow
$TemplateFiles = @(
    "data\templates\security\ssh_login.log",
    "data\templates\security\firewall.log",
    "data\templates\system\kernel.log",
    "data\templates\network\cisco_asa.log",
    "data\templates\application\apache_access.log"
)

foreach ($Template in $TemplateFiles) {
    if (Test-Path $Template) {
        $LineCount = (Get-Content $Template | Measure-Object -Line).Lines
        Write-Host "✓ $Template ($LineCount 行)" -ForegroundColor Green
    } else {
        Write-Host "✗ $Template" -ForegroundColor Red
    }
}

# 检查配置文件
Write-Host "`n检查配置文件..." -ForegroundColor Yellow
if (Test-Path "data\variables\placeholders.yaml") {
    Write-Host "✓ 变量配置文件存在" -ForegroundColor Green
} else {
    Write-Host "✗ 变量配置文件缺失" -ForegroundColor Red
}

if (Test-Path "config.yaml") {
    Write-Host "✓ 主配置文件存在" -ForegroundColor Green
} else {
    Write-Host "✗ 主配置文件缺失" -ForegroundColor Red
}

# 统计信息
Write-Host "`n=== 项目统计 ===" -ForegroundColor Cyan

# 统计Go文件
$GoFiles = Get-ChildItem -Recurse -Filter "*.go"
Write-Host "Go源文件数量: $($GoFiles.Count)" -ForegroundColor White

# 统计代码行数
$TotalLines = 0
foreach ($File in $GoFiles) {
    $Lines = (Get-Content $File.FullName | Measure-Object -Line).Lines
    $TotalLines += $Lines
}
Write-Host "总代码行数: $TotalLines" -ForegroundColor White

# 统计模板文件
if (Test-Path "data\templates") {
    $TemplateCount = (Get-ChildItem -Recurse -Path "data\templates" -Filter "*.log" | Measure-Object).Count
    Write-Host "模板文件数量: $TemplateCount" -ForegroundColor White
}

# 统计配置文件
$ConfigCount = (Get-ChildItem -Recurse -Filter "*.yaml" | Measure-Object).Count
Write-Host "配置文件数量: $ConfigCount" -ForegroundColor White

# 检查结果总结
Write-Host "`n=== 检查结果 ===" -ForegroundColor Cyan

if ($MissingDirs.Count -eq 0 -and $MissingFiles.Count -eq 0) {
    Write-Host "✓ 项目结构完整!" -ForegroundColor Green
    Write-Host "✓ 所有必需文件都存在!" -ForegroundColor Green
    Write-Host "`n项目已准备就绪，可以进行构建和测试。" -ForegroundColor Green
    Write-Host "注意: 需要安装Go 1.21+才能构建项目。" -ForegroundColor Yellow
} else {
    Write-Host "✗ 项目结构不完整" -ForegroundColor Red
    
    if ($MissingDirs.Count -gt 0) {
        Write-Host "缺失目录: $($MissingDirs -join ', ')" -ForegroundColor Red
    }
    
    if ($MissingFiles.Count -gt 0) {
        Write-Host "缺失文件: $($MissingFiles -join ', ')" -ForegroundColor Red
    }
}

# 显示使用说明
Write-Host "`n=== 使用说明 ===" -ForegroundColor Cyan
Write-Host "1. 安装Go 1.21+" -ForegroundColor White
Write-Host "2. 运行 'go mod tidy' 下载依赖" -ForegroundColor White
Write-Host "3. 运行 'go build -o syslog_sender.exe ./cmd/syslog_sender' 构建" -ForegroundColor White
Write-Host "4. 运行 './syslog_sender.exe --help' 查看帮助" -ForegroundColor White
Write-Host "5. 运行 './syslog_sender.exe --interactive' 进入交互模式" -ForegroundColor White

Write-Host "`n项目检查完成!" -ForegroundColor Green
#!/usr/bin/env pwsh
# Syslog发送工具构建脚本

Param(
    [string]$Target = "windows",
    [string]$Arch = "amd64",
    [switch]$Release
)

# 设置构建参数
$BuildTime = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$GitCommit = try { git rev-parse --short HEAD 2>$null } catch { "unknown" }
$Version = "1.0.0"

# 设置输出目录
$OutputDir = "./bin"
if (!(Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force
}

# 设置构建标志
$LdFlags = @(
    "-X 'main.Version=$Version'",
    "-X 'main.BuildTime=$BuildTime'",
    "-X 'main.GitCommit=$GitCommit'"
)

if ($Release) {
    $LdFlags += "-s -w"  # 去除调试信息
}

# 设置环境变量
$env:GOOS = $Target
$env:GOARCH = $Arch
$env:CGO_ENABLED = "0"

# 构建可执行文件
$OutputName = "syslog_sender"
if ($Target -eq "windows") {
    $OutputName += ".exe"
}

$OutputPath = Join-Path $OutputDir "${OutputName}_${Target}_${Arch}"
if ($Target -eq "windows") {
    $OutputPath += ".exe"
}

Write-Host "构建目标: $Target/$Arch" -ForegroundColor Green
Write-Host "输出文件: $OutputPath" -ForegroundColor Green
Write-Host "版本信息: $Version ($GitCommit)" -ForegroundColor Green

# 执行构建
try {
    $BuildCmd = "go build -ldflags `"$($LdFlags -join ' ')`" -o `"$OutputPath`" ./cmd/syslog_sender"
    Write-Host "执行命令: $BuildCmd" -ForegroundColor Yellow
    
    Invoke-Expression $BuildCmd
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "构建成功!" -ForegroundColor Green
        
        # 显示文件信息
        if (Test-Path $OutputPath) {
            $FileInfo = Get-Item $OutputPath
            Write-Host "文件大小: $([math]::Round($FileInfo.Length / 1MB, 2)) MB" -ForegroundColor Cyan
        }
    } else {
        Write-Host "构建失败!" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "构建过程中发生错误: $_" -ForegroundColor Red
    exit 1
}

# 构建多平台版本（如果指定了Release）
if ($Release) {
    Write-Host "`n开始构建多平台版本..." -ForegroundColor Yellow
    
    $Platforms = @(
        @{OS="linux"; Arch="amd64"},
        @{OS="linux"; Arch="arm64"},
        @{OS="darwin"; Arch="amd64"},
        @{OS="darwin"; Arch="arm64"},
        @{OS="windows"; Arch="amd64"}
    )
    
    foreach ($Platform in $Platforms) {
        $env:GOOS = $Platform.OS
        $env:GOARCH = $Platform.Arch
        
        $PlatformOutput = "syslog_sender_$($Platform.OS)_$($Platform.Arch)"
        if ($Platform.OS -eq "windows") {
            $PlatformOutput += ".exe"
        }
        
        $PlatformPath = Join-Path $OutputDir $PlatformOutput
        
        Write-Host "构建 $($Platform.OS)/$($Platform.Arch)..." -ForegroundColor Cyan
        
        try {
            $PlatformCmd = "go build -ldflags `"$($LdFlags -join ' ')`" -o `"$PlatformPath`" ./cmd/syslog_sender"
            Invoke-Expression $PlatformCmd
            
            if ($LASTEXITCODE -eq 0) {
                Write-Host "✓ $($Platform.OS)/$($Platform.Arch) 构建成功" -ForegroundColor Green
            } else {
                Write-Host "✗ $($Platform.OS)/$($Platform.Arch) 构建失败" -ForegroundColor Red
            }
        } catch {
            Write-Host "✗ $($Platform.OS)/$($Platform.Arch) 构建错误: $_" -ForegroundColor Red
        }
    }
}

Write-Host "`n构建完成!" -ForegroundColor Green
# 简单的项目结构检查脚本

Write-Host "=== Syslog发送工具项目检查 ===" -ForegroundColor Green

# 检查主要文件
$files = @(
    "go.mod",
    "main.go", 
    "config.yaml",
    "README.md"
)

Write-Host "`n检查主要文件:" -ForegroundColor Yellow
foreach ($file in $files) {
    if (Test-Path $file) {
        Write-Host "✓ $file" -ForegroundColor Green
    } else {
        Write-Host "✗ $file" -ForegroundColor Red
    }
}

# 检查目录
$dirs = @(
    "cmd",
    "pkg",
    "data",
    "scripts"
)

Write-Host "`n检查主要目录:" -ForegroundColor Yellow
foreach ($dir in $dirs) {
    if (Test-Path $dir) {
        Write-Host "✓ $dir" -ForegroundColor Green
    } else {
        Write-Host "✗ $dir" -ForegroundColor Red
    }
}

# 统计文件
Write-Host "`n项目统计:" -ForegroundColor Cyan
$goFiles = Get-ChildItem -Recurse -Filter "*.go"
Write-Host "Go文件数量: $($goFiles.Count)" -ForegroundColor White

$yamlFiles = Get-ChildItem -Recurse -Filter "*.yaml"
Write-Host "配置文件数量: $($yamlFiles.Count)" -ForegroundColor White

if (Test-Path "data\templates") {
    $templateFiles = Get-ChildItem -Recurse -Path "data\templates" -Filter "*.log"
    Write-Host "模板文件数量: $($templateFiles.Count)" -ForegroundColor White
}

Write-Host "`n项目检查完成!" -ForegroundColor Green
Write-Host "注意: 需要安装Go 1.21+才能构建项目" -ForegroundColor Yellow
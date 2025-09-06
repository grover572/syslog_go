#!/bin/bash

# 定义要测试的平台和架构组合
PLATFORMS=(
    "linux amd64"
    "linux arm64"
    "darwin amd64"
    "darwin arm64"
    "windows amd64"
)

# 创建输出目录
OUTPUT_DIR="dist"
mkdir -p "$OUTPUT_DIR"

# 清理旧的构建文件
rm -rf "$OUTPUT_DIR"/*

# 遍历所有平台和架构组合进行构建
for platform in "${PLATFORMS[@]}"; do
    # 分割平台和架构
    read -r os arch <<< "$platform"
    
    # 设置输出文件名
    if [ "$os" = "windows" ]; then
        output_name="$OUTPUT_DIR/syslog_go_${os}_${arch}.exe"
    else
        output_name="$OUTPUT_DIR/syslog_go_${os}_${arch}"
    fi
    
    echo "正在构建 $os/$arch..."
    
    # 执行构建
    GOOS=$os GOARCH=$arch go build -o "$output_name"
    
    # 检查构建结果
    if [ $? -eq 0 ]; then
        echo "✅ $os/$arch 构建成功"
    else
        echo "❌ $os/$arch 构建失败"
    fi
done

echo "\n构建完成！构建结果在 $OUTPUT_DIR 目录中"
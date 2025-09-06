#!/bin/bash

# 检查rpmbuild命令
if ! command -v rpmbuild &> /dev/null; then
    echo "错误: 未找到rpmbuild命令，请先安装rpm构建工具"
    if [[ "$(uname)" == "Darwin" ]]; then
        echo "在macOS上运行: brew install rpm"
    else
        echo "在CentOS/RHEL上运行: sudo yum install rpm-build"
        echo "在Fedora上运行: sudo dnf install rpm-build"
    fi
    exit 1
fi

# 设置变量
NAME="syslog_go"
VERSION="1.0.0"
RELEASE="1"

# 创建打包目录
RPM_ROOT="$HOME/rpmbuild"
mkdir -p "$RPM_ROOT"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# 创建源码包
cd ..
tar czf "${RPM_ROOT}/SOURCES/${NAME}-${VERSION}.tar.gz" ${NAME}/

# 复制spec文件
cp "${NAME}/${NAME}.spec" "${RPM_ROOT}/SPECS/"

# 构建RPM包
rpmbuild -ba "${RPM_ROOT}/SPECS/${NAME}.spec"

echo "RPM包已构建完成，请查看 ${RPM_ROOT}/RPMS/ 目录"
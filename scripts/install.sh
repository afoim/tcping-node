#!/bin/bash

PORT=39001
COMMAND="install"

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        install|uninstall)
            COMMAND="$1"
            shift
            ;;
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        *)
            echo "用法: $0 [install|uninstall] [-p 端口号]"
            exit 1
            ;;
    esac
done

INSTALL_DIR="/opt/tcping-node"
SERVICE_NAME="tcping-agent"
GITHUB_REPO="afoim/tcping-node"

install_service() {
    # 创建安装目录
    mkdir -p $INSTALL_DIR

    # 下载最新release
    LATEST_RELEASE=$(curl -s https://api.github.com/repos/$GITHUB_REPO/releases/latest)
    DOWNLOAD_URL=$(echo "$LATEST_RELEASE" | grep "browser_download_url.*agent" | cut -d '"' -f 4)
    curl -L "$DOWNLOAD_URL" -o "$INSTALL_DIR/agent"
    chmod +x "$INSTALL_DIR/agent"

    # 创建服务文件
    cat > "/etc/systemd/system/$SERVICE_NAME.service" << EOF
[Unit]
Description=TCPing Agent Service
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/agent -p $PORT
Restart=always

[Install]
WantedBy=multi-user.target
EOF

    # 重新加载systemd并启动服务
    systemctl daemon-reload
    systemctl enable $SERVICE_NAME
    systemctl start $SERVICE_NAME
    
    echo "安装完成！服务已启动在端口 $PORT"
}

uninstall_service() {
    # 停止并禁用服务
    systemctl stop $SERVICE_NAME
    systemctl disable $SERVICE_NAME

    # 删除服务文件和安装目录
    rm -f "/etc/systemd/system/$SERVICE_NAME.service"
    rm -rf $INSTALL_DIR

    systemctl daemon-reload
    
    echo "卸载完成！"
}

# 需要root权限
if [ "$EUID" -ne 0 ]; then 
    echo "请使用root权限运行此脚本"
    exit 1
fi

# 执行命令
if [ "$COMMAND" = "install" ]; then
    install_service
else
    uninstall_service
fi

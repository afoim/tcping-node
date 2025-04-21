#!/bin/bash

# 配置
REPO="afoim/tcping-node"
BINARY_NAME="agent"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="tcping-agent"
CONFIG_DIR="/etc/tcping-node"
DEFAULT_PORT=8081

# 确保以root权限运行
if [ "$EUID" -ne 0 ]; then 
    echo "请使用root权限运行此脚本"
    exit 1
fi

# 创建配置目录
mkdir -p "$CONFIG_DIR"

# 检查现有服务
check_existing_service() {
    if systemctl is-active --quiet $SERVICE_NAME; then
        echo "检测到已安装的TCPing Agent服务"
        echo "1) 升级服务"
        echo "2) 查看服务状态"
        echo "3) 修改端口"
        echo "4) 退出"
        read -p "请选择操作 (1-4): " choice
        case $choice in
            1)
                echo "正在升级..."
                download_and_install
                systemctl restart $SERVICE_NAME
                echo "升级完成"
                ;;
            2)
                systemctl status $SERVICE_NAME
                ;;
            3)
                configure_port
                systemctl restart $SERVICE_NAME
                echo "端口已更新"
                ;;
            *)
                exit 0
                ;;
        esac
        exit 0
    fi
}

# 配置端口
configure_port() {
    if [ -f "$CONFIG_DIR/config" ]; then
        source "$CONFIG_DIR/config"
        current_port=$PORT
    else
        current_port=$DEFAULT_PORT
    fi
    
    read -p "请输入服务端口 (当前: $current_port): " input_port
    if [[ $input_port =~ ^[0-9]+$ ]]; then
        echo "PORT=$input_port" > "$CONFIG_DIR/config"
        return $input_port
    else
        echo "使用默认端口: $current_port"
        echo "PORT=$current_port" > "$CONFIG_DIR/config"
        return $current_port
    fi
}

# 安装系统服务
install_service() {
    local port=$1
    cat > "/etc/systemd/system/$SERVICE_NAME.service" << EOF
[Unit]
Description=TCPing Node Agent
After=network.target

[Service]
ExecStart=$INSTALL_DIR/$BINARY_NAME -p $port
Restart=always
User=root

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable $SERVICE_NAME
    systemctl start $SERVICE_NAME
}

# 下载和安装
download_and_install() {
    # 创建临时目录
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"

    echo "正在获取最新版本信息..."
    LATEST_TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "$LATEST_TAG" ]; then
        echo "无法获取最新版本信息，使用默认版本 latest"
        LATEST_TAG="latest"
    fi

    echo "最新版本: $LATEST_TAG"
    DOWNLOAD_URL="https://mirror.ghproxy.com/https://github.com/$REPO/releases/download/$LATEST_TAG/$BINARY_NAME"
    echo "下载地址: $DOWNLOAD_URL"

    echo "正在下载..."
    wget -q "$DOWNLOAD_URL" -O "$BINARY_NAME" || {
        echo "下载失败"
        rm -rf "$TMP_DIR"
        exit 1
    }

    chmod +x "$BINARY_NAME"
    mv "$BINARY_NAME" "$INSTALL_DIR/"
    rm -rf "$TMP_DIR"
}

# 显示帮助信息
show_help() {
    echo "用法: $0 [命令]"
    echo "命令:"
    echo "  install    - 安装或更新agent"
    echo "  uninstall  - 卸载agent"
    echo "  start      - 启动服务"
    echo "  stop       - 停止服务"
    echo "  restart    - 重启服务"
    echo "  status     - 查看服务状态"
    echo "  port       - 更改端口"
    echo "  version    - 显示当前版本"
}

# 卸载函数
uninstall() {
    echo "正在卸载TCPing Agent..."
    
    # 停止并删除服务
    if systemctl is-active --quiet $SERVICE_NAME; then
        systemctl stop $SERVICE_NAME
        systemctl disable $SERVICE_NAME
    fi
    
    # 删除服务文件
    rm -f "/etc/systemd/system/$SERVICE_NAME.service"
    systemctl daemon-reload
    
    # 删除二进制文件
    rm -f "$INSTALL_DIR/$BINARY_NAME"
    
    # 删除配置目录
    rm -rf "$CONFIG_DIR"
    
    echo "卸载完成！"
}

# 主程序
main() {
    check_existing_service
    download_and_install
    configure_port
    install_service $PORT
    echo "安装完成！服务已启动，端口: $PORT"
    systemctl status $SERVICE_NAME
}

# 主命令处理
case "$1" in
    "install")
        main
        ;;
    "uninstall")
        uninstall
        ;;
    "start")
        systemctl start $SERVICE_NAME
        ;;
    "stop")
        systemctl stop $SERVICE_NAME
        ;;
    "restart")
        systemctl restart $SERVICE_NAME
        ;;
    "status")
        systemctl status $SERVICE_NAME
        ;;
    "port")
        configure_port
        systemctl restart $SERVICE_NAME
        ;;
    "version")
        echo "当前版本: $(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')"
        ;;
    *)
        show_help
        ;;
esac

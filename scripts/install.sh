#!/bin/bash

# 配置
BINARY_NAME="tcping-agent"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="tcping-agent"
CONFIG_DIR="/etc/tcping-node"
DEFAULT_PORT=8081
DOWNLOAD_URL="https://mirror.ghproxy.com/https://github.com/afoim/tcping-node/releases/download/latest/agent"

# 确保以root权限运行
if [ "$EUID" -ne 0 ]; then 
    echo "请使用root权限运行此脚本"
    exit 1
fi

# 创建配置目录
mkdir -p "$CONFIG_DIR"

# 读取配置文件
if [ -f "$CONFIG_DIR/config" ]; then
    source "$CONFIG_DIR/config"
else
    PORT=$DEFAULT_PORT
    echo "PORT=$PORT" > "$CONFIG_DIR/config"
fi

# 显示帮助信息
show_help() {
    echo "用法: $0 [命令]"
    echo "命令:"
    echo "  install    - 安装或更新agent"
    echo "  start      - 启动服务"
    echo "  stop       - 停止服务"
    echo "  restart    - 重启服务"
    echo "  status     - 查看服务状态"
    echo "  port       - 更改端口"
    echo "  version    - 显示当前版本"
}

# 安装systemd服务
install_service() {
    cat > "/etc/systemd/system/$SERVICE_NAME.service" << EOF
[Unit]
Description=TCPing Node Agent
After=network.target

[Service]
ExecStart=$INSTALL_DIR/$BINARY_NAME -p \$PORT
Environment="PORT=$PORT"
EnvironmentFile=$CONFIG_DIR/config
Restart=always
User=root

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable $SERVICE_NAME
}

# 安装agent
install_agent() {
    echo "正在下载agent..."
    wget -q -O "$INSTALL_DIR/$BINARY_NAME" "$DOWNLOAD_URL" || {
        echo "下载失败，请检查网络连接"
        exit 1
    }
    
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    install_service
    echo "安装完成"
    echo "当前端口: $PORT"
    systemctl start $SERVICE_NAME
    show_ip_info
}

# 添加IP信息显示函数
show_ip_info() {
    echo -e "\n节点信息:"
    echo "----------------------------------------"
    # 获取内网IP
    local internal_ips=$(ip -4 addr show | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | grep -v '127.0.0.1')
    echo "内网地址:"
    echo "$internal_ips" | while read -r ip; do
        echo "http://$ip:$PORT"
    done
    
    # 获取外网IP
    echo -e "\n外网地址:"
    local external_ip=$(curl -s ip.sb || curl -s ifconfig.me)
    if [ ! -z "$external_ip" ]; then
        echo "http://$external_ip:$PORT"
    else
        echo "无法获取外网IP"
    fi
    echo "----------------------------------------"
}

# 更改端口
change_port() {
    read -p "请输入新的端口号 (当前: $PORT): " new_port
    if [[ $new_port =~ ^[0-9]+$ ]]; then
        sed -i "s/PORT=.*/PORT=$new_port/" "$CONFIG_DIR/config"
        PORT=$new_port
        systemctl restart $SERVICE_NAME
        echo "端口已更改为: $new_port"
    else
        echo "无效的端口号"
    fi
}

# 获取版本
get_version() {
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        echo "Agent路径: $INSTALL_DIR/$BINARY_NAME"
        echo "监听端口: $PORT"
        echo "服务状态: $(systemctl is-active $SERVICE_NAME)"
        show_ip_info
    else
        echo "Agent未安装"
    fi
}

# 主命令处理
case "$1" in
    "install")
        install_agent
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
        change_port
        ;;
    "version")
        get_version
        ;;
    *)
        show_help
        ;;
esac

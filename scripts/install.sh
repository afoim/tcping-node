#!/bin/bash

# 配置
BINARY_NAME="tcping-agent"
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

# 读取配置文件
if [ -f "$CONFIG_DIR/config" ]; then
    source "$CONFIG_DIR/config"
else
    PORT=$DEFAULT_PORT
    echo "PORT=$PORT" > "$CONFIG_DIR/config"
fi

# Github加速镜像列表
GITHUB_MIRRORS=(
    "https://github.com"
    "https://download.fastgit.org"
    "https://mirror.ghproxy.com/https://github.com"
)

# 询问是否使用加速镜像
select_mirror() {
    echo "请选择下载源:"
    echo "1) Github 原地址"
    echo "2) FastGit 镜像"
    echo "3) GHProxy 镜像"
    read -p "请选择 (1-3, 默认1): " mirror_choice
    
    case "$mirror_choice" in
        2)
            echo "使用 FastGit 镜像"
            GITHUB_URL=${GITHUB_MIRRORS[1]}
            ;;
        3)
            echo "使用 GHProxy 镜像"
            GITHUB_URL=${GITHUB_MIRRORS[2]}
            ;;
        *)
            echo "使用 Github 原地址"
            GITHUB_URL=${GITHUB_MIRRORS[0]}
            ;;
    esac
}

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
    select_mirror
    echo "正在下载agent..."
    
    # 获取最新release的tag
    LATEST_TAG=$(curl -s "https://api.github.com/repos/afoim/tcping-node/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$LATEST_TAG" ]; then
        LATEST_TAG="v1.0.0"  # 如果获取失败，使用默认版本
    fi
    
    DOWNLOAD_URL="$GITHUB_URL/afoim/tcping-node/releases/download/$LATEST_TAG/agent"
    echo "下载地址: $DOWNLOAD_URL"
    
    # 使用临时文件下载
    TMP_FILE=$(mktemp)
    if wget -q -O "$TMP_FILE" "$DOWNLOAD_URL"; then
        # 检查文件类型
        FILE_TYPE=$(file -b "$TMP_FILE")
        if [[ $FILE_TYPE == *"ELF"* ]]; then  # 验证是否为Linux可执行文件
            mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
            chmod +x "$INSTALL_DIR/$BINARY_NAME"
            install_service
            echo "安装完成"
            echo "当前端口: $PORT"
            systemctl start $SERVICE_NAME
        else
            echo "错误: 下载的文件不是有效的可执行文件"
            echo "文件类型: $FILE_TYPE"
            rm -f "$TMP_FILE"
            exit 1
        fi
    else
        echo "下载失败，请检查网络连接或尝试其他镜像"
        rm -f "$TMP_FILE"
        exit 1
    fi
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

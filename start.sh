#!/bin/bash

# Video Hunter 启动脚本

echo "🎬 启动 Video Hunter..."

# 检查是否在正确的目录
if [ ! -f "main.go" ]; then
    echo "❌ 错误：请在 video-hunter 目录中运行此脚本"
    echo "   当前目录: $(pwd)"
    echo "   请运行: cd /path/to/video-hunter && ./start.sh"
    exit 1
fi

# 检查程序是否存在
if [ ! -f "video-hunter" ]; then
    echo "🔨 编译程序..."
    go build -o video-hunter main.go
    if [ $? -ne 0 ]; then
        echo "❌ 编译失败"
        exit 1
    fi
fi

# 添加执行权限
chmod +x video-hunter

# 检查端口是否被占用
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null ; then
    echo "⚠️  端口 8080 已被占用，正在停止旧服务..."
    pkill -f video-hunter
    sleep 2
fi

# 启动服务
echo "🚀 启动服务..."
./video-hunter &

# 等待服务启动
sleep 3

# 检查服务状态
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ Video Hunter 启动成功！"
    echo "🌐 访问地址: http://localhost:8080"
    echo "📖 快速开始: 查看 QUICK_START.md"
    echo ""
    echo "按 Ctrl+C 停止服务"
    
    # 等待用户中断
    wait
else
    echo "❌ 服务启动失败"
    exit 1
fi 
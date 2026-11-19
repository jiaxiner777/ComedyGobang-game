#!/bin/bash

echo "🎭 喜剧五子棋启动中..."
echo

# 检查是否存在编译好的可执行文件
if [ ! -f "comedy-gobang" ]; then
    echo "正在编译游戏..."
    go build -o comedy-gobang
    if [ $? -ne 0 ]; then
        echo "编译失败！请检查Go环境是否正确安装。"
        exit 1
    fi
fi

echo "🌐 启动服务器..."
echo "游戏将在浏览器中打开: http://localhost:8080"
echo

# 尝试打开默认浏览器
if command -v xdg-open > /dev/null; then
    xdg-open http://localhost:8080
elif command -v open > /dev/null; then
    open http://localhost:8080
fi

# 启动服务器
./comedy-gobang

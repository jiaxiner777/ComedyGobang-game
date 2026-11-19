#!/bin/bash

echo "🎭 喜剧五子棋 - 构建脚本"
echo "================================"

# 设置版本信息
VERSION=${1:-"1.0.0"}
BUILD_TIME=$(date +"%Y-%m-%d %H:%M:%S")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "版本: $VERSION"
echo "构建时间: $BUILD_TIME"
echo "Git提交: $GIT_COMMIT"
echo

# 清理旧的构建文件
echo "🧹 清理旧文件..."
rm -f comedy-gobang*
rm -rf dist/

# 创建分发目录
mkdir -p dist/windows
mkdir -p dist/linux
mkdir -p dist/macOS

# 构建Windows版本
echo "🔨 构建Windows版本..."
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT" -o dist/windows/comedy-gobang.exe

# 构建Linux版本
echo "🔨 构建Linux版本..."
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT" -o dist/linux/comedy-gobang

# 构建macOS版本
echo "🔨 构建macOS版本..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT" -o dist/macOS/comedy-gobang

# 复制配置文件和文档
echo "📋 复制配置文件..."
cp config.json dist/windows/
cp config.json dist/linux/
cp config.json dist/macOS/

cp README.md dist/windows/
cp README.md dist/linux/
cp README.md dist/macOS/

# 创建启动脚本
echo "📜 创建启动脚本..."

# Windows启动脚本
cat > dist/windows/start.bat << 'EOF'
@echo off
echo 🎭 喜剧五子棋启动中...
echo.
echo 游戏将在浏览器中打开: http://localhost:8080
echo.

REM 启动服务器并自动打开浏览器
start http://localhost:8080
comedy-gobang.exe

pause
EOF

# Linux/macOS启动脚本
cat > dist/linux/start.sh << 'EOF'
#!/bin/bash

echo "🎭 喜剧五子棋启动中..."
echo

# 尝试打开默认浏览器
if command -v xdg-open > /dev/null; then
    xdg-open http://localhost:8080
elif command -v open > /dev/null; then
    open http://localhost:8080
fi

# 启动服务器
./comedy-gobang
EOF

cp dist/linux/start.sh dist/macOS/

# 设置执行权限
chmod +x dist/linux/comedy-gobang
chmod +x dist/linux/start.sh
chmod +x dist/macOS/comedy-gobang
chmod +x dist/macOS/start.sh

# 创建压缩包
echo "📦 创建压缩包..."
cd dist

# Windows压缩包
cd windows
zip -r ../../comedy-gobang-windows-v$VERSION.zip .
cd ..

# Linux压缩包
cd linux
tar -czf ../../comedy-gobang-linux-v$VERSION.tar.gz .
cd ..

# macOS压缩包
cd macOS
tar -czf ../../comedy-gobang-macos-v$VERSION.tar.gz .
cd ..

cd ..

# 显示构建结果
echo
echo "✅ 构建完成！"
echo "================================"
echo "构建产物:"
ls -la *.zip *.tar.gz 2>/dev/null || echo "没有找到压缩包"
echo
echo "目录结构:"
tree dist/ 2>/dev/null || find dist/ -type f | sort

echo
echo "🚀 部署说明:"
echo "1. 解压对应平台的压缩包"
echo "2. 运行启动脚本 (start.bat 或 start.sh)"
echo "3. 浏览器会自动打开游戏页面"
echo
echo "🌐 游戏地址: http://localhost:8080"

@echo off
echo 🎭 喜剧五子棋启动中...
echo.

REM 检查是否存在编译好的exe文件
if not exist "comedy-gobang.exe" (
    echo 正在编译游戏...
    go build -o comedy-gobang.exe
    if %errorlevel% neq 0 (
        echo 编译失败！请检查Go环境是否正确安装。
        pause
        exit /b 1
    )
)

echo 🌐 启动服务器...
echo 游戏将在浏览器中打开: http://localhost:8080
echo.

REM 启动服务器并自动打开浏览器
start http://localhost:8080
comedy-gobang.exe

pause

package main

import (
	"comedy-gobang/internal/room"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

//go:embed frontend/dist/*
var frontendFS embed.FS

func main() {
	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	// 创建房间管理器
	roomManager := room.NewManager()

	// 创建 Gin 路由器
	r := gin.Default()

	// 设置 API 路由
	api := r.Group("/api")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "message": "喜剧五子棋服务器运行中"})
		})

		api.GET("/rooms", func(c *gin.Context) {
			rooms := roomManager.GetRoomList()
			c.JSON(200, gin.H{"rooms": rooms})
		})

		api.POST("/rooms", func(c *gin.Context) {
			room := roomManager.CreateRoom()
			c.JSON(200, gin.H{"room": room})
		})
	}

	// 设置 WebSocket 路由
	r.GET("/ws", roomManager.HandleWebSocket)

	// 嵌入前端文件
	frontendFiles, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		log.Fatal("无法获取前端文件:", err)
	}

	// 静态文件服务
	r.StaticFS("/static", http.FS(frontendFiles))

	// 所有其他路由返回 index.html (SPA 支持)
	r.NoRoute(func(c *gin.Context) {
		if c.Request.URL.Path == "/" {
			index, err := frontendFiles.Open("index.html")
			if err != nil {
				c.String(404, "页面未找到")
				return
			}
			defer index.Close()

			content, err := fs.ReadFile(frontendFiles, "index.html")
			if err != nil {
				c.String(500, "服务器错误")
				return
			}

			c.Data(200, "text/html; charset=utf-8", content)
		} else {
			c.FileFromFS(c.Request.URL.Path, http.FS(frontendFiles))
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("喜剧五子棋服务器启动成功!\n")
	fmt.Printf("请访问: http://localhost:%s\n", port)
	fmt.Printf("准备开始一场尴尬的对决吧!\n")

	log.Fatal(r.Run(":" + port))
}

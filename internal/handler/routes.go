package handler

import (
	"net/http"

	"video-hunter/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有路由
func RegisterRoutes(r *gin.Engine, svc *service.Service) {
	// 静态文件
	r.Static("/static", "./web/static")

	// 模板文件
	r.LoadHTMLGlob("web/templates/*")

	// 直接下载到本地（最高优先级）
	r.POST("/direct-download", svc.DirectDownload)

	// 主页
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Video Hunter - 智能视频下载器",
		})
	})

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "video-hunter",
		})
	})

	// API路由组
	api := r.Group("/api")
	{
		// 下载相关API
		api.POST("/download", svc.CreateDownload)
		api.GET("/downloads", svc.GetDownloads)
		api.GET("/downloads/:id", svc.GetDownload)
		api.POST("/downloads/:id/cancel", svc.CancelDownload)
		api.POST("/downloads/clear", svc.ClearDownloads)
		api.GET("/downloads/:id/download", svc.DownloadFile)

		// 视频信息API
		api.GET("/video-info", svc.GetVideoInfo)
	}

	// WebSocket
	r.GET("/ws", svc.HandleWebSocket)
}

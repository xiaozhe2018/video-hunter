package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"video-hunter/internal/config"
	"video-hunter/internal/handler"
	"video-hunter/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	configPath = flag.String("config", "", "配置文件路径")
	port       = flag.Int("port", 0, "服务器端口")
	host       = flag.String("host", "", "服务器地址")
)

func main() {
	flag.Parse()

	// 设置默认配置路径
	if *configPath == "" {
		*configPath = "configs/config.yaml"
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 命令行参数覆盖配置
	if *port > 0 {
		cfg.Server.Port = *port
	}
	if *host != "" {
		cfg.Server.Host = *host
	}

	// 设置日志
	setupLogger(cfg)

	// 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	// 创建服务
	svc := service.NewService(cfg)

	// 创建路由
	router := gin.Default()

	// 设置CORS
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 注册路由
	handler.RegisterRoutes(router, svc)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	// 启动服务器
	go func() {
		logrus.Infof("🚀 Video Hunter 服务启动在 http://%s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("🛑 正在关闭服务器...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Errorf("服务器关闭失败: %v", err)
	}

	logrus.Info("✅ 服务器已关闭")
}

// setupLogger 设置日志
func setupLogger(cfg *config.Config) {
	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// 设置日志格式
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 设置日志输出
	if cfg.Log.File != "" {
		logrus.SetOutput(&lumberjack.Logger{
			Filename:   cfg.Log.File,
			MaxSize:    cfg.Log.MaxSize, // MB
			MaxBackups: cfg.Log.MaxBackups,
			MaxAge:     cfg.Log.MaxAge, // days
			Compress:   true,
		})
	}

	// 同时输出到控制台
	logrus.AddHook(&ConsoleHook{})
}

// ConsoleHook 控制台日志钩子
type ConsoleHook struct{}

func (h *ConsoleHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *ConsoleHook) Fire(entry *logrus.Entry) error {
	// 如果输出不是标准输出，则同时输出到控制台
	if entry.Logger.Out != os.Stdout {
		line, err := entry.String()
		if err != nil {
			return err
		}
		fmt.Print(line)
	}
	return nil
}

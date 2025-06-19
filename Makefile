# Video Hunter Makefile

.PHONY: help build clean test deps dev web cli install

# 变量定义
BINARY_NAME=video-hunter
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

# 默认目标
help: ## 显示帮助信息
	@echo "🎬 Video Hunter - 智能视频下载器"
	@echo ""
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

# 依赖管理
deps: ## 安装Go依赖
	@echo "📦 安装Go依赖..."
	go mod tidy
	go mod download

# 构建
build: deps ## 构建可执行文件
	@echo "🔨 构建 ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} .

# 构建Web版本
web: deps ## 构建Web服务
	@echo "🌐 构建Web服务..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-web .

# 构建CLI版本
cli: deps ## 构建CLI工具
	@echo "💻 构建CLI工具..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-cli .

# 开发模式
dev: ## 启动开发模式
	@echo "🚀 启动开发模式..."
	@go run .

# 运行测试
test: ## 运行测试
	@echo "🧪 运行测试..."
	go test -v ./...

# 清理
clean: ## 清理构建文件
	@echo "🧹 清理构建文件..."
	rm -rf ${BUILD_DIR}
	go clean

# 安装
install: build ## 安装到系统
	@echo "📥 安装 ${BINARY_NAME}..."
	sudo cp ${BUILD_DIR}/${BINARY_NAME} /usr/local/bin/

# 发布构建
release: clean ## 构建发布版本
	@echo "📦 构建发布版本..."
	@mkdir -p ${BUILD_DIR}
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 .
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm64 .
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 .
	
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 .
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe .

# 格式化代码
fmt: ## 格式化Go代码
	@echo "🎨 格式化代码..."
	go fmt ./...

# 代码检查
lint: ## 运行代码检查
	@echo "🔍 运行代码检查..."
	golangci-lint run

# 生成文档
docs: ## 生成API文档
	@echo "📚 生成文档..."
	@mkdir -p docs
	swag init -g main.go -o docs

# Docker构建
docker-build: ## 构建Docker镜像
	@echo "🐳 构建Docker镜像..."
	docker build -t video-hunter:latest .

# Docker运行
docker-run: ## 运行Docker容器
	@echo "🐳 运行Docker容器..."
	docker run -p 8080:8080 -v $(PWD)/downloads:/app/downloads video-hunter:latest

# 创建配置文件
config: ## 创建默认配置文件
	@echo "⚙️ 创建默认配置文件..."
	@mkdir -p configs
	@echo "# 🎬 Video Hunter 配置文件" > configs/config.yaml
	@echo "# 智能视频下载器 - 基于 Go 和 yt-dlp 的现代化视频下载工具" >> configs/config.yaml
	@echo "" >> configs/config.yaml
	@echo "# 服务器配置" >> configs/config.yaml
	@echo "server:" >> configs/config.yaml
	@echo "  # 服务端口 (1-65535)" >> configs/config.yaml
	@echo "  port: 8080" >> configs/config.yaml
	@echo "  # 监听地址 (0.0.0.0 表示监听所有网络接口)" >> configs/config.yaml
	@echo "  host: \"0.0.0.0\"" >> configs/config.yaml
	@echo "  # 运行模式 (debug/release)" >> configs/config.yaml
	@echo "  mode: \"debug\"" >> configs/config.yaml
	@echo "" >> configs/config.yaml
	@echo "# 日志配置" >> configs/config.yaml
	@echo "log:" >> configs/config.yaml
	@echo "  # 日志级别 (debug/info/warn/error)" >> configs/config.yaml
	@echo "  level: \"info\"" >> configs/config.yaml
	@echo "  # 日志文件路径" >> configs/config.yaml
	@echo "  file: \"logs/video-hunter.log\"" >> configs/config.yaml
	@echo "  # 单个日志文件最大大小 (MB)" >> configs/config.yaml
	@echo "  max_size: 100" >> configs/config.yaml
	@echo "  # 日志文件保留天数" >> configs/config.yaml
	@echo "  max_age: 30" >> configs/config.yaml
	@echo "  # 保留的日志文件备份数量" >> configs/config.yaml
	@echo "  max_backups: 10" >> configs/config.yaml
	@echo "" >> configs/config.yaml
	@echo "# 下载器配置" >> configs/config.yaml
	@echo "downloader:" >> configs/config.yaml
	@echo "  # 下载文件保存目录" >> configs/config.yaml
	@echo "  output_dir: \"./downloads\"" >> configs/config.yaml
	@echo "  # 下载失败时的最大重试次数" >> configs/config.yaml
	@echo "  max_retries: 3" >> configs/config.yaml
	@echo "  # 下载超时时间 (秒)" >> configs/config.yaml
	@echo "  timeout: 300" >> configs/config.yaml
	@echo "  # 最大并发下载数" >> configs/config.yaml
	@echo "  max_concurrent: 3" >> configs/config.yaml
	@echo "" >> configs/config.yaml
	@echo "# yt-dlp 配置" >> configs/config.yaml
	@echo "ytdlp:" >> configs/config.yaml
	@echo "  # yt-dlp 命令路径" >> configs/config.yaml
	@echo "  # 支持以下格式:" >> configs/config.yaml
	@echo "  # - \"yt-dlp\" (如果 yt-dlp 已安装到系统 PATH)" >> configs/config.yaml
	@echo "  # - \"python3 -m yt_dlp\" (使用 pip 安装的 yt-dlp)" >> configs/config.yaml
	@echo "  # - \"/usr/bin/python3 -m yt_dlp\" (指定 python3 路径)" >> configs/config.yaml
	@echo "  path: \"yt-dlp\"" >> configs/config.yaml
	@echo "  # 用户代理字符串 (用于反爬虫)" >> configs/config.yaml
	@echo "  user_agent: \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36\"" >> configs/config.yaml
	@echo "  # Cookie 文件路径 (可选，用于需要登录的网站)" >> configs/config.yaml
	@echo "  cookies_file: \"\"" >> configs/config.yaml
	@echo "  # 代理设置 (可选，格式: http://proxy:port 或 socks5://proxy:port)" >> configs/config.yaml
	@echo "  proxy: \"\"" >> configs/config.yaml
	@echo "  # 默认下载格式 (best/worst/720p/1080p/4k 等)" >> configs/config.yaml
	@echo "  format: \"best\"" >> configs/config.yaml
	@echo "  # 是否提取音频 (true/false)" >> configs/config.yaml
	@echo "  extract_audio: false" >> configs/config.yaml
	@echo "  # 音频格式 (mp3/m4a/wav 等)" >> configs/config.yaml
	@echo "  audio_format: \"mp3\"" >> configs/config.yaml
	@echo "  # 音频质量 (128K/192K/320K 等)" >> configs/config.yaml
	@echo "  audio_quality: \"192K\"" >> configs/config.yaml
	@echo "" >> configs/config.yaml
	@echo "# 数据库配置" >> configs/config.yaml
	@echo "database:" >> configs/config.yaml
	@echo "  # 数据库驱动 (sqlite/mysql/postgresql)" >> configs/config.yaml
	@echo "  driver: \"sqlite\"" >> configs/config.yaml
	@echo "  # 数据库连接字符串" >> configs/config.yaml
	@echo "  dsn: \"./data/video-hunter.db\"" >> configs/config.yaml
	@echo "" >> configs/config.yaml
	@echo "# 安全配置" >> configs/config.yaml
	@echo "security:" >> configs/config.yaml
	@echo "  # CORS 允许的源 ([\"*\"] 表示允许所有源)" >> configs/config.yaml
	@echo "  cors_origins: [\"*\"]" >> configs/config.yaml
	@echo "  # 速率限制 (每分钟最大请求数)" >> configs/config.yaml
	@echo "  rate_limit: 100" >> configs/config.yaml
	@echo "  # 速率限制时间窗口" >> configs/config.yaml
	@echo "  rate_limit_window: \"1m\"" >> configs/config.yaml

# 初始化项目
init: config ## 初始化项目
	@echo "🎯 初始化项目..."
	@mkdir -p downloads temp logs web/static web/templates
	@echo "✅ 项目初始化完成！"
	@echo "📝 请查看 configs/config.yaml 进行配置"
	@echo "🚀 运行 'make dev' 启动开发模式"

# 显示版本信息
version: ## 显示版本信息
	@echo "Version: ${VERSION}"
	@echo "Build Time: $(shell date)"
	@echo "Go Version: $(shell go version)" 
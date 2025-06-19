# 🎬 Video Hunter - 智能视频下载器

一个基于 Go 和 yt-dlp 的现代化视频下载工具，支持多种视频平台，提供美观的 Web 界面。

## ✨ 特性

- 🎯 **多平台支持**: YouTube, Pinterest, Bilibili, Twitter, Instagram, TikTok, SpankBang 等
- 🌐 **现代化 Web UI**: 响应式设计，支持实时进度显示
- ⚡ **高性能**: 基于 yt-dlp，支持多线程下载
- 🔧 **易于配置**: YAML 配置文件，支持环境变量
- 📊 **实时监控**: 下载进度、速度、剩余时间显示
- 🎨 **美观界面**: 使用 Tailwind CSS，现代化设计
- 🔒 **安全可靠**: 支持 CORS、速率限制等安全特性
- 🛡️ **稳定架构**: 统一使用 yt-dlp，避免自定义解析的bug

## 🚀 快速开始

详细的快速开始指南请查看 [docs/QUICK_START.md](docs/QUICK_START.md)

### 前置要求

- Go 1.19+
- Python3（推荐使用系统自带的 python3）
- pip 安装 yt-dlp：
  ```bash
  python3 -m pip install -U yt-dlp
  ```

### 配置

编辑 `configs/config.yaml`，配置 yt-dlp 路径：

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  mode: "debug"

log:
  level: "info"
  file: "logs/video-hunter.log"
  max_size: 100
  max_age: 30
  max_backups: 10

downloader:
  output_dir: "./downloads"
  max_retries: 3
  timeout: 300
  max_concurrent: 3

ytdlp:
  # yt-dlp 命令路径 - 支持多种格式
  path: "python3 -m yt_dlp"  # 推荐：使用 pip 安装的 yt-dlp
  # path: "yt-dlp"           # 可选：如果 yt-dlp 已安装到系统 PATH
  user_agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
  format: "best"
  extract_audio: false
  audio_format: "mp3"
  audio_quality: "192K"

database:
  driver: "sqlite"
  dsn: "./data/video-hunter.db"

security:
  cors_origins: ["*"]
  rate_limit: 100
  rate_limit_window: "1m"
```

### 安装 Video Hunter

```bash
# 克隆项目
git clone <repository-url>
cd video-hunter

# 安装依赖
go mod download

# 编译
go build -o video-hunter main.go

# 运行
./video-hunter
```

### 使用启动脚本

```bash
# 使用便捷启动脚本
./start.sh
```

### 使用 Docker

```bash
# 构建镜像
docker build -t video-hunter .

# 运行容器
docker run -p 8080:8080 -v $(pwd)/downloads:/app/downloads video-hunter
```

## 📖 使用说明

### Web 界面

1. 启动服务后访问 `http://localhost:8080`
2. 在输入框中粘贴视频 URL
3. 选择下载格式和质量
4. 点击"开始下载"
5. 在下载列表中查看进度

### API 接口

#### 获取视频信息
```bash
curl "http://localhost:8080/api/video-info?url=<视频URL>"
```

#### 创建下载任务
```bash
curl -X POST http://localhost:8080/api/download \
  -H "Content-Type: application/json" \
  -d '{"url":"<视频URL>","format":"best"}'
```

#### 获取下载列表
```bash
curl http://localhost:8080/api/downloads
```

#### 取消下载
```bash
curl -X POST http://localhost:8080/api/downloads/<任务ID>/cancel
```

#### 清空下载记录
```bash
curl -X POST http://localhost:8080/api/downloads/clear
```

## 📝 更新日志

详细更新历史请查看 [docs/CHANGELOG.md](docs/CHANGELOG.md)

### v1.1.0 (2025-06-18)

- 🔧 **架构优化**: 移除有问题的QinAV专用下载器，统一使用yt-dlp
- 🛡️ **稳定性提升**: 避免自定义解析代码的bug，提高系统稳定性
- 📦 **代码简化**: 减少代码复杂度，更易维护
- 🎯 **功能增强**: yt-dlp支持更多网站，功能更强大
- 🚀 **性能优化**: 统一下载器架构，减少资源占用

### v1.0.0 (2025-06-17)

- ✨ 初始版本发布
- 🌐 现代化 Web 界面
- 🎯 支持多平台视频下载
- ⚡ 基于 yt-dlp 的高性能下载
- 📊 实时进度监控
- 🔧 灵活的配置系统

## 🏗️ 项目架构

### 当前架构 (v1.1.0)
```
Service
└── ytdlp下载器 (统一处理所有网站) ✅ 稳定可靠
```

### 优势
1. **代码更简洁** - 移除了复杂的网站特定逻辑
2. **更稳定** - 统一使用yt-dlp，避免自定义解析的bug
3. **更易维护** - 减少了代码复杂度
4. **功能更强大** - yt-dlp支持更多网站，包括YouTube、Pinterest、Bilibili等

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## ⚠️ 免责声明

- 本工具仅供学习和个人使用
- 请遵守相关网站的使用条款和版权法律
- 开发者不对使用本工具造成的任何问题负责
- 请尊重内容创作者的权益

## 🆘 支持

如果遇到问题，请：

1. 查看 [Issues](../../issues) 是否有类似问题
2. 检查日志文件 `logs/video-hunter.log`
3. 确保 yt-dlp 已正确安装并可用
4. 提交新的 Issue 并附上详细的错误信息
5. 查看 [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) 获取常见问题解决方案

---

**享受下载视频的乐趣！** 🎬✨ 

## SpankBang 支持说明
- 自动加反爬虫请求头，无需 Cookie、无需 impersonate
- 只需粘贴 SpankBang 视频链接即可下载
- 详细说明见 [docs/SPANKBANG_SUPPORT.md](docs/SPANKBANG_SUPPORT.md)

## 本地下载功能
- 支持将视频直接下载到用户本地
- 详细说明见 [docs/LOCAL_DOWNLOAD_FEATURE.md](docs/LOCAL_DOWNLOAD_FEATURE.md)

## 文件名处理
- 自动处理特殊字符和非法文件名
- 详细说明见 [docs/FILENAME_HANDLING.md](docs/FILENAME_HANDLING.md)

## 配置优化
- 详细的配置选项和优化建议
- 详细说明见 [docs/CONFIG_OPTIMIZATION.md](docs/CONFIG_OPTIMIZATION.md)

## 项目结构优化
- 优化的项目结构和代码组织
- 详细说明见 [docs/PROJECT_STRUCTURE_OPTIMIZATION.md](docs/PROJECT_STRUCTURE_OPTIMIZATION.md)

## 常见问题
1. **403 Forbidden**
   - 确认 python3 路径和 yt-dlp 安装一致
   - 网络需能正常访问 SpankBang
   - yt-dlp 需为 pip 安装最新版
2. **下载速度慢/中断**
   - 检查网络和磁盘空间
   - 可选择低清晰度格式
3. **找不到视频信息**
   - 检查链接是否有效，或目标站点反爬虫升级
4. **impersonate 报错**
   - 本项目已无需 impersonate，自动加请求头即可

## 🏗️ 项目结构

```
video-hunter/
├── cmd/                    # 命令行工具
│   ├── cli/               # CLI 版本
│   └── web/               # Web 版本
├── configs/               # 配置文件
│   └── config.yaml        # 主配置文件
├── data/                  # 数据文件
├── docs/                  # 文档
├── downloads/             # 下载目录
├── internal/              # 内部包
│   ├── config/            # 配置管理
│   ├── downloader/        # 下载器
│   │   ├── types.go       # 类型定义
│   │   └── ytdlp_downloader.go  # yt-dlp下载器
│   ├── handler/           # HTTP处理器
│   └── service/           # 业务逻辑
├── logs/                  # 日志文件
├── scripts/               # 脚本文件
├── temp/                  # 临时文件
├── web/                   # Web前端
│   ├── static/            # 静态资源
│   └── templates/         # HTML模板
├── Dockerfile             # Docker配置
├── Makefile               # 构建脚本
├── main.go                # 主程序
├── go.mod                 # Go模块
├── go.sum                 # 依赖校验
├── README.md              # 项目说明
├── QUICK_START.md         # 快速开始
├── CHANGELOG.md           # 更新日志
├── SPANKBANG_SUPPORT.md   # SpankBang支持说明
└── start.sh               # 启动脚本
```

## 📚 相关文档

- [快速开始指南](QUICK_START.md) - 立即上手使用
- [更新日志](CHANGELOG.md) - 详细的版本更新历史
- [SpankBang支持说明](SPANKBANG_SUPPORT.md) - SpankBang网站下载指南

## 🌟 项目亮点

- **🎯 多平台支持**: 支持YouTube、Pinterest、Bilibili、Twitter、Instagram、TikTok、SpankBang等主流视频平台
- **🌐 现代化界面**: 响应式Web界面，支持实时进度显示
- **⚡ 高性能**: 基于yt-dlp的高性能下载引擎
- **🛡️ 稳定可靠**: 统一使用yt-dlp，避免自定义解析的bug
- **🔧 易于部署**: 支持Docker一键部署
- **📊 实时监控**: 下载进度、速度、剩余时间实时显示

## 🚀 快速体验

```bash
# 克隆项目
git clone <repository-url>
cd video-hunter

# 使用启动脚本（推荐）
./start.sh

# 或手动启动
go build -o video-hunter main.go
./video-hunter
```

然后访问 http://localhost:8080 开始使用！

---

**享受下载视频的乐趣！** 🎬✨ 

## 抖音视频下载说明

抖音视频下载功能已经优化，不再需要访问Chrome Safe Storage。主要改进包括：

1. 移除了 `--cookies-from-browser chrome` 参数，避免访问浏览器的安全存储
2. 使用移动端 User-Agent 提高下载成功率
3. 添加 `--force-generic-extractor` 参数尝试通用下载方式
4. 使用 aria2c 作为下载器提高下载速度和稳定性
5. 优化错误提示，提供更友好的用户体验
6. 自动修正抖音链接格式，提高下载成功率
7. 当无法获取视频信息时，禁用下载按钮并提供明确的解决方案

**注意**：由于抖音的反爬虫机制，部分视频可能需要特殊处理：
- 优先使用移动端分享的短链接（以 v.douyin.com 开头）
- 确保链接格式正确，例如：https://v.douyin.com/xxxxxx/
- 尝试使用抖音官方分享功能获取新的链接

## 使用方法

1. 确保已安装 yt-dlp 和 ffmpeg
2. 启动服务：`./video-hunter`
3. 访问 http://localhost:8080
4. 输入视频链接并下载

## 支持的平台

- YouTube
- Bilibili
- 抖音
- 其他 yt-dlp 支持的平台

## 配置

详细配置请参考 `configs/config.yaml` 文件。
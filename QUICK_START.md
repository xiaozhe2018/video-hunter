# 🚀 Video Hunter 快速开始指南

## 立即开始

### 方法一：使用启动脚本（推荐）
```bash
# 在 video-hunter 目录中
./start.sh
```

### 方法二：手动启动
```bash
# 1. 确保在正确的目录
cd video-hunter

# 2. 编译程序（如果需要）
go build -o video-hunter main.go

# 3. 启动服务
./video-hunter
```

### 2. 访问 Web 界面
打开浏览器访问：http://localhost:8080

### 3. 下载视频
1. 复制视频 URL（支持多种平台）
2. 粘贴到输入框
3. 选择下载格式（推荐：最佳质量）
4. 点击"开始下载"
5. 在下载列表中查看进度



## 功能演示

### 测试视频下载
```bash
# 获取视频信息
curl "http://localhost:8080/api/video-info?url=https://www.youtube.com/watch?v=dQw4w9WgXcQ"

# 创建下载任务
curl -X POST http://localhost:8080/api/download \
  -H "Content-Type: application/json" \
  -d '{"url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","format":"best"}'
```

## 支持的平台

- ✅ **YouTube** - 完全支持
- ✅ **Pinterest** - 完全支持
- ✅ **Bilibili** - 完全支持
- ✅ **Twitter** - 完全支持
- ✅ **Instagram** - 完全支持
- ✅ **TikTok** - 完全支持
- ✅ **SpankBang** - 完全支持（自动反爬虫）
- ✅ **其他 yt-dlp 支持的网站**

## 🆕 v1.1.0 更新亮点

### 架构优化
- 🛡️ **稳定性提升**: 避免自定义解析代码的bug，提高系统稳定性
- 📦 **代码简化**: 减少代码复杂度，更易维护
- 🚀 **性能优化**: 统一下载器架构，减少资源占用

### 优势
1. **更稳定** - 统一使用yt-dlp，避免自定义解析的bug
2. **更简洁** - 移除了复杂的网站特定逻辑
3. **更易维护** - 减少了代码复杂度
4. **功能更强大** - yt-dlp支持更多网站

## 常见问题

### Q: 服务启动失败？
A: 检查端口8080是否被占用，或使用其他端口：
```bash
./video-hunter -port 9090
```

### Q: 下载失败？
A: 确保已安装 yt-dlp：
```bash
# macOS
python3 -m pip install -U yt-dlp

# Ubuntu/Debian
sudo apt install yt-dlp
```

### Q: 权限问题？
A: 给程序添加执行权限：
```bash
chmod +x video-hunter
chmod +x start.sh
```

### Q: 模板文件找不到？
A: 确保在正确的目录运行：
```bash
# 检查当前目录
pwd
# 应该显示: /path/to/video-hunter

# 检查文件是否存在
ls -la main.go web/templates/index.html
```

### Q: yt-dlp 路径问题？
A: 检查配置文件中的python3路径：
```bash
# 查看当前python3路径
which python3

# 编辑配置文件
vim configs/config.yaml
# 确保 ytdlp.path 指向正确的python3路径
```

## 文件位置

- **下载目录**: `./downloads/`
- **日志文件**: `./logs/video-hunter.log`
- **配置文件**: `./configs/config.yaml`

## 下一步

- 📖 查看完整文档：[README.md](README.md)
- 🔧 自定义配置：[configs/config.yaml](configs/config.yaml)
- 🐛 报告问题：提交 Issue
- 📝 查看更新日志：[README.md#更新日志](README.md#更新日志)

---

**享受下载视频的乐趣！** 🎬✨ 
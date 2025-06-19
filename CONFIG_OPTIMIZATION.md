# 🎯 Video Hunter 配置文件优化说明

## 📋 优化概述

本次优化主要针对 `configs/config.yaml` 配置文件，提升了配置的可读性、可维护性和易用性。

## ✨ 主要改进

### 1. 📝 添加详细注释
- 为每个配置项添加了中文注释说明
- 说明了配置项的作用和可选值
- 提供了配置示例和最佳实践

### 2. 🔧 简化 yt-dlp 路径配置
**优化前：**
```yaml
ytdlp:
  path: "/Library/Developer/CommandLineTools/usr/bin/python3 -m yt_dlp"
```

**优化后：**
```yaml
ytdlp:
  # yt-dlp 命令路径
  # 支持以下格式:
  # - "yt-dlp" (如果 yt-dlp 已安装到系统 PATH)
  # - "python3 -m yt_dlp" (使用 pip 安装的 yt-dlp)
  # - "/usr/bin/python3 -m yt_dlp" (指定 python3 路径)
  path: "python3 -m yt_dlp"
```

### 3. 🎯 统一配置格式
- 确保 YAML 配置文件、代码默认配置和 Makefile 生成的配置保持一致
- 更新了相关文档中的配置示例

## 🔄 相关文件更新

### 代码文件
- `internal/config/config.go` - 更新默认配置值
- `Makefile` - 更新配置生成模板

### 文档文件
- `README.md` - 更新配置示例
- `SPANKBANG_SUPPORT.md` - 更新配置说明

## 🚀 使用建议

### yt-dlp 路径配置选项

1. **推荐配置** (使用 pip 安装的 yt-dlp)：
   ```yaml
   ytdlp:
     path: "python3 -m yt_dlp"
   ```

2. **系统 PATH 配置** (如果 yt-dlp 已安装到系统 PATH)：
   ```yaml
   ytdlp:
     path: "yt-dlp"
   ```

3. **指定 Python 路径** (如果需要使用特定 Python 版本)：
   ```yaml
   ytdlp:
     path: "/usr/bin/python3 -m yt_dlp"
   ```

## 📊 配置项说明

### 服务器配置
- `port`: 服务端口 (1-65535)
- `host`: 监听地址 (0.0.0.0 表示监听所有网络接口)
- `mode`: 运行模式 (debug/release)

### 日志配置
- `level`: 日志级别 (debug/info/warn/error)
- `file`: 日志文件路径
- `max_size`: 单个日志文件最大大小 (MB)
- `max_age`: 日志文件保留天数
- `max_backups`: 保留的日志文件备份数量

### 下载器配置
- `output_dir`: 下载文件保存目录
- `max_retries`: 下载失败时的最大重试次数
- `timeout`: 下载超时时间 (秒)
- `max_concurrent`: 最大并发下载数

### yt-dlp 配置
- `path`: yt-dlp 命令路径
- `user_agent`: 用户代理字符串 (用于反爬虫)
- `cookies_file`: Cookie 文件路径 (可选)
- `proxy`: 代理设置 (可选)
- `format`: 默认下载格式
- `extract_audio`: 是否提取音频
- `audio_format`: 音频格式
- `audio_quality`: 音频质量

### 数据库配置
- `driver`: 数据库驱动 (sqlite/mysql/postgresql)
- `dsn`: 数据库连接字符串

### 安全配置
- `cors_origins`: CORS 允许的源
- `rate_limit`: 速率限制 (每分钟最大请求数)
- `rate_limit_window`: 速率限制时间窗口

## 🔍 验证配置

确保配置正确后，可以通过以下方式验证：

1. **编译程序**：
   ```bash
   go build -o video-hunter main.go
   ```

2. **启动服务**：
   ```bash
   ./video-hunter
   ```

3. **检查日志**：
   ```bash
   tail -f logs/video-hunter.log
   ```

## 📝 注意事项

1. **路径配置**：确保 yt-dlp 路径配置正确，建议使用相对路径
2. **权限设置**：确保程序有权限访问配置的目录
3. **网络访问**：确保网络能正常访问目标视频网站
4. **依赖安装**：确保已正确安装 yt-dlp

## 🎉 优化效果

- ✅ **可读性提升**：详细的中文注释让配置更易理解
- ✅ **易用性增强**：简化的路径配置减少了配置难度
- ✅ **一致性保证**：统一了所有相关文件的配置格式
- ✅ **维护性改善**：清晰的注释和结构便于后续维护

---

**享受优化的配置体验！** 🎬✨ 
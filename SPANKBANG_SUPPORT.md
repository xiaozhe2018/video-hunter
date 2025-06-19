# SpankBang 支持说明

## 概述

Video Hunter 现已完整支持 SpankBang 视频下载，无需 Cookie、无需 impersonate，只需粘贴视频链接即可。

## 技术实现

- 自动为 SpankBang 视频请求加上反爬虫必需的请求头（User-Agent、Accept、Accept-Language 等）
- 基于 yt-dlp，调用 python3 -m yt_dlp，兼容 macOS/Linux
- 进度、速度、状态实时显示，支持断点续传

## 配置要点

- `configs/config.yaml` 中 `ytdlp.path` 配置 yt-dlp 命令路径，例如：
  ```yaml
  ytdlp:
    # 推荐：使用 pip 安装的 yt-dlp
    path: "python3 -m yt_dlp"
    # 可选：如果 yt-dlp 已安装到系统 PATH
    # path: "yt-dlp"
  ```
- 不需要 `--impersonate` 参数，自动加请求头即可

## 使用方法

### Web 界面
1. 访问 [http://localhost:8080](http://localhost:8080)
2. 粘贴 SpankBang 视频链接
3. 选择格式（推荐 best）
4. 点击下载，进度实时显示

### API 示例
- 获取视频信息：
  ```bash
  curl "http://localhost:8080/api/video-info?url=https://spankbang.com/cnn3y-qn55d9/playlist/yumiaomiao"
  ```
- 创建下载任务：
  ```bash
  curl -X POST http://localhost:8080/api/download \
    -H "Content-Type: application/json" \
    -d '{"url":"https://spankbang.com/cnn3y-qn55d9/playlist/yumiaomiao","format":"best"}'
  ```

## 常见问题与排查

1. **403 Forbidden**
   - 确认 python3 路径和 pip 安装的 yt-dlp 一致
   - 不要用 Homebrew 版 yt-dlp，必须用 pip 安装
   - 网络需能正常访问 SpankBang
2. **impersonate 报错**
   - 本项目已无需 impersonate，自动加请求头即可
3. **下载速度慢/中断**
   - 检查网络和磁盘空间
   - 可选择低清晰度格式
4. **找不到视频信息**
   - 检查链接是否有效，或目标站点反爬虫升级

## 日志查看
```bash
tail -f logs/video-hunter.log
```

## 进阶说明
- 支持多线程下载、断点续传、自动重命名
- 下载中会生成 `.part` 临时文件，下载完成后自动重命名
- 支持多站点（YouTube、Pinterest、Bilibili、SpankBang等）

## 更新历史
- **2025-06-17**: 完整支持 SpankBang 视频下载，无需 impersonate

---

**仅供学习交流，请勿用于非法用途！** 
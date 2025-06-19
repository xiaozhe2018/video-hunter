# 🔧 Video Hunter 故障排除指南

## 🚨 常见问题及解决方案

### 1. 500 Internal Server Error - 获取视频信息失败

**错误现象：**
```
GET http://localhost:8080/api/video-info?url=... 500 (Internal Server Error)
获取视频信息失败: Error: HTTP 500
```

**可能原因：**
- yt-dlp 路径配置错误
- yt-dlp 未正确安装
- Python 环境问题

**解决方案：**

#### 步骤1：检查 yt-dlp 安装
```bash
# 检查 yt-dlp 是否在系统 PATH 中
which yt-dlp

# 检查 yt-dlp 版本
yt-dlp --version

# 如果上述命令失败，检查 pip 安装
python3 -m pip list | grep yt-dlp
```

#### 步骤2：更新配置文件
根据检测结果，更新 `configs/config.yaml`：

**如果 yt-dlp 在系统 PATH 中：**
```yaml
ytdlp:
  path: "yt-dlp"
```

**如果使用 pip 安装的 yt-dlp：**
```yaml
ytdlp:
  path: "python3 -m yt_dlp"
```

**如果需要指定 Python 路径：**
```yaml
ytdlp:
  path: "/usr/bin/python3 -m yt_dlp"
```

#### 步骤3：安装 yt-dlp（如果未安装）
```bash
# 使用 Homebrew (macOS)
brew install yt-dlp

# 或使用 pip
python3 -m pip install -U yt-dlp
```

#### 步骤4：重启服务
```bash
# 停止服务
pkill -f video-hunter

# 重新启动
./video-hunter
```

### 2. 下载失败 - exit status 1

**错误现象：**
```
下载失败: exit status 1
```

**可能原因：**
- 网络连接问题
- 目标网站反爬虫机制
- 磁盘空间不足
- 权限问题

**解决方案：**

#### 检查网络连接
```bash
# 测试网络连接
ping google.com

# 检查代理设置
echo $http_proxy
echo $https_proxy
```

#### 检查磁盘空间
```bash
# 检查磁盘空间
df -h

# 检查下载目录权限
ls -la downloads/
```

#### 更新用户代理
在 `configs/config.yaml` 中更新用户代理：
```yaml
ytdlp:
  user_agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
```

### 3. WebSocket 连接问题

**错误现象：**
```
WebSocket连接已建立
读取WebSocket消息失败: websocket: close 1001 (going away)
```

**解决方案：**
- 这是正常的连接行为，不影响功能
- 浏览器会自动重连
- 如果频繁出现，检查网络稳定性

### 4. 端口被占用

**错误现象：**
```
bind: address already in use
```

**解决方案：**
```bash
# 查找占用端口的进程
lsof -i :8080

# 杀死占用进程
kill -9 <PID>

# 或使用其他端口
./video-hunter -port 9090
```

### 5. 配置文件问题

**错误现象：**
```
读取配置文件失败
```

**解决方案：**
```bash
# 重新生成配置文件
make config

# 或手动创建
cp configs/config.yaml configs/config.yaml.backup
make config
```

## 🔍 调试技巧

### 1. 查看详细日志
```bash
# 实时查看日志
tail -f logs/video-hunter.log

# 查看最近的错误
grep "ERROR" logs/video-hunter.log | tail -10
```

### 2. 测试 yt-dlp 命令
```bash
# 测试 yt-dlp 是否能正常工作
yt-dlp --dump-json "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

# 测试特定网站
yt-dlp --dump-json "https://spankbang.com/60lfb/video/ol"
```

### 3. 使用测试脚本
```bash
# 运行API测试
./test_api.sh
```

## 📋 检查清单

在报告问题前，请确认：

- [ ] yt-dlp 已正确安装并可用
- [ ] 配置文件中的路径设置正确
- [ ] 网络连接正常
- [ ] 磁盘空间充足
- [ ] 服务已重启以应用新配置
- [ ] 查看了日志文件中的错误信息

## 🆘 获取帮助

如果问题仍未解决：

1. **查看日志文件**：`logs/video-hunter.log`
2. **运行测试脚本**：`./test_api.sh`
3. **检查系统信息**：
   ```bash
   echo "Python版本: $(python3 --version)"
   echo "yt-dlp版本: $(yt-dlp --version)"
   echo "Go版本: $(go version)"
   ```
4. **提交 Issue**：包含错误日志和系统信息

## 🎯 预防措施

1. **定期更新 yt-dlp**：
   ```bash
   brew upgrade yt-dlp
   # 或
   python3 -m pip install -U yt-dlp
   ```

2. **备份配置文件**：
   ```bash
   cp configs/config.yaml configs/config.yaml.backup
   ```

3. **监控日志**：
   ```bash
   tail -f logs/video-hunter.log
   ```

---

**希望这个指南能帮助您解决问题！** 🎬✨ 
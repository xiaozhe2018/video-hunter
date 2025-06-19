package service

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"video-hunter/internal/config"
	"video-hunter/internal/downloader"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Service 服务层
type Service struct {
	config     *config.Config
	ytdlp      *downloader.YtdlpDownloader
	douyin     *downloader.DouyinDownloader // 添加抖音下载器
	downloads  map[string]*downloader.DownloadResponse
	mu         sync.RWMutex
	wsClients  map[*websocket.Conn]bool
	wsMutex    sync.RWMutex
	downloadCh chan *downloadTask
	upgrader   websocket.Upgrader
}

// downloadTask 下载任务
type downloadTask struct {
	ID  string
	Req *downloader.DownloadRequest
}

// NewService 创建新的服务实例
func NewService(cfg *config.Config) *Service {
	s := &Service{
		config:     cfg,
		ytdlp:      downloader.NewYtdlpDownloader(cfg),
		douyin:     downloader.NewDouyinDownloader(), // 初始化抖音下载器
		downloads:  make(map[string]*downloader.DownloadResponse),
		wsClients:  make(map[*websocket.Conn]bool),
		downloadCh: make(chan *downloadTask, 100), // 增加缓冲区大小到100
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	// 启动下载工作池
	for i := 0; i < 10; i++ { // 增加工作协程数量到10
		go s.downloadWorker()
	}

	return s
}

// CreateDownload 创建下载任务
func (s *Service) CreateDownload(c *gin.Context) {
	var req downloader.DownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 生成下载ID
	downloadID := uuid.New().String()
	req.TaskID = downloadID // 设置任务ID

	// 创建下载响应
	download := &downloader.DownloadResponse{
		ID:       downloadID,
		Status:   downloader.StatusPending,
		Progress: 0,
		Created:  time.Now(),
		Updated:  time.Now(),
		File:     req.Output,
		Metadata: make(map[string]string),
	}

	// 保存save_to_local信息
	if req.SaveToLocal {
		download.Metadata["save_to_local"] = "true"
	}

	// 保存下载记录
	s.mu.Lock()
	s.downloads[downloadID] = download
	s.mu.Unlock()

	// 异步开始下载
	go s.startDownload(downloadID, &req)

	c.JSON(http.StatusOK, download)
}

// GetDownloads 获取所有下载任务
func (s *Service) GetDownloads(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	downloads := make([]*downloader.DownloadResponse, 0, len(s.downloads))
	for _, download := range s.downloads {
		downloads = append(downloads, download)
	}

	c.JSON(http.StatusOK, downloads)
}

// GetDownload 获取单个下载任务
func (s *Service) GetDownload(c *gin.Context) {
	id := c.Param("id")

	s.mu.RLock()
	download, exists := s.downloads[id]
	s.mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "下载任务不存在"})
		return
	}

	c.JSON(http.StatusOK, download)
}

// CancelDownload 取消下载任务
func (s *Service) CancelDownload(c *gin.Context) {
	id := c.Param("id")

	s.mu.Lock()
	defer s.mu.Unlock()

	download, exists := s.downloads[id]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "下载任务不存在"})
		return
	}

	download.Status = downloader.StatusCancelled
	download.Updated = time.Now()

	c.JSON(http.StatusOK, gin.H{"message": "下载已取消"})
}

// ClearDownloads 清空所有下载记录
func (s *Service) ClearDownloads(c *gin.Context) {
	s.mu.Lock()
	s.downloads = make(map[string]*downloader.DownloadResponse)
	s.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"message": "已清空所有下载记录"})
}

// GetVideoInfo 获取视频信息
func (s *Service) GetVideoInfo(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少URL参数"})
		return
	}

	var info *downloader.VideoInfo
	var err error

	// 判断是否为抖音链接
	if s.isDouyinURL(url) {
		// 使用抖音下载器获取视频信息
		info, err = s.douyin.GetVideoInfo(url)
	} else {
		// 使用yt-dlp下载器获取其他视频信息
		info, err = s.ytdlp.GetVideoInfo(url)
	}

	if err != nil {
		logrus.Errorf("获取视频信息失败: %v", err)

		// 对抖音视频提供更友好的错误信息
		if strings.Contains(url, "douyin.com") || strings.Contains(url, "v.douyin.com") {
			// 如果是抖音视频，提供更友好的错误信息
			errMsg := "抖音视频需要特殊处理，请尝试以下方法："
			solutions := []string{
				"使用移动端分享的链接（以v.douyin.com开头的短链接）",
				"确保链接格式正确，例如：https://v.douyin.com/xxxxxx/",
				"尝试使用抖音官方分享功能获取新的链接",
			}

			// 针对特定错误提供具体建议
			if strings.Contains(err.Error(), "Fresh cookies") || strings.Contains(err.Error(), "cookies") {
				errMsg = "抖音视频需要登录信息，请尝试以下方法："
				solutions = append(solutions, "使用移动端APP分享的链接")
			}

			// 检查是否是特定链接格式的问题
			if strings.Contains(err.Error(), "Unsupported URL") {
				errMsg = "抖音链接格式不正确，请尝试以下方法："
				solutions = append(solutions, "确保链接完整且正确")
				solutions = append(solutions, "使用短链接格式：https://v.douyin.com/xxxxxx/")
			}

			c.JSON(http.StatusOK, gin.H{
				"title": "抖音视频",
				"formats": []map[string]interface{}{
					{
						"format_id":  "best",
						"ext":        "mp4",
						"resolution": "最佳质量",
					},
				},
				"special_note":   errMsg,
				"solutions":      solutions,
				"can_download":   false,
				"error_type":     "douyin_access",
				"original_error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取视频信息失败"})
		return
	}

	c.JSON(http.StatusOK, info)
}

// isDouyinURL 判断是否为抖音链接
func (s *Service) isDouyinURL(urlStr string) bool {
	patterns := []string{
		`^https?://v\.douyin\.com/`,
		`^https?://www\.douyin\.com/video/`,
		`^https?://www\.douyin\.com/share/video/`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, urlStr)
		if matched {
			return true
		}
	}
	return false
}

// HandleWebSocket WebSocket处理器
func (s *Service) HandleWebSocket(c *gin.Context) {
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("WebSocket升级失败: %v", err)
		return
	}
	defer conn.Close()

	// 添加到客户端列表
	s.wsMutex.Lock()
	s.wsClients[conn] = true
	s.wsMutex.Unlock()

	logrus.Info("WebSocket连接已建立")

	// 发送当前所有下载状态
	s.mu.RLock()
	for id, download := range s.downloads {
		message := map[string]interface{}{
			"type":     "progress",
			"id":       id,
			"progress": download.Progress,
			"speed":    download.Speed,
			"eta":      download.ETA,
			"status":   download.Status,
			"file":     download.File,
		}
		if err := conn.WriteJSON(message); err != nil {
			logrus.Errorf("发送初始状态失败: %v", err)
		}
	}
	s.mu.RUnlock()

	// 保持连接并处理消息
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			logrus.Errorf("读取WebSocket消息失败: %v", err)
			break
		}
	}

	// 连接关闭时清理
	s.wsMutex.Lock()
	delete(s.wsClients, conn)
	s.wsMutex.Unlock()
}

// downloadWorker 下载工作协程
func (s *Service) downloadWorker() {
	for task := range s.downloadCh {
		s.processDownload(task.ID, task.Req)
	}
}

// processDownload 处理下载任务
func (s *Service) processDownload(id string, req *downloader.DownloadRequest) {
	logrus.Infof("开始处理下载任务: %s, URL: %s", id, req.URL)

	// 获取下载记录
	s.mu.RLock()
	download, exists := s.downloads[id]
	s.mu.RUnlock()

	if !exists {
		logrus.Errorf("下载任务不存在: %s", id)
		return
	}

	// 更新状态为下载中
	s.mu.Lock()
	download.Status = downloader.StatusDownloading
	download.Updated = time.Now()
	s.mu.Unlock()

	// 广播进度更新
	s.broadcastProgress(id, download)

	// 判断是否为抖音链接
	if s.isDouyinURL(req.URL) {
		// 使用抖音下载器
		resp, err := s.douyin.Download(req)
		if err == nil && resp != nil {
			// 更新下载信息
			s.mu.Lock()
			download.Status = resp.Status
			download.Progress = resp.Progress
			download.Speed = resp.Speed
			download.ETA = resp.ETA
			download.File = resp.File
			download.Error = resp.Error
			download.Updated = time.Now()
			s.mu.Unlock()

			// 广播进度更新
			s.broadcastProgress(id, download)
			return
		}
		// 如果抖音下载器失败，记录错误
		logrus.Errorf("抖音下载器失败: %v，将尝试使用yt-dlp下载器", err)
	}

	// 使用yt-dlp下载器作为后备
	s.processYtdlpDownload(id, req, download)
}

// processYtdlpDownload 使用yt-dlp处理下载
func (s *Service) processYtdlpDownload(id string, req *downloader.DownloadRequest, download *downloader.DownloadResponse) {
	// 使用任务ID作为文件名前缀
	if req.Output == "" {
		req.Output = filepath.Join(s.config.Downloader.OutputDir, id+"_video.mp4")
	} else {
		// 如果有指定输出文件名，在前面加上任务ID
		dir := filepath.Dir(req.Output)
		base := filepath.Base(req.Output)
		req.Output = filepath.Join(dir, id+"_"+base)
	}

	// 上次进度更新时间
	var lastProgressTime time.Time
	// 上次进度值
	var lastProgress float64

	// 创建进度回调
	progressCallback := func(progress *downloader.DownloadResponse) {
		// 获取当前时间
		now := time.Now()

		// 仅在以下情况下更新和广播进度：
		// 1. 首次更新
		// 2. 距离上次更新已经过去至少1秒
		// 3. 进度变化超过1%
		// 4. 进度达到100%（完成）
		if lastProgressTime.IsZero() ||
			now.Sub(lastProgressTime) >= time.Second ||
			progress.Progress-lastProgress >= 1.0 ||
			progress.Progress >= 100.0 {

			s.mu.Lock()
			download.Progress = progress.Progress
			download.Speed = progress.Speed
			download.ETA = progress.ETA
			download.Updated = now
			s.mu.Unlock()

			// 发送进度更新
			s.broadcastProgress(id, download)

			// 更新上次进度时间和值
			lastProgressTime = now
			lastProgress = progress.Progress

			// 仅在进度变化显著时记录日志
			logrus.Infof("下载进度 [%s]: %.1f%% %s", id, progress.Progress, progress.Speed)
		}
	}

	// 使用yt-dlp下载器
	actualFile, err := s.ytdlp.Download(req, progressCallback)

	s.mu.Lock()
	if err != nil {
		download.Status = downloader.StatusFailed
		download.Error = err.Error()
		logrus.Errorf("下载失败 [%s]: %v", id, err)
	} else {
		// 验证文件是否真的存在
		if _, fileErr := os.Stat(actualFile); fileErr != nil {
			download.Status = downloader.StatusFailed
			download.Error = fmt.Sprintf("下载完成但文件不存在: %v", fileErr)
			logrus.Errorf("下载完成但文件不存在 [%s]: %v", id, fileErr)
		} else {
			// 确保文件名包含任务ID
			if !strings.Contains(filepath.Base(actualFile), id) {
				// 如果文件名不包含任务ID，尝试重命名文件
				dir := filepath.Dir(actualFile)
				ext := filepath.Ext(actualFile)
				base := strings.TrimSuffix(filepath.Base(actualFile), ext)
				newPath := filepath.Join(dir, id+"_"+base+ext)

				if renameErr := os.Rename(actualFile, newPath); renameErr == nil {
					actualFile = newPath
					logrus.Infof("文件已重命名为: %s", actualFile)
				} else {
					logrus.Warnf("无法重命名文件 [%s]: %v", id, renameErr)
				}
			}

			// 检查是否有对应的音频文件
			if strings.Contains(actualFile, ".f") && strings.HasSuffix(actualFile, ".mp4") {
				videoFile := actualFile
				audioFile := ""

				// 尝试查找对应的音频文件
				filePattern := strings.Replace(actualFile, ".f", ".f", 1)
				filePattern = strings.TrimSuffix(filePattern, filepath.Ext(filePattern)) + ".*"
				matches, _ := filepath.Glob(filePattern)

				for _, match := range matches {
					if match != actualFile && (strings.Contains(match, ".m4a") || strings.Contains(match, ".mp3") || strings.Contains(match, ".aac") || strings.Contains(match, ".opus")) {
						audioFile = match
						break
					}
				}

				// 如果找到了音频文件，尝试合并
				if audioFile != "" {
					logrus.Infof("找到对应的音频文件: %s", audioFile)
					mergedPath := filepath.Join(filepath.Dir(videoFile), id+"_merged"+filepath.Ext(videoFile))

					// 使用ffmpeg合并视频和音频
					cmd := exec.Command("ffmpeg", "-i", videoFile, "-i", audioFile, "-c", "copy", mergedPath)
					var stderr bytes.Buffer
					cmd.Stderr = &stderr
					if err := cmd.Run(); err == nil {
						logrus.Infof("成功合并视频和音频到: %s", mergedPath)
						actualFile = mergedPath
					} else {
						logrus.Errorf("合并视频和音频失败: %v, 错误输出: %s", err, stderr.String())
					}
				}
			}

			// 更新下载状态
			download.Status = downloader.StatusCompleted
			download.Progress = 100
			download.File = actualFile
			logrus.Infof("下载完成 [%s]: %s", id, download.File)
		}
	}
	download.Updated = time.Now()
	s.mu.Unlock()

	// 广播进度更新
	s.broadcastProgress(id, download)
}

// startDownload 开始下载任务
func (s *Service) startDownload(id string, req *downloader.DownloadRequest) {
	s.downloadCh <- &downloadTask{
		ID:  id,
		Req: req,
	}
}

// broadcastProgress 广播下载进度
func (s *Service) broadcastProgress(id string, download *downloader.DownloadResponse) {
	// 记录日志
	logrus.Infof("下载进度 [%s]: %.1f%% %s", id, download.Progress, download.Speed)

	// 准备要发送的消息
	message := map[string]interface{}{
		"type":     "progress",
		"id":       id,
		"progress": download.Progress,
		"speed":    download.Speed,
		"eta":      download.ETA,
		"status":   download.Status,
		"file":     download.File,
		"error":    download.Error,
		"updated":  download.Updated,
	}

	// 广播到所有WebSocket客户端
	s.wsMutex.RLock()
	for client := range s.wsClients {
		err := client.WriteJSON(message)
		if err != nil {
			logrus.Errorf("发送WebSocket消息失败: %v", err)
			client.Close()
			delete(s.wsClients, client)
		}
	}
	s.wsMutex.RUnlock()
}

// DownloadFile 下载文件到用户本地
func (s *Service) DownloadFile(c *gin.Context) {
	id := c.Param("id")
	logrus.Infof("DownloadFile API被调用，ID: %s", id)

	s.mu.RLock()
	download, exists := s.downloads[id]
	s.mu.RUnlock()

	if !exists {
		logrus.Errorf("下载任务不存在: %s", id)
		c.JSON(http.StatusNotFound, gin.H{"error": "下载任务不存在"})
		return
	}

	if download.Status != downloader.StatusCompleted {
		logrus.Errorf("下载尚未完成: %s, 状态: %s", id, download.Status)
		c.JSON(http.StatusBadRequest, gin.H{"error": "下载尚未完成"})
		return
	}

	// 检查文件是否存在
	if download.File == "" {
		logrus.Errorf("文件路径为空 [%s]", id)
		c.JSON(http.StatusNotFound, gin.H{"error": "文件路径为空"})
		return
	}

	logrus.Infof("原始文件路径: %s", download.File)

	// 构建绝对路径
	var absPath string
	if filepath.IsAbs(download.File) {
		absPath = download.File
	} else {
		// 如果是相对路径，转换为绝对路径
		var err error
		absPath, err = filepath.Abs(download.File)
		if err != nil {
			logrus.Errorf("无法获取绝对路径 [%s]: %v", id, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取绝对路径"})
			return
		}
	}

	logrus.Infof("绝对文件路径: %s", absPath)

	// 检查文件是否存在
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		logrus.Errorf("文件不存在或无法访问 [%s]: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在或无法访问"})
		return
	}

	logrus.Infof("文件存在，大小: %d bytes", fileInfo.Size())

	// 生成友好的文件名
	filename := filepath.Base(absPath)

	// 移除任务ID前缀，使文件名更友好
	if strings.Contains(filename, id+"_") {
		// 移除任务ID前缀
		filename = strings.Replace(filename, id+"_", "", 1)
	}

	// 如果文件名包含格式标识符（如.f100026），尝试移除它
	if strings.Contains(filename, ".f") {
		parts := strings.Split(filename, ".f")
		if len(parts) > 1 {
			// 查找第二个点的位置
			secondPart := parts[1]
			dotIndex := strings.Index(secondPart, ".")
			if dotIndex > 0 {
				// 重建文件名，移除格式标识符
				ext := secondPart[dotIndex:]
				filename = parts[0] + ext
			}
		}
	}

	// 如果是合并后的文件，移除"_merged"后缀
	filename = strings.Replace(filename, "_merged", "", 1)

	logrus.Infof("设置下载文件名: %s", filename)

	// 设置响应头，指定文件名
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// 发送文件
	c.File(absPath)
	logrus.Info("文件下载响应已发送")
}

// DirectDownload 直接下载到本地（不保存服务器）
func (s *Service) DirectDownload(c *gin.Context) {
	logrus.Info("DirectDownload API被调用")

	type reqBody struct {
		URL    string `json:"url"`
		Format string `json:"format"`
	}
	var req reqBody
	if err := c.ShouldBindJSON(&req); err != nil || req.URL == "" {
		logrus.Errorf("参数解析失败: %v, URL为空: %v", err, req.URL == "")
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少URL参数或参数格式错误"})
		return
	}

	logrus.Infof("开始下载视频: %s, 格式: %s", req.URL, req.Format)

	// 临时文件名
	tmpDir := os.TempDir()
	filename := uuid.New().String() + ".mp4"
	filePath := filepath.Join(tmpDir, filename)

	dlReq := &downloader.DownloadRequest{
		URL:    req.URL,
		Format: req.Format,
		Output: filePath,
	}

	// 下载到临时文件
	logrus.Info("调用yt-dlp下载器")
	actualFile, err := s.ytdlp.Download(dlReq, nil)
	if err != nil {
		logrus.Errorf("下载失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "下载失败: " + err.Error()})
		return
	}

	logrus.Infof("下载完成，文件路径: %s", actualFile)

	// 设置响应头，返回文件流
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=\"video.mp4\"")
	c.File(filePath)

	// 下载完成后删除临时文件
	go func() {
		time.Sleep(10 * time.Second)
		os.Remove(filePath)
		logrus.Infof("临时文件已删除: %s", filePath)
	}()
}

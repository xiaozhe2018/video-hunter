package downloader

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// DouyinDownloader 抖音下载器
type DouyinDownloader struct {
	client    *http.Client
	tasks     map[string]*DownloadResponse
	taskMutex sync.RWMutex
}

// NewDouyinDownloader 创建新的抖音下载器实例
func NewDouyinDownloader() *DouyinDownloader {
	return &DouyinDownloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		tasks: make(map[string]*DownloadResponse),
	}
}

// Download 实现下载接口
func (d *DouyinDownloader) Download(req *DownloadRequest) (*DownloadResponse, error) {
	if req == nil {
		return nil, errors.New("下载请求不能为空")
	}

	// 验证URL是否为抖音链接
	if !d.isDouyinURL(req.URL) {
		return nil, errors.New("不支持的URL格式")
	}

	// 获取视频信息
	videoInfo, err := d.GetVideoInfo(req.URL)
	if err != nil {
		return nil, fmt.Errorf("获取视频信息失败: %w", err)
	}

	// 创建下载响应
	resp := &DownloadResponse{
		ID:       req.TaskID,
		Status:   StatusPending,
		Progress: 0,
		Created:  time.Now(),
		Updated:  time.Now(),
		Title:    videoInfo.Title,
	}

	// 保存任务信息
	d.taskMutex.Lock()
	d.tasks[req.TaskID] = resp
	d.taskMutex.Unlock()

	// 异步开始下载
	go d.startDownload(req, resp, videoInfo)

	return resp, nil
}

// GetProgress 获取下载进度
func (d *DouyinDownloader) GetProgress(id string) (*DownloadResponse, error) {
	d.taskMutex.RLock()
	defer d.taskMutex.RUnlock()

	if resp, ok := d.tasks[id]; ok {
		return resp, nil
	}
	return nil, errors.New("任务不存在")
}

// Cancel 取消下载
func (d *DouyinDownloader) Cancel(id string) error {
	d.taskMutex.Lock()
	defer d.taskMutex.Unlock()

	if resp, ok := d.tasks[id]; ok {
		resp.Status = StatusCancelled
		return nil
	}
	return errors.New("任务不存在")
}

// GetVideoInfo 获取视频信息
func (d *DouyinDownloader) GetVideoInfo(urlStr string) (*VideoInfo, error) {
	// 解析并清理URL
	cleanURL, err := d.cleanURL(urlStr)
	if err != nil {
		return nil, err
	}

	// 发送请求获取页面内容
	req, err := http.NewRequest("GET", cleanURL, nil)
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析视频信息
	return d.parseVideoInfo(string(body))
}

// isDouyinURL 判断是否为抖音链接
func (d *DouyinDownloader) isDouyinURL(urlStr string) bool {
	patterns := []string{
		`^https?://v\.douyin\.com/`,
		`^https?://www\.douyin\.com/video/`,
		`^https?://www\.douyin\.com/share/video/`,
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, urlStr); matched {
			return true
		}
	}
	return false
}

// cleanURL 清理并规范化URL
func (d *DouyinDownloader) cleanURL(urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	// 如果是短链接，进行重定向获取真实链接
	if strings.Contains(parsedURL.Host, "v.douyin.com") {
		req, err := http.NewRequest("GET", urlStr, nil)
		if err != nil {
			return "", err
		}
		resp, err := d.client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		return resp.Request.URL.String(), nil
	}

	return urlStr, nil
}

// parseVideoInfo 从页面内容解析视频信息
func (d *DouyinDownloader) parseVideoInfo(content string) (*VideoInfo, error) {
	// 提取视频标题
	titleRegex := regexp.MustCompile(`<title[^>]*>(.*?)</title>`)
	titleMatch := titleRegex.FindStringSubmatch(content)
	title := "未知标题"
	if len(titleMatch) > 1 {
		title = strings.TrimSpace(titleMatch[1])
	}

	// 提取视频数据
	videoDataRegex := regexp.MustCompile(`<script id="RENDER_DATA" type="application/json">(.*?)</script>`)
	videoDataMatch := videoDataRegex.FindStringSubmatch(content)
	if len(videoDataMatch) < 2 {
		return nil, errors.New("无法找到视频数据")
	}

	// 解码视频数据
	decodedData, err := url.QueryUnescape(videoDataMatch[1])
	if err != nil {
		return nil, err
	}

	// 解析JSON数据
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(decodedData), &data); err != nil {
		return nil, err
	}

	// 构建视频信息
	info := &VideoInfo{
		Title:       title,
		CanDownload: true,
		Formats:     make([]VideoFormat, 0),
	}

	// 添加默认格式
	info.Formats = append(info.Formats, VideoFormat{
		FormatID:   "best",
		Extension:  "mp4",
		Resolution: "最佳质量",
		Quality:    "高清",
	})

	return info, nil
}

// startDownload 开始下载过程
func (d *DouyinDownloader) startDownload(req *DownloadRequest, resp *DownloadResponse, info *VideoInfo) {
	// 更新状态为下载中
	resp.Status = StatusDownloading
	resp.Updated = time.Now()

	// TODO: 实现实际的下载逻辑
	// 1. 获取视频真实下载地址
	// 2. 使用适当的下载方法（支持断点续传）
	// 3. 更新下载进度，注意控制更新频率
	//    - 使用时间间隔控制（例如每秒最多更新一次）
	//    - 只在进度有明显变化时更新（例如变化超过1%）
	// 4. 处理下载完成或失败的情况

	// 临时：标记为失败
	resp.Status = StatusFailed
	resp.Error = "下载功能尚未实现"
	resp.Updated = time.Now()
}

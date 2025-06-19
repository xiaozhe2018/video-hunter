package downloader

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"video-hunter/internal/config"

	"github.com/sirupsen/logrus"
)

// YtdlpDownloader 使用yt-dlp的下载器实现
type YtdlpDownloader struct {
	config *config.Config
}

// Config 下载器配置
type Config struct {
	YtdlpPath  string
	OutputDir  string
	UserAgent  string
	MaxRetries int
	Timeout    int
}

// NewYtdlpDownloader 创建新的yt-dlp下载器
func NewYtdlpDownloader(config *config.Config) *YtdlpDownloader {
	if config.YtDlp.Path == "" {
		config.YtDlp.Path = "yt-dlp" // 默认使用系统 PATH 中的 yt-dlp
	}

	// 如果是相对路径，尝试在系统 PATH 中查找
	if !filepath.IsAbs(config.YtDlp.Path) && !strings.Contains(config.YtDlp.Path, string(os.PathSeparator)) {
		if fullPath, err := exec.LookPath(config.YtDlp.Path); err == nil {
			config.YtDlp.Path = fullPath
		}
	}

	return &YtdlpDownloader{
		config: config,
	}
}

// GetVideoInfo 获取视频信息
func (y *YtdlpDownloader) GetVideoInfo(url string) (*VideoInfo, error) {
	// 对抖音视频使用专用解析方法
	if strings.Contains(url, "douyin.com") || strings.Contains(url, "v.douyin.com") {
		logrus.Info("检测到抖音视频，使用专用解析方法")

		// 将抖音链接转换为标准格式
		url = y.convertDouyinUrl(url)

		// 尝试直接获取抖音视频地址
		videoURL, title, err := y.getDouyinRealUrl(url)
		if err == nil && videoURL != "" {
			// 成功获取到视频地址，构造VideoInfo
			info := &VideoInfo{
				Title:    title,
				Duration: "未知",
				Formats: []VideoFormat{
					{
						FormatID:   "best",
						Extension:  "mp4",
						Resolution: "最佳质量",
						URL:        videoURL,
					},
				},
				Metadata: map[string]string{
					"source":     "douyin",
					"direct_url": "true",
					"video_url":  videoURL,
					"parsed_by":  "direct_api",
				},
			}
			return info, nil
		}

		// 如果是短链接解析失败，返回特定的错误信息
		if strings.Contains(url, "v.douyin.com") {
			// 构造用户友好的错误信息
			info := &VideoInfo{
				Title: "抖音视频",
				Formats: []VideoFormat{
					{
						FormatID:   "best",
						Extension:  "mp4",
						Resolution: "最佳质量",
					},
				},
				SpecialNote: "抖音短链接解析失败，请尝试以下方法：",
				ErrorType:   "douyin_short_link",
				Solutions: []string{
					"使用完整的抖音视频链接（以www.douyin.com/video/开头）",
					"确保短链接格式正确且有效",
					"尝试使用抖音官方分享功能获取新的链接",
					"使用抖音APP内的复制链接功能获取链接",
				},
			}
			return info, fmt.Errorf("抖音短链接解析失败: %v", err)
		}

		// 如果直接获取失败，记录错误并继续使用yt-dlp
		logrus.Warnf("直接获取抖音视频信息失败: %v，将尝试使用yt-dlp", err)
	}

	if y.config.YtDlp.Path == "" {
		return nil, fmt.Errorf("yt-dlp 路径未配置，请检查 config.yaml 的 ytdlp.path")
	}

	// 检查文件是否存在且可执行
	if _, err := os.Stat(y.config.YtDlp.Path); err != nil {
		return nil, fmt.Errorf("yt-dlp 路径无效或不可访问: %s", y.config.YtDlp.Path)
	}

	args := []string{
		"--dump-json",
		"--no-playlist",
		"--user-agent", y.config.YtDlp.UserAgent,
		"--no-warnings",
	}

	// 为特定网站添加特殊请求头
	if strings.Contains(url, "spankbang.com") {
		args = append(args,
			"--add-header", "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"--add-header", "Accept-Language: en-US,en;q=0.5",
			"--add-header", "Accept-Encoding: gzip, deflate",
			"--add-header", "DNT: 1",
			"--add-header", "Connection: keep-alive",
			"--add-header", "Upgrade-Insecure-Requests: 1",
		)
	} else if strings.Contains(url, "bilibili.com") {
		// 为B站添加特殊请求头
		bilibiliHeaders := map[string]string{
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
			"Accept-Language":           "zh-CN,zh;q=0.9,en;q=0.8",
			"Accept-Encoding":           "gzip, deflate, br",
			"DNT":                       "1",
			"Connection":                "keep-alive",
			"Upgrade-Insecure-Requests": "1",
			"Sec-Fetch-Dest":            "document",
			"Sec-Fetch-Mode":            "navigate",
			"Sec-Fetch-Site":            "none",
		}

		// 添加B站特殊请求头
		for key, value := range bilibiliHeaders {
			args = append(args, "--add-header", fmt.Sprintf("%s: %s", key, value))
		}

		// 添加额外参数
		args = append(args, "--no-check-certificate")
	} else if strings.Contains(url, "douyin.com") || strings.Contains(url, "v.douyin.com") {
		// 将抖音链接转换为短链接
		url = y.convertDouyinUrl(url)

		// 为抖音添加特殊请求头
		douyinHeaders := map[string]string{
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
			"Accept-Language":           "zh-CN,zh;q=0.9,en;q=0.8",
			"Accept-Encoding":           "gzip, deflate, br",
			"DNT":                       "1",
			"Connection":                "keep-alive",
			"Upgrade-Insecure-Requests": "1",
			"Sec-Fetch-Dest":            "document",
			"Sec-Fetch-Mode":            "navigate",
			"Sec-Fetch-Site":            "none",
		}

		// 添加抖音特殊请求头
		for key, value := range douyinHeaders {
			args = append(args, "--add-header", fmt.Sprintf("%s: %s", key, value))
		}

		// 添加抖音特定参数，不使用--cookies-from-browser
		args = append(args, "--extractor-args", "douyin:app_version=9.9.10")
		args = append(args, "--extractor-args", "douyin:device_platform=android")

		// 使用移动端UA，这对抖音下载很重要
		mobileUA := "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36"
		args = append(args, "--add-header", "User-Agent: "+mobileUA)
		args = append(args, "--user-agent", mobileUA)

		// 添加额外参数
		args = append(args, "--no-check-certificate")
	} else if strings.Contains(url, "pinterest.com") {
		// 为Pinterest添加特殊请求头
		pinterestHeaders := map[string]string{
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
			"Accept-Language":           "en-US,en;q=0.9",
			"Accept-Encoding":           "gzip, deflate, br",
			"DNT":                       "1",
			"Connection":                "keep-alive",
			"Upgrade-Insecure-Requests": "1",
		}

		// 添加Pinterest特殊请求头
		for key, value := range pinterestHeaders {
			args = append(args, "--add-header", fmt.Sprintf("%s: %s", key, value))
		}
	}

	args = append(args, url)
	cmd := exec.Command(y.config.YtDlp.Path, args...)

	// 打印调试信息
	fmt.Printf("DEBUG: 执行命令: %s %v\n", y.config.YtDlp.Path, args)

	// 捕获错误输出
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		// 构造用户友好的错误信息
		errMsg := stderr.String()
		info := &VideoInfo{
			Title: "视频信息获取失败",
			Formats: []VideoFormat{
				{
					FormatID:   "best",
					Extension:  "mp4",
					Resolution: "最佳质量",
				},
			},
			CanDownload: false,
		}

		// 根据错误信息提供不同的解决方案
		if strings.Contains(errMsg, "douyin") || strings.Contains(url, "douyin.com") {
			info.Title = "抖音视频"
			info.ErrorType = "douyin_access"
			info.SpecialNote = "抖音视频需要登录信息，请尝试以下方法："
			info.Solutions = []string{
				"使用移动端分享的链接（以v.douyin.com开头的短链接）",
				"确保链接格式正确，例如：https://v.douyin.com/xxxxxx/",
				"尝试使用抖音官方分享功能获取新的链接",
				"使用移动端APP分享的链接",
			}
		} else if strings.Contains(errMsg, "bilibili") || strings.Contains(url, "bilibili.com") {
			info.Title = "哔哩哔哩视频"
			info.ErrorType = "bilibili_access"
			info.SpecialNote = "哔哩哔哩视频可能需要登录信息，请尝试以下方法："
			info.Solutions = []string{
				"确保链接格式正确",
				"尝试使用哔哩哔哩官方分享功能获取新的链接",
				"尝试使用不同的视频清晰度",
			}
		} else {
			info.ErrorType = "general_error"
			info.SpecialNote = "视频信息获取失败，请尝试以下方法："
			info.Solutions = []string{
				"检查网络连接",
				"确保链接格式正确",
				"尝试使用其他视频源",
				"链接可能已失效，请尝试获取新的链接",
			}
		}

		// 添加原始错误信息
		info.Metadata = map[string]string{
			"original_error": fmt.Sprintf("获取视频信息失败: %v, stderr: %s", err, errMsg),
		}

		return info, fmt.Errorf("获取视频信息失败: %w, stderr: %s", err, stderr.String())
	}

	// 检查输出是否为空
	if len(output) == 0 {
		return nil, fmt.Errorf("获取视频信息失败: 输出为空")
	}

	// 尝试解析JSON
	var videoData map[string]interface{}
	if err := json.Unmarshal(output, &videoData); err != nil {
		// 记录原始输出以便调试
		logrus.Errorf("JSON解析错误: %v, 原始输出: %s", err, string(output))
		return nil, fmt.Errorf("解析视频信息失败: %w", err)
	}

	info := &VideoInfo{
		Title:       getString(videoData, "title"),
		Duration:    getString(videoData, "duration_string"),
		Formats:     []VideoFormat{},
		Metadata:    make(map[string]string),
		CanDownload: true, // 确保设置为可下载
	}

	// 提取格式信息
	if formats, ok := videoData["formats"].([]interface{}); ok {
		for _, format := range formats {
			if formatMap, ok := format.(map[string]interface{}); ok {
				videoFormat := VideoFormat{
					FormatID:   getString(formatMap, "format_id"),
					Extension:  getString(formatMap, "ext"),
					Resolution: getString(formatMap, "resolution"),
					Filesize:   getInt64(formatMap, "filesize"),
					URL:        getString(formatMap, "url"),
					Quality:    getString(formatMap, "quality"),
				}
				info.Formats = append(info.Formats, videoFormat)
			}
		}
	}

	return info, nil
}

// Download 下载视频
func (y *YtdlpDownloader) Download(req *DownloadRequest, progressCallback func(*DownloadResponse)) (string, error) {
	// 对B站视频使用专用下载方法
	if strings.Contains(req.URL, "bilibili.com") {
		logrus.Info("检测到B站视频，使用专用下载方法")
		return y.DownloadBilibili(req, progressCallback)
	}

	// 对抖音视频使用专用下载方法
	if strings.Contains(req.URL, "douyin.com") || strings.Contains(req.URL, "v.douyin.com") {
		logrus.Info("检测到抖音视频，使用专用下载方法")
		return y.DownloadDouyin(req, progressCallback)
	}

	if y.config.YtDlp.Path == "" {
		return "", fmt.Errorf("yt-dlp 路径未配置，请检查 config.yaml 的 ytdlp.path")
	}

	// 检查文件是否存在且可执行
	if _, err := os.Stat(y.config.YtDlp.Path); err != nil {
		return "", fmt.Errorf("yt-dlp 路径无效或不可访问: %s", y.config.YtDlp.Path)
	}

	// 基本参数
	args := []string{
		"--no-playlist",
		"--user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"--no-warnings",
		"-v",        // 添加详细输出
		"--newline", // 确保每行都有换行符，便于解析
	}

	// 为SpankBang添加特殊请求头
	if strings.Contains(req.URL, "spankbang.com") {
		spankbangHeaders := map[string]string{
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"Accept-Language":           "en-US,en;q=0.5",
			"Accept-Encoding":           "gzip, deflate",
			"DNT":                       "1",
			"Connection":                "keep-alive",
			"Upgrade-Insecure-Requests": "1",
			"Sec-Fetch-Dest":            "document",
			"Sec-Fetch-Mode":            "navigate",
			"Sec-Fetch-Site":            "none",
			"Sec-Fetch-User":            "?1",
			"Cache-Control":             "max-age=0",
		}

		// 添加SpankBang特殊请求头
		for key, value := range spankbangHeaders {
			args = append(args, "--add-header", fmt.Sprintf("%s: %s", key, value))
		}

		// 添加Referer
		if req.Referer == "" {
			args = append(args, "--referer", "https://spankbang.com/")
		}
	} else if strings.Contains(req.URL, "bilibili.com") {
		// 为B站添加特殊请求头
		bilibiliHeaders := map[string]string{
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
			"Accept-Language":           "zh-CN,zh;q=0.9,en;q=0.8",
			"Accept-Encoding":           "gzip, deflate, br",
			"DNT":                       "1",
			"Connection":                "keep-alive",
			"Upgrade-Insecure-Requests": "1",
			"Sec-Fetch-Dest":            "document",
			"Sec-Fetch-Mode":            "navigate",
			"Sec-Fetch-Site":            "none",
		}

		// 添加B站特殊请求头
		for key, value := range bilibiliHeaders {
			args = append(args, "--add-header", fmt.Sprintf("%s: %s", key, value))
		}

		// 如果用户提供了cookies，使用它
		if req.Cookies != "" {
			args = append(args, "--cookies", req.Cookies)
		}
	}

	// 添加自定义请求头
	if req.Headers != nil {
		for key, value := range req.Headers {
			args = append(args, "--add-header", fmt.Sprintf("%s: %s", key, value))
		}
	}

	// 添加Cookies
	if req.Cookies != "" {
		args = append(args, "--cookies", req.Cookies)
	}

	// 添加Referer
	if req.Referer != "" {
		args = append(args, "--referer", req.Referer)
	}

	// 添加格式选择
	if req.Format != "" {
		// 对B站视频使用特殊格式处理
		if strings.Contains(req.URL, "bilibili.com") {
			// 对于B站视频，使用可用的最佳格式而不是特定格式
			// 这样可以避免请求需要会员的格式
			if req.Format == "best" {
				// 使用更通用的格式选择，确保同时下载视频和音频
				args = append(args, "-f", "bestvideo+bestaudio/best")
			} else {
				// 如果用户指定了具体格式，尝试使用它，但添加备选项
				args = append(args, "-f", req.Format+"/bestvideo+bestaudio/best")
			}

			// 确保合并视频和音频
			args = append(args, "--merge-output-format", "mp4")

			// 检查系统中是否安装了ffmpeg
			_, err := exec.LookPath("ffmpeg")
			if err == nil {
				// 只有在ffmpeg存在的情况下才添加ffmpeg相关参数
				args = append(args, "--ffmpeg-location", "ffmpeg")
			} else {
				logrus.Warnf("系统中未安装ffmpeg，视频和音频将不会被合并: %v", err)
				logrus.Warnf("请安装ffmpeg以获得带音频的视频")
			}

			// 添加额外的参数以确保正确下载
			args = append(args, "--no-check-certificate")
		} else {
			// 其他网站使用正常的格式选择
			args = append(args, "-f", req.Format)
		}
	}

	// 添加其他选项
	if req.Options != nil {
		for key, value := range req.Options {
			if value != "" {
				args = append(args, fmt.Sprintf("--%s", key), value)
			} else {
				args = append(args, fmt.Sprintf("--%s", key))
			}
		}
	}

	// 生成带有任务ID前缀的输出文件名
	outputTemplate := ""
	if req.Output != "" {
		// 如果提供了输出路径，在文件名前添加任务ID
		dir := filepath.Dir(req.Output)
		filename := filepath.Base(req.Output)
		outputTemplate = filepath.Join(dir, fmt.Sprintf("%s_%s", req.TaskID, filename))
	} else {
		// 默认输出路径
		outputTemplate = filepath.Join(y.config.Downloader.OutputDir, fmt.Sprintf("%s_%(title)s.%(ext)s", req.TaskID))
	}
	args = append(args, "-o", outputTemplate)

	// 添加URL
	args = append(args, req.URL)

	// 创建独立的命令实例
	cmd := exec.Command(y.config.YtDlp.Path, args...)
	logrus.Infof("执行命令: %s %v", y.config.YtDlp.Path, args)

	// 创建管道读取输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("创建输出管道失败: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("创建错误输出管道失败: %v", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("启动下载失败: %v", err)
	}

	// 创建一个WaitGroup来等待所有goroutine完成
	var wg sync.WaitGroup
	wg.Add(2)

	// 收集错误信息
	var stderrOutput strings.Builder
	var stdoutOutput strings.Builder
	var actualFilePath string

	// 处理标准输出
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			stdoutOutput.WriteString(line + "\n")

			// 检查是否包含文件路径信息
			if strings.Contains(line, "Destination:") {
				parts := strings.SplitN(line, "Destination:", 2)
				if len(parts) > 1 {
					actualFilePath = strings.TrimSpace(parts[1])
					logrus.Infof("检测到实际下载文件路径: %s", actualFilePath)
				}
			}

			// 解析进度信息并调用回调
			if strings.Contains(line, "%") {
				progress := parseProgress(line)
				if progressCallback != nil {
					logrus.Debugf("发送进度更新: %f%% %s", progress.Progress, progress.Speed)
					progressCallback(progress)
				}
			}

			// 记录所有输出到日志
			logrus.Debug("yt-dlp stdout:", line)
		}
	}()

	// 处理错误输出
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			stderrOutput.WriteString(line + "\n")

			// 检查错误输出中是否包含文件路径信息
			if strings.Contains(line, "[download] Destination:") {
				parts := strings.SplitN(line, "[download] Destination:", 2)
				if len(parts) > 1 {
					actualFilePath = strings.TrimSpace(parts[1])
					logrus.Infof("从stderr检测到实际下载文件路径: %s", actualFilePath)
				}
			}

			// 解析进度信息并调用回调
			if strings.Contains(line, "%") {
				progress := parseProgress(line)
				if progressCallback != nil {
					logrus.Debugf("从stderr发送进度更新: %f%% %s", progress.Progress, progress.Speed)
					progressCallback(progress)
				}
			}

			logrus.Debug("yt-dlp stderr:", line)
		}
	}()

	// 等待命令完成
	err = cmd.Wait()
	wg.Wait()

	if err != nil {
		// 如果有错误，返回详细的错误信息
		errMsg := fmt.Sprintf("下载失败: %v\n命令: %s %v\n标准输出:\n%s\n错误输出:\n%s",
			err, y.config.YtDlp.Path, args, stdoutOutput.String(), stderrOutput.String())
		logrus.Error(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	// 如果没有从输出中获取到实际文件路径，尝试查找最近创建的文件
	if actualFilePath == "" {
		logrus.Warn("未从输出中检测到实际文件路径，尝试查找最近创建的文件")
		foundPath, err := y.findActualFile(outputTemplate)
		if err != nil {
			logrus.Errorf("查找实际文件失败: %v", err)
			// 如果找不到实际文件，返回错误
			return "", fmt.Errorf("下载可能失败，无法找到下载的文件: %v", err)
		}
		actualFilePath = foundPath
		logrus.Infof("找到最近创建的文件: %s", actualFilePath)

		// 验证找到的文件是否与当前任务匹配
		if !strings.Contains(filepath.Base(actualFilePath), req.TaskID) {
			logrus.Errorf("找到的文件 %s 与当前任务ID %s 不匹配", actualFilePath, req.TaskID)
			return "", fmt.Errorf("下载失败，找到的文件与当前任务不匹配")
		}
	}

	// 确保文件存在且可访问
	if _, err := os.Stat(actualFilePath); err != nil {
		logrus.Warnf("检测到的文件路径不存在或不可访问: %s, 错误: %v", actualFilePath, err)
		// 尝试使用相对路径
		relPath := filepath.Join(y.config.Downloader.OutputDir, filepath.Base(actualFilePath))
		if _, err := os.Stat(relPath); err == nil {
			actualFilePath = relPath
			logrus.Infof("使用相对路径访问文件: %s", actualFilePath)
		} else {
			// 如果仍然找不到，返回模板路径
			logrus.Warnf("无法访问下载的文件，返回模板路径: %s", outputTemplate)
			return outputTemplate, nil
		}
	}

	// 返回实际下载的文件路径
	return actualFilePath, nil
}

// processOutputFilename 处理输出文件名
func (y *YtdlpDownloader) processOutputFilename(filename string) string {
	// 清理文件名，移除不安全的字符
	filename = y.sanitizeFilename(filename)

	// 检查是否已经包含扩展名
	if strings.Contains(filename, ".%(ext)s") {
		// 已经包含yt-dlp模板，直接返回
		return filename
	}

	// 检查是否有其他扩展名
	if strings.Contains(filename, ".") {
		// 有扩展名但不是yt-dlp模板，替换为yt-dlp模板
		parts := strings.Split(filename, ".")
		if len(parts) > 1 {
			baseName := strings.Join(parts[:len(parts)-1], ".")
			return baseName + ".%(ext)s"
		}
	}

	// 没有扩展名，添加yt-dlp模板
	return filename + ".%(ext)s"
}

// sanitizeFilename 清理文件名，移除不安全的字符
func (y *YtdlpDownloader) sanitizeFilename(filename string) string {
	// 移除或替换不安全的字符
	unsafeChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := filename

	for _, char := range unsafeChars {
		result = strings.ReplaceAll(result, char, "_")
	}

	// 移除前后空格
	result = strings.TrimSpace(result)

	// 如果文件名为空，使用默认名称
	if result == "" {
		result = "video"
	}

	return result
}

// monitorProgress 监控下载进度
func (y *YtdlpDownloader) monitorProgress(stdout, stderr interface{}, callback ProgressCallback) {
	// 解析yt-dlp的输出来获取真实进度
	scanner := bufio.NewScanner(stdout.(interface {
		Read(p []byte) (n int, err error)
	}).(interface {
		Read(p []byte) (n int, err error)
	}))

	progressRegex := regexp.MustCompile(`(\d+\.?\d*)%`)
	speedRegex := regexp.MustCompile(`(\d+\.?\d*)([KMGT]iB/s|B/s)`)
	etaRegex := regexp.MustCompile(`ETA (\d+:\d+)`)

	for scanner.Scan() {
		line := scanner.Text()

		// 解析进度
		if progressMatches := progressRegex.FindStringSubmatch(line); len(progressMatches) > 1 {
			progress, _ := strconv.ParseFloat(progressMatches[1], 64)

			// 解析速度
			speed := ""
			if speedMatches := speedRegex.FindStringSubmatch(line); len(speedMatches) > 2 {
				speed = speedMatches[1] + " " + speedMatches[2]
			}

			// 解析ETA
			eta := ""
			if etaMatches := etaRegex.FindStringSubmatch(line); len(etaMatches) > 1 {
				eta = etaMatches[1]
			}

			if callback != nil {
				callback(&DownloadResponse{
					Progress: progress,
					Speed:    speed,
					ETA:      eta,
				})
			}
		}
	}
}

// 辅助函数
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt64(data map[string]interface{}, key string) int64 {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case float64:
			return int64(v)
		case int:
			return int64(v)
		case int64:
			return v
		}
	}
	return 0
}

// findActualFile 查找实际下载的文件
func (y *YtdlpDownloader) findActualFile(outputTemplate string) (string, error) {
	// 从输出模板中提取任务ID
	templateBase := filepath.Base(outputTemplate)
	var taskID string

	// 尝试从模板中提取任务ID
	parts := strings.SplitN(templateBase, "_", 2)
	if len(parts) > 1 {
		taskID = parts[0]
	}

	// 如果没有任务ID，使用旧的方法查找最近创建的文件
	if taskID == "" {
		return y.findLatestFile()
	}

	// 扫描输出目录，查找匹配任务ID前缀的文件
	files, err := os.ReadDir(y.config.Downloader.OutputDir)
	if err != nil {
		return "", fmt.Errorf("读取输出目录失败: %w", err)
	}

	// 系统文件黑名单
	systemFiles := map[string]bool{
		".DS_Store":  true,
		"Thumbs.db":  true,
		".gitignore": true,
	}

	// 查找匹配任务ID的文件
	var matchedFiles []string
	var videoFile string
	var audioFile string
	var mergedFile string

	for _, file := range files {
		if !file.IsDir() {
			// 跳过系统文件
			if systemFiles[file.Name()] {
				continue
			}

			// 跳过隐藏文件（以 . 开头的文件）
			if strings.HasPrefix(file.Name(), ".") {
				continue
			}

			// 检查文件名是否以任务ID开头
			if strings.HasPrefix(file.Name(), taskID+"_") {
				filePath := filepath.Join(y.config.Downloader.OutputDir, file.Name())
				matchedFiles = append(matchedFiles, filePath)

				// 检查是否是视频文件
				if strings.HasSuffix(file.Name(), ".mp4") {
					// 如果是最终合并的文件（不包含格式ID），优先使用
					if !strings.Contains(file.Name(), ".f") {
						mergedFile = filePath
					} else {
						videoFile = filePath
					}
				}

				// 检查是否是音频文件
				if strings.HasSuffix(file.Name(), ".m4a") ||
					strings.HasSuffix(file.Name(), ".aac") ||
					strings.HasSuffix(file.Name(), ".mp3") {
					audioFile = filePath
				}
			}
		}
	}

	// 如果找到了合并后的文件，直接返回
	if mergedFile != "" {
		logrus.Infof("找到合并后的文件: %s", mergedFile)
		return mergedFile, nil
	}

	// 如果找到了视频和音频文件，尝试合并它们
	if videoFile != "" && audioFile != "" {
		logrus.Infof("找到分离的视频文件 %s 和音频文件 %s，尝试合并", videoFile, audioFile)

		// 检查系统中是否安装了ffmpeg
		_, err := exec.LookPath("ffmpeg")
		if err != nil {
			logrus.Warnf("系统中未安装ffmpeg，无法合并视频和音频: %v", err)
			logrus.Warnf("请安装ffmpeg以获得带音频的视频，现在将返回无声视频")
			return videoFile, nil
		}

		// 创建合并后的文件名
		mergedFilename := taskID + "_merged.mp4"
		mergedPath := filepath.Join(y.config.Downloader.OutputDir, mergedFilename)

		// 使用FFmpeg合并文件
		cmd := exec.Command("ffmpeg", "-i", videoFile, "-i", audioFile, "-c:v", "copy", "-c:a", "aac", "-strict", "experimental", "-y", mergedPath)

		// 设置标准错误输出，以便记录错误信息
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err = cmd.Run()
		if err == nil {
			logrus.Infof("成功合并视频和音频到: %s", mergedPath)
			return mergedPath, nil
		} else {
			logrus.Errorf("合并视频和音频失败: %v, 错误输出: %s", err, stderr.String())
			// 合并失败，仍然返回视频文件，至少用户可以看到画面
			logrus.Warnf("合并失败，返回视频文件: %s", videoFile)
			return videoFile, nil
		}
	}

	// 如果只找到视频文件，返回视频文件
	if videoFile != "" {
		logrus.Infof("只找到视频文件: %s", videoFile)
		return videoFile, nil
	}

	// 如果只找到音频文件，返回音频文件
	if audioFile != "" {
		logrus.Infof("只找到音频文件: %s", audioFile)
		return audioFile, nil
	}

	// 如果找到了任何匹配的文件，返回第一个
	if len(matchedFiles) > 0 {
		logrus.Infof("找到匹配任务ID %s 的文件: %s", taskID, matchedFiles[0])
		return matchedFiles[0], nil
	}

	// 如果没有找到匹配任务ID的文件，回退到查找最近创建的文件
	logrus.Warnf("未找到匹配任务ID %s 的文件，尝试查找最近创建的文件", taskID)
	return y.findLatestFile()
}

// findLatestFile 查找最近创建的文件（作为备选方案）
func (y *YtdlpDownloader) findLatestFile() (string, error) {
	// 扫描输出目录，查找最近创建的文件
	files, err := os.ReadDir(y.config.Downloader.OutputDir)
	if err != nil {
		return "", fmt.Errorf("读取输出目录失败: %w", err)
	}

	var latestFile os.FileInfo
	var latestTime time.Time

	// 系统文件黑名单
	systemFiles := map[string]bool{
		".DS_Store":  true,
		"Thumbs.db":  true,
		".gitignore": true,
	}

	// 查找最近创建的文件
	for _, file := range files {
		if !file.IsDir() {
			// 跳过系统文件
			if systemFiles[file.Name()] {
				continue
			}

			// 跳过隐藏文件（以 . 开头的文件）
			if strings.HasPrefix(file.Name(), ".") {
				continue
			}

			fileInfo, err := file.Info()
			if err != nil {
				continue
			}

			// 检查文件是否是最近创建的
			if fileInfo.ModTime().After(latestTime) {
				latestTime = fileInfo.ModTime()
				latestFile = fileInfo
			}
		}
	}

	if latestFile == nil {
		return "", fmt.Errorf("未找到下载的文件")
	}

	// 确保文件是最近5分钟内创建的
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	if latestFile.ModTime().Before(fiveMinutesAgo) {
		return "", fmt.Errorf("未找到最近5分钟内创建的文件，最新文件创建于 %s", latestFile.ModTime().Format(time.RFC3339))
	}

	// 返回相对于工作区根目录的路径
	return filepath.Join(y.config.Downloader.OutputDir, latestFile.Name()), nil
}

// DownloadBilibili 专门处理B站视频下载
func (y *YtdlpDownloader) DownloadBilibili(req *DownloadRequest, progressCallback func(*DownloadResponse)) (string, error) {
	if y.config.YtDlp.Path == "" {
		return "", fmt.Errorf("yt-dlp 路径未配置，请检查 config.yaml 的 ytdlp.path")
	}

	// 检查文件是否存在且可执行
	if _, err := os.Stat(y.config.YtDlp.Path); err != nil {
		return "", fmt.Errorf("yt-dlp 路径无效或不可访问: %s", y.config.YtDlp.Path)
	}

	// 基本参数 - 确保不包含--postprocessor-args
	args := []string{
		"--no-playlist",
		"--user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"--no-warnings",
		"-v",        // 添加详细输出
		"--newline", // 确保每行都有换行符，便于解析
	}

	// 为B站添加特殊请求头
	bilibiliHeaders := map[string]string{
		"Connection":                "keep-alive",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
		"Accept-Language":           "zh-CN,zh;q=0.9,en;q=0.8",
		"Accept-Encoding":           "gzip, deflate, br",
		"DNT":                       "1",
		"Sec-Fetch-Dest":            "document",
	}

	// 添加B站特殊请求头
	for key, value := range bilibiliHeaders {
		args = append(args, "--add-header", fmt.Sprintf("%s: %s", key, value))
	}

	// 如果用户提供了cookies，使用它
	if req.Cookies != "" {
		args = append(args, "--cookies", req.Cookies)
	}

	// 添加自定义请求头
	if req.Headers != nil {
		for key, value := range req.Headers {
			args = append(args, "--add-header", fmt.Sprintf("%s: %s", key, value))
		}
	}

	// 添加格式选择 - 确保不使用--list-formats参数
	if req.Format == "best" || req.Format == "" {
		// 使用更通用的格式选择，确保同时下载视频和音频
		args = append(args, "-f", "bestvideo+bestaudio/best")
	} else {
		// 如果用户指定了具体格式，尝试使用它，但添加备选项
		args = append(args, "-f", req.Format+"/bestvideo+bestaudio/best")
	}

	// 确保合并视频和音频
	args = append(args, "--merge-output-format", "mp4")

	// 检查系统中是否安装了ffmpeg
	_, err := exec.LookPath("ffmpeg")
	if err == nil {
		// 只有在ffmpeg存在的情况下才添加ffmpeg相关参数
		args = append(args, "--ffmpeg-location", "ffmpeg")
	} else {
		logrus.Warnf("系统中未安装ffmpeg，视频和音频将不会被合并: %v", err)
		logrus.Warnf("请安装ffmpeg以获得带音频的视频")
	}

	// 添加额外的参数以确保正确下载
	args = append(args, "--no-check-certificate")

	// 生成带有任务ID前缀的输出文件名
	outputTemplate := ""
	if req.Output != "" {
		// 如果提供了输出路径，在文件名前添加任务ID
		dir := filepath.Dir(req.Output)
		filename := filepath.Base(req.Output)
		outputTemplate = filepath.Join(dir, fmt.Sprintf("%s_%s", req.TaskID, filename))
	} else {
		// 默认输出路径
		outputTemplate = filepath.Join(y.config.Downloader.OutputDir, fmt.Sprintf("%s_%(title)s.%(ext)s", req.TaskID))
	}
	args = append(args, "-o", outputTemplate)

	// 添加URL
	args = append(args, req.URL)

	// 创建独立的命令实例
	cmd := exec.Command(y.config.YtDlp.Path, args...)
	logrus.Infof("执行命令: %s %v", y.config.YtDlp.Path, args)

	// 创建管道读取输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("创建输出管道失败: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("创建错误输出管道失败: %v", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("启动下载失败: %v", err)
	}

	// 创建一个WaitGroup来等待所有goroutine完成
	var wg sync.WaitGroup
	wg.Add(2)

	// 收集错误信息
	var stderrOutput strings.Builder
	var stdoutOutput strings.Builder
	var actualFilePath string
	var videoFile string
	var audioFile string

	// 处理标准输出
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			stdoutOutput.WriteString(line + "\n")

			// 检查是否包含文件路径信息
			if strings.Contains(line, "Destination:") {
				parts := strings.SplitN(line, "Destination:", 2)
				if len(parts) > 1 {
					filePath := strings.TrimSpace(parts[1])
					actualFilePath = filePath
					logrus.Infof("检测到实际下载文件路径: %s", filePath)

					// 判断是视频还是音频文件
					if strings.HasSuffix(filePath, ".mp4") {
						videoFile = filePath
					} else if strings.HasSuffix(filePath, ".m4a") || strings.HasSuffix(filePath, ".aac") || strings.HasSuffix(filePath, ".mp3") {
						audioFile = filePath
					}
				}
			}

			// 解析进度信息并调用回调
			if strings.Contains(line, "%") {
				progress := parseProgress(line)
				if progressCallback != nil {
					progressCallback(progress)
				}
			}

			// 记录所有输出到日志
			logrus.Debug("yt-dlp stdout:", line)
		}
	}()

	// 处理错误输出
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			stderrOutput.WriteString(line + "\n")

			// 检查错误输出中是否包含文件路径信息
			if strings.Contains(line, "[download] Destination:") {
				parts := strings.SplitN(line, "[download] Destination:", 2)
				if len(parts) > 1 {
					filePath := strings.TrimSpace(parts[1])
					actualFilePath = filePath
					logrus.Infof("从stderr检测到实际下载文件路径: %s", filePath)

					// 判断是视频还是音频文件
					if strings.HasSuffix(filePath, ".mp4") {
						videoFile = filePath
					} else if strings.HasSuffix(filePath, ".m4a") || strings.HasSuffix(filePath, ".aac") || strings.HasSuffix(filePath, ".mp3") {
						audioFile = filePath
					}
				}
			}

			// 解析进度信息并调用回调
			if strings.Contains(line, "%") {
				progress := parseProgress(line)
				if progressCallback != nil {
					progressCallback(progress)
				}
			}

			logrus.Debug("yt-dlp stderr:", line)
		}
	}()

	// 等待命令完成
	err = cmd.Wait()
	wg.Wait()

	if err != nil {
		// 如果有错误，但是已经下载了文件，尝试手动合并
		if videoFile != "" && audioFile != "" {
			logrus.Warnf("yt-dlp合并失败，但已下载视频和音频文件，尝试手动合并")
			mergedFile := filepath.Join(y.config.Downloader.OutputDir, req.TaskID+"_merged.mp4")
			if mergedPath, mergeErr := y.mergeVideoAndAudio(videoFile, audioFile, mergedFile); mergeErr == nil {
				logrus.Infof("手动合并成功: %s", mergedPath)
				return mergedPath, nil
			} else {
				logrus.Errorf("手动合并失败: %v", mergeErr)
				// 合并失败，返回视频文件（没有声音）
				return videoFile, nil
			}
		}

		// 如果是cookies错误，尝试提供更明确的错误信息
		if strings.Contains(stderrOutput.String(), "cookies") || strings.Contains(stderrOutput.String(), "Fresh cookies") {
			return "", fmt.Errorf("下载抖音视频需要登录信息。请尝试通过浏览器下载或使用移动端分享的链接。错误信息: %v", err)
		}

		// 如果没有下载任何文件，返回错误
		errMsg := fmt.Sprintf("下载失败: %v\n命令: %s %v\n标准输出:\n%s\n错误输出:\n%s",
			err, y.config.YtDlp.Path, args, stdoutOutput.String(), stderrOutput.String())
		logrus.Error(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	// 如果没有从输出中获取到实际文件路径，尝试查找最近创建的文件
	if actualFilePath == "" {
		logrus.Warn("未从输出中检测到实际文件路径，尝试查找最近创建的文件")
		foundPath, err := y.findActualFile(outputTemplate)
		if err != nil {
			logrus.Errorf("查找实际文件失败: %v", err)
			// 如果找不到实际文件，返回错误
			return "", fmt.Errorf("下载可能失败，无法找到下载的文件: %v", err)
		}
		actualFilePath = foundPath
		logrus.Infof("找到最近创建的文件: %s", actualFilePath)

		// 验证找到的文件是否与当前任务匹配
		if !strings.Contains(filepath.Base(actualFilePath), req.TaskID) {
			logrus.Errorf("找到的文件 %s 与当前任务ID %s 不匹配", actualFilePath, req.TaskID)
			return "", fmt.Errorf("下载失败，找到的文件与当前任务不匹配")
		}
	}

	// 检查是否需要手动合并视频和音频
	if videoFile != "" && audioFile != "" && !strings.Contains(actualFilePath, "_merged") {
		logrus.Info("检测到分离的视频和音频文件，尝试手动合并")
		mergedFile := filepath.Join(y.config.Downloader.OutputDir, req.TaskID+"_merged.mp4")
		if mergedPath, mergeErr := y.mergeVideoAndAudio(videoFile, audioFile, mergedFile); mergeErr == nil {
			logrus.Infof("手动合并成功: %s", mergedPath)
			return mergedPath, nil
		} else {
			logrus.Errorf("手动合并失败: %v", mergeErr)
			// 合并失败，返回视频文件（没有声音）
			return videoFile, nil
		}
	}

	return actualFilePath, nil
}

// extractSpeed 从下载进度行中提取速度信息
func extractSpeed(line string) string {
	speedRegex := regexp.MustCompile(`(\d+\.?\d*[KMG]?i?B/s)`)
	matches := speedRegex.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractETA 从下载进度行中提取剩余时间信息
func extractETA(line string) string {
	etaRegex := regexp.MustCompile(`ETA:?\s*(\d+:\d+)`)
	matches := etaRegex.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// convertDouyinUrl 处理抖音链接
func (y *YtdlpDownloader) convertDouyinUrl(url string) string {
	// 如果是短链接，直接返回
	if strings.Contains(url, "v.douyin.com") {
		return url
	}

	// 如果是标准链接，检查格式并修正
	if strings.Contains(url, "douyin.com/video/") {
		// 提取视频ID
		re := regexp.MustCompile(`douyin\.com/video/(\d+)`)
		matches := re.FindStringSubmatch(url)
		if len(matches) > 1 {
			videoID := matches[1]
			// 构建标准链接格式
			return "https://www.douyin.com/video/" + videoID
		}
	}

	// 对于其他格式，保持不变
	return url
}

// isValidURL 检查URL是否有效
func (y *YtdlpDownloader) isValidURL(urlStr string) bool {
	// 创建HTTP客户端，只检查头信息
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 不跟随重定向
		},
	}

	// 创建HEAD请求
	req, err := http.NewRequest("HEAD", urlStr, nil)
	if err != nil {
		logrus.Debugf("创建HEAD请求失败: %v", err)
		return false
	}

	// 添加请求头
	req.Header.Set("User-Agent", y.config.Douyin.MobileUA)

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		logrus.Debugf("发送HEAD请求失败: %v", err)
		return false
	}
	defer resp.Body.Close()

	// 检查状态码
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// getDouyinVideoByOfficialAPI 使用官方API获取抖音视频
func (y *YtdlpDownloader) getDouyinVideoByOfficialAPI(videoID string) (string, string, error) {
	// 构造API请求URL
	apiURL := fmt.Sprintf("https://www.iesdouyin.com/web/api/v2/aweme/iteminfo/?item_ids=%s", videoID)

	// 设置请求头，模拟移动端浏览器
	client := &http.Client{
		Timeout: time.Duration(y.config.Douyin.APITimeout) * time.Second,
	}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置移动端User-Agent
	req.Header.Set("User-Agent", y.config.Douyin.MobileUA)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://www.douyin.com/")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("请求API失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析JSON响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("解析JSON失败: %w", err)
	}

	// 检查API返回的状态码
	if statusCode, ok := result["status_code"].(float64); ok && statusCode != 0 {
		return "", "", fmt.Errorf("API返回错误状态码: %v", statusCode)
	}

	// 提取视频标题
	title := "抖音视频"
	if itemList, ok := result["item_list"].([]interface{}); ok && len(itemList) > 0 {
		if item, ok := itemList[0].(map[string]interface{}); ok {
			if desc, ok := item["desc"].(string); ok && desc != "" {
				title = desc
			}
		}
	} else {
		return "", "", fmt.Errorf("API返回数据中没有视频信息")
	}

	// 尝试提取无水印视频地址
	videoURL := ""

	if itemList, ok := result["item_list"].([]interface{}); ok && len(itemList) > 0 {
		if item, ok := itemList[0].(map[string]interface{}); ok {
			// 尝试获取视频信息
			if video, ok := item["video"].(map[string]interface{}); ok {
				// 尝试获取无水印播放地址
				if playAddr, ok := video["play_addr"].(map[string]interface{}); ok {
					if urlList, ok := playAddr["url_list"].([]interface{}); ok && len(urlList) > 0 {
						for _, u := range urlList {
							if url, ok := u.(string); ok {
								// 替换域名，尝试获取无水印版本
								noWatermarkURL := strings.Replace(url, "playwm", "play", 1)
								// 验证URL是否有效
								if y.isValidURL(noWatermarkURL) {
									videoURL = noWatermarkURL
									break
								}
							}
						}
					}
				}
			}
		}
	}

	if videoURL == "" {
		return "", "", fmt.Errorf("无法从API响应中提取视频地址")
	}

	logrus.Infof("获取到抖音视频地址(官方API): %s", videoURL)

	// 返回视频地址和标题
	return videoURL, title, nil
}

// getDouyinVideoByMobileAPI 使用移动端API获取抖音视频
func (y *YtdlpDownloader) getDouyinVideoByMobileAPI(videoID string) (string, string, error) {
	// 构造移动端API请求URL
	apiURL := fmt.Sprintf("https://aweme.snssdk.com/aweme/v1/aweme/detail/?aweme_id=%s", videoID)

	// 设置请求头，模拟移动端APP
	client := &http.Client{
		Timeout: time.Duration(y.config.Douyin.APITimeout) * time.Second,
	}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置移动端APP的User-Agent和请求头
	req.Header.Set("User-Agent", "com.ss.android.ugc.aweme/800 (Linux; U; Android 10; zh_CN; Pixel 4; Build/QQ3A.200805.001; Cronet/58.0.2991.0)")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("X-Khronos", fmt.Sprintf("%d", time.Now().Unix()))
	req.Header.Set("X-Gorgon", "8404e4a20000"+fmt.Sprintf("%08x", time.Now().UnixNano()%0x100000000))

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("请求移动端API失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析JSON响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("解析JSON失败: %w", err)
	}

	// 提取视频标题和地址
	title := "抖音视频"
	videoURL := ""

	if awemeDetail, ok := result["aweme_detail"].(map[string]interface{}); ok {
		// 提取标题
		if desc, ok := awemeDetail["desc"].(string); ok && desc != "" {
			title = desc
		}

		// 提取视频地址
		if video, ok := awemeDetail["video"].(map[string]interface{}); ok {
			if playAddr, ok := video["play_addr"].(map[string]interface{}); ok {
				if urlList, ok := playAddr["url_list"].([]interface{}); ok && len(urlList) > 0 {
					for _, u := range urlList {
						if url, ok := u.(string); ok {
							// 验证URL是否有效
							if y.isValidURL(url) {
								videoURL = url
								break
							}
						}
					}
				}
			}
		}
	}

	if videoURL == "" {
		return "", "", fmt.Errorf("无法从移动端API响应中提取视频地址")
	}

	logrus.Infof("获取到抖音视频地址(移动端API): %s", videoURL)

	// 返回视频地址和标题
	return videoURL, title, nil
}

// getDouyinVideoByThirdPartyAPI 使用第三方解析服务获取抖音视频
func (y *YtdlpDownloader) getDouyinVideoByThirdPartyAPI(url string) (string, string, error) {
	// 这里可以添加多个第三方解析服务，如果一个失败可以尝试另一个
	services := []string{
		"https://api.douyin.wtf/api?url=",
		"https://api.scxs.cn/?url=",
		"https://api.oick.cn/douyin/api.php?url=",
		"https://api.3jx.top/api/douyin.php?url=",
		"https://tenapi.cn/v2/video?url=",
		"https://api.douyin.qlike.cn/api.php?url=",
		"https://api.lyfzn.com/douyin/api/?url=",
		"https://api.douyin.city/api?url=",
		"https://api.kit9.cn/api/douyin_parse/api.php?url=",
		"https://api.tikhub.io/douyin/?url=",
	}

	var lastError error
	for _, service := range services {
		videoURL, title, err := y.callThirdPartyAPI(service, url)
		if err == nil && videoURL != "" {
			return videoURL, title, nil
		}
		logrus.Debugf("第三方API %s 解析失败: %v", service, err)
		lastError = err
	}

	return "", "", fmt.Errorf("所有第三方解析服务均失败: %v", lastError)
}

// callThirdPartyAPI 调用第三方解析API
func (y *YtdlpDownloader) callThirdPartyAPI(serviceURL, videoURL string) (string, string, error) {
	// 构造API请求URL
	apiURL := serviceURL + url.QueryEscape(videoURL)

	logrus.Debugf("请求第三方API: %s", apiURL)

	// 设置请求头
	client := &http.Client{
		Timeout: time.Duration(y.config.Douyin.APITimeout) * time.Second,
	}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", y.config.Douyin.MobileUA)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://www.douyin.com/")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("请求第三方API失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 记录响应内容，用于调试
	logrus.Debugf("第三方API响应: %s", string(body))

	// 检查响应是否为空
	if len(body) == 0 {
		return "", "", fmt.Errorf("第三方API返回空响应")
	}

	// 解析JSON响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// 尝试直接从响应中提取URL（有些API可能直接返回URL文本）
		if strings.HasPrefix(strings.TrimSpace(string(body)), "http") {
			return strings.TrimSpace(string(body)), "抖音视频", nil
		}
		return "", "", fmt.Errorf("解析JSON失败: %w, 响应: %s", err, string(body))
	}

	// 尝试提取视频URL和标题
	// 注意：不同的第三方API返回格式可能不同，这里需要根据实际情况调整
	videoURL = ""
	title := "抖音视频"

	// 检查API是否返回成功状态
	if code, ok := result["code"].(float64); ok && code != 200 && code != 0 {
		if msg, ok := result["msg"].(string); ok {
			return "", "", fmt.Errorf("API返回错误: %s", msg)
		}
		return "", "", fmt.Errorf("API返回错误代码: %v", code)
	}

	// 尝试提取视频URL - 处理常见的API响应格式
	// 1. data.url 格式
	if data, ok := result["data"].(map[string]interface{}); ok {
		// 提取视频URL
		if v, ok := data["url"].(string); ok && v != "" {
			videoURL = v
		} else if v, ok := data["video_url"].(string); ok && v != "" {
			videoURL = v
		} else if v, ok := data["play_url"].(string); ok && v != "" {
			videoURL = v
		} else if v, ok := data["video"].(string); ok && v != "" {
			videoURL = v
		} else if v, ok := data["nwm_video_url"].(string); ok && v != "" {
			videoURL = v
		} else if v, ok := data["mp4"].(string); ok && v != "" {
			videoURL = v
		}

		// 提取标题
		if t, ok := data["title"].(string); ok && t != "" {
			title = t
		} else if t, ok := data["desc"].(string); ok && t != "" {
			title = t
		} else if t, ok := data["text"].(string); ok && t != "" {
			title = t
		}
	}

	// 2. 根级别格式
	if videoURL == "" {
		if v, ok := result["url"].(string); ok && v != "" {
			videoURL = v
		} else if v, ok := result["video_url"].(string); ok && v != "" {
			videoURL = v
		} else if v, ok := result["play_url"].(string); ok && v != "" {
			videoURL = v
		} else if v, ok := result["video"].(string); ok && v != "" {
			videoURL = v
		} else if v, ok := result["nwm_video_url"].(string); ok && v != "" {
			videoURL = v
		} else if v, ok := result["mp4"].(string); ok && v != "" {
			videoURL = v
		}

		if t, ok := result["title"].(string); ok && t != "" {
			title = t
		} else if t, ok := result["desc"].(string); ok && t != "" {
			title = t
		} else if t, ok := result["text"].(string); ok && t != "" {
			title = t
		}
	}

	// 3. 处理特殊格式
	if videoURL == "" {
		// 遍历所有字段，尝试找到URL
		for _, value := range result {
			if strValue, ok := value.(string); ok {
				if strings.HasPrefix(strValue, "http") && (strings.Contains(strValue, ".mp4") || strings.Contains(strValue, "video")) {
					videoURL = strValue
					break
				}
			}
		}
	}

	if videoURL == "" {
		return "", "", fmt.Errorf("无法从第三方API响应中提取视频地址")
	}

	logrus.Infof("获取到抖音视频地址(第三方API): %s", videoURL)

	// 返回视频地址和标题
	return videoURL, title, nil
}

// extractVideoIDFromHTML 从HTML中提取视频ID
func (y *YtdlpDownloader) extractVideoIDFromHTML(html string) string {
	// 尝试多种正则表达式模式
	patterns := []string{
		`"aweme_id"\s*:\s*"(\d+)"`,
		`"itemId"\s*:\s*"(\d+)"`,
		`"video_id"\s*:\s*"(\d+)"`,
		`\/video\/(\d+)`,
		`"id"\s*:\s*"(\d+)"`,
		`"awemeId"\s*:\s*"(\d+)"`,
		`"aweme":\{"id":"(\d+)"`,
		`data-id="(\d+)"`,
		`videoId\s*:\s*['"](\d+)['"]`,
		`\/share\/video\/(\d+)`,
		`"id":\s*(\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// getDouyinVideoByDirectCurl 使用curl命令获取抖音视频的真实地址
func (y *YtdlpDownloader) getDouyinVideoByDirectCurl(url string) (string, string, error) {
	// 生成临时文件名
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("douyin_%d.json", time.Now().UnixNano()))

	// 构造curl命令
	args := []string{
		"-s", "-L",
		"-A", y.config.Douyin.MobileUA,
		"-H", "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
		"-H", "Accept-Language: zh-CN,zh;q=0.9,en;q=0.8",
		"-H", "Connection: keep-alive",
		"-H", "Upgrade-Insecure-Requests: 1",
		"-H", "Sec-Fetch-Mode: navigate",
		"-H", "Sec-Fetch-Site: none",
		"-H", "Sec-Fetch-Dest: document",
		"-H", "DNT: 1",
		"-H", "Referer: https://www.douyin.com/",
		"--max-time", fmt.Sprintf("%d", y.config.Douyin.APITimeout),
		"-o", tempFile,
		url,
	}

	// 执行curl命令
	cmd := exec.Command("curl", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	logrus.Debugf("执行curl命令: curl %v", args)

	if err := cmd.Run(); err != nil {
		// 如果失败，尝试使用不同的User-Agent重试
		logrus.Warnf("curl命令执行失败，尝试使用不同的User-Agent重试: %v, stderr: %s", err, stderr.String())

		// 使用不同的User-Agent重试
		alternativeUA := "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1"
		args[2] = alternativeUA

		cmd = exec.Command("curl", args...)
		stderr.Reset()
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return "", "", fmt.Errorf("curl命令执行失败: %w, stderr: %s", err, stderr.String())
		}
	}

	// 读取临时文件
	defer os.Remove(tempFile)
	content, err := os.ReadFile(tempFile)
	if err != nil {
		return "", "", fmt.Errorf("读取临时文件失败: %w", err)
	}

	// 尝试从HTML中提取视频信息
	html := string(content)

	// 1. 尝试提取视频ID
	videoID := y.extractVideoIDFromHTML(html)
	if videoID != "" {
		logrus.Infof("从HTML中提取到视频ID: %s", videoID)
		// 使用视频ID尝试获取视频信息
		return y.getDouyinVideoByOfficialAPI(videoID)
	}

	// 2. 直接尝试提取视频URL
	videoURL := y.extractVideoURLFromHTML(html)
	if videoURL != "" {
		logrus.Infof("从HTML中直接提取到视频URL: %s", videoURL)
		// 提取标题
		title := y.extractTitleFromHTML(html)
		if title == "" {
			title = "抖音视频"
		}
		return videoURL, title, nil
	}

	return "", "", fmt.Errorf("无法从HTML中提取视频信息")
}

// extractVideoURLFromHTML 从HTML中提取视频URL
func (y *YtdlpDownloader) extractVideoURLFromHTML(html string) string {
	// 尝试多种正则表达式模式
	patterns := []string{
		`"playAddr":\s*\[\s*"([^"]+)"`,
		`"play_addr":\s*\{\s*"url_list":\s*\[\s*"([^"]+)"`,
		`"url":"([^"]+\.mp4[^"]*)"`,
		`src="([^"]+\.mp4[^"]*)"`,
		`"downloadAddr":\s*\[\s*"([^"]+)"`,
		`"download_addr":\s*\{\s*"url_list":\s*\[\s*"([^"]+)"`,
		`"video":\s*\{\s*"play_addr":\s*\{\s*"url_list":\s*\[\s*"([^"]+)"`,
		`"playApi":\s*"([^"]+)"`,
		`"videoUrl":\s*"([^"]+)"`,
		`"video_url":\s*"([^"]+)"`,
		`"playUrl":\s*"([^"]+)"`,
		`"play_url":\s*"([^"]+)"`,
		`<video[^>]+src="([^"]+)"`,
		`<source[^>]+src="([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			// 解码URL
			decodedURL, err := url.QueryUnescape(matches[1])
			if err == nil {
				return decodedURL
			}
			return matches[1]
		}
	}

	return ""
}

// extractTitleFromHTML 从HTML中提取视频标题
func (y *YtdlpDownloader) extractTitleFromHTML(html string) string {
	// 尝试多种正则表达式模式
	patterns := []string{
		`<title>([^<]+)</title>`,
		`"desc":"([^"]+)"`,
		`"title":"([^"]+)"`,
		`"description":"([^"]+)"`,
		`<meta\s+name="description"\s+content="([^"]+)"`,
		`<meta\s+property="og:title"\s+content="([^"]+)"`,
		`<meta\s+property="og:description"\s+content="([^"]+)"`,
		`<h1[^>]*>([^<]+)</h1>`,
		`"text":"([^"]+)"`,
		`"content":"([^"]+)"`,
		`"videoTitle":"([^"]+)"`,
		`"video_title":"([^"]+)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			// 清理标题，去除多余的空白字符
			title := strings.TrimSpace(matches[1])
			// 如果标题太长，截断它
			if len(title) > 100 {
				title = title[:97] + "..."
			}
			return title
		}
	}

	return ""
}

// getDouyinVideoByWebAPI 使用抖音Web API获取视频信息
func (y *YtdlpDownloader) getDouyinVideoByWebAPI(videoID string) (string, string, error) {
	// 构造API请求URL
	apiURL := fmt.Sprintf("https://www.douyin.com/aweme/v1/web/aweme/detail/?aweme_id=%s&aid=1128&version_name=23.5.0&device_platform=web", videoID)

	// 设置请求头，模拟Web浏览器
	client := &http.Client{
		Timeout: time.Duration(y.config.Douyin.APITimeout) * time.Second,
	}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置Web浏览器的User-Agent和请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://www.douyin.com/video/"+videoID)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Origin", "https://www.douyin.com")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("请求Web API失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("读取响应失败: %w", err)
	}

	// 解析JSON响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", fmt.Errorf("解析JSON失败: %w", err)
	}

	// 检查API返回的状态码
	if statusCode, ok := result["status_code"].(float64); ok && statusCode != 0 {
		return "", "", fmt.Errorf("Web API返回错误状态码: %v", statusCode)
	}

	// 提取视频标题
	title := "抖音视频"
	if awemeDetail, ok := result["aweme_detail"].(map[string]interface{}); ok {
		if desc, ok := awemeDetail["desc"].(string); ok && desc != "" {
			title = desc
		}
	} else {
		return "", "", fmt.Errorf("Web API返回数据中没有视频信息")
	}

	// 尝试提取无水印视频地址
	videoURL := ""
	if awemeDetail, ok := result["aweme_detail"].(map[string]interface{}); ok {
		if video, ok := awemeDetail["video"].(map[string]interface{}); ok {
			// 尝试获取无水印播放地址
			if playAddr, ok := video["play_addr"].(map[string]interface{}); ok {
				if urlList, ok := playAddr["url_list"].([]interface{}); ok && len(urlList) > 0 {
					for _, u := range urlList {
						if url, ok := u.(string); ok {
							// 替换域名，尝试获取无水印版本
							noWatermarkURL := strings.Replace(url, "playwm", "play", 1)
							// 验证URL是否有效
							if y.isValidURL(noWatermarkURL) {
								videoURL = noWatermarkURL
								break
							}
						}
					}
				}
			}
		}
	}

	if videoURL == "" {
		return "", "", fmt.Errorf("无法从Web API响应中提取视频地址")
	}

	logrus.Infof("获取到抖音视频地址(Web API): %s", videoURL)

	// 返回视频地址和标题
	return videoURL, title, nil
}

// mergeVideoAndAudio 使用ffmpeg手动合并视频和音频文件
func (y *YtdlpDownloader) mergeVideoAndAudio(videoFile, audioFile, outputFile string) (string, error) {
	// 检查文件是否存在
	if _, err := os.Stat(videoFile); err != nil {
		return "", fmt.Errorf("视频文件不存在: %v", err)
	}
	if _, err := os.Stat(audioFile); err != nil {
		return "", fmt.Errorf("音频文件不存在: %v", err)
	}

	// 检查系统中是否安装了ffmpeg
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		logrus.Warnf("系统中未安装ffmpeg，无法合并视频和音频: %v", err)
		logrus.Warnf("请安装ffmpeg以获得带音频的视频，现在将返回无声视频")
		return videoFile, nil // 返回视频文件而不是错误
	}

	// 使用ffmpeg合并
	cmd := exec.Command("ffmpeg", "-i", videoFile, "-i", audioFile, "-c:v", "copy", "-c:a", "aac", "-strict", "experimental", outputFile)
	logrus.Infof("执行ffmpeg合并命令: %v", cmd.Args)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg合并失败: %v\n错误输出: %s", err, stderr.String())
	}

	// 检查输出文件是否存在
	if _, err := os.Stat(outputFile); err != nil {
		return "", fmt.Errorf("合并后的文件不存在: %v", err)
	}

	return outputFile, nil
}

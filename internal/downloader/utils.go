package downloader

import (
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// getString 从map中获取字符串
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getInt64 从map中获取int64
func getInt64(data map[string]interface{}, key string) int64 {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case int:
			return int64(v)
		case int64:
			return v
		case float64:
			return int64(v)
		case string:
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				return i
			}
		}
	}
	return 0
}

// extractSpeed 从输出行中提取下载速度
func extractSpeed(line string) string {
	speedRegex := regexp.MustCompile(`(\d+\.?\d*\s*[KMG]?i?B/s)`)
	if matches := speedRegex.FindStringSubmatch(line); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractETA 从输出行中提取预计剩余时间
func extractETA(line string) string {
	etaRegex := regexp.MustCompile(`ETA\s+(\d+:\d+)`)
	if matches := etaRegex.FindStringSubmatch(line); len(matches) > 1 {
		return matches[1]
	}
	return ""
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
		return false
	}

	// 添加请求头
	req.Header.Set("User-Agent", y.config.Douyin.MobileUA)

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 检查状态码
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// sanitizeFilename 清理文件名，移除不安全字符
func (y *YtdlpDownloader) sanitizeFilename(filename string) string {
	// 替换Windows和Unix系统中不允许的文件名字符
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\n", "\r"}
	result := filename

	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}

	// 限制长度
	if len(result) > 200 {
		result = result[:197] + "..."
	}

	return result
}

// processOutputFilename 处理输出文件名
func (y *YtdlpDownloader) processOutputFilename(filename string) string {
	// 如果文件名为空，返回空字符串
	if filename == "" {
		return ""
	}

	// 清理文件名
	filename = y.sanitizeFilename(filename)

	// 检查是否有扩展名
	if !strings.Contains(filename, ".") {
		// 如果没有扩展名，添加通配符扩展名
		filename = filename + ".%(ext)s"
	} else {
		// 如果有扩展名，替换为通配符扩展名
		ext := filepath.Ext(filename)
		filename = strings.TrimSuffix(filename, ext) + ".%(ext)s"
	}

	return filename
}

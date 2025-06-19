package downloader

import (
	"regexp"
	"strconv"
	"time"
)

// DownloadRequest 下载请求
type DownloadRequest struct {
	URL         string            `json:"url"`
	Output      string            `json:"output,omitempty"`
	Format      string            `json:"format,omitempty"`
	Threads     int               `json:"threads,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	SaveToLocal bool              `json:"save_to_local,omitempty"` // 是否保存到用户本地
	Headers     map[string]string `json:"headers,omitempty"`
	Cookies     string            `json:"cookies,omitempty"`
	Referer     string            `json:"referer,omitempty"`
	Options     map[string]string `json:"options,omitempty"`
	TaskID      string            `json:"task_id,omitempty"` // 任务ID
}

// DownloadResponse 下载响应
type DownloadResponse struct {
	ID       string            `json:"id"`
	Status   DownloadStatus    `json:"status"`
	Progress float64           `json:"progress"`
	Speed    string            `json:"speed,omitempty"`
	ETA      string            `json:"eta,omitempty"`
	File     string            `json:"file,omitempty"`
	Size     int64             `json:"size,omitempty"`
	Error    string            `json:"error,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Created  time.Time         `json:"created"`
	Updated  time.Time         `json:"updated"`
	Title    string            `json:"title,omitempty"`
}

// DownloadStatus 下载状态
type DownloadStatus string

const (
	StatusPending     DownloadStatus = "pending"
	StatusDownloading DownloadStatus = "downloading"
	StatusCompleted   DownloadStatus = "completed"
	StatusFailed      DownloadStatus = "failed"
	StatusCancelled   DownloadStatus = "cancelled"
)

// VideoInfo 视频信息
type VideoInfo struct {
	Title       string            `json:"title"`
	Duration    string            `json:"duration,omitempty"`
	Formats     []VideoFormat     `json:"formats"`
	Thumbnail   string            `json:"thumbnail,omitempty"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	SpecialNote string            `json:"special_note,omitempty"` // 特殊提示信息
	ErrorType   string            `json:"error_type,omitempty"`   // 错误类型
	Solutions   []string          `json:"solutions,omitempty"`    // 解决方案建议
	CanDownload bool              `json:"can_download"`           // 是否可以下载
}

// VideoFormat 视频格式
type VideoFormat struct {
	FormatID   string `json:"format_id"`
	Extension  string `json:"extension"`
	Resolution string `json:"resolution,omitempty"`
	Filesize   int64  `json:"filesize,omitempty"`
	URL        string `json:"url,omitempty"`
	Quality    string `json:"quality"`
}

// Downloader 下载器接口
type Downloader interface {
	Download(req *DownloadRequest) (*DownloadResponse, error)
	GetProgress(id string) (*DownloadResponse, error)
	Cancel(id string) error
	GetVideoInfo(url string) (*VideoInfo, error)
}

// ProgressCallback 进度回调函数
type ProgressCallback func(progress *DownloadResponse)

// parseProgress 解析进度信息
func parseProgress(line string) *DownloadResponse {
	resp := &DownloadResponse{}

	// 解析进度百分比
	progressRegex := regexp.MustCompile(`(\d+\.?\d*)%`)
	if matches := progressRegex.FindStringSubmatch(line); len(matches) > 1 {
		if pct, err := strconv.ParseFloat(matches[1], 64); err == nil {
			resp.Progress = pct
		}
	}

	// 解析下载速度
	speedRegex := regexp.MustCompile(`(\d+\.?\d*\s*[KMG]?i?B/s)`)
	if matches := speedRegex.FindStringSubmatch(line); len(matches) > 1 {
		resp.Speed = matches[1]
	}

	// 解析预计剩余时间
	etaRegex := regexp.MustCompile(`ETA\s+(\d+:\d+)`)
	if matches := etaRegex.FindStringSubmatch(line); len(matches) > 1 {
		resp.ETA = matches[1]
	}

	// 解析文件大小
	sizeRegex := regexp.MustCompile(`(\d+\.?\d*\s*[KMG]?i?B)\s+of\s+(\d+\.?\d*\s*[KMG]?i?B)`)
	if matches := sizeRegex.FindStringSubmatch(line); len(matches) > 2 {
		// 可以添加下载大小信息到元数据
		if resp.Metadata == nil {
			resp.Metadata = make(map[string]string)
		}
		resp.Metadata["downloaded"] = matches[1]
		resp.Metadata["total_size"] = matches[2]
	}

	return resp
}

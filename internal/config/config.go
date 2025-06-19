package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config 应用配置结构体
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Log        LogConfig        `mapstructure:"log"`
	Downloader DownloaderConfig `mapstructure:"downloader"`
	YtDlp      YtDlpConfig      `mapstructure:"ytdlp"`
	Aria2      Aria2Config      `mapstructure:"aria2"`
	Douyin     DouyinConfig     `mapstructure:"douyin"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Security   SecurityConfig   `mapstructure:"security"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
	Mode string `mapstructure:"mode"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
}

// DownloaderConfig 下载器配置
type DownloaderConfig struct {
	OutputDir     string `mapstructure:"output_dir"`
	MaxRetries    int    `mapstructure:"max_retries"`
	Timeout       int    `mapstructure:"timeout"`
	MaxConcurrent int    `mapstructure:"max_concurrent"`
}

// YtDlpConfig yt-dlp 配置
type YtDlpConfig struct {
	Path         string `mapstructure:"path"`
	UserAgent    string `mapstructure:"user_agent"`
	CookiesFile  string `mapstructure:"cookies_file"`
	Proxy        string `mapstructure:"proxy"`
	Format       string `mapstructure:"format"`
	ExtractAudio bool   `mapstructure:"extract_audio"`
	AudioFormat  string `mapstructure:"audio_format"`
	AudioQuality string `mapstructure:"audio_quality"`
}

// Aria2Config aria2c 配置
type Aria2Config struct {
	Path           string `mapstructure:"path"`
	MaxConnections int    `mapstructure:"max_connections"`
	MinSplitSize   int    `mapstructure:"min_split_size"`
	Continue       bool   `mapstructure:"continue"`
}

// DouyinConfig 抖音下载配置
type DouyinConfig struct {
	EnableDirectAPI bool   `mapstructure:"enable_direct_api"`
	UseMobileUA     bool   `mapstructure:"use_mobile_ua"`
	MobileUA        string `mapstructure:"mobile_ua"`
	APITimeout      int    `mapstructure:"api_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	CorsOrigins       []string      `mapstructure:"cors_origins"`
	RateLimit         int           `mapstructure:"rate_limit"`
	RateLimitWindow   string        `mapstructure:"rate_limit_window"`
	RateLimitDuration time.Duration `mapstructure:"-"`
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	var config Config

	// 设置默认值
	setDefaults()

	// 设置配置文件路径
	viper.SetConfigFile(configPath)

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// 配置文件不存在，创建默认配置文件
			logrus.Warn("配置文件不存在，将创建默认配置文件")
			if err := createDefaultConfig(configPath); err != nil {
				return nil, fmt.Errorf("创建默认配置文件失败: %w", err)
			}
			// 重新读取配置文件
			if err := viper.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("读取配置文件失败: %w", err)
			}
		} else {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	// 解析配置文件
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 处理特殊配置
	if err := processConfig(&config); err != nil {
		return nil, fmt.Errorf("处理配置失败: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认值
func setDefaults() {
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.mode", "debug")

	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.file", "logs/video-hunter.log")
	viper.SetDefault("log.max_size", 100)
	viper.SetDefault("log.max_age", 30)
	viper.SetDefault("log.max_backups", 10)

	viper.SetDefault("downloader.output_dir", "./downloads")
	viper.SetDefault("downloader.max_retries", 3)
	viper.SetDefault("downloader.timeout", 300)
	viper.SetDefault("downloader.max_concurrent", 3)

	viper.SetDefault("ytdlp.path", "yt-dlp")
	viper.SetDefault("ytdlp.user_agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	viper.SetDefault("ytdlp.cookies_file", "")
	viper.SetDefault("ytdlp.proxy", "")
	viper.SetDefault("ytdlp.format", "best")
	viper.SetDefault("ytdlp.extract_audio", false)
	viper.SetDefault("ytdlp.audio_format", "mp3")
	viper.SetDefault("ytdlp.audio_quality", "192K")

	viper.SetDefault("aria2.path", "aria2c")
	viper.SetDefault("aria2.max_connections", 16)
	viper.SetDefault("aria2.min_split_size", 1)
	viper.SetDefault("aria2.continue", true)

	viper.SetDefault("douyin.enable_direct_api", true)
	viper.SetDefault("douyin.use_mobile_ua", true)
	viper.SetDefault("douyin.mobile_ua", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36")
	viper.SetDefault("douyin.api_timeout", 10)

	viper.SetDefault("database.driver", "sqlite")
	viper.SetDefault("database.dsn", "./data/video-hunter.db")

	viper.SetDefault("security.cors_origins", []string{"*"})
	viper.SetDefault("security.rate_limit", 100)
	viper.SetDefault("security.rate_limit_window", "1m")
}

// createDefaultConfig 创建默认配置文件
func createDefaultConfig(configPath string) error {
	// 创建目录
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 创建配置文件
	if err := viper.SafeWriteConfigAs(configPath); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// processConfig 处理配置
func processConfig(config *Config) error {
	// 处理日志文件路径
	if config.Log.File != "" {
		dir := filepath.Dir(config.Log.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建日志目录失败: %w", err)
		}
	}

	// 处理下载目录
	if config.Downloader.OutputDir != "" {
		if err := os.MkdirAll(config.Downloader.OutputDir, 0755); err != nil {
			return fmt.Errorf("创建下载目录失败: %w", err)
		}
	}

	// 处理速率限制时间窗口
	if config.Security.RateLimitWindow != "" {
		duration, err := time.ParseDuration(config.Security.RateLimitWindow)
		if err != nil {
			return fmt.Errorf("解析速率限制时间窗口失败: %w", err)
		}
		config.Security.RateLimitDuration = duration
	}

	return nil
}

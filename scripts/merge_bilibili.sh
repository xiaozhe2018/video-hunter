#!/bin/bash

# 检查ffmpeg是否安装
if ! command -v ffmpeg &> /dev/null; then
    echo "错误: ffmpeg 未安装，请先安装 ffmpeg"
    echo "可以使用 'brew install ffmpeg' (macOS) 或 'apt install ffmpeg' (Ubuntu/Debian) 安装"
    exit 1
fi

# 检查参数
if [ $# -lt 3 ]; then
    echo "用法: $0 <视频文件路径> <音频文件路径> <输出文件路径>"
    echo "例如: $0 ./downloads/video.mp4 ./downloads/audio.m4a ./downloads/merged.mp4"
    exit 1
fi

VIDEO_FILE="$1"
AUDIO_FILE="$2"
OUTPUT_FILE="$3"

# 检查文件是否存在
if [ ! -f "$VIDEO_FILE" ]; then
    echo "错误: 视频文件 '$VIDEO_FILE' 不存在"
    exit 1
fi

if [ ! -f "$AUDIO_FILE" ]; then
    echo "错误: 音频文件 '$AUDIO_FILE' 不存在"
    exit 1
fi

# 合并视频和音频
echo "正在合并视频和音频文件..."
ffmpeg -i "$VIDEO_FILE" -i "$AUDIO_FILE" -c:v copy -c:a aac -strict experimental "$OUTPUT_FILE"

# 检查是否成功
if [ $? -eq 0 ]; then
    echo "合并成功！输出文件: $OUTPUT_FILE"
    echo "文件大小: $(du -h "$OUTPUT_FILE" | cut -f1)"
else
    echo "合并失败，请检查错误信息"
    exit 1
fi 
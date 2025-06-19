#!/bin/bash

# 检查ffmpeg是否安装
if ! command -v ffmpeg &> /dev/null; then
    echo "错误: ffmpeg 未安装，请先安装 ffmpeg"
    echo "可以使用 'brew install ffmpeg' (macOS) 或 'apt install ffmpeg' (Ubuntu/Debian) 安装"
    exit 1
fi

# 设置下载目录，默认为当前目录下的downloads
DOWNLOAD_DIR="${1:-./downloads}"

# 检查下载目录是否存在
if [ ! -d "$DOWNLOAD_DIR" ]; then
    echo "错误: 下载目录 '$DOWNLOAD_DIR' 不存在"
    echo "用法: $0 [下载目录路径]"
    exit 1
fi

echo "正在扫描目录: $DOWNLOAD_DIR"

# 找出所有可能的任务ID（文件名前缀）
task_ids=$(find "$DOWNLOAD_DIR" -type f -name "*.mp4" -o -name "*.m4a" -o -name "*.aac" | grep -v "_merged" | sed -E 's/.*\/([^_]+)_.*/\1/' | sort | uniq)

if [ -z "$task_ids" ]; then
    echo "未找到需要合并的文件"
    exit 0
fi

echo "找到以下任务ID:"
echo "$task_ids"

# 遍历每个任务ID
for task_id in $task_ids; do
    echo "处理任务ID: $task_id"
    
    # 查找对应的视频和音频文件
    video_file=$(find "$DOWNLOAD_DIR" -type f -name "${task_id}_*.mp4" | grep -v "_merged" | head -1)
    audio_file=$(find "$DOWNLOAD_DIR" -type f -name "${task_id}_*.m4a" -o -name "${task_id}_*.aac" | head -1)
    
    # 如果同时找到视频和音频文件
    if [ -n "$video_file" ] && [ -n "$audio_file" ]; then
        echo "找到匹配的文件:"
        echo "视频: $video_file"
        echo "音频: $audio_file"
        
        # 检查是否已经有合并后的文件
        merged_file="$DOWNLOAD_DIR/${task_id}_merged.mp4"
        if [ -f "$merged_file" ]; then
            echo "已存在合并文件: $merged_file，跳过"
            continue
        fi
        
        # 合并视频和音频
        echo "正在合并视频和音频文件..."
        ffmpeg -i "$video_file" -i "$audio_file" -c:v copy -c:a aac -strict experimental -y "$merged_file"
        
        # 检查是否成功
        if [ $? -eq 0 ]; then
            echo "合并成功！输出文件: $merged_file"
            echo "文件大小: $(du -h "$merged_file" | cut -f1)"
        else
            echo "合并失败，请检查错误信息"
        fi
    else
        echo "未找到匹配的视频和音频文件，跳过"
    fi
    
    echo "------------------------"
done

echo "处理完成" 
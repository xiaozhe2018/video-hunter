// Video Hunter Web App
class VideoHunterApp {
    constructor() {
        this.ws = null;
        this.downloads = new Map();
        this.currentDownloadId = null;
        this.pollingInterval = null;
        this.init();
    }

    init() {
        this.bindEvents();
        this.connectWebSocket();
        this.loadDownloads();
        this.startProgressPolling();
    }

    bindEvents() {
        // 下载按钮点击
        document.getElementById('downloadBtn').addEventListener('click', () => {
            this.startDownload();
        });

        // 阻止表单自动提交，防止多次下载
        document.getElementById('downloadForm').addEventListener('submit', function(e) {
            e.preventDefault();
        });

        // URL输入变化时获取视频信息
        document.getElementById('url').addEventListener('input', (e) => {
            this.debounce(() => this.getVideoInfo(e.target.value), 1000);
        });

        // 粘贴按钮
        document.getElementById('pasteBtn').addEventListener('click', () => {
            navigator.clipboard.readText().then(text => {
                document.getElementById('url').value = text;
                this.getVideoInfo(text);
            }).catch(() => {
                this.showNotification('无法访问剪贴板', 'error');
            });
        });

        // 清空URL按钮
        document.getElementById('clearUrlBtn').addEventListener('click', () => {
            document.getElementById('url').value = '';
            this.hideVideoInfo();
        });

        // 清空按钮
        document.getElementById('clearBtn').addEventListener('click', () => {
            this.clearForm();
        });

        // 刷新按钮
        document.getElementById('refreshBtn').addEventListener('click', () => {
            this.loadDownloads();
        });

        // 清空所有按钮
        document.getElementById('clearAllBtn').addEventListener('click', () => {
            this.clearAllDownloads();
        });

        // 设置按钮
        document.getElementById('settingsBtn').addEventListener('click', () => {
            this.showSettings();
        });

        // 帮助按钮
        document.getElementById('helpBtn').addEventListener('click', () => {
            this.showHelp();
        });
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            console.log('WebSocket连接已建立');
            this.showNotification('连接成功', 'success');
        };
        
        this.ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.handleWebSocketMessage(data);
        };
        
        this.ws.onclose = () => {
            console.log('WebSocket连接已关闭');
            this.showNotification('连接断开，正在重连...', 'warning');
            // 5秒后重连
            setTimeout(() => this.connectWebSocket(), 5000);
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket错误:', error);
            this.showNotification('连接错误', 'error');
        };
    }

    handleWebSocketMessage(data) {
        console.log('收到WebSocket消息:', data);
        
        // 处理进度更新消息
        if (data.type === 'progress') {
            const download = data;
            
            // 更新内存中的数据
            const existingDownload = this.downloads.get(download.id);
            if (existingDownload) {
                // 合并新数据到现有数据
                this.downloads.set(download.id, {
                    ...existingDownload,
                    progress: download.progress,
                    speed: download.speed,
                    eta: download.eta,
                    status: download.status,
                    file: download.file || existingDownload.file,
                    updated: new Date().toISOString()
                });
                
                // 更新UI中的进度条
                this.updateDownloadProgress(this.downloads.get(download.id));
                
                // 如果下载完成，显示通知
                if (download.status === 'completed' && existingDownload.status !== 'completed') {
                    this.showNotification(`下载完成: ${download.file || '未知文件'}`, 'success');
                }
                
                // 如果下载失败，显示通知
                if (download.status === 'failed' && existingDownload.status !== 'failed') {
                    this.showNotification(`下载失败: ${download.error || '未知错误'}`, 'error');
                }
            } else {
                // 如果是新的下载，添加到内存中并更新列表
                this.downloads.set(download.id, download);
                this.updateDownloadList();
            }
        }
    }

    async startDownload() {
        const url = document.getElementById('url').value.trim();
        const format = document.getElementById('format').value;

        if (!url) {
            this.showNotification('请输入视频URL', 'error');
            return;
        }

        // 显示加载中
        this.showLoading('正在创建下载任务...');

        try {
            // 创建下载任务
            const createResponse = await fetch('/api/download', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ url, format })
            });

            if (!createResponse.ok) {
                throw new Error(`HTTP ${createResponse.status}: ${createResponse.statusText}`);
            }

            const downloadTask = await createResponse.json();
            const downloadId = downloadTask.id;

            // 将下载任务添加到内存中
            this.downloads.set(downloadId, downloadTask);
            
            // 更新下载列表，添加新的下载项
            this.updateDownloadList();
            
            // 隐藏加载中
            this.hideLoading();
            
            // 清空表单
            this.clearForm();
            
            // 显示通知
            this.showNotification('下载任务已创建，可在下载列表中查看进度', 'success');
            
            // 滚动到下载列表
            document.getElementById('downloadList').scrollIntoView({ behavior: 'smooth' });
            
            // 不再使用轮询，改为通过WebSocket接收实时进度更新
        } catch (error) {
            this.hideLoading();
            this.showNotification(`创建下载任务失败: ${error.message}`, 'error');
            console.error('下载错误:', error);
        }
    }

    async getVideoInfo(url) {
        if (!url || url.length < 10) {
            this.hideVideoInfo();
            return;
        }

        try {
            const response = await fetch(`/api/video-info?url=${encodeURIComponent(url)}`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const info = await response.json();
            this.displayVideoInfo(info);
        } catch (error) {
            console.error('获取视频信息失败:', error);
            this.hideVideoInfo();
        }
    }

    displayVideoInfo(info) {
        const container = document.getElementById('videoInfo');
        const content = document.getElementById('videoInfoContent');
        const downloadBtn = document.getElementById('downloadBtn');

        // 重置下载按钮状态
        downloadBtn.disabled = false;
        downloadBtn.classList.remove('bg-gray-400', 'cursor-not-allowed');
        downloadBtn.classList.add('bg-blue-600', 'hover:bg-blue-700');

        // 检查是否有特殊提示信息（针对抖音视频）
        if (info.special_note) {
            let html = `
                <div class="p-4 mb-4 rounded-lg bg-yellow-50 border border-yellow-200">
                    <h4 class="font-medium text-yellow-800 mb-2">
                        <i class="fas fa-exclamation-triangle mr-2"></i>${info.special_note}
                    </h4>
            `;

            // 如果有解决方案，显示解决方案列表
            if (info.solutions && info.solutions.length > 0) {
                html += `
                    <ul class="list-disc pl-5 text-yellow-700 text-sm">
                        ${info.solutions.map(solution => `<li>${solution}</li>`).join('')}
                    </ul>
                `;
            }

            html += `</div>`;

            // 如果指示不能下载，禁用下载按钮
            if (info.can_download === false) {
                downloadBtn.disabled = true;
                downloadBtn.classList.remove('bg-blue-600', 'hover:bg-blue-700');
                downloadBtn.classList.add('bg-gray-400', 'cursor-not-allowed');
            }

            content.innerHTML = html;
            container.classList.remove('hidden');
            return;
        }

        let html = `
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                    <p><strong>标题:</strong> ${info.title || '未知'}</p>
                    <p><strong>时长:</strong> ${info.duration || '未知'}</p>
                </div>
                <div>
                    <p><strong>格式数量:</strong> ${info.formats ? info.formats.length : 0}</p>
                    <p><strong>最佳质量:</strong> ${this.getBestQuality(info.formats)}</p>
                </div>
            </div>
        `;

        if (info.formats && info.formats.length > 0) {
            html += `
                <div class="mt-4">
                    <p class="font-medium mb-2">可用格式:</p>
                    <div class="grid grid-cols-2 md:grid-cols-4 gap-2 text-xs">
                        ${info.formats.slice(0, 8).map(format => `
                            <div class="bg-white p-2 rounded border">
                                <div class="font-medium">${format.format_id}</div>
                                <div class="text-gray-500">${format.resolution || '未知'}</div>
                                <div class="text-gray-400">${format.extension}</div>
                            </div>
                        `).join('')}
                    </div>
                </div>
            `;
        }

        content.innerHTML = html;
        container.classList.remove('hidden');
    }

    hideVideoInfo() {
        document.getElementById('videoInfo').classList.add('hidden');
    }

    getBestQuality(formats) {
        if (!formats || formats.length === 0) return '未知';
        
        const best = formats.find(f => f.quality === 'best') || formats[0];
        return best.resolution || best.format_id || '未知';
    }

    async loadDownloads() {
        try {
            const response = await fetch('/api/downloads');
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const downloads = await response.json();
            this.downloads.clear();
            downloads.forEach(download => {
                this.downloads.set(download.id, download);
            });

            this.updateDownloadList();
        } catch (error) {
            console.error('加载下载列表失败:', error);
            this.showNotification('加载下载列表失败', 'error');
        }
    }

    updateDownloadList() {
        const container = document.getElementById('downloadList');
        const emptyState = document.getElementById('emptyState');
        const countElement = document.getElementById('downloadCount');

        if (this.downloads.size === 0) {
            // 检查emptyState元素是否存在
            if (emptyState) {
                container.innerHTML = emptyState.outerHTML;
            } else {
                // 如果emptyState不存在，创建一个新的空状态显示
                container.innerHTML = `
                    <div class="text-center text-gray-500 py-8" id="emptyState">
                        <i class="fas fa-inbox text-4xl mb-4"></i>
                        <p>暂无下载任务</p>
                        <p class="text-sm">开始下载视频后，任务将显示在这里</p>
                    </div>
                `;
            }
            countElement.textContent = '0';
            return;
        }

        countElement.textContent = this.downloads.size.toString();

        const downloadsHtml = Array.from(this.downloads.values())
            .sort((a, b) => new Date(b.created) - new Date(a.created))
            .map(download => this.createDownloadItem(download))
            .join('');

        container.innerHTML = downloadsHtml;
    }

    createDownloadItem(download) {
        const statusClass = this.getStatusClass(download.status);
        const statusIcon = this.getStatusIcon(download.status);
        const progress = download.progress || 0;
        const created = new Date(download.created).toLocaleString('zh-CN');

        return `
            <div class="download-item bg-gray-50 rounded-lg p-4 border-l-4 ${statusClass}" data-id="${download.id}">
                <div class="flex items-center justify-between mb-3">
                    <div class="flex items-center space-x-3">
                        <div class="w-8 h-8 rounded-full bg-blue-100 flex items-center justify-center">
                            <i class="${statusIcon} text-blue-600"></i>
                        </div>
                        <div>
                            <h4 class="font-medium text-gray-800">${download.file || '未知文件'}</h4>
                            <p class="text-sm text-gray-500">ID: ${download.id}</p>
                        </div>
                    </div>
                    <div class="text-right">
                        <p class="text-sm font-medium ${statusClass}">${this.getStatusText(download.status)}</p>
                        <p class="text-xs text-gray-500">${created}</p>
                    </div>
                </div>
                
                <div class="mb-3">
                    <div class="flex justify-between text-sm text-gray-600 mb-1">
                        <span>进度: <span class="progress-text">${progress.toFixed(1)}%</span></span>
                        <span class="speed-text">${download.speed || ''} <span class="eta-text">${download.eta ? `ETA: ${download.eta}` : ''}</span></span>
                    </div>
                    <div class="w-full bg-gray-200 rounded-full h-2">
                        <div class="progress-bar bg-blue-600 h-2 rounded-full transition-all duration-300" style="width: ${progress}%"></div>
                    </div>
                </div>

                <div class="flex justify-between items-center">
                    <div class="flex space-x-2">
                        ${download.status === 'downloading' ? `
                            <button onclick="app.cancelDownload('${download.id}')" 
                                    class="text-red-600 hover:text-red-800 text-sm">
                                <i class="fas fa-stop mr-1"></i>取消
                            </button>
                        ` : ''}
                        ${download.status === 'completed' ? `
                            <button onclick="app.openFile('${download.file}')" 
                                    class="text-green-600 hover:text-green-800 text-sm">
                                <i class="fas fa-folder-open mr-1"></i>打开文件
                            </button>
                        ` : ''}
                    </div>
                    <div class="text-xs text-gray-500">
                        更新时间: <span class="update-time">${new Date(download.updated).toLocaleTimeString('zh-CN')}</span>
                    </div>
                </div>
                
                ${download.error ? `
                    <div class="mt-2 p-2 bg-red-50 border border-red-200 rounded text-sm text-red-700">
                        <i class="fas fa-exclamation-triangle mr-1"></i>错误: ${download.error}
                    </div>
                ` : ''}
            </div>
        `;
    }

    getStatusClass(status) {
        switch (status) {
            case 'pending': return 'border-yellow-500';
            case 'downloading': return 'border-blue-500';
            case 'completed': return 'border-green-500';
            case 'failed': return 'border-red-500';
            case 'cancelled': return 'border-gray-500';
            default: return 'border-gray-300';
        }
    }

    getStatusIcon(status) {
        switch (status) {
            case 'pending': return 'fas fa-clock';
            case 'downloading': return 'fas fa-download';
            case 'completed': return 'fas fa-check-circle';
            case 'failed': return 'fas fa-times-circle';
            case 'cancelled': return 'fas fa-ban';
            default: return 'fas fa-question-circle';
        }
    }

    getStatusText(status) {
        switch (status) {
            case 'pending': return '等待中';
            case 'downloading': return '下载中';
            case 'completed': return '已完成';
            case 'failed': return '失败';
            case 'cancelled': return '已取消';
            default: return '未知';
        }
    }

    async cancelDownload(id) {
        try {
            const response = await fetch(`/api/downloads/${id}/cancel`, {
                method: 'POST'
            });

            if (response.ok) {
                this.showNotification('下载已取消', 'success');
                this.loadDownloads();
            } else {
                throw new Error(`HTTP ${response.status}`);
            }
        } catch (error) {
            this.showNotification('取消下载失败', 'error');
            console.error('取消下载错误:', error);
        }
    }

    async clearAllDownloads() {
        // 使用更友好的确认对话框
        const confirmed = await this.showConfirmDialog(
            '确认清空',
            '确定要清空所有下载记录吗？此操作不可撤销。',
            '清空所有',
            '取消'
        );
        
        if (!confirmed) {
            return;
        }

        this.showLoading('正在清空下载记录...');

        try {
            const response = await fetch('/api/downloads/clear', {
                method: 'POST'
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            const result = await response.json();
            
            // 清空下载记录
            this.downloads.clear();
            
            // 安全地更新UI
            this.updateDownloadList();
            
            this.hideLoading();
            this.showNotification(result.message || '已清空所有下载记录', 'success');
        } catch (error) {
            this.hideLoading();
            this.showNotification(`清空下载记录失败: ${error.message}`, 'error');
            console.error('清空下载错误:', error);
        }
    }

    clearForm() {
        document.getElementById('downloadForm').reset();
        this.hideVideoInfo();
    }

    showLoading(text = '正在处理...') {
        document.getElementById('loadingText').textContent = text;
        document.getElementById('loadingOverlay').classList.remove('hidden');
        document.getElementById('loadingOverlay').classList.add('flex');
    }

    hideLoading() {
        document.getElementById('loadingOverlay').classList.add('hidden');
        document.getElementById('loadingOverlay').classList.remove('flex');
    }

    showNotification(message, type = 'info') {
        const notification = document.getElementById('notification');
        const text = document.getElementById('notificationText');
        const icon = document.getElementById('notificationIcon');

        text.textContent = message;

        // 设置图标和样式
        switch (type) {
            case 'success':
                icon.className = 'fas fa-check-circle text-green-500';
                break;
            case 'error':
                icon.className = 'fas fa-exclamation-circle text-red-500';
                break;
            case 'warning':
                icon.className = 'fas fa-exclamation-triangle text-yellow-500';
                break;
            default:
                icon.className = 'fas fa-info-circle text-blue-500';
        }

        notification.classList.remove('hidden');
        
        // 3秒后自动隐藏
        setTimeout(() => {
            this.hideNotification();
        }, 3000);
    }

    hideNotification() {
        document.getElementById('notification').classList.add('hidden');
    }

    showSettings() {
        // 加载当前设置
        this.loadSettings();
        document.getElementById('settingsModal').classList.remove('hidden');
        document.getElementById('settingsModal').classList.add('flex');
    }

    closeSettings() {
        console.log('closeSettings被调用');
        document.getElementById('settingsModal').classList.add('hidden');
        document.getElementById('settingsModal').classList.remove('flex');
    }

    loadSettings() {
        // 从localStorage加载设置，如果没有则使用默认值
        const settings = JSON.parse(localStorage.getItem('videoHunterSettings') || '{}');
        
        document.getElementById('downloadDir').value = settings.downloadDir || './downloads';
        document.getElementById('maxRetries').value = settings.maxRetries || 3;
        document.getElementById('timeout').value = settings.timeout || 300;
        document.getElementById('maxConcurrent').value = settings.maxConcurrent || 3;
    }

    saveSettings() {
        console.log('saveSettings被调用');
        try {
            // 收集设置值
            const settings = {
                downloadDir: document.getElementById('downloadDir').value,
                maxRetries: parseInt(document.getElementById('maxRetries').value),
                timeout: parseInt(document.getElementById('timeout').value),
                maxConcurrent: parseInt(document.getElementById('maxConcurrent').value)
            };

            // 验证设置
            if (settings.maxRetries < 0 || settings.maxRetries > 10) {
                this.showNotification('最大重试次数必须在0-10之间', 'error');
                return;
            }

            if (settings.timeout < 30 || settings.timeout > 1800) {
                this.showNotification('超时时间必须在30-1800秒之间', 'error');
                return;
            }

            if (settings.maxConcurrent < 1 || settings.maxConcurrent > 10) {
                this.showNotification('最大并发数必须在1-10之间', 'error');
                return;
            }

            // 保存到localStorage
            localStorage.setItem('videoHunterSettings', JSON.stringify(settings));
            
            this.closeSettings();
            this.showNotification('设置已保存', 'success');
        } catch (error) {
            this.showNotification('保存设置失败: ' + error.message, 'error');
        }
    }

    showHelp() {
        document.getElementById('helpModal').classList.remove('hidden');
        document.getElementById('helpModal').classList.add('flex');
    }

    closeHelp() {
        console.log('closeHelp被调用');
        document.getElementById('helpModal').classList.add('hidden');
        document.getElementById('helpModal').classList.remove('flex');
    }

    // 新增：显示确认对话框
    showConfirmDialog(title, message, confirmText = '确定', cancelText = '取消') {
        return new Promise((resolve) => {
            // 创建模态框
            const modal = document.createElement('div');
            modal.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50';
            modal.innerHTML = `
                <div class="bg-white rounded-lg p-6 max-w-md mx-4 w-full">
                    <div class="flex items-center mb-4">
                        <i class="fas fa-question-circle text-blue-500 mr-3 text-xl"></i>
                        <h3 class="text-lg font-bold text-gray-800">${title}</h3>
                    </div>
                    <p class="text-gray-700 mb-6">${message}</p>
                    <div class="flex justify-end space-x-2">
                        <button id="cancelBtn" class="px-4 py-2 bg-gray-200 text-gray-700 rounded hover:bg-gray-300 transition-colors">
                            ${cancelText}
                        </button>
                        <button id="confirmBtn" class="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition-colors">
                            ${confirmText}
                        </button>
                    </div>
                </div>
            `;

            // 添加到页面
            document.body.appendChild(modal);

            // 绑定事件
            const confirmBtn = modal.querySelector('#confirmBtn');
            const cancelBtn = modal.querySelector('#cancelBtn');

            const cleanup = () => {
                document.body.removeChild(modal);
            };

            confirmBtn.addEventListener('click', () => {
                cleanup();
                resolve(true);
            });

            cancelBtn.addEventListener('click', () => {
                cleanup();
                resolve(false);
            });

            // 点击背景关闭
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    cleanup();
                    resolve(false);
                }
            });

            // ESC键关闭
            const handleEsc = (e) => {
                if (e.key === 'Escape') {
                    cleanup();
                    resolve(false);
                    document.removeEventListener('keydown', handleEsc);
                }
            };
            document.addEventListener('keydown', handleEsc);
        });
    }

    showStatus(message, type = 'info') {
        const statusBar = document.getElementById('statusBar');
        const statusText = document.getElementById('statusText');
        
        statusText.textContent = message;
        statusBar.classList.remove('hidden');
        
        // 5秒后自动隐藏
        setTimeout(() => {
            this.hideStatus();
        }, 5000);
    }

    hideStatus() {
        document.getElementById('statusBar').classList.add('hidden');
    }

    debounce(func, wait) {
        clearTimeout(this.debounceTimer);
        this.debounceTimer = setTimeout(func, wait);
    }

    startProgressPolling() {
        // 清除之前的轮询
        if (this.pollingInterval) {
            clearInterval(this.pollingInterval);
        }

        // 每2秒轮询一次，只更新进度，不重新渲染整个列表
        this.pollingInterval = setInterval(() => {
            if (this.downloads.size > 0) {
                this.updateProgressOnly();
            }
        }, 2000);
    }

    // 新增：只更新进度，不重新渲染整个列表
    async updateProgressOnly() {
        try {
            const response = await fetch('/api/downloads');
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const downloads = await response.json();
            
            // 只更新进度条和状态，不重新渲染整个列表
            downloads.forEach(download => {
                const existingDownload = this.downloads.get(download.id);
                if (existingDownload) {
                    // 更新内存中的数据
                    this.downloads.set(download.id, download);
                    
                    // 只更新进度条和状态显示
                    this.updateDownloadProgress(download);
                } else {
                    // 如果是新的下载，添加到内存中
                    this.downloads.set(download.id, download);
                }
            });

            // 更新下载数量
            const countElement = document.getElementById('downloadCount');
            if (countElement) {
                countElement.textContent = this.downloads.size.toString();
            }

        } catch (error) {
            console.error('更新进度失败:', error);
            // 不显示错误通知，避免干扰用户体验
        }
    }

    // 新增：只更新单个下载项的进度
    updateDownloadProgress(download) {
        const downloadElement = document.querySelector(`[data-id="${download.id}"]`);
        if (!downloadElement) {
            return;
        }

        // 更新进度条
        const progressBar = downloadElement.querySelector('.progress-bar');
        const progressText = downloadElement.querySelector('.progress-text');
        const speedText = downloadElement.querySelector('.speed-text');
        const etaText = downloadElement.querySelector('.eta-text');
        const updateTime = downloadElement.querySelector('.update-time');
        const statusText = downloadElement.querySelector('.font-medium');

        if (progressBar) {
            progressBar.style.width = `${download.progress || 0}%`;
        }
        if (progressText) {
            progressText.textContent = `${(download.progress || 0).toFixed(1)}%`;
        }
        if (speedText) {
            speedText.textContent = download.speed || '';
        }
        if (etaText) {
            etaText.textContent = download.eta ? `ETA: ${download.eta}` : '';
        }
        if (updateTime) {
            updateTime.textContent = new Date(download.updated).toLocaleTimeString('zh-CN');
        }
        if (statusText) {
            statusText.textContent = this.getStatusText(download.status);
            statusText.className = `text-sm font-medium ${this.getStatusClass(download.status)}`;
        }

        // 如果状态发生变化，更新图标和边框
        const statusIcon = downloadElement.querySelector('.w-8.h-8 i');
        const downloadItem = downloadElement;
        
        if (statusIcon) {
            statusIcon.className = this.getStatusIcon(download.status) + ' text-blue-600';
        }
        
        // 更新边框颜色
        downloadItem.className = downloadItem.className.replace(/border-\w+-500/g, '');
        downloadItem.classList.add(this.getStatusClass(download.status));

        // 如果下载完成，检查是否需要自动下载到本地
        // if (download.status === 'completed') {
        //     // 检查是否设置了保存到本地
        //     if (download.metadata && download.metadata.save_to_local === 'true') {
        //         this.triggerFileDownload(download.id);
        //     }
        // }

        // 如果下载完成或失败，显示相应的按钮
        this.updateDownloadButtons(downloadElement, download);
    }

    // 新增：触发文件下载
    async triggerFileDownload(downloadId) {
        try {
            // 创建一个隐藏的下载链接
            const downloadUrl = `/api/downloads/${downloadId}/download`;
            const link = document.createElement('a');
            link.href = downloadUrl;
            link.download = ''; // 让浏览器自动处理文件名
            link.style.display = 'none';
            
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
            
            this.showNotification('文件已开始下载到本地', 'success');
        } catch (error) {
            console.error('触发文件下载失败:', error);
            this.showNotification('自动下载失败，请手动下载', 'error');
        }
    }

    // 新增：更新下载按钮
    updateDownloadButtons(downloadElement, download) {
        const buttonContainer = downloadElement.querySelector('.flex.space-x-2');
        if (!buttonContainer) {
            return;
        }

        // 根据状态更新按钮
        if (download.status === 'downloading') {
            buttonContainer.innerHTML = `
                <button onclick="app.cancelDownload('${download.id}')" 
                        class="text-red-600 hover:text-red-800 text-sm">
                    <i class="fas fa-stop mr-1"></i>取消
                </button>
            `;
        } else if (download.status === 'completed') {
            let buttons = '';
            // 始终显示"下载到本地"按钮
            buttons += `
                <button onclick="app.triggerFileDownload('${download.id}')" 
                        class="text-blue-600 hover:text-blue-800 text-sm mr-2">
                    <i class="fas fa-download mr-1"></i>下载到本地
                </button>
            `;
            // 也可保留打开文件按钮（如有需要）
            // buttons += `
            //     <button onclick="app.openFile('${download.file}')" 
            //             class="text-green-600 hover:text-green-800 text-sm mr-2">
            //         <i class="fas fa-folder-open mr-1"></i>打开文件
            //     </button>
            // `;
            buttonContainer.innerHTML = buttons;
        } else {
            buttonContainer.innerHTML = '';
        }

        // 如果有错误信息，显示错误
        const errorContainer = downloadElement.querySelector('.bg-red-50');
        if (download.error) {
            if (!errorContainer) {
                const errorDiv = document.createElement('div');
                errorDiv.className = 'mt-2 p-2 bg-red-50 border border-red-200 rounded text-sm text-red-700';
                errorDiv.innerHTML = `<i class="fas fa-exclamation-triangle mr-1"></i>错误: ${download.error}`;
                downloadElement.appendChild(errorDiv);
            } else {
                errorContainer.innerHTML = `<i class="fas fa-exclamation-triangle mr-1"></i>错误: ${download.error}`;
            }
        } else if (errorContainer) {
            errorContainer.remove();
        }
    }

    openFile(filename) {
        // 这里应该实现打开文件的功能
        this.showNotification(`文件: ${filename}`, 'info');
    }

    stopProgressPolling() {
        if (this.pollingInterval) {
            clearInterval(this.pollingInterval);
            this.pollingInterval = null;
        }
    }
}

// 全局函数
function hideNotification() {
    if (window.app) {
        app.hideNotification();
    } else {
        console.warn('app对象未初始化');
    }
}

function closeSettings() {
    console.log('closeSettings被调用');
    if (window.app) {
        app.closeSettings();
    } else {
        console.warn('app对象未初始化，尝试直接关闭模态框');
        const modal = document.getElementById('settingsModal');
        if (modal) {
            modal.classList.add('hidden');
            modal.classList.remove('flex');
        }
    }
}

function saveSettings() {
    console.log('saveSettings被调用');
    if (window.app) {
        app.saveSettings();
    } else {
        console.warn('app对象未初始化，无法保存设置');
        alert('应用未完全加载，请稍后再试');
    }
}

function closeHelp() {
    console.log('closeHelp被调用');
    if (window.app) {
        app.closeHelp();
    } else {
        console.warn('app对象未初始化，尝试直接关闭模态框');
        const modal = document.getElementById('helpModal');
        if (modal) {
            modal.classList.add('hidden');
            modal.classList.remove('flex');
        }
    }
}

// 初始化应用
let app;
document.addEventListener('DOMContentLoaded', () => {
    console.log('DOM加载完成，初始化VideoHunterApp');
    app = new VideoHunterApp();
    window.app = app; // 确保全局可访问
    console.log('VideoHunterApp初始化完成');
}); 
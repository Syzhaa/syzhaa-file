// Auto-split from room.html inline script. Shared globals via window scope.

function handleDragOver(e) {
    e.preventDefault();
    e.currentTarget.classList.add('drag-over');
}

function handleDragLeave(e) {
    e.currentTarget.classList.remove('drag-over');
}

function handleDrop(e) {
    e.preventDefault();
    e.currentTarget.classList.remove('drag-over');
    const files = e.dataTransfer.files;
    uploadFiles(files);
}

function handleFileSelect(e) {
    const files = e.target.files;
    uploadFiles(files);
}

async function uploadFiles(files) {
    const progressEl = document.getElementById('upload-progress-inline');
    const queueList = document.getElementById('upload-queue-list');
    const totalText = document.getElementById('upload-total-text');
    
    progressEl.classList.remove('hidden');
    uploadAbortController = new AbortController();
    uploadStartTime = Date.now();
    uploadedBytes = 0;
    
    const totalFiles = files.length;
    let completedFiles = 0;
    
    // Create queue items for all files
    queueList.innerHTML = '';
    const fileItems = [];
    
    for (let i = 0; i < files.length; i++) {
        const file = files[i];
        const itemId = `upload-item-${i}`;
        
        const itemHTML = `
            <div id="${itemId}" class="bg-white rounded-xl p-3 border border-line">
                <div class="flex items-center gap-3 mb-2">
                    <input type="checkbox" id="${itemId}-checkbox"
                           class="w-[18px] h-[18px] rounded border-line accent-[#F6821F] cursor-pointer shrink-0"
                           checked>
                    <span id="${itemId}-icon" class="material-symbols-outlined text-muted text-[22px] shrink-0">pending</span>
                    <div class="flex-1 min-w-0">
                        <p class="font-semibold text-[13px] truncate">${file.name}</p>
                        <p class="text-xs text-muted">${formatFileSize(file.size)}</p>
                    </div>
                    <span id="${itemId}-speed" class="text-xs font-bold text-brand-600 shrink-0 tabular-nums"></span>
                    <span id="${itemId}-status" class="text-xs font-semibold text-muted shrink-0">Menunggu</span>
                </div>
                <div class="w-full bg-line rounded-full h-1.5">
                    <div id="${itemId}-bar" class="bg-brand-500 h-1.5 rounded-full transition-all" style="width: 0%"></div>
                </div>
            </div>
        `;
        
        queueList.innerHTML += itemHTML;
        fileItems.push({ id: itemId, file: file });
    }
    
    totalText.textContent = `0/${totalFiles}`;
    
    // Upload files sequentially
    for (let i = 0; i < files.length; i++) {
        const file = files[i];
        const fileId = generateUUID();
        const itemId = `upload-item-${i}`;
        
        // Check if file is selected
        const checkbox = document.getElementById(`${itemId}-checkbox`);
        if (!checkbox.checked) {
            // Skip this file - mark as skipped
            document.getElementById(`${itemId}-icon`).textContent = 'remove_circle';
            document.getElementById(`${itemId}-icon`).className = 'material-symbols-outlined text-gray-400 text-xl';
            document.getElementById(`${itemId}-status`).textContent = 'Skipped';
            document.getElementById(`${itemId}-status`).className = 'text-xs font-semibold text-gray-500';
            continue;
        }
        
        // Update to uploading state
        document.getElementById(`${itemId}-icon`).textContent = 'upload_file';
        document.getElementById(`${itemId}-icon`).className = 'material-symbols-outlined text-brand-500 text-[22px]';
        document.getElementById(`${itemId}-status`).textContent = 'Uploading...';
        document.getElementById(`${itemId}-status`).className = 'text-xs font-semibold text-brand-600';
        checkbox.disabled = true;
        
        try {
            await uploadFileInChunks(file, fileId, currentRoom.id, (progress, bytesUploaded, mbps) => {
                document.getElementById(`${itemId}-bar`).style.width = progress + '%';
                document.getElementById(`${itemId}-status`).textContent = `${progress}%`;
                const speedEl = document.getElementById(`${itemId}-speed`);
                if (speedEl && mbps > 0) {
                    speedEl.textContent = mbps >= 10 ? Math.round(mbps) + ' Mbps' : mbps.toFixed(1) + ' Mbps';
                }
            });
            
            // Update to completed state
            completedFiles++;
            document.getElementById(`${itemId}-icon`).textContent = 'check_circle';
            document.getElementById(`${itemId}-icon`).className = 'material-symbols-outlined text-green-600 text-xl';
            document.getElementById(`${itemId}-status`).textContent = 'Completed';
            document.getElementById(`${itemId}-status`).className = 'text-xs font-semibold text-green-600';
            document.getElementById(`${itemId}-speed`).textContent = '';
            document.getElementById(`${itemId}-bar`).style.width = '100%';
            document.getElementById(`${itemId}-bar`).className = 'bg-green-500 h-1.5 rounded-full transition-all';
            
            totalText.textContent = `${completedFiles}/${totalFiles}`;
            
        } catch (error) {
            if (error.message === 'Upload cancelled') {
                document.getElementById(`${itemId}-icon`).textContent = 'cancel';
                document.getElementById(`${itemId}-icon`).className = 'material-symbols-outlined text-orange-600 text-xl';
                document.getElementById(`${itemId}-status`).textContent = 'Cancelled';
                document.getElementById(`${itemId}-status`).className = 'text-xs font-semibold text-orange-600';
                console.log('Upload cancelled by user');
                break;
            }
            
            // Update to error state
            document.getElementById(`${itemId}-icon`).textContent = 'error';
            document.getElementById(`${itemId}-icon`).className = 'material-symbols-outlined text-red-600 text-xl';
            document.getElementById(`${itemId}-status`).textContent = 'Failed';
            document.getElementById(`${itemId}-status`).className = 'text-xs font-semibold text-red-600';
            
            console.error('Upload error:', error);
            showInfoModal(error.message === 'Upload failed' ? `Gagal upload ${file.name}` : error.message);
        }
    }
    
    // Reload content once after all uploads
    if (completedFiles > 0) {
        await loadTotalFileCount();
        await loadRoomContent();
        updateTotalFileDisplay();
    }
    
    // Auto-hide after 2 seconds if all completed
    if (completedFiles === totalFiles) {
        setTimeout(() => {
            progressEl.classList.add('hidden');
        }, 2000);
    }
}

async function uploadFileInChunks(file, fileId, roomId, onProgress) {
    const chunkSize = 10 * 1024 * 1024; // 10MB (increased for faster upload)
    const totalChunks = Math.ceil(file.size / chunkSize);
    let totalUploaded = 0;
    let speedMbps = 0;
    
    for (let i = 0; i < totalChunks; i++) {
        if (uploadAbortController.signal.aborted) {
            throw new Error('Upload cancelled');
        }
        
        const start = i * chunkSize;
        const end = Math.min(start + chunkSize, file.size);
        const chunk = file.slice(start, end);
        
        const formData = new FormData();
        formData.append('file', chunk);
        formData.append('chunkIndex', i);
        formData.append('totalChunks', totalChunks);
        formData.append('fileId', fileId);
        formData.append('originalName', file.name);
        formData.append('mimeType', file.type);
        formData.append('totalSize', file.size);
        formData.append('folderId', currentFolderId || '');
        
        const chunkStart = performance.now();
        const response = await fetch(`/api/upload/${roomId}`, {
            method: 'POST',
            body: formData,
            signal: uploadAbortController.signal
        });
        const chunkSecs = Math.max((performance.now() - chunkStart) / 1000, 0.01);
        // Mbps = (bytes * 8) / (seconds * 1e6), smoothed
        const instMbps = (chunk.size * 8) / (chunkSecs * 1e6);
        speedMbps = speedMbps === 0 ? instMbps : speedMbps * 0.7 + instMbps * 0.3;
        
        if (!response.ok) {
            let msg = 'Upload failed';
            try {
                const errData = await response.json();
                if (errData.error) msg = errData.error;
            } catch {}
            throw new Error(msg);
        }
        
        totalUploaded += chunk.size;
        const progress = Math.round(((i + 1) / totalChunks) * 100);
        onProgress(progress, totalUploaded, speedMbps);
    }
}

function cancelUpload() {
    if (uploadAbortController) {
        uploadAbortController.abort();
        document.getElementById('upload-progress-inline').classList.add('hidden');
    }
}

function toggleSelectAllUpload() {
    const checkboxes = document.querySelectorAll('[id^="upload-item-"][id$="-checkbox"]');
    if (checkboxes.length === 0) return;
    
    const allChecked = Array.from(checkboxes).every(cb => cb.checked && !cb.disabled);
    checkboxes.forEach(cb => {
        if (!cb.disabled) {
            cb.checked = !allChecked;
        }
    });
}

function formatSpeed(bytesPerSecond) {
    if (bytesPerSecond < 1024) {
        return Math.round(bytesPerSecond) + ' B/s';
    } else if (bytesPerSecond < 1024 * 1024) {
        return (bytesPerSecond / 1024).toFixed(1) + ' KB/s';
    } else {
        return (bytesPerSecond / 1024 / 1024).toFixed(2) + ' MB/s';
    }
}

function updateTotalFileDisplay() {
    const totalFileEl = document.getElementById('total-files-count');
    if (totalFileEl) {
        totalFileEl.textContent = totalFilesInRoom;
    }
}


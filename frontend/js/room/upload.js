// Auto-split from room.html inline script. Shared globals via window scope.

// --- Sesi upload (localStorage) ---
// Tiap file yang diupload dicatat fileId-nya. Kalau halaman di-refresh,
// fileId yang sama dipakai lagi + chunk yang sudah sampai di server
// dilewati -> upload MELANJUTKAN, tidak mengulang dari 0.
const UPLOAD_SESSION_TTL = 7 * 24 * 3600 * 1000; // 7 hari

function uploadSessionKey(file, roomId) {
    return `af_up_${roomId}_${file.size}_${file.lastModified}_${file.name}`;
}
function getUploadSession(file, roomId) {
    try {
        const raw = localStorage.getItem(uploadSessionKey(file, roomId));
        if (!raw) return null;
        const s = JSON.parse(raw);
        if (Date.now() - s.createdAt > UPLOAD_SESSION_TTL) {
            localStorage.removeItem(uploadSessionKey(file, roomId));
            return null;
        }
        return s;
    } catch { return null; }
}
function setUploadSession(file, roomId, fileId) {
    try {
        localStorage.setItem(uploadSessionKey(file, roomId), JSON.stringify({
            fileId, fileName: file.name, fileSize: file.size, createdAt: Date.now(),
        }));
    } catch {}
}
function clearUploadSession(file, roomId) {
    try { localStorage.removeItem(uploadSessionKey(file, roomId)); } catch {}
}
// Daftar sesi upload yang belum selesai di room ini (untuk banner pengingat).
function listUnfinishedUploadSessions(roomId) {
    const out = [];
    try {
        const prefix = `af_up_${roomId}_`;
        for (let i = 0; i < localStorage.length; i++) {
            const k = localStorage.key(i);
            if (k && k.startsWith(prefix)) {
                const s = JSON.parse(localStorage.getItem(k));
                if (s && Date.now() - s.createdAt < UPLOAD_SESSION_TTL) out.push(s);
            }
        }
    } catch {}
    return out;
}
// Banner pengingat: dipanggil setelah renderRoom() — kalau ada upload yang
// belum selesai, tampilkan supaya user tahu tinggal pilih file yang sama.
function showUnfinishedUploadSessions() {
    try {
        if (!currentRoom || !currentRoom.id) return;
        const old = document.getElementById('upload-resume-banner');
        if (old) old.remove();
        const sessions = listUnfinishedUploadSessions(currentRoom.id);
        if (!sessions.length) return;
        const panel = document.getElementById('upload-progress-inline');
        if (!panel) return;
        const banner = document.createElement('div');
        banner.id = 'upload-resume-banner';
        banner.className = 'mb-4 p-3 bg-amber-50 border border-amber-200 rounded-xl flex items-start gap-2';
        banner.innerHTML =
            '<span class="material-symbols-outlined text-amber-600 text-[20px] shrink-0">history</span>' +
            '<div class="text-[13px] text-ink min-w-0">' +
            '<p class="font-semibold mb-1">Ada upload yang belum selesai:</p>' +
            sessions.map(s =>
                `<p class="truncate text-muted">• ${escapeHtml(s.fileName)} (${formatFileSize(s.fileSize)})</p>`
            ).join('') +
            '<p class="mt-1 text-muted">Pilih file yang sama untuk melanjutkan dari terakhir, tidak mengulang dari awal.</p>' +
            '</div>';
        panel.parentNode.insertBefore(banner, panel);
    } catch (e) { console.warn('resume banner:', e); }
}
function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

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
                        <p id="${itemId}-sizes" class="text-xs text-muted">0 B / ${formatFileSize(file.size)}</p>
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

        // --- Resume: pakai sesi lama kalau ada, tanya server chunk mana yang sudah sampai ---
        let session = getUploadSession(file, currentRoom.id);
        if (!session) {
            const newFileId = generateUUID();
            setUploadSession(file, currentRoom.id, newFileId);
            session = { fileId: newFileId };
        }
        const fileId = session.fileId;
        let skipChunks = new Set();
        try {
            const st = await api.get(`/api/upload/${currentRoom.id}/chunks?fileId=${encodeURIComponent(fileId)}`, { noRedirect: true });
            if (st && st.success && Array.isArray(st.uploaded)) skipChunks = new Set(st.uploaded);
        } catch (e) { console.warn('chunk status:', e); }
        const resumed = skipChunks.size > 0;
        if (resumed) {
            document.getElementById(`${itemId}-status`).textContent = 'Melanjutkan...';
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
                document.getElementById(`${itemId}-sizes`).textContent =
                    `${formatFileSize(bytesUploaded)} / ${formatFileSize(file.size)}`;
                const speedEl = document.getElementById(`${itemId}-speed`);
                if (speedEl && mbps > 0) {
                    speedEl.textContent = mbps >= 10 ? Math.round(mbps) + ' Mbps' : mbps.toFixed(1) + ' Mbps';
                }
            }, skipChunks);
            clearUploadSession(file, currentRoom.id);
            
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

async function uploadFileInChunks(file, fileId, roomId, onProgress, skipChunks) {
    // 5MB chunks: small enough to finish on slow mobile links and cheap to
    // retry, big enough to keep request count sane (~160 reqs for 800MB).
    const chunkSize = 5 * 1024 * 1024;
    const totalChunks = Math.ceil(file.size / chunkSize);
    skipChunks = skipChunks || new Set();

    // --- Real-time progress & honest speed meter ---
    // fetch() can't report upload progress, so chunks go through XHR which
    // fires upload.onprogress per byte. A 500ms ticker turns that into a
    // smooth bar + speed measured from actual bytes on the wire.
    let doneBytes = 0;      // bytes dari chunk yang sudah selesai
    let chunkLoaded = 0;    // bytes chunk berjalan yang sudah terkirim
    let speedBps = 0;       // EMA kecepatan (byte/detik)
    let lastTickBytes = 0;
    let lastTickAt = performance.now();

    const renderLive = () => {
        const now = performance.now();
        const dt = Math.max((now - lastTickAt) / 1000, 0.05);
        const nowLoaded = doneBytes + chunkLoaded;
        const instBps = Math.max(0, (nowLoaded - lastTickBytes) / dt);
        // EMA: responsif tapi tidak lompat-lompat; macet -> turun ke 0 (jujur)
        speedBps = speedBps === 0 ? instBps : speedBps * 0.6 + instBps * 0.4;
        lastTickBytes = nowLoaded;
        lastTickAt = now;
        const progress = Math.min(100, Math.round((nowLoaded / file.size) * 100));
        onProgress(progress, nowLoaded, (speedBps * 8) / 1e6);
    };
    const ticker = setInterval(() => {
        if (uploadAbortController.signal.aborted) return;
        renderLive();
    }, 500);

    try {
        for (let i = 0; i < totalChunks; i++) {
            if (uploadAbortController.signal.aborted) {
                throw new Error('Upload cancelled');
            }

            const start = i * chunkSize;
            const end = Math.min(start + chunkSize, file.size);
            const chunkBytes = end - start;

            // Chunk sudah sampai di server (sesi resume) -> lewati tanpa kirim ulang
            if (skipChunks.has(i)) {
                doneBytes += chunkBytes;
                renderLive();
                continue;
            }

            const chunk = file.slice(start, end);
            chunkLoaded = 0;

            await uploadChunkWithRetry(chunk, i, totalChunks, fileId, file, roomId,
                uploadAbortController.signal, (loaded) => { chunkLoaded = loaded; });

            doneBytes += chunk.size;
            chunkLoaded = 0;
            renderLive();
        }
    } finally {
        clearInterval(ticker);
    }
}

// Satu chunk via XHR (biar ada upload.onprogress). Resolve kalau 2xx,
// reject dengan {status, message} kalau HTTP error, Error kalau network/abort.
function postChunkXHR(roomId, formData, signal, onBytes) {
    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open('POST', `/api/upload/${roomId}`);

        xhr.upload.onprogress = (e) => {
            if (e.lengthComputable) onBytes(e.loaded);
        };
        xhr.onload = async () => {
            if (xhr.status >= 200 && xhr.status < 300) { resolve(); return; }
            let msg = 'Upload failed';
            try {
                const errData = JSON.parse(xhr.responseText);
                if (errData.error) msg = errData.error;
            } catch {}
            reject({ status: xhr.status, message: msg });
        };
        xhr.onerror = () => reject(new Error('network'));
        xhr.ontimeout = () => reject(new Error('timeout'));
        xhr.onabort = () => reject(new Error('Upload cancelled'));
        signal.addEventListener('abort', () => xhr.abort(), { once: true });
        xhr.send(formData);
    });
}

// Upload satu chunk dengan retry (tahan koneksi HP yang putus-nyambung).
// Retry untuk: network error, 429 (rate limit), 5xx.
// TIDAK retry untuk 4xx permanen (400/401/403/404/413) — langsung gagal.
async function uploadChunkWithRetry(chunk, i, totalChunks, fileId, file, roomId, signal, onBytes) {
    const maxAttempts = 4;
    let lastError = null;

    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
        if (signal.aborted) throw new Error('Upload cancelled');
        onBytes(0);

        const formData = new FormData();
        formData.append('file', chunk);
        formData.append('chunkIndex', i);
        formData.append('totalChunks', totalChunks);
        formData.append('fileId', fileId);
        formData.append('originalName', file.name);
        formData.append('mimeType', file.type);
        formData.append('totalSize', file.size);
        formData.append('folderId', currentFolderId || '');

        try {
            await postChunkXHR(roomId, formData, signal, onBytes);
            return;
        } catch (err) {
            if (err && err.message === 'Upload cancelled') throw new Error('Upload cancelled');
            const status = err && err.status;
            // HTTP 4xx selain 429 = permanen, langsung lempar
            if (status && status !== 429 && status < 500) {
                throw new Error(err.message || 'Upload failed');
            }
            lastError = (err instanceof Error) ? err : new Error((err && err.message) || 'Upload failed');
        }

        if (attempt < maxAttempts) {
            // Exponential backoff: 1s, 2s, 4s
            await new Promise(res => setTimeout(res, 1000 * Math.pow(2, attempt - 1)));
        }
    }

    throw lastError || new Error('Upload failed');
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


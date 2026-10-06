// Auto-split from room.html inline script. Shared globals via window scope.

// Dialog pilihan: unduh satu per satu atau sebagai ZIP
function chooseDownloadMode(count) {
    return new Promise((resolve) => {
        const bd = document.createElement('div');
        bd.style.cssText = 'position:fixed;inset:0;background:rgba(0,0,0,.45);z-index:100;display:flex;align-items:center;justify-content:center;padding:16px';
        bd.innerHTML = `
            <div style="background:#fff;border-radius:16px;padding:20px;max-width:340px;width:100%;box-shadow:0 20px 60px rgba(0,0,0,.25)">
                <h3 style="margin:0 0 4px;font-size:16px;font-weight:700;color:#1f2937">Unduh ${count} file</h3>
                <p style="margin:0 0 16px;font-size:13px;color:#6b7280">Pilih cara mengunduh:</p>
                <div style="display:flex;flex-direction:column;gap:10px">
                    <button id="dl-onebyone" style="display:flex;align-items:center;gap:12px;padding:12px 16px;border:1.5px solid #e5e7eb;border-radius:12px;background:#fff;cursor:pointer;text-align:left">
                        <span class="material-symbols-outlined" style="color:#F6821F;font-size:24px">download</span>
                        <span><b style="font-size:14px;color:#1f2937">Satu per satu</b><br><small style="color:#6b7280">Tiap file diunduh terpisah</small></span>
                    </button>
                    <button id="dl-zip" style="display:flex;align-items:center;gap:12px;padding:12px 16px;border:none;border-radius:12px;background:#1a1a2e;color:#fff;cursor:pointer;text-align:left">
                        <span class="material-symbols-outlined" style="font-size:24px">folder_zip</span>
                        <span><b style="font-size:14px">Sebagai ZIP</b><br><small style="opacity:.7">Digabung jadi satu file zip</small></span>
                    </button>
                    <button id="dl-cancel" style="padding:10px;border:none;border-radius:12px;background:#f3f4f6;color:#374151;font-weight:600;cursor:pointer">Batal</button>
                </div>
            </div>`;
        document.body.appendChild(bd);
        const close = (v) => { bd.remove(); resolve(v); };
        bd.querySelector('#dl-onebyone').onclick = () => close('onebyone');
        bd.querySelector('#dl-zip').onclick = () => close('zip');
        bd.querySelector('#dl-cancel').onclick = () => close(null);
        bd.onclick = (e) => { if (e.target === bd) close(null); };
    });
}

// Unduh banyak file satu per satu (trigger download browser berurutan)
async function downloadOneByOne(files) {
    for (const file of files) {
        const a = document.createElement('a');
        a.href = `/d/${file.id}`;
        a.download = file.original_name || '';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        // jeda biar browser tidak memblokir multi-download
        await new Promise(r => setTimeout(r, 600));
    }
}

async function downloadAllAsZip() {
    if (currentFiles.length === 0) {
        showInfoModal('Tidak ada file untuk didownload');
        return;
    }

    const mode = await chooseDownloadMode(currentFiles.length);
    if (mode === 'onebyone') {
        downloadOneByOne(currentFiles);
        return;
    }
    if (mode !== 'zip') return;
    
    const progressEl = document.getElementById('download-progress');
    const queueList = document.getElementById('download-queue-list');
    const totalText = document.getElementById('download-total-text');
    
    progressEl.classList.remove('hidden');
    downloadAbortController = new AbortController();
    
    const totalFiles = currentFiles.length;
    let completedFiles = 0;
    
    // Create queue items for all files
    queueList.innerHTML = '';
    
    for (let i = 0; i < currentFiles.length; i++) {
        const file = currentFiles[i];
        const itemId = `download-item-${i}`;
        
        const itemHTML = `
            <div id="${itemId}" class="bg-wash rounded-xl p-3 border border-line">
                <div class="flex items-center gap-3 mb-2">
                    <input type="checkbox" id="${itemId}-checkbox"
                           class="w-[18px] h-[18px] rounded border-line accent-[#F6821F] cursor-pointer shrink-0"
                           checked>
                    <span id="${itemId}-icon" class="material-symbols-outlined text-muted text-[22px] shrink-0">pending</span>
                    <div class="flex-1 min-w-0">
                        <p class="font-semibold text-[13px] truncate">${file.original_name}</p>
                        <p class="text-xs text-muted">${formatFileSize(file.size)}</p>
                    </div>
                    <span id="${itemId}-status" class="text-xs font-semibold text-muted shrink-0">Menunggu</span>
                </div>
                <div class="w-full bg-line rounded-full h-1.5">
                    <div id="${itemId}-bar" class="bg-brand-500 h-1.5 rounded-full transition-all" style="width: 0%"></div>
                </div>
            </div>
        `;
        
        queueList.innerHTML += itemHTML;
    }
    
    totalText.textContent = `0/${totalFiles}`;
    
    try {
        const zip = new JSZip();
        
        for (let i = 0; i < currentFiles.length; i++) {
            const file = currentFiles[i];
            const itemId = `download-item-${i}`;
            
            if (downloadAbortController.signal.aborted) {
                throw new Error('Download cancelled');
            }
            
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
            
            // Update to downloading state
            document.getElementById(`${itemId}-icon`).textContent = 'download';
            document.getElementById(`${itemId}-icon`).className = 'material-symbols-outlined text-green-600 text-xl';
            document.getElementById(`${itemId}-status`).textContent = 'Downloading...';
            document.getElementById(`${itemId}-status`).className = 'text-xs font-semibold text-green-600';
            document.getElementById(`${itemId}-bar`).style.width = '50%';
            checkbox.disabled = true;
            
            try {
                const response = await fetch(`/d/${file.id}`, {
                    signal: downloadAbortController.signal
                });
                
                if (!response.ok) throw new Error(`Failed to download ${file.original_name}`);
                
                const blob = await response.blob();
                zip.file(file.original_name, blob);
                
                // Update to completed state
                completedFiles++;
                document.getElementById(`${itemId}-icon`).textContent = 'check_circle';
                document.getElementById(`${itemId}-icon`).className = 'material-symbols-outlined text-green-600 text-xl';
                document.getElementById(`${itemId}-status`).textContent = 'Completed';
                document.getElementById(`${itemId}-status`).className = 'text-xs font-semibold text-green-600';
                document.getElementById(`${itemId}-bar`).style.width = '100%';
                
                totalText.textContent = `${completedFiles}/${totalFiles}`;
                
            } catch (error) {
                // Update to error state
                document.getElementById(`${itemId}-icon`).textContent = 'error';
                document.getElementById(`${itemId}-icon`).className = 'material-symbols-outlined text-red-600 text-xl';
                document.getElementById(`${itemId}-status`).textContent = 'Failed';
                document.getElementById(`${itemId}-status`).className = 'text-xs font-semibold text-red-600';
                throw error;
            }
        }
        
        // Show ZIP creation status
        const zipItemHTML = `
            <div class="bg-brand-50 rounded-xl p-3 border border-brand-100">
                <div class="flex items-center gap-3">
                    <div class="spinner w-5 h-5 border-[3px] border-brand-200 border-t-brand-500 rounded-full shrink-0"></div>
                    <div class="flex-1">
                        <p class="font-semibold text-sm text-ink">Membuat arsip ZIP...</p>
                        <p class="text-xs text-muted">Mohon tunggu</p>
                    </div>
                </div>
            </div>
        `;
        queueList.innerHTML += zipItemHTML;
        
        const zipBlob = await zip.generateAsync({
            type: 'blob',
            compression: 'DEFLATE',
            compressionOptions: { level: 6 }
        });
        
        const url = URL.createObjectURL(zipBlob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `room-${currentRoom.pin}-files.zip`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        
        // Auto-hide after 2 seconds
        setTimeout(() => {
            progressEl.classList.add('hidden');
        }, 2000);
        
    } catch (error) {
        if (error.message === 'Download cancelled') {
            console.log('Download cancelled by user');
        } else {
            console.error('Download error:', error);
            showInfoModal('Gagal download files: ' + error.message);
        }
        progressEl.classList.add('hidden');
    }
}

async function downloadSelected() {
    const selected = getSelectedFiles();
    
    if (selected.length === 0) {
        showInfoModal('Pilih file terlebih dahulu');
        return;
    }
    
    if (selected.length === 1) {
        // Single file - direct download
        window.location.href = `/d/${selected[0].id}`;
        return;
    }

    const mode = await chooseDownloadMode(selected.length);
    if (mode === 'onebyone') {
        downloadOneByOne(selected);
        return;
    }
    if (mode !== 'zip') return;
    
    // Multiple files - download as ZIP
    try {
        const zip = new JSZip();
        const progressEl = document.getElementById('download-progress');
        const progressBar = document.getElementById('download-bar');
        const progressText = document.getElementById('download-text');
        
        progressEl.classList.remove('hidden');
        let completed = 0;
        
        for (const file of selected) {
            const response = await fetch(`/d/${file.id}`);
            const blob = await response.blob();
            zip.file(file.original_name, blob);
            
            completed++;
            const percent = Math.round((completed / selected.length) * 100);
            progressBar.style.width = `${percent}%`;
            progressText.textContent = `${percent}% - ${completed}/${selected.length} files`;
        }
        
        const zipBlob = await zip.generateAsync({ type: 'blob' });
        const url = URL.createObjectURL(zipBlob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `selected-files-${Date.now()}.zip`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        
        progressEl.classList.add('hidden');
        
        // Clear selection after download
        selectedFiles.clear();
        document.querySelectorAll('.file-checkbox').forEach(cb => cb.checked = false);
        updateSelectionUI();
        
    } catch (error) {
        console.error('Download error:', error);
        showInfoModal('Gagal mendownload file');
        document.getElementById('download-progress').classList.add('hidden');
    }
}

function cancelDownload() {
    if (downloadAbortController) {
        downloadAbortController.abort();
        document.getElementById('download-progress').classList.add('hidden');
    }
}

function toggleSelectAllDownload() {
    const checkboxes = document.querySelectorAll('[id^="download-item-"][id$="-checkbox"]');
    if (checkboxes.length === 0) return;
    
    const allChecked = Array.from(checkboxes).every(cb => cb.checked && !cb.disabled);
    checkboxes.forEach(cb => {
        if (!cb.disabled) {
            cb.checked = !allChecked;
        }
    });
}

function getSelectedFiles() {
    return currentFiles.filter(file => selectedFiles.has(file.id));
}

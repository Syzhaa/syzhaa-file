// Auto-split from room.html inline script. Shared globals via window scope.

function createFileCard(file) {
    if (currentView === 'card') {
        return createFileCardView(file);
    } else {
        return createFileListView(file);
    }
}

function createFileListView(file) {
    const fileSize = formatFileSize(file.size);
    const icon = getFileIcon(file.original_name);
    const canDownload = !currentRoom || (currentRoom.permission || 'both') !== 'view';
    const canDelete = !currentRoom || currentRoom.allow_delete !== false;

    return `
        <div class="file-item flex items-center gap-3.5 p-3.5 bg-white border border-line rounded-xl hover:border-muted transition-colors" data-file-id="${file.id}">
            <input type="checkbox"
                   class="file-checkbox w-[18px] h-[18px] rounded border-line accent-[#F6821F] shrink-0 cursor-pointer"
                   onchange="toggleFileSelection('${file.id}')">
            <div class="w-11 h-11 rounded-lg bg-wash border border-line flex items-center justify-center text-2xl shrink-0">${icon}</div>
            <div class="flex-1 min-w-0">
                <h4 class="font-semibold text-ink text-[15px] truncate">${file.original_name}</h4>
                <p class="text-[13px] text-muted mt-0.5">${fileSize} &middot; ${file.downloads} unduhan</p>
            </div>
            <div class="flex gap-2 shrink-0">
                ${canDownload ? `
                <a href="/d/${file.id}"
                   class="px-3.5 py-2 bg-brand-500 text-white rounded-lg hover:bg-brand-600 text-[13px] font-semibold transition-colors"
                   download>
                    Unduh
                </a>` : ''}
                ${canDelete ? `
                <button onclick="deleteFile('${file.id}')"
                        class="p-2 text-muted hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                        aria-label="Hapus">
                    <span class="material-symbols-outlined text-[20px] block">delete</span>
                </button>` : ''}
            </div>
        </div>
    `;
}

function createFileCardView(file) {
    const fileSize = formatFileSize(file.size);
    const icon = getFileIcon(file.original_name);
    const isImage = isImageFile(file.original_name);
    const thumbnailUrl = isImage ? `/d/${file.id}` : null;
    const canDownload = !currentRoom || (currentRoom.permission || 'both') !== 'view';
    const canDelete = !currentRoom || currentRoom.allow_delete !== false;

    return `
        <div class="file-item bg-white border border-line rounded-xl overflow-hidden hover:shadow-card hover:border-muted transition-all" data-file-id="${file.id}">
            <div class="aspect-[16/10] bg-wash flex items-center justify-center relative overflow-hidden border-b border-line">
                <input type="checkbox"
                       class="file-checkbox absolute top-2.5 left-2.5 w-[18px] h-[18px] rounded border-line accent-[#F6821F] z-10 cursor-pointer bg-white"
                       onchange="toggleFileSelection('${file.id}')">
                ${isImage ?
                    `<img src="${thumbnailUrl}"
                          alt="${file.original_name}"
                          class="w-full h-full object-cover"
                          onerror="this.style.display='none'; this.nextElementSibling.style.display='flex';">
                     <div class="hidden w-full h-full items-center justify-center">
                         <span class="text-5xl">${icon}</span>
                     </div>` :
                    `<span class="text-5xl">${icon}</span>`
                }
            </div>
            <div class="p-3.5">
                <h4 class="font-semibold text-ink text-[14px] truncate mb-0.5">${file.original_name}</h4>
                <p class="text-xs text-muted mb-3">${fileSize} &middot; ${file.downloads} unduhan</p>
                <div class="flex gap-2">
                    ${canDownload ? `
                    <a href="/d/${file.id}"
                       class="flex-1 text-center px-3 py-2 bg-brand-500 text-white rounded-lg hover:bg-brand-600 text-[13px] font-semibold transition-colors"
                       download>
                        Unduh
                    </a>` : ''}
                    ${canDelete ? `
                    <button onclick="deleteFile('${file.id}')"
                            class="px-2.5 py-2 text-muted hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                            aria-label="Hapus">
                        <span class="material-symbols-outlined text-[20px] block">delete</span>
                    </button>` : ''}
                </div>
            </div>
        </div>
    `;
}

function switchView(view) {
    currentView = view;

    // Update button states
    const listBtn = document.getElementById('view-list-btn');
    const cardBtn = document.getElementById('view-card-btn');
    const activeCls = ['bg-white', 'shadow-card', 'text-ink'];
    const idleCls = ['text-muted'];

    if (view === 'list') {
        listBtn.classList.add(...activeCls);
        listBtn.classList.remove(...idleCls);
        cardBtn.classList.add(...idleCls);
        cardBtn.classList.remove(...activeCls);

        // List layout
        document.getElementById('file-list').className = 'space-y-2.5';
    } else {
        cardBtn.classList.add(...activeCls);
        cardBtn.classList.remove(...idleCls);
        listBtn.classList.add(...idleCls);
        listBtn.classList.remove(...activeCls);

        // Grid layout
        document.getElementById('file-list').className = 'grid grid-cols-2 lg:grid-cols-3 gap-3.5';
    }

    // Re-render files and folders
    displayContent();
}

function renderFileList() {
    const fileListEl = document.getElementById('file-list');
    if (currentFiles.length === 0) {
        fileListEl.innerHTML = '<p class="text-center text-muted py-10 text-sm col-span-full">Belum ada file yang diupload</p>';
    } else {
        fileListEl.innerHTML = currentFiles.map(file => createFileCard(file)).join('');
    }
}

function isImageFile(filename) {
    const ext = filename.split('.').pop().toLowerCase();
    return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg'].includes(ext);
}

function copyLink() {
    navigator.clipboard.writeText(window.location.href).then(() => {
        showInfoModal('Link berhasil disalin!', true);
    });
}





function formatFileSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

function getFileIcon(filename) {
    const ext = filename.split('.').pop().toLowerCase();
    const icons = {
        'pdf': '📄', 'doc': '📝', 'docx': '📝', 'txt': '📝',
        'jpg': '🖼️', 'jpeg': '🖼️', 'png': '🖼️', 'gif': '🖼️', 'webp': '🖼️',
        'mp4': '🎥', 'mov': '🎥', 'avi': '🎥',
        'mp3': '🎵', 'wav': '🎵',
        'zip': '🗜️', 'rar': '🗜️',
        'xls': '📊', 'xlsx': '📊'
    };
    return icons[ext] || '📦';
}

async function deleteFile(fileId) {
    showConfirmModal('Yakin ingin menghapus file ini?', 'Hapus', async () => {
    
    try {
        await api.del(`/api/file/${fileId}`);
        await loadTotalFileCount();
        await loadRoomContent();
        updateTotalFileDisplay();
    } catch (error) {
        console.error('Delete error:', error);
        showInfoModal('Gagal menghapus file');
    }
    });
}

function generateUUID() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
        const r = Math.random() * 16 | 0;
        const v = c == 'x' ? r : (r & 0x3 | 0x8);
        return v.toString(16);
    });
}

function toggleFileSelection(fileId) {
    if (selectedFiles.has(fileId)) {
        selectedFiles.delete(fileId);
    } else {
        selectedFiles.add(fileId);
    }
    updateSelectionUI();
}

function toggleSelectAll() {
    const selectAllCheckbox = document.getElementById('select-all');
    const fileCheckboxes = document.querySelectorAll('.file-checkbox');
    
    if (selectAllCheckbox.checked) {
        // Select all
        selectedFiles.clear();
        currentFiles.forEach(file => selectedFiles.add(file.id));
        fileCheckboxes.forEach(cb => cb.checked = true);
    } else {
        // Deselect all
        selectedFiles.clear();
        fileCheckboxes.forEach(cb => cb.checked = false);
    }
    
    updateSelectionUI();
}

function updateSelectionUI() {
    const count = selectedFiles.size;
    const countEl = document.getElementById('selected-count');
    const downloadBtn = document.getElementById('download-selected-btn');
    const selectAllCheckbox = document.getElementById('select-all');
    
    // Update count
    countEl.textContent = `${count} dipilih`;
    
    // Update button state
    if (count > 0) {
        downloadBtn.disabled = false;
        downloadBtn.innerHTML = `<span class="material-symbols-outlined text-[20px]">download</span> Unduh ${count}`;
    } else {
        downloadBtn.disabled = true;
        downloadBtn.innerHTML = '<span class="material-symbols-outlined text-[20px]">download</span> Unduh';
    }
    
    // Update select all checkbox
    if (count === 0) {
        selectAllCheckbox.checked = false;
        selectAllCheckbox.indeterminate = false;
    } else if (count === currentFiles.length) {
        selectAllCheckbox.checked = true;
        selectAllCheckbox.indeterminate = false;
    } else {
        selectAllCheckbox.checked = false;
        selectAllCheckbox.indeterminate = true;
    }
    
    // Update visual selection state on cards
    document.querySelectorAll('.file-item').forEach(card => {
        const fileId = card.getAttribute('data-file-id');
        if (selectedFiles.has(fileId)) {
            card.classList.add('ring-2', 'ring-brand-500', 'border-brand-500');
        } else {
            card.classList.remove('ring-2', 'ring-brand-500', 'border-brand-500');
        }
    });
}


async function loadFiles() {
    try {
        const url = currentFolderId
            ? `/api/room/${roomId}?folder_id=${currentFolderId}`
            : `/api/room/${roomId}`;
        
        const data = await api.get(url);
        
        if (data.room) {
            currentFiles = data.files || [];
        }
    } catch (error) {
        console.error('Load files error:', error);
    }
}

function displayContent() {
    const fileListEl = document.getElementById('file-list');
    
    let html = '';
    
    if (currentFolders.length === 0 && currentFiles.length === 0) {
        html = '<p class="text-center text-muted py-10 text-sm">Belum ada file atau folder</p>';
    } else {
        if (currentFolders.length > 0) {
            html += currentFolders.map(folder => createFolderCard(folder)).join('');
        }
        if (currentFiles.length > 0) {
            html += currentFiles.map(file => createFileCard(file)).join('');
        }
    }
    
    fileListEl.innerHTML = html;
    renderBreadcrumb();
}


// Auto-split from room.html inline script. Shared globals via window scope.

async function createFolder() {
    const folderName = document.getElementById('folder-name-input').value.trim();
    
    if (!folderName) {
        showInfoModal('Nama folder tidak boleh kosong');
        return;
    }
    
    try {
        const data = await api.post(`/api/folder/create/${roomId}`, {
            name: folderName,
            parent_id: currentFolderId || ''
        });
        
        const data = await response.json();
        
        if (data.success) {
            closeCreateFolderModal();
            await loadRoomContent();
        } else {
            showInfoModal('Gagal membuat folder');
        }
    } catch (error) {
        console.error('Create folder error:', error);
        showInfoModal('Gagal membuat folder');
    }
}

async function loadFolders() {
    try {
        const url = currentFolderId 
            ? `/api/folders/${roomId}?parent_id=${currentFolderId}`
            : `/api/folders/${roomId}`;
        
        const data = await api.get(url);
        
        if (data.success) {
            currentFolders = data.folders || [];
        }
    } catch (error) {
        console.error('Load folders error:', error);
    }
}

async function loadRoomContent() {
    await Promise.all([loadFolders(), loadFiles()]);
    displayContent();
}

function createFolderCard(folder) {
    if (currentView === 'card') {
        return `
            <div class="file-item bg-white border border-line rounded-xl overflow-hidden hover:shadow-card hover:border-muted transition-all cursor-pointer"
                 onclick="openFolder('${folder.id}', '${folder.name}')">
                <div class="aspect-[16/10] bg-brand-50 flex items-center justify-center border-b border-brand-100">
                    <span class="material-symbols-outlined text-6xl text-brand-500">folder</span>
                </div>
                <div class="p-3.5 flex items-center gap-2">
                    <div class="flex-1 min-w-0">
                        <h4 class="font-semibold text-ink text-[14px] truncate">${folder.name}</h4>
                        <p class="text-xs text-muted mt-0.5">${new Date(folder.created_at).toLocaleString('id-ID')}</p>
                    </div>
                    <button onclick="event.stopPropagation(); deleteFolder('${folder.id}')"
                            class="p-2 text-muted hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors shrink-0"
                            aria-label="Hapus folder">
                        <span class="material-symbols-outlined text-[20px] block">delete</span>
                    </button>
                </div>
            </div>
        `;
    } else {
        return `
            <div class="file-item bg-white border border-line rounded-xl hover:border-muted hover:shadow-card transition-all cursor-pointer"
                 onclick="openFolder('${folder.id}', '${folder.name}')">
                <div class="flex items-center gap-3.5 p-3.5">
                    <div class="w-11 h-11 bg-brand-50 border border-brand-100 rounded-lg flex items-center justify-center shrink-0">
                        <span class="material-symbols-outlined text-[28px] text-brand-500">folder</span>
                    </div>
                    <div class="flex-1 min-w-0">
                        <h4 class="font-semibold text-ink text-[15px] truncate">${folder.name}</h4>
                        <p class="text-[13px] text-muted mt-0.5">${new Date(folder.created_at).toLocaleString('id-ID')}</p>
                    </div>
                    <button onclick="event.stopPropagation(); deleteFolder('${folder.id}')"
                            class="p-2 text-muted hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors shrink-0"
                            aria-label="Hapus folder">
                        <span class="material-symbols-outlined text-[20px] block">delete</span>
                    </button>
                </div>
            </div>
        `;
    }
}

function openFolder(folderId, folderName = 'Folder') {
    navigateToFolder(folderId, folderName);
}

function navigateToFolder(folderId, folderName) {
    folderPath.push({ id: folderId, name: folderName });
    currentFolderId = folderId;
    loadRoomContent();
    renderBreadcrumb();
}

function navigateBack() {
    if (folderPath.length > 0) {
        folderPath.pop();
        currentFolderId = folderPath.length > 0 ? folderPath[folderPath.length - 1].id : null;
        loadRoomContent();
        renderBreadcrumb();
    }
}

function navigateToRoot() {
    folderPath = [];
    currentFolderId = null;
    loadRoomContent();
    renderBreadcrumb();
}

function renderBreadcrumb() {
    const navBar = document.getElementById('folder-nav-bar');
    const pathDisplay = document.getElementById('folder-path-display');
    
    if (folderPath.length === 0) {
        navBar.classList.add('hidden');
        return;
    }
    
    navBar.classList.remove('hidden');
    
    let pathHTML = `
        <button onclick="navigateToRoot()" class="text-brand-600 hover:text-brand-700 font-semibold">
            <span class="material-symbols-outlined text-lg align-middle">home</span>
        </button>
    `;
    
    folderPath.forEach((folder, index) => {
        const isLast = index === folderPath.length - 1;
        if (isLast) {
            pathHTML += `
                <span class="text-gray-400">/</span>
                <span class="text-gray-800 font-semibold">${folder.name}</span>
            `;
        } else {
            pathHTML += `
                <span class="text-gray-400">/</span>
                <button onclick="navigateToFolder('${folder.id}', '${folder.name}')" 
                        class="text-brand-600 hover:text-brand-700">${folder.name}</button>
            `;
        }
    });
    
    pathDisplay.innerHTML = pathHTML;
}

async function deleteFolder(folderId) {
    showConfirmModal('Hapus folder ini? Semua file dan subfolder di dalamnya akan ikut terhapus.', 'Hapus', async () => {
    
    try {
        await api.del(`/api/folder/${folderId}`);
        
        await loadTotalFileCount();
        await loadRoomContent();
        updateTotalFileDisplay();
    } catch (error) {
        console.error('Delete folder error:', error);
        showInfoModal('Gagal menghapus folder');
    }
    });
}

function openCreateFolderModal() {
    document.getElementById('create-folder-modal').classList.remove('hidden');
    document.getElementById('folder-name-input').value = '';
    document.getElementById('folder-name-input').focus();
}

function closeCreateFolderModal() {
    document.getElementById('create-folder-modal').classList.add('hidden');
}


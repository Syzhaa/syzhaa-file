// Media Preview Enhancement for AmbilFile
// Auto-enhances file list with card layout and image thumbnails

(function() {
    'use strict';
    
    // Configuration
    const IMAGE_EXTENSIONS = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico'];
    const VIDEO_EXTENSIONS = ['mp4', 'webm', 'mov', 'avi', 'mkv'];
    
    // Check if file is image
    function isImage(filename) {
        const ext = filename.split('.').pop().toLowerCase();
        return IMAGE_EXTENSIONS.includes(ext);
    }
    
    // Check if file is video
    function isVideo(filename) {
        const ext = filename.split('.').pop().toLowerCase();
        return VIDEO_EXTENSIONS.includes(ext);
    }
    
    // Get file icon emoji
    function getFileIcon(filename, mimetype) {
        if (isImage(filename)) return '🖼️';
        if (isVideo(filename)) return '🎥';
        if (mimetype.includes('pdf')) return '📄';
        if (mimetype.includes('zip') || mimetype.includes('rar')) return '🗜️';
        if (mimetype.includes('audio')) return '🎵';
        if (mimetype.includes('text')) return '📝';
        return '📦';
    }
    
    // Format file size
    function formatSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
    }
    
    // Create media card HTML
    function createMediaCard(file) {
        const icon = getFileIcon(file.original_name, file.mimetype);
        const isImageFile = isImage(file.original_name);
        const isVideoFile = isVideo(file.original_name);
        const downloadUrl = `/d/${file.id}`;
        
        const previewHtml = isImageFile 
            ? `<div class="aspect-video bg-slate-200 rounded overflow-hidden mb-3">
                   <img src="${downloadUrl}" alt="${file.original_name}" 
                        class="w-full h-full object-cover hover:scale-105 transition-transform"
                        onerror="this.parentElement.innerHTML='<div class=\\'flex items-center justify-center h-full text-6xl\\'>${icon}</div>'">
               </div>`
            : isVideoFile
            ? `<div class="aspect-video bg-slate-900 rounded overflow-hidden mb-3 flex items-center justify-center text-6xl">
                   🎥
               </div>`
            : `<div class="aspect-video bg-slate-100 rounded overflow-hidden mb-3 flex items-center justify-center text-6xl">
                   ${icon}
               </div>`;
        
        return `
            <div class="bg-white brutal-border brutal-shadow rounded-lg overflow-hidden hover:translate-x-1 hover:translate-y-1 hover:shadow-none transition-all duration-150">
                ${previewHtml}
                <div class="p-4">
                    <p class="font-semibold text-slate-900 truncate mb-2" title="${file.original_name}">
                        ${file.original_name}
                    </p>
                    <div class="flex items-center justify-between text-sm text-slate-600 mb-3">
                        <span>${formatSize(file.size)}</span>
                        <span>↓ ${file.downloads || 0}</span>
                    </div>
                    <div class="flex gap-2">
                        <a href="${downloadUrl}" 
                           class="flex-1 px-3 py-2 bg-slate-900 text-white text-center font-semibold rounded brutal-border hover:bg-slate-800 text-sm">
                            Download
                        </a>
                        <button onclick="deleteFile('${file.id}')"
                                class="px-3 py-2 bg-red-500 text-white font-semibold rounded brutal-border hover:bg-red-600 text-sm">
                            🗑️
                        </button>
                    </div>
                </div>
            </div>
        `;
    }
    
    // Override original renderFiles function if exists
    const originalRenderFiles = window.renderFiles;
    
    window.renderFiles = function(files) {
        const container = document.getElementById('file-list');
        
        if (!files || files.length === 0) {
            container.innerHTML = '<p class="text-slate-600 text-center py-8">Belum ada file. Upload yang pertama!</p>';
            return;
        }
        
        // Check if card view enabled (can be toggled later)
        const useCardView = localStorage.getItem('fileViewMode') !== 'list';
        
        if (useCardView) {
            // Card grid layout
            container.className = 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6';
            container.innerHTML = files.map(file => createMediaCard(file)).join('');
        } else {
            // Fallback to original list view if defined
            if (originalRenderFiles) {
                originalRenderFiles(files);
            } else {
                // Simple list fallback
                container.className = 'space-y-4';
                container.innerHTML = files.map(file => `
                    <div class="flex items-center justify-between p-4 bg-white brutal-border rounded">
                        <div>
                            <p class="font-semibold text-slate-900">${file.original_name}</p>
                            <p class="text-sm text-slate-600">${formatSize(file.size)} • ${file.downloads || 0} downloads</p>
                        </div>
                        <div class="flex gap-2">
                            <a href="/d/${file.id}" class="px-4 py-2 bg-slate-900 text-white font-semibold rounded">Download</a>
                            <button onclick="deleteFile('${file.id}')" class="px-4 py-2 bg-red-500 text-white font-semibold rounded">Delete</button>
                        </div>
                    </div>
                `).join('');
            }
        }
    };
    
    // Add view toggle button if not exists
    function addViewToggle() {
        const existingToggle = document.getElementById('viewToggleBtn');
        if (existingToggle) return;
        
        const fileListContainer = document.getElementById('file-list-container');
        if (!fileListContainer) return;
        
        const toggleBtn = document.createElement('button');
        toggleBtn.id = 'viewToggleBtn';
        toggleBtn.className = 'mb-4 px-4 py-2 bg-slate-200 text-slate-900 font-semibold rounded brutal-border';
        toggleBtn.textContent = localStorage.getItem('fileViewMode') === 'list' ? '🎴 Card View' : '📋 List View';
        toggleBtn.onclick = function() {
            const current = localStorage.getItem('fileViewMode');
            const newMode = current === 'list' ? 'card' : 'list';
            localStorage.setItem('fileViewMode', newMode);
            toggleBtn.textContent = newMode === 'list' ? '🎴 Card View' : '📋 List View';
            
            // Re-render with new mode
            if (window.currentRoomFiles) {
                window.renderFiles(window.currentRoomFiles);
            }
        };
        
        fileListContainer.insertBefore(toggleBtn, fileListContainer.firstChild);
    }
    
    // Initialize when DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', addViewToggle);
    } else {
        addViewToggle();
    }
    
    console.log('✅ Media Preview Enhancement loaded');
})();

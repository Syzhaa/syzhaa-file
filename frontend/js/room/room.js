// Auto-split from room.html inline script. Shared globals via window scope.

let roomId = null;
let currentRoom = null;
let currentFiles = [];
let totalFilesInRoom = 0;
let currentView = 'list'; // 'list' or 'card'
let uploadAbortController = null;
let uploadStartTime = null;
let uploadedBytes = 0;
let countdownInterval = null;

// Load room on page load
window.addEventListener('DOMContentLoaded', async () => {
    const urlParams = new URLSearchParams(window.location.search);
    roomId = urlParams.get('id');
    
    if (!roomId) {
        showInfoModal('Room ID tidak ditemukan');
        window.location.href = '/';
        return;
    }
    
    await loadRoom(roomId);
});





// Generic info modal — pengganti alert()


// Generic confirm modal — pengganti confirm()
let confirmCallback = null;
document.addEventListener('DOMContentLoaded', () => {
    const yesBtn = document.getElementById('confirm-yes-btn');
    if (yesBtn) {
        yesBtn.addEventListener('click', () => {
            const cb = confirmCallback;
            closeConfirmModal();
            if (cb) cb();
        });
    }
});











// Countdown Timer


// Download All as ZIP
let downloadAbortController = null;






// Drag & Drop




// Upload files








// ============================================
// MULTI-SELECT FUNCTIONALITY
// ============================================

let selectedFiles = new Set();






// ============================================
// FOLDER FUNCTIONALITY
// ============================================

let currentFolderId = null;
let currentFolders = [];
let folderPath = []; // [{id: 'xxx', name: 'Folder Name'}]




















// Legacy: keep for compatibility












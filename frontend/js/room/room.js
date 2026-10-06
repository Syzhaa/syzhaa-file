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

// Load room info + files. (Restored: hilang saat refactor 3B.)
async function loadRoom(roomId) {
    try {
        const data = await api.get(`/api/room/${roomId}`);

        if (!data.room) {
            showErrorModal(
                'Room Tidak Ditemukan',
                'Room mungkin sudah kadaluarsa atau tidak pernah ada.'
            );
            return;
        }

        currentRoom = data.room;
        currentRoom.permission = data.permission || 'both';
        currentRoom.allow_delete = data.allow_delete !== false;
        // Update quota label
        if (data.quota_info && data.quota_info.label) {
            const ql = document.getElementById('quota-label');
            if (ql) ql.textContent = 'Maksimal ' + data.quota_info.label + ' total per ruangan';
        }
        await loadTotalFileCount();
        renderRoom();
        await loadRoomContent();
    } catch (error) {
        console.error('Error loading room:', error);
        showErrorModal(
            'Gagal Memuat Room',
            'Terjadi kesalahan saat memuat room. Silakan coba lagi.'
        );
    }
}

async function loadTotalFileCount() {
    try {
        const data = await api.get(`/api/room/${roomId}`);
        totalFilesInRoom = (data.files && data.files.length) || 0;
    } catch (error) {
        console.error('Error loading total file count:', error);
        totalFilesInRoom = 0;
    }
}





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












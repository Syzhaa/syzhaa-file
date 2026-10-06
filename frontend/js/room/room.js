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
let homeDestination = '/'; // tujuan tombol home: '/' anonim, '/user' atau '/admin' kalau login

// Kalau sudah login, tombol home di room mengarah ke dashboard
// (user -> /user, admin -> /admin). Non-login tetap ke landing page.
async function updateHomeLinks() {
    try {
        const data = await api.get('/user/me', { noRedirect: true });
        if (data && data.user && !data.error) homeDestination = '/user';
    } catch { /* bukan sesi user */ }
    if (homeDestination === '/') {
        try {
            const adm = await api.get('/admin/me', { noRedirect: true });
            if (adm && adm.admin && !adm.error) homeDestination = '/admin';
        } catch { /* tetap landing */ }
    }
    document.querySelectorAll('[data-home-link]').forEach(a => { a.href = homeDestination; });
}

// Load room on page load
window.addEventListener('DOMContentLoaded', async () => {
    const urlParams = new URLSearchParams(window.location.search);
    roomId = urlParams.get('id');
    
    if (!roomId) {
        showInfoModal('Room ID tidak ditemukan');
        window.location.href = '/';
        return;
    }
    
    updateHomeLinks(); // jalan paralel, tidak menghambat load room
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
        // Simpan info kuota di currentRoom supaya renderRoom() bisa pakai
        // (jangan update DOM di sini — renderRoom() me-render ulang semuanya)
        currentRoom.quota_label = (data.quota_info && data.quota_info.label) ? data.quota_info.label : '1 GB';
        currentRoom.quota_unlimited = !!(data.quota_info && data.quota_info.unlimited);
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












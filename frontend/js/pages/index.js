// Extracted from index.html


// Modal
function openModal(name) {
    document.getElementById('mobile-menu').classList.remove('open');
    document.getElementById('modal-' + name).classList.add('active');
}
function closeModal(name) {
    document.getElementById('modal-' + name).classList.remove('active');
}
document.addEventListener('click', (e) => {
    if (e.target.classList.contains('modal')) e.target.classList.remove('active');
});
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') document.querySelectorAll('.modal.active').forEach(m => m.classList.remove('active'));
});

// Info modal — pengganti alert()
function showInfoModal(message, success = false) {
    const badge = document.getElementById('info-icon');
    const glyph = document.getElementById('info-icon-glyph');
    if (success) {
        badge.className = 'mx-auto w-12 h-12 rounded-xl flex items-center justify-center bg-green-50 border border-green-100 text-green-500';
        glyph.textContent = 'check_circle';
    } else {
        badge.className = 'mx-auto w-12 h-12 rounded-xl flex items-center justify-center bg-red-50 border border-red-100 text-red-500';
        glyph.textContent = 'error';
    }
    document.getElementById('info-text').textContent = message;
    openModal('info');
}

// Hamburger
document.getElementById('hamburger').addEventListener('click', () => {
    document.getElementById('mobile-menu').classList.toggle('open');
});
document.querySelectorAll('#mobile-menu a').forEach(a => {
    a.addEventListener('click', () => document.getElementById('mobile-menu').classList.remove('open'));
});

// Create Room
async function createRoom() {
    const btn = document.getElementById('createRoomBtn');
    const expiryValue = parseInt(document.getElementById('expiryValue').value);
    const expiryUnit = parseInt(document.getElementById('expiryUnit').value);
    const expiryMinutes = expiryValue * expiryUnit;
    if (!expiryMinutes || expiryMinutes < 10 || expiryMinutes > 10080) {
        showInfoModal('Durasi tidak valid (10 menit - 7 hari)');
        return;
    }
    btn.disabled = true;
    btn.textContent = 'Membuat...';
    try {
        const response = await api.post('/api/room/create', { expiry_minutes: expiryMinutes });
        const data = await response.json();
        if (data.success) {
            if (data.owner_token) {
                localStorage.setItem('room_owner_token_' + data.room_id, data.owner_token);
            }
            window.location.href = '/room.html?id=' + data.room_id;
        } else {
            showInfoModal('Gagal membuat ruangan: ' + (data.error || 'Unknown error'));
        }
    } catch (error) {
        console.error('Error creating room:', error);
        showInfoModal('Terjadi kesalahan saat membuat ruangan');
    }
    btn.disabled = false;
    btn.textContent = 'Buat Ruangan';
}

// Join Room with PIN
async function joinRoom() {
    const pin = document.getElementById('pinInput').value.trim();
    if (pin.length !== 6 || !/^\d+$/.test(pin)) {
        showInfoModal('PIN harus 6 digit angka');
        return;
    }
    try {
        const response = await api.post('/api/room/pin', { pin });
        const data = await response.json();
        if (data.success) {
            if (data.owner_token) {
                localStorage.setItem('room_owner_token_' + data.room_id, data.owner_token);
            }
            window.location.href = '/room.html?id=' + data.room_id;
        } else {
            showInfoModal('PIN tidak valid atau ruangan sudah kadaluarsa');
        }
    } catch (error) {
        console.error('Error joining room:', error);
        showInfoModal('Terjadi kesalahan saat join ruangan');
    }
}

// Session check: kalau sudah login, langsung ke dashboard
async function checkUserSession() {
    try {
        const response = await api.get('/user/me');
        if (response.ok) {
            const data = await response.json();
            if (data.user && !data.error) {
                window.location.href = '/user/';
                return;
            }
        }
    } catch (error) { /* tetap tampil landing */ }
    // Tidak login: update tombol jadi Dashboard link (fallback)
    try {
        const btn = document.getElementById('auth-button');
        if (btn) {
            btn.textContent = 'Masuk';
            btn.onclick = () => window.location.href = '/user-login.html';
        }
    } catch {}
}
checkUserSession();

// Live stats
async function loadStats() {
    try {
        const res = await api.get('/api/stats');
        if (!res.ok) return;
        const s = await res.json();
        const totalRooms = (s.total_rooms || 0) + (s.deleted_rooms || 0);
        const totalFiles = (s.total_files || 0) + (s.deleted_files || 0);
        const totalBytes = (s.total_bytes || 0) + (s.deleted_bytes || 0);
        document.getElementById('stat-rooms').textContent = totalRooms.toLocaleString('id-ID');
        document.getElementById('stat-files').textContent = totalFiles.toLocaleString('id-ID');
        document.getElementById('stat-size').textContent = formatBytes(totalBytes);
        document.getElementById('live-stats').style.display = 'flex';
    } catch {}
}
function formatBytes(b) {
    if (b < 1024) return b + ' B';
    const u = ['KB','MB','GB','TB'];
    let i = -1;
    do { b /= 1024; i++; } while (b >= 1024 && i < u.length - 1);
    return b.toFixed(1) + ' ' + u[i];
}
loadStats();

// Service worker
if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
        navigator.serviceWorker.register('/sw.js').catch(err => console.warn('SW failed:', err));
    });
}

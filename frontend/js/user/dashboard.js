// User dashboard — ringkasan statistik saja.
// Daftar ruangan pindah ke halaman khusus: /user/rooms.html

function showInfoModal(message, success = false) {
    const icon = document.getElementById('info-modal-icon');
    const glyph = document.getElementById('info-modal-glyph');
    icon.className = 'modal-icon ' + (success ? 'ok' : 'err');
    glyph.textContent = success ? 'check_circle' : 'error';
    document.getElementById('info-modal-msg').textContent = message;
    document.getElementById('info-modal').classList.add('open');
}
function closeInfoModal() {
    document.getElementById('info-modal').classList.remove('open');
}

let confirmCallback = null;
function showConfirmModal(message, onYes) {
    document.getElementById('confirm-modal-msg').textContent = message;
    confirmCallback = onYes;
    document.getElementById('confirm-modal').classList.add('open');
}
function closeConfirmModal() {
    document.getElementById('confirm-modal').classList.remove('open');
    confirmCallback = null;
}
document.getElementById('confirm-modal-yes').addEventListener('click', () => {
    const cb = confirmCallback;
    closeConfirmModal();
    if (cb) cb();
});

function setText(id, val) {
    const el = document.getElementById(id);
    if (el) el.textContent = val;
}

async function load() {
    try {
        const me = await api('/user/me');
        const u = me.user;
        setText('user-name', u.name);
        setText('side-user-name', u.name);
        setText('greeting', 'Halo, ' + u.name);
    } catch { return; }

    try {
        const keys = await api('/user/api-keys');
        setText('stat-keys', (keys.api_keys || []).length);
    } catch {}

    try {
        const rooms = await api('/user/rooms');
        const list = rooms.rooms || [];
        const now = new Date();
        const active = list.filter(r => new Date(r.expires_at) > now).length;
        setText('stat-rooms', list.length);
        setText('stat-rooms-active', active);
        setText('stat-rooms-expired', list.length - active);
    } catch {}
}

async function logout() {
    await api.post('/auth/user/logout');
    window.location.href = '/';
}

function openCreateRoomSheet() {
    document.getElementById('create-room-sheet').classList.add('open');
}

function closeCreateRoomSheet() {
    document.getElementById('create-room-sheet').classList.remove('open');
}

async function submitCreateRoomSheet() {
    const btn = document.getElementById('sheet-create-room-btn');
    const val = parseInt(document.getElementById('sheet-expiry-value').value);
    const unit = parseInt(document.getElementById('sheet-expiry-unit').value);
    const minutes = val * unit;
    if (!minutes || minutes < 10 || minutes > 10080) {
        showInfoModal('Durasi tidak valid (10 menit - 7 hari)');
        return;
    }
    btn.disabled = true;
    btn.textContent = 'Membuat...';
    try {
        const data = await api.post('/api/room/create', { expiry_minutes: minutes });
        if (data.room_id) {
            if (data.owner_token) {
                localStorage.setItem('room_owner_token_' + data.room_id, data.owner_token);
            }
            window.location.href = '/room.html?id=' + data.room_id;
        } else {
            showInfoModal('Gagal membuat ruangan');
            btn.disabled = false;
            btn.textContent = 'Buat Ruangan';
        }
    } catch {
        showInfoModal('Tidak bisa terhubung ke server');
        btn.disabled = false;
        btn.textContent = 'Buat Ruangan';
    }
}

load();

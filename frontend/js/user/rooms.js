// Ruangan Saya — halaman khusus daftar room user.

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

function esc(s) { const d = document.createElement('div'); d.textContent = s; return d.innerHTML; }

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
        const rooms = await api('/user/rooms');
        const list = rooms.rooms || [];
        window._allRooms = list;
        const now = new Date();
        const active = list.filter(r => new Date(r.expires_at) > now).length;
        setText('stat-rooms', list.length);
        setText('stat-rooms-active', active);
        setText('stat-rooms-expired', list.length - active);
        renderRooms(window._roomFilter || 'all');
    } catch {
        setText('rooms-list', '');
        document.getElementById('rooms-list').innerHTML = '<div class="empty">Gagal memuat ruangan.</div>';
    }
}

function renderRooms(filter) {
    window._roomFilter = filter;
    const list = window._allRooms || [];
    const now = new Date();
    let filtered = list;
    if (filter === 'active') filtered = list.filter(r => new Date(r.expires_at) > now);
    if (filter === 'expired') filtered = list.filter(r => new Date(r.expires_at) <= now);

    document.querySelectorAll('#room-filter .filter-btn').forEach(b => {
        const isActive = b.dataset.filter === filter;
        b.style.background = isActive ? '#1a1a2e' : '#fff';
        b.style.color = isActive ? '#fff' : '#6b7280';
    });

    const rl = document.getElementById('rooms-list');
    if (!rl) return;
    if (!filtered.length) {
        const msg = filter === 'all' ? 'Belum ada ruangan. <a href="/user/create-room.html" style="color:#F6821F;font-weight:600">Buat ruangan</a> dulu.'
            : filter === 'active' ? 'Tidak ada room aktif.'
            : 'Tidak ada room kadaluarsa.';
        rl.innerHTML = '<div class="empty">' + msg + '</div>';
    } else {
        rl.innerHTML = filtered.map(rm => {
            const isActive = new Date(rm.expires_at) > now;
            const badge = isActive
                ? '<span style="font-size:11px;font-weight:700;padding:2px 8px;border-radius:99px;background:#dcfce7;color:#15803d;">Aktif</span>'
                : '<span style="font-size:11px;font-weight:700;padding:2px 8px;border-radius:99px;background:#f1f2f6;color:#9ca3af;">Kadaluarsa</span>';
            return `<div class="roomrow">
                <span class="material-symbols-outlined" style="color:#9ca3af">folder</span>
                <div style="flex:1">
                    <div class="pin">${esc(rm.pin)} ${badge}</div>
                    <div class="meta">Kadaluarsa ${new Date(rm.expires_at).toLocaleString('id-ID')}</div>
                </div>
                ${isActive ? `<a href="/room.html?id=${esc(rm.id)}">Buka</a>` : ''}
                <button onclick="deleteRoom('${esc(rm.id)}')" title="Hapus" style="border:0;background:#fee2e2;color:#dc2626;border-radius:8px;padding:6px 8px;cursor:pointer;display:inline-flex;align-items:center;font-family:inherit;">
                    <span class="material-symbols-outlined" style="font-size:18px">delete</span>
                </button>
            </div>`;
        }).join('');
    }
}

function filterRooms(f) { renderRooms(f); }

function deleteRoom(roomId) {
    showConfirmModal('Hapus room ini? Semua file di dalamnya akan ikut terhapus dan tidak bisa dikembalikan.', async () => {
        try {
            const headers = {};
            const ownerToken = localStorage.getItem('room_owner_token_' + roomId);
            if (ownerToken) headers['X-Room-Token'] = ownerToken;
            const res = await api('/api/room/' + roomId, { method: 'DELETE', headers: headers });
            if (!res) throw new Error('gagal');
            window._allRooms = (window._allRooms || []).filter(r => r.id !== roomId);
            renderRooms(window._roomFilter || 'all');
            const list = window._allRooms;
            const now = new Date();
            const active = list.filter(r => new Date(r.expires_at) > now).length;
            setText('stat-rooms', list.length);
            setText('stat-rooms-active', active);
            setText('stat-rooms-expired', list.length - active);
            showInfoModal('Room berhasil dihapus.', true);
        } catch {
            showInfoModal('Gagal menghapus room.');
        }
    });
}

async function logout() {
    await api.post('/auth/user/logout');
    window.location.href = '/';
}

load();

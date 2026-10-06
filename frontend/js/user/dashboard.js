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

load();

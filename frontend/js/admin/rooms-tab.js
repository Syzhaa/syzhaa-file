// Admin dashboard module. Shared globals via window scope.

async function createRoom() {
    const expiryValue = parseInt(document.getElementById('expiryValue').value);
    const expiryUnit = parseInt(document.getElementById('expiryUnit').value);
    const expiryMinutes = expiryValue * expiryUnit;
    
    if (expiryMinutes < 10 || expiryMinutes > 10080) {
        showToast('Invalid expiry duration (10 min - 7 days)', 'error');
        return;
    }
    
    const data = await apiCall('/api/v1/rooms', {
        method: 'POST',
        body: JSON.stringify({
            expiry_minutes: expiryMinutes
        })
    });
    
    if (data && data.success) {
        showToast('Room created successfully!', 'success');
        closeModal('createRoom');
        
        // Show room details in modal
        const roomLink = `https://ambilfile.web.id/?room=${data.room_id}`;
        const pinLink = `https://ambilfile.web.id/?pin=${data.pin}`;
        
        document.getElementById('roomPinDisplay').value = data.pin;
        document.getElementById('roomLinkDisplay').value = roomLink;
        document.getElementById('pinLinkDisplay').value = pinLink;
        openModal('roomCreated');
        
        loadStats();
    } else {
        showToast('Failed to create room', 'error');
    }
}

async function joinRoom() {
    const pin = document.getElementById('pinInput').value.trim();
    
    if (pin.length !== 6 || !/^\d+$/.test(pin)) {
        showToast('PIN harus 6 digit angka', 'error');
        return;
    }
    
    try {
        const data = await api.post('/api/room/pin', { pin });
        
        if (data.success) {
            closeModal('joinPin');
            window.location.href = `/room.html?id=${data.room_id}`;
        } else {
            showToast('PIN tidak valid atau ruangan sudah kadaluarsa', 'error');
        }
    } catch (error) {
        console.error('Error joining room:', error);
        showToast('Terjadi kesalahan saat join ruangan', 'error');
    }
}

async function createAdminRoom() {
    const btn = document.getElementById('admin-create-room-btn');
    const val = parseInt(document.getElementById('admin-expiry-value').value);
    const unit = parseInt(document.getElementById('admin-expiry-unit').value);
    const minutes = val * unit;
    if (!minutes || minutes < 10 || minutes > 10080) {
        showToast('Durasi tidak valid (10 menit - 7 hari)', 'error');
        return;
    }
    btn.disabled = true;
    btn.textContent = 'Membuat...';
    try {
        const data = await apiCall('/api/room/create', {
            method: 'POST',
            body: JSON.stringify({ expiry_minutes: minutes })
        });
        if (data && data.room_id) {
            if (data.owner_token) {
                localStorage.setItem('room_owner_token_' + data.room_id, data.owner_token);
            }
            window.location.href = '/room.html?id=' + data.room_id;
        } else {
            showToast('Gagal membuat ruangan', 'error');
            btn.disabled = false;
            btn.textContent = 'Buat Ruangan';
        }
    } catch {
        showToast('Tidak bisa terhubung ke server', 'error');
        btn.disabled = false;
        btn.textContent = 'Buat Ruangan';
    }
}

async function loadAdminRooms() {
    try {
        const data = await apiCall('/admin/rooms');
        _adminRooms = (data && data.rooms) || [];
        renderAdminRooms();
    } catch {
        document.getElementById('admin-rooms-list').innerHTML = '<p class="text-outline">Gagal memuat.</p>';
    }
}

function renderAdminRooms() {
    const now = new Date();
    let list = _adminRooms;
    const f = _adminRoomFilter;
    if (f === 'active') list = list.filter(r => new Date(r.expires_at) > now);
    if (f === 'expired') list = list.filter(r => new Date(r.expires_at) <= now);
    if (f === 'mine') list = list.filter(r => r.is_admin);
    
    document.querySelectorAll('#admin-room-filter button').forEach(b => {
        const on = b.dataset.filter === f;
        b.style.background = on ? '#1a1a2e' : '#fff';
        b.style.color = on ? '#fff' : '#6b7280';
    });
    
    const el = document.getElementById('admin-rooms-list');
    if (!list.length) {
        el.innerHTML = '<p class="text-outline">Tidak ada ruangan.</p>';
        return;
    }
    el.innerHTML = list.map(r => {
        const active = new Date(r.expires_at) > now;
        const badge = active
            ? '<span class="text-xs font-bold px-2 py-1 rounded-full" style="background:#dcfce7;color:#15803d;">Aktif</span>'
            : '<span class="text-xs font-bold px-2 py-1 rounded-full" style="background:#f1f2f6;color:#9ca3af;">Kadaluarsa</span>';
        const owner = r.is_admin ? '<span class="text-xs font-bold px-2 py-1 rounded-full" style="background:#fff4e8;color:#F6821F;">Admin</span>' : '';
        return `<div class="bg-white p-4 rounded-2xl border border-outline-variant flex items-center gap-4">
            <span class="material-symbols-outlined" style="color:#9ca3af">folder</span>
            <div class="flex-1">
                <div class="font-mono font-bold" style="letter-spacing:2px">${r.pin} ${badge} ${owner}</div>
                <div class="text-xs text-outline">${r.file_count} file &middot; ${r.owner_email || 'Anonim'} &middot; Kadaluarsa ${new Date(r.expires_at).toLocaleString('id-ID')}</div>
            </div>
            ${active ? `<a href="/room.html?id=${r.id}" class="text-sm font-semibold" style="color:#F6821F">Buka</a>` : ''}
            <button onclick="deleteAdminRoom('${r.id}')" class="text-sm font-semibold px-3 py-1.5 rounded-lg" style="background:#fee2e2;color:#dc2626;border:0;cursor:pointer;">Hapus</button>
        </div>`;
    }).join('');
}

function filterAdminRooms(f) {
    _adminRoomFilter = f;
    renderAdminRooms();
}

async function deleteAdminRoom(roomId) {
    if (!confirm('Hapus room ini? Semua file di dalamnya akan ikut terhapus dan tidak bisa dikembalikan.')) return;
    try {
        const headers = {};
        const ownerToken = localStorage.getItem('room_owner_token_' + roomId);
        if (ownerToken) headers['X-Room-Token'] = ownerToken;
        const res = await api('/api/room/' + roomId, { method: 'DELETE', headers: headers });
        if (!res.ok) throw new Error('gagal');
        _adminRooms = _adminRooms.filter(r => r.id !== roomId);
        renderAdminRooms();
        alert('Room berhasil dihapus.');
    } catch {
        alert('Gagal menghapus room.');
    }
}


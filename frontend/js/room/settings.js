// Auto-split from room.html inline script. Shared globals via window scope.

async function setRoomPermission(perm) {
    try {
        const res = await fetch(`/api/room/${currentRoom.id}/settings`, {
            method: 'PUT',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({permission: perm})
        });
        if (res.ok) {
            currentRoom.permission = perm;
            renderShareSettings();
            await loadRoomContent();
        }
    } catch (e) {
        showInfoModal('Gagal memperbarui izin');
    }
}

async function toggleAllowDelete() {
    try {
        const newVal = !(currentRoom.allow_delete !== false);
        const res = await fetch(`/api/room/${currentRoom.id}/settings`, {
            method: 'PUT',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({allow_delete: newVal})
        });
        if (res.ok) {
            currentRoom.allow_delete = newVal;
            renderShareSettings();
            await loadRoomContent();
        }
    } catch (e) {
        showInfoModal('Gagal memperbarui pengaturan');
    }
}

function openShareModal() {
    if (!currentRoom) return;
    document.getElementById('share-link-input').value = window.location.href;
    
    const pin = currentRoom.pin || 'N/A';
    const expiry = currentRoom.expires_at ? new Date(currentRoom.expires_at).toLocaleString('id-ID') : 'N/A';
    document.getElementById('share-room-info').innerHTML = `
        <p><span class="font-medium text-ink">PIN:</span> ${pin}</p>
        <p><span class="font-medium text-ink">Kadaluarsa:</span> ${expiry}</p>
    `;
    
    renderShareSettings();
    document.getElementById('share-modal').classList.remove('hidden');
}

function closeShareModal() {
    document.getElementById('share-modal').classList.add('hidden');
}

function renderShareSettings() {
    const perm = currentRoom.permission || 'both';
    const allowDel = currentRoom.allow_delete !== false;
    const canDownload = perm !== 'view';
    
    // Permission buttons
    const permBtns = document.getElementById('share-perm-btns');
    const perms = [
        {val: 'both', label: 'Lihat + Download'},
        {val: 'view', label: 'Lihat saja'},
        {val: 'download', label: 'Download saja'},
    ];
    permBtns.innerHTML = perms.map(p => `
        <button onclick="setRoomPermission('${p.val}')"
                class="px-4 py-2 rounded-lg text-sm font-medium transition-colors ${perm === p.val ? 'bg-brand-500 text-white' : 'bg-slate-100 text-ink'}">
            ${p.label}
        </button>
    `).join('');
    
    // Delete toggle
    const delToggle = document.getElementById('share-delete-toggle');
    const delKnob = document.getElementById('share-delete-knob');
    delToggle.className = `w-12 h-7 rounded-full transition-colors relative shrink-0 ${allowDel ? 'bg-brand-500' : 'bg-slate-300'}`;
    delKnob.className = `absolute top-1 w-5 h-5 bg-white rounded-full shadow transition-all ${allowDel ? 'right-1' : 'left-1'}`;
    
    // Download toggle
    const dlToggle = document.getElementById('share-download-toggle');
    const dlKnob = document.getElementById('share-download-knob');
    dlToggle.className = `w-12 h-7 rounded-full transition-colors relative shrink-0 ${canDownload ? 'bg-brand-500' : 'bg-slate-300'}`;
    dlKnob.className = `absolute top-1 w-5 h-5 bg-white rounded-full shadow transition-all ${canDownload ? 'right-1' : 'left-1'}`;
}

async function copyShareLink() {
    const input = document.getElementById('share-link-input');
    try {
        await navigator.clipboard.writeText(input.value);
    } catch (err) {
        input.select();
        document.execCommand('copy');
    }
    closeShareModal();
    showInfoModal('Link ruangan berhasil disalin ke clipboard!', true);
}

async function toggleDownloadPermission() {
    const current = currentRoom.permission || 'both';
    // Toggle between 'both' and 'view'
    const newPerm = current === 'view' ? 'both' : 'view';
    await setRoomPermission(newPerm);
}

async function copyRoomLink() {
    openShareModal();
}

function showRoomInfo() {
    openShareModal();
}


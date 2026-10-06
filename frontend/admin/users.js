// Admin User Management JS
// Generic info modal — pengganti alert()
function showInfoModal(message, success = false) {
    const box = document.getElementById('infoModalIcon');
    const glyph = document.getElementById('infoModalGlyph');
    if (success) {
        box.className = 'w-12 h-12 rounded-xl flex items-center justify-center mx-auto mb-4 bg-green-50 border border-green-100 text-green-500';
        glyph.textContent = 'check_circle';
    } else {
        box.className = 'w-12 h-12 rounded-xl flex items-center justify-center mx-auto mb-4 bg-red-50 border border-red-100 text-red-500';
        glyph.textContent = 'error';
    }
    document.getElementById('infoModalMsg').textContent = message;
    document.getElementById('infoModal').classList.remove('hidden');
}
function closeInfoModal() {
    document.getElementById('infoModal').classList.add('hidden');
}

// Generic confirm modal — pengganti confirm()
let confirmCallback = null;
function showConfirmModal(message, yesLabel, onYes) {
    document.getElementById('confirmModalMsg').textContent = message;
    document.getElementById('confirmModalYes').textContent = yesLabel || 'Ya';
    confirmCallback = onYes;
    document.getElementById('confirmModal').classList.remove('hidden');
}
function closeConfirmModal() {
    document.getElementById('confirmModal').classList.add('hidden');
    confirmCallback = null;
}
document.getElementById('confirmModalYes').addEventListener('click', () => {
    const cb = confirmCallback;
    closeConfirmModal();
    if (cb) cb();
});

// Generic prompt modal — pengganti prompt()
let promptCallback = null;
function showPromptModal(message, placeholder, onOk) {
    document.getElementById('promptModalMsg').textContent = message;
    const input = document.getElementById('promptModalInput');
    input.value = '';
    input.placeholder = placeholder || '';
    promptCallback = onOk;
    document.getElementById('promptModal').classList.remove('hidden');
    setTimeout(() => input.focus(), 50);
}
function closePromptModal() {
    document.getElementById('promptModal').classList.add('hidden');
    promptCallback = null;
}
document.getElementById('promptModalYes').addEventListener('click', () => {
    const cb = promptCallback;
    const val = document.getElementById('promptModalInput').value;
    closePromptModal();
    if (cb) cb(val);
});

let allUsers = [];
let currentFilter = 'all';

// Wrapper around global api() (from /js/api.js) preserving legacy null-on-error behavior.
async function apiCall(endpoint, options = {}) {
    try {
        return await api(endpoint, options);
    } catch (e) {
        return null;
    }
}

async function loadAdmin() {
    const data = await apiCall('/admin/me');
    if (data) {
        document.getElementById('adminName').textContent = data.name;
        document.getElementById('adminAvatar').src = data.avatar_url;
    }
}

async function loadUsers(status = '') {
    const endpoint = status ? `/admin/users?status=${status}` : '/admin/users';
    const data = await apiCall(endpoint);
    if (data && data.users) {
        allUsers = data.users;
        updateCounts();
        renderUsers();
    }
}

function updateCounts() {
    const counts = { all: allUsers.length, pending: 0, approved: 0, rejected: 0, suspended: 0 };
    allUsers.forEach(u => counts[u.status]++);
    Object.keys(counts).forEach(key => {
        const el = document.getElementById(`count-${key}`);
        if (el) el.textContent = counts[key];
    });
}

function filterUsers(status) {
    currentFilter = status;
    document.querySelectorAll('[id^="tab-"]').forEach(btn => btn.classList.remove('ring-4', 'ring-blue-300'));
    document.getElementById(`tab-${status}`).classList.add('ring-4', 'ring-blue-300');
    renderUsers();
}

function renderUsers() {
    const filtered = currentFilter === 'all' ? allUsers : allUsers.filter(u => u.status === currentFilter);
    const container = document.getElementById('usersList');
    
    if (filtered.length === 0) {
        container.innerHTML = '<p class="text-center text-slate-600 py-8">No users found</p>';
        return;
    }
    
    container.innerHTML = filtered.map(user => {
        const statusColors = {
            pending: 'bg-yellow-500',
            approved: 'bg-green-500',
            rejected: 'bg-red-500',
            suspended: 'bg-orange-500'
        };
        const statusColor = statusColors[user.status] || 'bg-slate-500';
        
        const storageLimit = user.storage_limit_mb || '<span class="text-slate-500">default</span>';
        const durationLimit = user.max_file_duration_days || '<span class="text-slate-500">default</span>';
        
        // API key access approval
        const apiApproved = !!user.api_approved;
        const apiRequested = !!user.api_requested_at;
        const apiBadge = apiApproved
            ? '<span class="px-2 py-1 rounded text-xs font-semibold text-white bg-green-600">API ✓</span>'
            : apiRequested
            ? '<span class="px-2 py-1 rounded text-xs font-semibold text-white bg-yellow-500">API ⏳</span>'
            : '<span class="px-2 py-1 rounded text-xs font-semibold text-white bg-slate-400">API —</span>';
        const apiAction = !apiApproved
            ? `<button onclick="approveUserAPI('${user.id}')" class="px-3 py-2 bg-blue-500 text-white font-semibold rounded brutal-border-thin hover:bg-blue-600 text-sm">Setujui API</button>`
            : `<button onclick="revokeUserAPI('${user.id}')" class="px-3 py-2 bg-slate-500 text-white font-semibold rounded brutal-border-thin hover:bg-slate-600 text-sm">Cabut API</button>`;
        
        const actions = user.status === 'pending' 
            ? `<button onclick="showApproveModal('${user.id}')" class="px-3 py-2 bg-green-500 text-white font-semibold rounded brutal-border-thin hover:bg-green-600 text-sm">Approve</button>
               <button onclick="showRejectModal('${user.id}')" class="px-3 py-2 bg-red-500 text-white font-semibold rounded brutal-border-thin hover:bg-red-600 text-sm">Reject</button>`
            : user.status === 'approved'
            ? `<button onclick="suspendUser('${user.id}')" class="px-3 py-2 bg-orange-500 text-white font-semibold rounded brutal-border-thin hover:bg-orange-600 text-sm">Suspend</button>`
            : '';
        
        return `
            <div class="bg-white brutal-border-thin rounded p-4 hover:bg-slate-50">
                <div class="flex items-center justify-between">
                    <div class="flex items-center gap-4 flex-1">
                        <img src="${user.avatar_url}" class="w-12 h-12 rounded-full brutal-border-thin" alt="">
                        <div class="flex-1">
                            <div class="flex items-center gap-3 mb-2">
                                <span class="font-bold text-slate-900">${user.name}</span>
                                <span class="px-2 py-1 rounded text-xs font-semibold text-white ${statusColor}">${user.status}</span>
                                ${apiBadge}
                            </div>
                            <p class="text-sm text-slate-600">${user.email}</p>
                            <div class="text-xs text-slate-500 mt-1">
                                Registered: ${new Date(user.created_at).toLocaleDateString('id-ID')}
                                ${user.last_login ? ` • Last login: ${new Date(user.last_login).toLocaleDateString('id-ID')}` : ''}
                            </div>
                        </div>
                    </div>
                    <div class="text-right mr-4">
                        <p class="text-sm text-slate-600">Storage: ${storageLimit} MB</p>
                        <p class="text-sm text-slate-600">Duration: ${durationLimit} days</p>
                    </div>
                    <div class="flex gap-2">
                        ${apiAction}
                        ${actions}
                    </div>
                </div>
            </div>
        `;
    }).join('');
}

function showApproveModal(userId) {
    document.getElementById('approve-user-id').value = userId;
    document.getElementById('approveModal').classList.remove('hidden');
}

function closeApproveModal() {
    document.getElementById('approveModal').classList.add('hidden');
    document.getElementById('approve-storage').value = '';
    document.getElementById('approve-duration').value = '';
}

async function confirmApprove() {
    const userId = document.getElementById('approve-user-id').value;
    const storage = document.getElementById('approve-storage').value;
    const duration = document.getElementById('approve-duration').value;
    
    const payload = {};
    if (storage) payload.storage_limit_mb = parseInt(storage);
    if (duration) payload.max_file_duration_days = parseInt(duration);
    
    const data = await apiCall(`/admin/users/${userId}/approve`, { method: 'POST', body: JSON.stringify(payload) });
    if (data && data.success) {
        closeApproveModal();
        await loadUsers();
        showInfoModal('User berhasil disetujui!', true);
    }
}

function showRejectModal(userId) {
    document.getElementById('reject-user-id').value = userId;
    document.getElementById('rejectModal').classList.remove('hidden');
}

function closeRejectModal() {
    document.getElementById('rejectModal').classList.add('hidden');
    document.getElementById('reject-reason').value = '';
}

async function confirmReject() {
    const userId = document.getElementById('reject-user-id').value;
    const reason = document.getElementById('reject-reason').value;
    
    const data = await apiCall(`/admin/users/${userId}/reject`, { 
        method: 'POST', 
        body: JSON.stringify({ reason: reason || 'Registration rejected by admin' })
    });
    
    if (data && data.success) {
        closeRejectModal();
        await loadUsers();
        showInfoModal('User ditolak.', true);
    }
}

async function approveUserAPI(userId) {
    const data = await apiCall(`/admin/users/${userId}/api-approve`, { method: 'PUT' });
    if (data && data.success) {
        await loadUsers();
        showInfoModal('Akses API key disetujui.', true);
    }
}

async function revokeUserAPI(userId) {
    showConfirmModal('Cabut akses API key user ini? Key yang sudah dibuat tetap aktif.', 'Cabut', async () => {
        const data = await apiCall(`/admin/users/${userId}/api-revoke`, { method: 'PUT' });
        if (data && data.success) {
            await loadUsers();
            showInfoModal('Akses API key dicabut.', true);
        }
    });
}

async function suspendUser(userId) {
    showConfirmModal('Suspend user ini? Dia tidak akan bisa login sampai diaktifkan lagi.', 'Suspend', () => {
        showPromptModal('Alasan suspend (opsional):', 'Mis. pelanggaran ketentuan', async (reason) => {
            const data = await apiCall(`/admin/users/${userId}/suspend`, {
                method: 'POST',
                body: JSON.stringify({ reason: reason || 'Akun di-suspend oleh admin' })
            });

            if (data && data.success) {
                await loadUsers();
                showInfoModal('User di-suspend.', true);
            }
        });
    });
}

async function loadSettings() {
    const data = await apiCall('/admin/settings');
    if (data && data.settings) {
        document.getElementById('default-storage').value = data.settings.default_storage_limit_mb?.value || '2048';
        document.getElementById('anonymous-storage').value = data.settings.anonymous_storage_limit_mb?.value || '1024';
        document.getElementById('default-duration').value = data.settings.default_max_duration_days?.value || '7';
        document.getElementById('require-approval').value = data.settings.require_approval?.value || 'true';
    }
}

async function saveSettings() {
    const settings = [
        { key: 'default_storage_limit_mb', value: document.getElementById('default-storage').value },
        { key: 'anonymous_storage_limit_mb', value: document.getElementById('anonymous-storage').value },
        { key: 'default_max_duration_days', value: document.getElementById('default-duration').value },
        { key: 'require_approval', value: document.getElementById('require-approval').value }
    ];
    
    for (const setting of settings) {
        await apiCall('/admin/settings', { method: 'POST', body: JSON.stringify(setting) });
    }
    
    showInfoModal('Pengaturan berhasil disimpan!', true);
}

async function init() {
    try {
        await Promise.all([loadAdmin(), loadUsers(), loadSettings()]);
        document.getElementById('loading').classList.add('hidden');
        document.getElementById('app').classList.remove('hidden');
    } catch (error) {
        console.error('Init error:', error);
        window.location.href = '/admin/login.html';
    }
}

setInterval(() => loadUsers(), 30000);
init();

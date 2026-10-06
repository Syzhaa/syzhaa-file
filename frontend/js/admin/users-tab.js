// Admin dashboard module. Shared globals via window scope.

async function loadUsers() {
    const data = await apiCall('/admin/users');
    if (data && data.success && data.users) {
        users = data.users;
        renderUsersTable();
    }
}

function renderUsersTable() {
    const tbody = document.getElementById('usersTableBody');
    if (!tbody) return;
    
    if (users.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="px-6 py-12 text-center text-outline">No users found</td></tr>';
        return;
    }
    
    tbody.innerHTML = users.map(user => `
        <tr class="border-b border-outline-variant hover:bg-surface-container/50">
            <td class="px-6 py-4">
                <div class="flex items-center gap-3">
                    <img src="${user.avatar_url || `https://ui-avatars.com/api/?name=${encodeURIComponent(user.name)}`}" 
                         class="w-10 h-10 rounded-full border-2 border-outline-variant">
                    <div>
                        <p class="font-semibold text-sm">${user.name}</p>
                        <p class="text-xs text-outline">${user.email}</p>
                    </div>
                </div>
            </td>
            <td class="px-6 py-4">
                <span class="px-3 py-1 rounded-full text-xs font-semibold ${
                    user.status === 'approved' || user.status === 'active' ? 'bg-green-100 text-green-700' :
                    user.status === 'pending' ? 'bg-yellow-100 text-yellow-700' :
                    user.status === 'rejected' ? 'bg-red-100 text-red-700' :
                    'bg-gray-100 text-gray-700'
                }">
                    ${user.status}
                </span>
                ${user.api_requested_at && !user.api_approved ? `
                    <span class="ml-1 px-2.5 py-1 rounded-full text-xs font-semibold bg-blue-100 text-blue-700">Minta API</span>
                ` : ''}
            </td>
            <td class="px-6 py-4 text-sm text-outline">${formatDate(user.created_at)}</td>
            <td class="px-6 py-4 text-sm text-outline">${user.storage_used_mb ? formatBytes(user.storage_used_mb * 1024 * 1024) : '0 MB'}</td>
            <td class="px-6 py-4 text-right">
                <div class="relative inline-block">
                    <button onclick="toggleUserMenu('${user.id}', event)" class="p-2 rounded-lg hover:bg-gray-100 text-gray-500" title="Aksi">
                        <span class="material-symbols-outlined">more_vert</span>
                    </button>
                    <div id="umenu-${user.id}" class="hidden absolute right-0 top-full mt-1 w-56 bg-white rounded-xl shadow-xl border border-outline-variant py-1.5 z-30 text-left">
                        ${userMenuItems(user)}
                    </div>
                </div>
            </td>
        </tr>
    `).join('');

    // Mobile cards
    renderUsersCards();
}

// Item menu aksi per user (dipakai tabel desktop & kartu mobile)
function userMenuItems(user) {
    const item = (fn, icon, label, color) => `
        <button onclick="${fn}('${user.id}')" class="w-full flex items-center gap-3 px-4 py-2.5 text-sm font-medium ${color} hover:bg-gray-50 text-left">
            <span class="material-symbols-outlined text-xl">${icon}</span>${label}
        </button>`;
    let html = '';
    if (user.status === 'pending') {
        html += item('approveUser', 'check_circle', 'Setujui User', 'text-green-600');
        html += item('rejectUser', 'cancel', 'Tolak User', 'text-red-600');
    }
    if (user.status === 'approved' || user.status === 'active') {
        if (user.api_requested_at && !user.api_approved) {
            html += item('approveUserAPI', 'key', 'Setujui API Key', 'text-blue-600');
        }
        if (user.api_approved) {
            html += item('revokeUserAPI', 'key_off', 'Cabut Akses API', 'text-gray-600');
        }
        html += item('editUserQuota', 'storage', 'Edit Kuota', 'text-gray-700');
        html += item('suspendUser', 'block', 'Suspend User', 'text-orange-600');
    }
    if (!html) html = '<p class="px-4 py-2.5 text-sm text-outline">Tidak ada aksi</p>';
    return html;
}

function toggleUserMenu(userId, event) {
    event.stopPropagation();
    const menu = document.getElementById('umenu-' + userId);
    if (!menu) return;
    const wasHidden = menu.classList.contains('hidden');
    document.querySelectorAll('[id^="umenu-"]').forEach(m => m.classList.add('hidden'));
    if (wasHidden) menu.classList.remove('hidden');
}

// Tutup semua menu saat klik di luar
document.addEventListener('click', () => {
    document.querySelectorAll('[id^="umenu-"]').forEach(m => m.classList.add('hidden'));
});

function renderUsersCards() {
    const container = document.getElementById('usersCards');
    if (!container) return;

    if (users.length === 0) {
        container.innerHTML = '<div class="bg-white p-8 rounded-2xl border border-outline-variant text-center text-outline">Belum ada pengguna</div>';
        return;
    }

    container.innerHTML = users.map(user => {
        const statusColor = user.status === 'approved' || user.status === 'active' ? 'bg-green-100 text-green-700' :
            user.status === 'pending' ? 'bg-yellow-100 text-yellow-700' :
            user.status === 'rejected' ? 'bg-red-100 text-red-700' : 'bg-orange-100 text-orange-700';
        const avatar = user.avatar_url || `https://ui-avatars.com/api/?name=${encodeURIComponent(user.name)}&background=F6821F&color=fff`;
        return `
        <div class="bg-white p-4 rounded-2xl border border-outline-variant">
            <div class="flex items-center gap-3 mb-3">
                <img src="${avatar}" class="w-11 h-11 rounded-full">
                <div class="flex-1 min-w-0">
                    <p class="font-semibold text-sm truncate">${user.name}</p>
                    <p class="text-xs text-outline truncate">${user.email}</p>
                </div>
                <span class="px-2.5 py-1 rounded-full text-xs font-semibold ${statusColor}">${user.status}</span>
                ${user.api_requested_at && !user.api_approved ? `
                    <span class="px-2.5 py-1 rounded-full text-xs font-semibold bg-blue-100 text-blue-700">Minta API</span>
                ` : ''}
                <div class="relative">
                    <button onclick="toggleUserMenu('${user.id}', event)" class="p-2 rounded-lg hover:bg-gray-100 text-gray-500" title="Aksi">
                        <span class="material-symbols-outlined">more_vert</span>
                    </button>
                    <div id="umenu-${user.id}" class="hidden absolute right-0 top-full mt-1 w-56 bg-white rounded-xl shadow-xl border border-outline-variant py-1.5 z-30 text-left">
                        ${userMenuItems(user)}
                    </div>
                </div>
            </div>
            <div class="flex items-center justify-between text-xs text-outline mb-1">
                <span>Bergabung ${new Date(user.created_at).toLocaleDateString('id-ID')}</span>
                <span class="shrink-0 ml-2">· ${user.storage_limit_mb || 2048} MB</span>
            </div>
        </div>`;
    }).join('');
}

function approveUser(userId) {
    showConfirmModal(
        'Approve User',
        'Approve this user and grant them access to the platform?',
        async () => {
            const data = await apiCall(`/admin/users/${userId}/approve`, {
                method: 'POST'
            });
            
            if (data && data.success) {
                showToast('User approved successfully', 'success');
                loadUsers();
                loadPendingUsers();
                loadStats();
            } else {
                showToast('Failed to approve user', 'error');
            }
        }
    );
}

function rejectUser(userId) {
    rejectUserId = userId;
    document.getElementById('rejectReasonInput').value = '';
    openModal('rejectUser');
}

async function confirmRejectUser() {
    const reason = document.getElementById('rejectReasonInput').value.trim();
    closeModal('rejectUser');
    
    if (!rejectUserId) return;
    
    const data = await apiCall(`/admin/users/${rejectUserId}/reject`, {
        method: 'POST',
        body: JSON.stringify({ reason: reason || '' })
    });
    
    if (data && data.success) {
        showToast('User rejected', 'success');
        loadUsers();
        loadPendingUsers();
        loadStats();
    } else {
        showToast('Failed to reject user', 'error');
    }
    
    rejectUserId = null;
}

function suspendUser(userId) {
    showConfirmModal(
        'Suspend User',
        'Suspend this user? They will not be able to access the platform.',
        async () => {
            const data = await apiCall(`/admin/users/${userId}/suspend`, {
                method: 'POST'
            });
            
            if (data && data.success) {
                showToast('User suspended', 'success');
                loadUsers();
            } else {
                showToast('Failed to suspend user', 'error');
            }
        }
    );
}

async function approveUserAPI(userId) {
    const data = await apiCall(`/admin/users/${userId}/api-approve`, { method: 'PUT' });
    if (data && data.success) {
        showToast('Akses API disetujui', 'success');
        loadUsers();
        loadStats();
    } else {
        showToast(data && data.error ? data.error : 'Gagal menyetujui API', 'error');
    }
}

function revokeUserAPI(userId) {
    showConfirmModal(
        'Cabut Akses API',
        'Cabut persetujuan API user ini? Semua API key miliknya akan ikut dinonaktifkan.',
        async () => {
            const data = await apiCall(`/admin/users/${userId}/api-revoke`, { method: 'PUT' });
            if (data && data.success) {
                showToast('Akses API dicabut', 'success');
                loadUsers();
            } else {
                showToast('Gagal mencabut akses API', 'error');
            }
        }
    );
}

let quotaUserId = null;
function editUserQuota(userId) {
    const user = (typeof users !== 'undefined' ? users : []).find(u => u.id === userId);
    quotaUserId = userId;
    document.getElementById('quotaLimitInput').value = (user && user.storage_limit_mb) || 2048;
    document.getElementById('quotaUserName').textContent = user ? user.name : '';
    openModal('editQuota');
}

async function confirmEditQuota() {
    const limit = parseInt(document.getElementById('quotaLimitInput').value, 10);
    if (!limit || limit < 1) {
        showToast('Limit tidak valid', 'error');
        return;
    }
    closeModal('editQuota');
    if (!quotaUserId) return;
    const data = await apiCall(`/admin/users/${quotaUserId}/quotas`, {
        method: 'PUT',
        body: JSON.stringify({ storage_limit_mb: limit })
    });
    if (data && data.success) {
        showToast('Kuota diperbarui', 'success');
        loadUsers();
    } else {
        showToast('Gagal memperbarui kuota', 'error');
    }
    quotaUserId = null;
}


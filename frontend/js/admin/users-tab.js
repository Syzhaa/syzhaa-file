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
                    user.status === 'approved' ? 'bg-green-100 text-green-700' :
                    user.status === 'pending' ? 'bg-yellow-100 text-yellow-700' :
                    user.status === 'rejected' ? 'bg-red-100 text-red-700' :
                    'bg-gray-100 text-gray-700'
                }">
                    ${user.status}
                </span>
            </td>
            <td class="px-6 py-4 text-sm text-outline">${formatDate(user.created_at)}</td>
            <td class="px-6 py-4 text-sm text-outline">${user.storage_used_mb ? formatBytes(user.storage_used_mb * 1024 * 1024) : '0 MB'}</td>
            <td class="px-6 py-4 text-right">
                <div class="flex justify-end gap-2">
                    ${user.status === 'pending' ? `
                        <button onclick="approveUser('${user.id}')" class="px-3 py-1.5 bg-green-500 text-white text-xs font-semibold rounded-lg hover:bg-green-600">
                            Approve
                        </button>
                        <button onclick="rejectUser('${user.id}')" class="px-3 py-1.5 bg-red-500 text-white text-xs font-semibold rounded-lg hover:bg-red-600">
                            Reject
                        </button>
                    ` : user.status === 'approved' ? `
                        <button onclick="suspendUser('${user.id}')" class="px-3 py-1.5 bg-orange-500 text-white text-xs font-semibold rounded-lg hover:bg-orange-600">
                            Suspend
                        </button>
                    ` : ''}
                </div>
            </td>
        </tr>
    `).join('');

    // Mobile cards
    renderUsersCards();
}

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
            </div>
            <div class="flex items-center justify-between text-xs text-outline mb-3">
                <span>Bergabung ${new Date(user.created_at).toLocaleDateString('id-ID')}</span>
                <span>${user.storage_limit_mb || 2048} MB</span>
            </div>
            <div class="flex gap-2">
                ${user.status === 'pending' ? `
                    <button onclick="approveUser('${user.id}')" class="flex-1 px-3 py-2 bg-green-500 text-white text-sm font-semibold rounded-xl">Setujui</button>
                    <button onclick="rejectUser('${user.id}')" class="flex-1 px-3 py-2 bg-red-500 text-white text-sm font-semibold rounded-xl">Tolak</button>
                ` : ''}
                ${user.api_requested_at && !user.api_approved ? `
                    <button onclick="approveUserAPI('${user.id}')" class="flex-1 px-3 py-2 bg-blue-500 text-white text-sm font-semibold rounded-xl">Setujui API</button>
                ` : ''}
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


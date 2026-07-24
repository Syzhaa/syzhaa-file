// Admin User Management JS
let allUsers = [];
let currentFilter = 'all';

async function apiCall(endpoint, options = {}) {
    const response = await fetch(endpoint, {
        ...options,
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', ...options.headers }
    });
    if (response.status === 401) {
        window.location.href = '/admin/login.html';
        return null;
    }
    return response.json();
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
        alert('User approved successfully!');
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
        alert('User rejected');
    }
}

async function suspendUser(userId) {
    if (!confirm('Are you sure you want to suspend this user?')) return;
    
    const reason = prompt('Suspension reason (optional):');
    const data = await apiCall(`/admin/users/${userId}/suspend`, { 
        method: 'POST', 
        body: JSON.stringify({ reason: reason || 'Account suspended by admin' })
    });
    
    if (data && data.success) {
        await loadUsers();
        alert('User suspended');
    }
}

async function loadSettings() {
    const data = await apiCall('/admin/settings');
    if (data && data.settings) {
        document.getElementById('default-storage').value = data.settings.default_storage_limit_mb?.value || '5120';
        document.getElementById('default-duration').value = data.settings.default_max_duration_days?.value || '7';
        document.getElementById('require-approval').value = data.settings.require_approval?.value || 'true';
    }
}

async function saveSettings() {
    const settings = [
        { key: 'default_storage_limit_mb', value: document.getElementById('default-storage').value },
        { key: 'default_max_duration_days', value: document.getElementById('default-duration').value },
        { key: 'require_approval', value: document.getElementById('require-approval').value }
    ];
    
    for (const setting of settings) {
        await apiCall('/admin/settings', { method: 'POST', body: JSON.stringify(setting) });
    }
    
    alert('Settings saved successfully!');
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

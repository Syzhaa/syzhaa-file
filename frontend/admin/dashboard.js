// Admin Dashboard v2 - Modern Material Design 3
// AmbilFile Admin Panel

let currentUser = null;
let currentView = 'dashboard';
let stats = {};
let users = [];
let apiKeys = [];

// ============================================
// API HELPER FUNCTIONS
// ============================================

async function apiCall(endpoint, options = {}) {
    try {
        const response = await fetch(endpoint, {
            ...options,
            credentials: 'include',
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            }
        });
        
        if (response.status === 401) {
            window.location.href = '/admin/login.html';
            return null;
        }
        
        const data = await response.json();
        return data;
    } catch (error) {
        console.error('API call failed:', error);
        showToast('Network error occurred', 'error');
        return null;
    }
}

function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `fixed bottom-6 right-6 px-6 py-4 rounded-xl shadow-2xl text-white font-medium z-50 animate-slide-up ${
        type === 'error' ? 'bg-red-500' : 
        type === 'success' ? 'bg-green-500' : 
        'bg-primary'
    }`;
    toast.textContent = message;
    document.body.appendChild(toast);
    
    setTimeout(() => {
        toast.remove();
    }, 3000);
}

function formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

function formatDate(dateString) {
    if (!dateString) return 'N/A';
    const date = new Date(dateString);
    return date.toLocaleDateString('id-ID', { 
        year: 'numeric', 
        month: 'short', 
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// ============================================
// MODAL FUNCTIONS
// ============================================

function openModal(modalName) {
    const modal = document.getElementById(`modal-${modalName}`);
    if (modal) {
        modal.classList.add('active');
    }
}

function closeModal(modalName) {
    const modal = document.getElementById(`modal-${modalName}`);
    if (modal) {
        modal.classList.remove('active');
    }
}

// Copy API key to clipboard
function copyAPIKey() {
    const apiKeyInput = document.getElementById('apiKeyDisplay');
    apiKeyInput.select();
    navigator.clipboard.writeText(apiKeyInput.value).then(() => {
        showToast('✓ API key copied to clipboard!', 'success');
    }).catch(() => {
        // Fallback for older browsers
        document.execCommand('copy');
        showToast('✓ API key copied to clipboard!', 'success');
    });
}

// Generic copy text function
function copyText(elementId) {
    const input = document.getElementById(elementId);
    input.select();
    navigator.clipboard.writeText(input.value).then(() => {
        showToast('✓ Copied to clipboard!', 'success');
    }).catch(() => {
        // Fallback for older browsers
        document.execCommand('copy');
        showToast('✓ Copied to clipboard!', 'success');
    });
}

// Show confirmation modal
let confirmCallback = null;
function showConfirmModal(title, message, onConfirm) {
    document.getElementById('confirmTitle').textContent = title;
    document.getElementById('confirmMessage').textContent = message;
    confirmCallback = onConfirm;
    
    // Attach callback to confirm button
    const confirmBtn = document.getElementById('confirmButton');
    confirmBtn.onclick = () => {
        closeModal('confirm');
        if (confirmCallback) {
            confirmCallback();
            confirmCallback = null;
        }
    };
    
    openModal('confirm');
}

// Close modal when clicking outside
document.addEventListener('click', (e) => {
    if (e.target.classList.contains('modal')) {
        e.target.classList.remove('active');
    }
});

// ============================================
// NAVIGATION FUNCTIONS
// ============================================

function switchView(viewName) {
    // Update bottom nav active state FIRST (mobile) - inline style, bulletproof
    try {
        document.querySelectorAll('.bnav-item').forEach(item => {
            item.style.color = (item.dataset.bnav === viewName) ? '#F6821F' : '#737686';
        });
    } catch (e) {}

    // Hide all views
    document.querySelectorAll('[id^="view-"]').forEach(view => {
        view.classList.add('hidden');
    });
    
    // Show selected view
    const targetView = document.getElementById(`view-${viewName}`);
    if (targetView) {
        targetView.classList.remove('hidden');
        currentView = viewName;
    }
    
    // Update sidebar active state
    document.querySelectorAll('.sidebar-item').forEach(item => {
        item.classList.remove('active');
        if (item.dataset.view === viewName) {
            item.classList.add('active');
        }
    });
    
    // Load view-specific data
    if (viewName === 'users') {
        loadUsers();
    } else if (viewName === 'api-keys') {
        loadAPIKeys();
    } else if (viewName === 'settings') {
        apiCall('/admin/me').then(data => {
            if (data && data.admin) {
                document.getElementById('accountEmail').value = data.admin.email || '';
            }
        });
    }
}

// Sidebar navigation
document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('.sidebar-item').forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            const view = item.dataset.view;
            if (view) {
                switchView(view);
            }
        });
    });
});

// ============================================
// DASHBOARD LOADING
// ============================================

async function loadUser() {
    const data = await apiCall('/admin/me');
    if (data && data.success !== false) {
        currentUser = data;
        document.getElementById('userName').textContent = data.name || 'Admin';
        document.getElementById('userEmail').textContent = data.email || '';
        
        const avatar = document.getElementById('userAvatar');
        if (data.avatar_url) {
            avatar.src = data.avatar_url;
        } else {
            avatar.src = `https://ui-avatars.com/api/?name=${encodeURIComponent(data.name || 'Admin')}&background=0053db&color=fff`;
        }
    }
}

async function loadStats() {
    const data = await apiCall('/admin/stats');
    if (data && data.success) {
        stats = data.stats;
        
        // Update stat cards
        document.getElementById('stat-rooms').textContent = stats.total_rooms || 0;
        document.getElementById('stat-files').textContent = stats.total_files || 0;
        document.getElementById('stat-users').textContent = stats.total_users || 0;
        
        const storageGB = (stats.total_size || 0) / (1024 * 1024 * 1024);
        document.getElementById('stat-storage').textContent = storageGB.toFixed(2) + ' GB';
    }
}

async function loadPendingUsers() {
    const data = await apiCall('/admin/users?status=pending');
    if (data && data.success && data.users) {
        const pendingUsers = data.users.filter(u => u.status === 'pending');
        document.getElementById('pending-count').textContent = pendingUsers.length;
        const dot = document.getElementById('notif-dot');
        if (dot) dot.classList.toggle('hidden', pendingUsers.length === 0);

        const container = document.getElementById('pendingUsers');
        if (pendingUsers.length === 0) {
            container.innerHTML = '<p class="text-center text-outline py-8">Tidak ada yang menunggu persetujuan</p>';
            return;
        }
        
        container.innerHTML = pendingUsers.slice(0, 5).map(user => `
            <div class="flex items-center justify-between p-4 bg-surface-container rounded-xl border border-outline-variant">
                <div class="flex items-center gap-3">
                    <img src="${user.avatar_url || `https://ui-avatars.com/api/?name=${encodeURIComponent(user.name)}`}" 
                         class="w-10 h-10 rounded-full border-2 border-outline-variant">
                    <div>
                        <p class="font-semibold text-sm">${user.name}</p>
                        <p class="text-xs text-outline">${user.email}</p>
                    </div>
                </div>
                <div class="flex gap-2">
                    <button onclick="approveUser('${user.id}')" 
                            class="px-3 py-1.5 bg-green-500 text-white text-xs font-semibold rounded-lg hover:bg-green-600">
                        Approve
                    </button>
                    <button onclick="rejectUser('${user.id}')" 
                            class="px-3 py-1.5 bg-red-500 text-white text-xs font-semibold rounded-lg hover:bg-red-600">
                        Reject
                    </button>
                </div>
            </div>
        `).join('');
    }
}

async function loadRecentActivity() {
    const container = document.getElementById('recentActivity');
    const data = await apiCall('/admin/users');
    if (!data || !data.success || !data.users) {
        container.innerHTML = '<p class="text-center text-outline text-sm py-6">Belum ada data</p>';
        return;
    }
    const latest = data.users.slice(0, 5);
    if (!latest.length) {
        container.innerHTML = '<p class="text-center text-outline text-sm py-6">Belum ada pendaftar</p>';
        return;
    }
    const statusColor = { pending: 'text-amber-500', approved: 'text-green-500', rejected: 'text-red-500', suspended: 'text-gray-500' };
    const statusLabel = { pending: 'Menunggu', approved: 'Disetujui', rejected: 'Ditolak', suspended: 'Suspend' };
    container.innerHTML = latest.map(u => {
        const when = u.created_at ? new Date(u.created_at).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }) : '-';
        return `
        <div class="flex items-start gap-3 text-sm">
            <span class="material-symbols-outlined ${statusColor[u.status] || 'text-outline'}">account_circle</span>
            <div class="flex-1 min-w-0">
                <p class="font-medium truncate">${escHtml(u.name)}</p>
                <p class="text-xs text-outline">${escHtml(u.email)} · ${when} · ${statusLabel[u.status] || u.status}</p>
            </div>
        </div>`;
    }).join('');
}

function escHtml(s) {
    const d = document.createElement('div');
    d.textContent = s || '';
    return d.innerHTML;
}

// ============================================
// ROOM MANAGEMENT
// ============================================

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
        const response = await fetch('/api/room/pin', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ pin })
        });
        
        const data = await response.json();
        
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

// ============================================
// USER MANAGEMENT
// ============================================

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

// Store userId for rejection
let rejectUserId = null;

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

// ============================================
// API KEYS MANAGEMENT
// ============================================

async function loadAPIKeys() {
    try {
        const data = await apiCall('/admin/api-keys');
        if (data && data.success && data.api_keys) {
            apiKeys = data.api_keys;
        } else {
            apiKeys = [];
        }
    } catch {
        apiKeys = [];
    }
    renderAPIKeys();
}

function renderAPIKeys() {
    const container = document.getElementById('apiKeysList');
    if (!container) return;
    
    if (apiKeys.length === 0) {
        container.innerHTML = `
            <div class="bg-white p-12 rounded-2xl border border-outline-variant text-center">
                <span class="material-symbols-outlined text-6xl text-outline mb-4">key_off</span>
                <p class="text-outline">No API keys created yet</p>
                <button onclick="createAPIKey()" class="mt-4 px-6 py-2 bg-primary text-on-primary rounded-xl font-semibold hover:brightness-110">
                    Create Your First API Key
                </button>
            </div>
        `;
        return;
    }
    
    container.innerHTML = apiKeys.map(key => `
        <div class="bg-white p-5 rounded-2xl border border-outline-variant">
            <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-3 mb-4">
                <div class="flex-1 min-w-0">
                    <h4 class="font-semibold text-base mb-1 truncate">${key.name || 'Unnamed Key'}</h4>
                    <p class="text-xs text-outline mb-2">Dibuat: ${formatDate(key.created_at)}</p>
                    <div class="flex items-center gap-2 flex-wrap">
                        <code class="px-3 py-1.5 bg-surface-container rounded-lg text-xs font-mono">sfa_...${key.key_suffix || '****'}</code>
                        <span class="px-3 py-1 rounded-full text-xs font-semibold ${key.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-700'}">
                            ${key.is_active ? 'Aktif' : 'Nonaktif'}
                        </span>
                    </div>
                </div>
                <div class="flex gap-2 shrink-0">
                    <button onclick="toggleAPIKey('${key.id}', ${!key.is_active})" 
                            class="px-3 py-1.5 ${key.is_active ? 'bg-orange-500' : 'bg-green-500'} text-white text-xs font-semibold rounded-lg">
                        ${key.is_active ? 'Matikan' : 'Aktifkan'}
                    </button>
                    <button onclick="deleteAPIKey('${key.id}')" 
                            class="px-3 py-1.5 bg-red-500 text-white text-xs font-semibold rounded-lg">
                        Hapus
                    </button>
                </div>
            </div>
        </div>
    `).join('');
}

// Open modal to create API key
function createAPIKey() {
    document.getElementById('apiKeyNameInput').value = '';
    openModal('createApiKey');
}

// Confirm and create API key
async function confirmCreateAPIKey() {
    const name = document.getElementById('apiKeyNameInput').value.trim();
    closeModal('createApiKey');
    
    const data = await apiCall('/admin/api-keys', {
        method: 'POST',
        body: JSON.stringify({
            name: name || 'Unnamed Key'
        })
    });
    
    if (data && data.success && data.api_key) {
        showToast('API key created successfully!', 'success');
        
        // Show the full key in modal (only shown once)
        document.getElementById('apiKeyDisplay').value = data.api_key.key;
        openModal('apiKey');
        
        loadAPIKeys();
    } else {
        showToast('Failed to create API key', 'error');
    }
}

async function toggleAPIKey(keyId, activate) {
    const data = await apiCall(`/admin/api-keys/${keyId}/toggle`, {
        method: 'POST'
    });
    
    if (data && data.success) {
        showToast(`API key ${activate ? 'enabled' : 'disabled'}`, 'success');
        loadAPIKeys();
    } else {
        showToast('Failed to toggle API key', 'error');
    }
}

function deleteAPIKey(keyId) {
    showConfirmModal(
        'Delete API Key',
        'Delete this API key? This action cannot be undone.',
        async () => {
            const data = await apiCall(`/admin/api-keys/${keyId}`, {
                method: 'DELETE'
            });
            
            if (data && data.success) {
                showToast('API key deleted', 'success');
                loadAPIKeys();
            } else {
                showToast('Failed to delete API key', 'error');
            }
        }
    );
}

// ============================================
// LOGOUT
// ============================================

function logout() {
    showConfirmModal(
        'Logout',
        'Are you sure you want to logout?',
        async () => {
            await apiCall('/auth/logout', { method: 'POST' });
            window.location.href = '/';
        }
    );
}

// ============================================
// INITIALIZATION
// ============================================

async function initDashboard() {
    try {
        // Load user info
        await loadUser();
        
        // Load dashboard data
        await Promise.all([
            loadStats(),
            loadPendingUsers(),
            loadRecentActivity()
        ]);
        
        // Hide loading, show app
        document.getElementById('loading').classList.add('hidden');
        document.getElementById('app').classList.remove('hidden');
        
    } catch (error) {
        console.error('Dashboard initialization failed:', error);
        showToast('Failed to load dashboard', 'error');
    }
}

// Start the app when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initDashboard);
} else {
    initDashboard();
}

// Auto-refresh stats every 30 seconds
setInterval(() => {
    if (currentView === 'dashboard') {
        loadStats();
        loadPendingUsers();
    }
}, 30000);


// Save admin account changes (email / password)
async function saveAccount() {
    const email = document.getElementById('accountEmail').value.trim();
    const currentPw = document.getElementById('accountCurrentPw').value;
    const newPw = document.getElementById('accountNewPw').value;
    const newPw2 = document.getElementById('accountNewPw2').value;

    if (!currentPw) {
        showToast('Isi password saat ini untuk verifikasi', 'error');
        return;
    }
    if (newPw && newPw !== newPw2) {
        showToast('Password baru tidak sama', 'error');
        return;
    }
    if (newPw && newPw.length < 8) {
        showToast('Password baru minimal 8 karakter', 'error');
        return;
    }

    const data = await apiCall('/admin/account', {
        method: 'POST',
        body: JSON.stringify({ email, current_password: currentPw, new_password: newPw })
    });
    if (data && data.success) {
        showToast('Akun berhasil diperbarui', 'success');
        document.getElementById('accountCurrentPw').value = '';
        document.getElementById('accountNewPw').value = '';
        document.getElementById('accountNewPw2').value = '';
        if (data.admin && data.admin.email) {
            document.getElementById('accountEmail').value = data.admin.email;
            const emailEl = document.getElementById('userEmail');
            if (emailEl) emailEl.textContent = data.admin.email;
        }
    } else if (data) {
        showToast(data.error || 'Gagal menyimpan', 'error');
    }
}


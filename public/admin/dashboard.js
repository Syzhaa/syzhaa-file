// Admin Dashboard v2 - Modern Material Design 3
// Syzhaa File Admin Panel

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

// ============================================
// MODAL UTILITY FUNCTIONS
// ============================================

function openModal(modalId) {
    const modal = document.getElementById(`modal-${modalId}`);
    if (modal) {
        modal.classList.add('active');
    }
}

function closeModal(modalId) {
    const modal = document.getElementById(`modal-${modalId}`);
    if (modal) {
        modal.classList.remove('active');
    }
}

function showAlert(message, type = 'info') {
    const modal = document.getElementById('modal-alert');
    const icon = document.getElementById('alertIcon');
    const iconSpan = icon.querySelector('.material-symbols-outlined');
    const title = document.getElementById('alertTitle');
    const messageEl = document.getElementById('alertMessage');
    
    // Configure based on type
    if (type === 'error') {
        icon.className = 'flex-shrink-0 w-12 h-12 rounded-full bg-red-100 flex items-center justify-center';
        iconSpan.className = 'material-symbols-outlined text-2xl text-red-600';
        iconSpan.textContent = 'error';
        title.textContent = 'Error';
        title.className = 'text-xl font-headline font-bold mb-2 text-red-600';
    } else if (type === 'success') {
        icon.className = 'flex-shrink-0 w-12 h-12 rounded-full bg-green-100 flex items-center justify-center';
        iconSpan.className = 'material-symbols-outlined text-2xl text-green-600';
        iconSpan.textContent = 'check_circle';
        title.textContent = 'Success';
        title.className = 'text-xl font-headline font-bold mb-2 text-green-600';
    } else if (type === 'warning') {
        icon.className = 'flex-shrink-0 w-12 h-12 rounded-full bg-amber-100 flex items-center justify-center';
        iconSpan.className = 'material-symbols-outlined text-2xl text-amber-600';
        iconSpan.textContent = 'warning';
        title.textContent = 'Warning';
        title.className = 'text-xl font-headline font-bold mb-2 text-amber-600';
    } else {
        icon.className = 'flex-shrink-0 w-12 h-12 rounded-full bg-blue-100 flex items-center justify-center';
        iconSpan.className = 'material-symbols-outlined text-2xl text-blue-600';
        iconSpan.textContent = 'info';
        title.textContent = 'Information';
        title.className = 'text-xl font-headline font-bold mb-2 text-blue-600';
    }
    
    messageEl.textContent = message;
    openModal('alert');
}

let confirmCallback = null;
function showConfirm(message, onConfirm) {
    document.getElementById('confirmMessage').textContent = message;
    confirmCallback = onConfirm;
    
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

let promptCallback = null;
function showPrompt(message, onSubmit, placeholder = '', title = 'Input Required') {
    document.getElementById('promptTitle').textContent = title;
    document.getElementById('promptMessage').textContent = message;
    const input = document.getElementById('promptInput');
    input.value = '';
    input.placeholder = placeholder;
    promptCallback = onSubmit;
    
    const promptBtn = document.getElementById('promptButton');
    promptBtn.onclick = () => {
        const value = input.value.trim();
        closeModal('prompt');
        if (promptCallback) {
            promptCallback(value);
            promptCallback = null;
        }
    };
    
    // Allow Enter key to submit
    input.onkeypress = (e) => {
        if (e.key === 'Enter') {
            promptBtn.click();
        }
    };
    
    openModal('prompt');
    setTimeout(() => input.focus(), 100);
}

function copyAPIKeyFromModal() {
    const input = document.getElementById('apiKeyValue');
    input.select();
    navigator.clipboard.writeText(input.value).then(() => {
        showToast('✓ API key copied to clipboard!', 'success');
    }).catch(() => {
        document.execCommand('copy');
        showToast('✓ API key copied to clipboard!', 'success');
    });
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
    
    // Mobile menu toggle
    const menuToggle = document.getElementById('menuToggle');
    const sidebar = document.getElementById('sidebar');
    if (menuToggle && sidebar) {
        menuToggle.addEventListener('click', () => {
            sidebar.classList.toggle('-translate-x-full');
        });
    }
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
        
        const container = document.getElementById('pendingUsers');
        if (pendingUsers.length === 0) {
            container.innerHTML = '<p class="text-center text-outline py-8">No pending approvals</p>';
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
    // Mock recent activity for now
    const container = document.getElementById('recentActivity');
    container.innerHTML = `
        <div class="flex items-start gap-3 text-sm">
            <span class="material-symbols-outlined text-primary">check_circle</span>
            <div class="flex-1">
                <p class="font-medium">New user approved</p>
                <p class="text-xs text-outline">2 minutes ago</p>
            </div>
        </div>
        <div class="flex items-start gap-3 text-sm">
            <span class="material-symbols-outlined text-secondary">upload_file</span>
            <div class="flex-1">
                <p class="font-medium">Large file uploaded</p>
                <p class="text-xs text-outline">15 minutes ago</p>
            </div>
        </div>
        <div class="flex items-start gap-3 text-sm">
            <span class="material-symbols-outlined text-tertiary">key</span>
            <div class="flex-1">
                <p class="font-medium">API key created</p>
                <p class="text-xs text-outline">1 hour ago</p>
            </div>
        </div>
    `;
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
        
        // Show room details
        const roomLink = `https://file.syzhaa.my.id/?room=${data.room_id}`;
        const pinLink = `https://file.syzhaa.my.id/?pin=${data.pin}`;
        
        showAlert(`Room Created!\n\nPIN: ${data.pin}\n\nRoom Link: ${roomLink}\n\nPIN Link: ${pinLink}`, 'success');
        
        loadStats();
    } else {
        showToast('Failed to create room', 'error');
    }
}

async function joinRoom() {
    const pin = document.getElementById('pinInput').value.trim();
    
    if (pin.length !== 6) {
        showToast('PIN must be 6 digits', 'error');
        return;
    }
    
    window.location.href = `https://file.syzhaa.my.id/?pin=${pin}`;
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
}

async function approveUser(userId) {
    showConfirm(
        'Approve this user?',
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

async function rejectUser(userId) {
    showPrompt(
        'Enter rejection reason (optional):',
        async (reason) => {
            const data = await apiCall(`/admin/users/${userId}/reject`, {
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
        },
        'e.g. Incomplete information',
        'Reject User'
    );
}

async function suspendUser(userId) {
    showConfirm(
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
    const data = await apiCall('/admin/api-keys');
    if (data && data.success && data.api_keys) {
        apiKeys = data.api_keys;
        renderAPIKeys();
    }
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
        <div class="bg-white p-6 rounded-2xl border border-outline-variant">
            <div class="flex items-start justify-between mb-4">
                <div class="flex-1">
                    <h4 class="font-semibold text-lg mb-1">${key.name || 'Unnamed Key'}</h4>
                    <p class="text-sm text-outline mb-3">Created: ${formatDate(key.created_at)}</p>
                    <div class="flex items-center gap-2">
                        <code class="px-3 py-2 bg-surface-container rounded-lg text-xs font-mono">${key.key_prefix}...${key.key_suffix || '****'}</code>
                        <span class="px-3 py-1 rounded-full text-xs font-semibold ${key.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-700'}">
                            ${key.is_active ? 'Active' : 'Inactive'}
                        </span>
                    </div>
                </div>
                <div class="flex gap-2">
                    <button onclick="toggleAPIKey('${key.id}', ${!key.is_active})" 
                            class="px-3 py-1.5 ${key.is_active ? 'bg-orange-500' : 'bg-green-500'} text-white text-xs font-semibold rounded-lg hover:brightness-110">
                        ${key.is_active ? 'Disable' : 'Enable'}
                    </button>
                    <button onclick="deleteAPIKey('${key.id}')" 
                            class="px-3 py-1.5 bg-red-500 text-white text-xs font-semibold rounded-lg hover:bg-red-600">
                        Delete
                    </button>
                </div>
            </div>
            <div class="grid grid-cols-3 gap-4 pt-4 border-t border-outline-variant">
                <div>
                    <p class="text-xs text-outline">Rooms Created</p>
                    <p class="text-lg font-bold">${key.rooms_created || 0}</p>
                </div>
                <div>
                    <p class="text-xs text-outline">Files Uploaded</p>
                    <p class="text-lg font-bold">${key.files_uploaded || 0}</p>
                </div>
                <div>
                    <p class="text-xs text-outline">Last Used</p>
                    <p class="text-sm font-medium">${key.last_used_at ? formatDate(key.last_used_at) : 'Never'}</p>
                </div>
            </div>
        </div>
    `).join('');
}

async function createAPIKey() {
    showPrompt(
        'Enter a name for this API key (optional):',
        async (name) => {
            const data = await apiCall('/admin/api-keys', {
                method: 'POST',
                body: JSON.stringify({
                    name: name || 'Unnamed Key'
                })
            });
            
            if (data && data.success && data.api_key) {
                showToast('API key created successfully!', 'success');
                
                // Show the full key in modal (only shown once)
                document.getElementById('apiKeyValue').value = data.api_key.key;
                openModal('apiKeyDisplay');
                
                loadAPIKeys();
            } else {
                showToast('Failed to create API key', 'error');
            }
        },
        'My API Key',
        'Create API Key'
    );
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

async function deleteAPIKey(keyId) {
    showConfirm(
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

async function logout() {
    showConfirm(
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


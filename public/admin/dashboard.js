// Admin Dashboard JS
let currentUser = null;
let apiKeys = [];

// API Helper
async function apiCall(endpoint, options = {}) {
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
    
    return response.json();
}

// Load user info
async function loadUser() {
    const data = await apiCall('/admin/me');
    if (data) {
        currentUser = data;
        document.getElementById('userName').textContent = data.name;
        document.getElementById('userEmail').textContent = data.email;
        document.getElementById('userAvatar').src = data.avatar_url || '/default-avatar.png';
    }
}

// Load stats
async function loadStats() {
    const data = await apiCall('/admin/stats');
    if (data && data.stats) {
        document.getElementById('statTotalRooms').textContent = data.stats.total_rooms;
        document.getElementById('statActiveRooms').textContent = data.stats.active_rooms;
        document.getElementById('statTotalFiles').textContent = data.stats.total_files;
        document.getElementById('statTotalSize').textContent = formatBytes(data.stats.total_size);
    }
}

// Load API keys
async function loadAPIKeys() {
    const data = await apiCall('/admin/api-keys');
    if (data && data.keys) {
        apiKeys = data.keys;
        renderAPIKeys();
    }
}

// Render API keys
function renderAPIKeys() {
    const container = document.getElementById('apiKeysList');
    
    if (apiKeys.length === 0) {
        container.innerHTML = '<p class="text-slate-600 text-center py-8">Belum ada API keys. Buat yang pertama!</p>';
        return;
    }
    
    container.innerHTML = apiKeys.map(key => {
        const isExpired = key.expires_at && new Date(key.expires_at) < new Date();
        const statusColor = !key.is_active ? 'bg-slate-400' : isExpired ? 'bg-red-500' : 'bg-green-500';
        const statusText = !key.is_active ? 'Disabled' : isExpired ? 'Expired' : 'Active';
        
        return `
            <div class="brutal-border-thin rounded p-4 flex items-center justify-between hover:bg-slate-50">
                <div class="flex-1">
                    <div class="flex items-center gap-3 mb-2">
                        <span class="font-bold text-slate-900">${escapeHtml(key.name)}</span>
                        <span class="px-2 py-1 rounded text-xs font-semibold text-white ${statusColor}">${statusText}</span>
                    </div>
                    <div class="text-sm text-slate-600 space-y-1">
                        <p>Created: ${new Date(key.created_at).toLocaleString('id-ID')}</p>
                        ${key.expires_at ? `<p>Expires: ${new Date(key.expires_at).toLocaleString('id-ID')}</p>` : '<p>Expires: Never</p>'}
                        ${key.last_used_at ? `<p>Last used: ${new Date(key.last_used_at).toLocaleString('id-ID')}</p>` : '<p>Never used</p>'}
                    </div>
                </div>
                <div class="flex gap-2">
                    <button onclick="toggleAPIKey('${key.id}')" 
                            class="px-3 py-2 text-sm font-semibold rounded brutal-border-thin ${key.is_active ? 'bg-yellow-500 text-white' : 'bg-green-500 text-white'}">
                        ${key.is_active ? 'Disable' : 'Enable'}
                    </button>
                    <button onclick="deleteAPIKey('${key.id}')" 
                            class="px-3 py-2 text-sm font-semibold rounded brutal-border-thin bg-red-500 text-white hover:bg-red-600">
                        Delete
                    </button>
                </div>
            </div>
        `;
    }).join('');
}

// Show/hide modals
function showCreateKeyModal() {
    document.getElementById('createKeyModal').classList.remove('hidden');
}

function hideCreateKeyModal() {
    document.getElementById('createKeyModal').classList.add('hidden');
    document.getElementById('keyName').value = '';
    document.getElementById('keyExpiry').value = '';
}

// Create API key
async function createAPIKey() {
    const name = document.getElementById('keyName').value.trim();
    const expiryDays = document.getElementById('keyExpiry').value.trim();
    
    if (!name) {
        alert('Nama API key wajib diisi!');
        return;
    }
    
    const payload = { name };
    if (expiryDays && parseInt(expiryDays) > 0) {
        payload.expiry_days = parseInt(expiryDays);
    }
    
    const data = await apiCall('/admin/api-keys', {
        method: 'POST',
        body: JSON.stringify(payload)
    });
    
    if (data && data.success) {
        hideCreateKeyModal();
        document.getElementById('newAPIKey').textContent = data.api_key.key;
        document.getElementById('showKeyModal').classList.remove('hidden');
        await loadAPIKeys();
    } else {
        alert('Failed to create API key');
    }
}

// Copy and close key modal
function copyAndCloseKeyModal() {
    const keyText = document.getElementById('newAPIKey').textContent;
    navigator.clipboard.writeText(keyText).then(() => {
        document.getElementById('showKeyModal').classList.add('hidden');
        alert('API key copied to clipboard!');
    });
}

// Toggle API key
async function toggleAPIKey(id) {
    const data = await apiCall(`/admin/api-keys/${id}/toggle`, { method: 'POST' });
    if (data && data.success) {
        await loadAPIKeys();
    }
}

// Delete API key
async function deleteAPIKey(id) {
    if (!confirm('Yakin mau hapus API key ini? Tindakan ini tidak bisa dibatalkan!')) {
        return;
    }
    
    const data = await apiCall(`/admin/api-keys/${id}`, { method: 'DELETE' });
    if (data && data.success) {
        await loadAPIKeys();
    }
}

// Logout
async function logout() {
    await apiCall('/auth/logout', { method: 'POST' });
    window.location.href = '/admin/login.html';
}

// Utils
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Init
async function init() {
    try {
        await Promise.all([
            loadUser(),
            loadStats(),
            loadAPIKeys()
        ]);
        
        document.getElementById('loading').classList.add('hidden');
        document.getElementById('app').classList.remove('hidden');
    } catch (error) {
        console.error('Init error:', error);
        window.location.href = '/admin/login.html';
    }
}

// Auto-refresh stats every 30s
setInterval(() => {
    loadStats();
    loadAPIKeys();
}, 30000);

// Start
init();

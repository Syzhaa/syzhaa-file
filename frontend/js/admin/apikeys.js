// Admin dashboard module. Shared globals via window scope.

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

function createAPIKey() {
    document.getElementById('apiKeyNameInput').value = '';
    openModal('createApiKey');
}

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


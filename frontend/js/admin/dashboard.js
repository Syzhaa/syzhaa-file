// Admin dashboard module. Shared globals via window scope.

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

// Wrapper around global api() (from /js/api.js) preserving legacy null-on-error behavior.




// ============================================
// MODAL FUNCTIONS
// ============================================



// Copy API key to clipboard

// Generic copy text function

// Show confirmation modal
let confirmCallback = null;

// Close modal when clicking outside
document.addEventListener('click', (e) => {
    if (e.target.classList.contains('modal')) {
        e.target.classList.remove('active');
    }
});

// ============================================
// NAVIGATION FUNCTIONS
// ============================================


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






// ============================================
// ROOM MANAGEMENT
// ============================================



// ============================================
// USER MANAGEMENT
// ============================================





// Store userId for rejection
let rejectUserId = null;




// ============================================
// API KEYS MANAGEMENT
// ============================================



// Open modal to create API key

// Confirm and create API key



// ============================================
// LOGOUT
// ============================================


// ============================================
// INITIALIZATION
// ============================================


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



let _adminRooms = [];
let _adminRoomFilter = 'all';





// Load rooms when switching to rooms view
const _origSwitchView = switchView;
switchView = function(viewName) {
    _origSwitchView(viewName);
    if (viewName === 'rooms') loadAdminRooms();
};

async function apiCall(endpoint, options = {}) {
    try {
        return await api(endpoint, options);
    } catch (e) {
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

function escHtml(s) {
    const d = document.createElement('div');
    d.textContent = s || '';
    return d.innerHTML;
}

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

function switchView(viewName) {
    // Update URL hash for persistence across reloads
    try {
        history.replaceState(null, '', '#' + viewName);
    } catch (e) {}

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

        // Restore view from URL hash (persist active tab across reloads)
        const hash = (location.hash || '').replace('#', '');
        const validViews = ['dashboard', 'users', 'api-keys', 'settings'];
        if (validViews.includes(hash) && hash !== 'dashboard') {
            switchView(hash);
        }
        
    } catch (error) {
        console.error('Dashboard initialization failed:', error);
        showToast('Failed to load dashboard', 'error');
    }
}


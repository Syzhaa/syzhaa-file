// Admin dashboard module. Shared globals via window scope.

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
        const myRoomsEl = document.getElementById('stat-my-rooms');
        if (myRoomsEl) myRoomsEl.textContent = stats.admin_rooms || 0;
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


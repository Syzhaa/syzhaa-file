// layout.js — shared navigation components for AmbilFile.
// Injects bottom nav (mobile) and sidebar (desktop) with active state.
// Usage: <script src="/js/layout.js"></script> then Layout.init('dashboard')

(function () {
    const userNav = [
        { id: 'dashboard', href: '/user/', icon: 'dashboard', label: 'Dashboard' },
        { id: 'create-room', href: '/user/create-room.html', icon: 'add_circle', label: 'Buat Room' },
        { id: 'api-keys', href: '/user/api-keys.html', icon: 'key', label: 'API Key' },
        { id: 'account', href: '/user/account.html', icon: 'person', label: 'Akun' },
    ];

    const adminNav = [
        { id: 'dashboard', href: '/admin/', icon: 'dashboard', label: 'Dashboard' },
        { id: 'users', href: '/admin/users.html', icon: 'group', label: 'Users' },
        { id: 'rooms', href: '/admin/#rooms', icon: 'folder', label: 'Ruangan' },
        { id: 'settings', href: '/admin/#settings', icon: 'settings', label: 'Settings' },
    ];

    function buildBottomNav(items, activeId) {
        return '<nav class="bottom-nav">' + items.map(function (it) {
            const active = it.id === activeId ? ' active' : '';
            return '<a href="' + it.href + '" class="bnav-item' + active + '">' +
                '<span class="material-symbols-outlined">' + it.icon + '</span>' +
                '<span>' + it.label + '</span></a>';
        }).join('') + '</nav>';
    }

    function buildSidebar(items, activeId, userName, onLogout) {
        const menuItems = items.map(function (it) {
            const active = it.id === activeId ? ' active' : '';
            return '<a href="' + it.href + '" class="snav-item' + active + '">' +
                '<span class="material-symbols-outlined">' + it.icon + '</span>' +
                '<span>' + it.label + '</span></a>';
        }).join('');
        return '<aside class="sidebar">' +
            '<a href="/" class="logo">Ambil<span>File</span></a>' +
            '<nav>' + menuItems + '</nav>' +
            '<div class="sidebar-footer">' +
            '<span class="user-name">' + (userName || '') + '</span>' +
            '<button onclick="' + onLogout + '" class="logout-btn">Keluar</button>' +
            '</div></aside>';
    }

    window.Layout = {
        // For user pages: init('dashboard' | 'create-room' | 'api-keys' | 'account', userName)
        userNav: function (activeId, userName) {
            const nav = buildBottomNav(userNav, activeId);
            document.body.insertAdjacentHTML('beforeend', nav);
        },
        // For admin pages
        adminNav: function (activeId) {
            const nav = buildBottomNav(adminNav, activeId);
            document.body.insertAdjacentHTML('beforeend', nav);
        }
    };
})();

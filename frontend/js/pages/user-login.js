// Extracted from user-login.html

// Show OAuth error messages from redirect
        (function() {
            const params = new URLSearchParams(location.search);
            const errMap = {
                cancelled: 'Login Google dibatalkan.',
                oauth: 'Gagal login dengan Google, coba lagi.',
                rejected: 'Akunmu ditolak admin.',
                suspended: 'Akunmu dinonaktifkan. Hubungi admin.',
                server: 'Terjadi kesalahan server, coba lagi.'
            };
            const e = params.get('error');
            if (e && errMap[e]) {
                const el = document.getElementById('err');
                el.textContent = errMap[e];
                el.style.display = 'block';
            }
        })();
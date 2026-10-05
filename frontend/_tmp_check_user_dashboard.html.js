
        function showInfoModal(message, success = false) {
            const icon = document.getElementById('info-modal-icon');
            const glyph = document.getElementById('info-modal-glyph');
            icon.className = 'modal-icon ' + (success ? 'ok' : 'err');
            glyph.textContent = success ? 'check_circle' : 'error';
            document.getElementById('info-modal-msg').textContent = message;
            document.getElementById('info-modal').classList.add('open');
        }
        function closeInfoModal() {
            document.getElementById('info-modal').classList.remove('open');
        }

        async function api(path, opts = {}) {
            const res = await fetch(path, opts);
            if (res.status === 401) { window.location.href = '/user-login.html'; throw new Error('unauth'); }
            return res.json();
        }

        function esc(s) { const d = document.createElement('div'); d.textContent = s; return d.innerHTML; }

        async function load() {
            try {
                const me = await api('/user/me');
                const u = me.user;
                document.getElementById('user-name').textContent = u.name;
                document.getElementById('acc-name').textContent = u.name;
                document.getElementById('acc-email').textContent = u.email;
                document.getElementById('acc-since').textContent = new Date(u.created_at).toLocaleDateString('id-ID', { day: 'numeric', month: 'long', year: 'numeric' });
            } catch { return; }

            const keys = await api('/user/api-keys');
            const kl = document.getElementById('keys-list');
            if (!keys.api_keys || !keys.api_keys.length) {
                kl.innerHTML = '<div class="empty">Belum ada API key. Buat satu di bawah.</div>';
            } else {
                kl.innerHTML = keys.api_keys.map(k =>
                    `<div class="keyrow">
                        <div style="flex:1">
                            <div class="kname">${esc(k.name)}</div>
                            <div class="kdate">Dibuat ${new Date(k.created_at).toLocaleDateString('id-ID')}</div>
                        </div>
                        <button class="iconbtn danger" title="Hapus" onclick="deleteKey('${k.id}','${esc(k.name)}')">
                            <span class="material-symbols-outlined" style="font-size:20px">delete</span>
                        </button>
                    </div>`
                ).join('');
            }

            const rooms = await api('/user/rooms');
            const rl = document.getElementById('rooms-list');
            if (!rooms.rooms || !rooms.rooms.length) {
                rl.innerHTML = '<div class="empty">Belum ada ruangan. <a href="/" style="color:#F6821F;font-weight:600">Buat ruangan</a> dulu.</div>';
            } else {
                rl.innerHTML = rooms.rooms.map(rm =>
                    `<div class="roomrow">
                        <span class="material-symbols-outlined" style="color:#9ca3af">folder</span>
                        <div style="flex:1">
                            <div class="pin">${esc(rm.pin)}</div>
                            <div class="meta">Kadaluarsa ${new Date(rm.expires_at).toLocaleString('id-ID')}</div>
                        </div>
                        <a href="/room/${esc(rm.id)}">Buka</a>
                    </div>`
                ).join('');
            }
        }

        async function createKey() {
            const btn = document.getElementById('create-key-btn');
            const name = document.getElementById('key-name').value.trim();
            btn.disabled = true;
            try {
                const data = await api('/user/api-keys', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name: name || 'API Key' })
                });
                if (data.success) {
                    document.getElementById('newkey-val').textContent = data.api_key.key;
                    document.getElementById('newkey-box').style.display = 'block';
                    document.getElementById('key-name').value = '';
                    load();
                } else {
                    showInfoModal(data.error || 'Gagal membuat API key');
                }
            } catch (e) { if (e.message !== 'unauth') showInfoModal('Tidak bisa terhubung ke server'); }
            btn.disabled = false;
        }

        async function deleteKey(id, name) {
            if (!confirm(`Hapus API key "${name}"? Aplikasi yang memakainya akan berhenti bekerja.`)) return;
            try {
                const data = await api('/user/api-keys/' + id, { method: 'DELETE' });
                if (data.success) load();
                else showInfoModal(data.error || 'Gagal menghapus');
            } catch (e) { if (e.message !== 'unauth') showInfoModal('Tidak bisa terhubung ke server'); }
        }

        function copyNewKey() {
            const v = document.getElementById('newkey-val').textContent;
            navigator.clipboard.writeText(v).then(() => {
                const b = document.querySelector('#newkey-box .copybtn');
                b.textContent = 'Tersalin!';
                setTimeout(() => b.textContent = 'Salin', 1500);
            });
        }

        async function logout() {
            await fetch('/auth/user/logout', { method: 'POST' });
            window.location.href = '/';
        }

        load();
    
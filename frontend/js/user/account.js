// Extracted from user/account.html


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

        let confirmCallback = null;
        function showConfirmModal(message, onYes) {
            document.getElementById('confirm-modal-msg').textContent = message;
            confirmCallback = onYes;
            document.getElementById('confirm-modal').classList.add('open');
        }
        function closeConfirmModal() {
            document.getElementById('confirm-modal').classList.remove('open');
            confirmCallback = null;
        }
        document.getElementById('confirm-modal-yes').addEventListener('click', () => {
            const cb = confirmCallback;
            closeConfirmModal();
            if (cb) cb();
        });

        function esc(s) { const d = document.createElement('div'); d.textContent = s; return d.innerHTML; }

        function setText(id, val) {
            const el = document.getElementById(id);
            if (el) el.textContent = val;
        }
        function setDisplay(id, val) {
            const el = document.getElementById(id);
            if (el) el.style.display = val;
        }

        async function load() {
            try {
                const me = await api('/user/me');
                const u = me.user;
                setText('user-name', u.name);
                setText('side-user-name', u.name);
                setText('acc-name', u.name);
                setText('acc-email', u.email);
                setText('acc-since', new Date(u.created_at).toLocaleDateString('id-ID', { day: 'numeric', month: 'long', year: 'numeric' }));
                const usedBytes = me.storage_used_bytes || 0;
                const limitMB = u.storage_limit_mb || 2048;
                const limitBytes = limitMB * 1024 * 1024;
                const pct = limitBytes > 0 ? Math.min(100, Math.round(usedBytes / limitBytes * 100)) : 0;
                const fill = document.getElementById('storage-fill');
                if (fill) fill.style.width = pct + '%';
                setText('storage-label', (me.storage_used_label || '0 MB') + ' / ' + limitMB + ' MB');
            } catch { return; }
        }

        async function requestApiAccess() {
            const btn = document.getElementById('request-api-btn');
            btn.disabled = true;
            btn.textContent = 'Mengirim...';
            try {
                const res = await api('/user/api-keys/request', { method: 'POST' });
                if (res.success) {
                    document.getElementById('api-locked-msg').textContent = 'Permintaan terkirim! Tunggu persetujuan admin ya.';
                    btn.style.display = 'none';
                } else {
                    showInfoModal(res.error || 'Gagal mengirim permintaan');
                    btn.disabled = false;
                    btn.textContent = 'Minta Persetujuan';
                }
            } catch {
                showInfoModal('Tidak bisa terhubung ke server');
                btn.disabled = false;
                btn.textContent = 'Minta Persetujuan';
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
            showConfirmModal(`Hapus API key "${name}"? Aplikasi yang memakainya akan berhenti bekerja.`, async () => {
                try {
                    const data = await api('/user/api-keys/' + id, { method: 'DELETE' });
                    if (data.success) load();
                    else showInfoModal(data.error || 'Gagal menghapus');
                } catch (e) { if (e.message !== 'unauth') showInfoModal('Tidak bisa terhubung ke server'); }
            });
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
            await api.post('/auth/user/logout');
            window.location.href = '/';
        }

        load();
    
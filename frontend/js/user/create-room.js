// Extracted from user/create-room.html


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
                setText('user-name', me.user.name);
                setText('side-user-name', me.user.name);
            } catch {}
        }

        async function createRoom() {
            const btn = document.getElementById('createRoomBtn');
            const expiryValue = parseInt(document.getElementById('expiryValue').value);
            const expiryUnit = parseInt(document.getElementById('expiryUnit').value);
            const expiryMinutes = expiryValue * expiryUnit;
            if (!expiryMinutes || expiryMinutes < 10 || expiryMinutes > 10080) {
                showInfoModal('Durasi tidak valid (10 menit - 7 hari)');
                return;
            }
            btn.disabled = true;
            btn.textContent = 'Membuat...';
            try {
                const data = await api.post('/api/room/create', { expiry_minutes: expiryMinutes });
                if (data.room_id) {
                    window.location.href = '/room.html?id=' + data.room_id;
                } else {
                    showInfoModal('Gagal membuat ruangan: ' + (data.error || 'Unknown error'));
                    btn.disabled = false;
                    btn.textContent = 'Buat Ruangan';
                }
            } catch {
                showInfoModal('Tidak bisa terhubung ke server');
                btn.disabled = false;
                btn.textContent = 'Buat Ruangan';
            }
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
    

        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        brand: { 50:'#FFF7ED',100:'#FFEDD5',500:'#F6821F',600:'#E06F0A',700:'#C05E08' },
                        ink: '#101828',
                        muted: '#667085',
                        line: '#EAECF0',
                        wash: '#F9FAFB'
                    },
                    fontFamily: { sans: ['Inter','system-ui','sans-serif'] },
                    boxShadow: {
                        card: '0 1px 2px rgba(16,24,40,.06), 0 1px 3px rgba(16,24,40,.1)',
                        pop: '0 12px 32px rgba(16,24,40,.12)'
                    }
                }
            }
        }
    

// Modal
function openModal(name) {
    document.getElementById('mobile-menu').classList.remove('open');
    document.getElementById('modal-' + name).classList.add('active');
}
function closeModal(name) {
    document.getElementById('modal-' + name).classList.remove('active');
}
document.addEventListener('click', (e) => {
    if (e.target.classList.contains('modal')) e.target.classList.remove('active');
});
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') document.querySelectorAll('.modal.active').forEach(m => m.classList.remove('active'));
});

// Info modal — pengganti alert()
function showInfoModal(message, success = false) {
    const badge = document.getElementById('info-icon');
    const glyph = document.getElementById('info-icon-glyph');
    if (success) {
        badge.className = 'mx-auto w-12 h-12 rounded-xl flex items-center justify-center bg-green-50 border border-green-100 text-green-500';
        glyph.textContent = 'check_circle';
    } else {
        badge.className = 'mx-auto w-12 h-12 rounded-xl flex items-center justify-center bg-red-50 border border-red-100 text-red-500';
        glyph.textContent = 'error';
    }
    document.getElementById('info-text').textContent = message;
    openModal('info');
}

// Hamburger
document.getElementById('hamburger').addEventListener('click', () => {
    document.getElementById('mobile-menu').classList.toggle('open');
});
document.querySelectorAll('#mobile-menu a').forEach(a => {
    a.addEventListener('click', () => document.getElementById('mobile-menu').classList.remove('open'));
});

// Create Room
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
        const response = await fetch('/api/room/create', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ expiry_minutes: expiryMinutes })
        });
        const data = await response.json();
        if (data.success) {
            window.location.href = '/room.html?id=' + data.room_id;
        } else {
            showInfoModal('Gagal membuat ruangan: ' + (data.error || 'Unknown error'));
        }
    } catch (error) {
        console.error('Error creating room:', error);
        showInfoModal('Terjadi kesalahan saat membuat ruangan');
    }
    btn.disabled = false;
    btn.textContent = 'Buat Ruangan';
}

// Join Room with PIN
async function joinRoom() {
    const pin = document.getElementById('pinInput').value.trim();
    if (pin.length !== 6 || !/^\d+$/.test(pin)) {
        showInfoModal('PIN harus 6 digit angka');
        return;
    }
    try {
        const response = await fetch('/api/room/pin', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ pin })
        });
        const data = await response.json();
        if (data.success) {
            window.location.href = '/room.html?id=' + data.room_id;
        } else {
            showInfoModal('PIN tidak valid atau ruangan sudah kadaluarsa');
        }
    } catch (error) {
        console.error('Error joining room:', error);
        showInfoModal('Terjadi kesalahan saat join ruangan');
    }
}

// Session check (user auth belum tersedia di backend — tombol tetap "Masuk")
async function checkUserSession() {
    try {
        const response = await fetch('/user/me');
        if (response.ok) {
            const data = await response.json();
            if (data.user && !data.error) {
                const btn = document.getElementById('auth-button');
                btn.textContent = 'Dashboard';
                btn.onclick = () => window.location.href = '/user/dashboard';
            }
        }
    } catch (error) { /* tetap tampil "Masuk" */ }
}
checkUserSession();

// Service worker
if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
        navigator.serviceWorker.register('/sw.js').catch(err => console.warn('SW failed:', err));
    });
}

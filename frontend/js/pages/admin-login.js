// Extracted from admin/login.html

document.getElementById('loginForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const btn = document.getElementById('loginBtn');
    const err = document.getElementById('errorMsg');
    err.classList.add('hidden');
    btn.disabled = true;
    btn.textContent = 'Memproses...';
    try {
        const data = await api.post('/auth/login', {
            email: document.getElementById('email').value.trim(),
            password: document.getElementById('password').value
        });
        if (data.success) {
            window.location.href = '/admin';
        } else {
            err.textContent = data.error || 'Login gagal. Periksa email dan password.';
            err.classList.remove('hidden');
        }
    } catch (e) {
        err.textContent = 'Terjadi kesalahan koneksi. Coba lagi.';
        err.classList.remove('hidden');
    }
    btn.disabled = false;
    btn.textContent = 'Masuk';
});
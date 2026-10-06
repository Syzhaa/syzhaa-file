// Admin dashboard module. Shared globals via window scope.

async function saveAccount() {
    const email = document.getElementById('accountEmail').value.trim();
    const currentPw = document.getElementById('accountCurrentPw').value;
    const newPw = document.getElementById('accountNewPw').value;
    const newPw2 = document.getElementById('accountNewPw2').value;

    if (!currentPw) {
        showToast('Isi password saat ini untuk verifikasi', 'error');
        return;
    }
    if (newPw && newPw !== newPw2) {
        showToast('Password baru tidak sama', 'error');
        return;
    }
    if (newPw && newPw.length < 8) {
        showToast('Password baru minimal 8 karakter', 'error');
        return;
    }

    const data = await apiCall('/admin/account', {
        method: 'POST',
        body: JSON.stringify({ email, current_password: currentPw, new_password: newPw })
    });
    if (data && data.success) {
        showToast('Akun berhasil diperbarui', 'success');
        document.getElementById('accountCurrentPw').value = '';
        document.getElementById('accountNewPw').value = '';
        document.getElementById('accountNewPw2').value = '';
        if (data.admin && data.admin.email) {
            document.getElementById('accountEmail').value = data.admin.email;
            const emailEl = document.getElementById('userEmail');
            if (emailEl) emailEl.textContent = data.admin.email;
        }
    } else if (data) {
        showToast(data.error || 'Gagal menyimpan', 'error');
    }
}


// Auto-split from room.html inline script. Shared globals via window scope.

function showErrorModal(title, message) {
    document.getElementById('error-title').textContent = title;
    document.getElementById('error-message').textContent = message;
    document.getElementById('error-modal').classList.remove('hidden');
}

function closeErrorModal() {
    document.getElementById('error-modal').classList.add('hidden');
    window.location.href = '/';
}

function showInfoModal(message, success = false) {
    const wrap = document.getElementById('info-icon-wrap');
    const icon = document.getElementById('info-icon');
    if (success) {
        wrap.className = 'w-14 h-14 bg-green-50 border border-green-100 rounded-2xl flex items-center justify-center mx-auto mb-4';
        icon.className = 'material-symbols-outlined text-3xl text-green-500';
        icon.textContent = 'check_circle';
    } else {
        wrap.className = 'w-14 h-14 bg-red-50 border border-red-100 rounded-2xl flex items-center justify-center mx-auto mb-4';
        icon.className = 'material-symbols-outlined text-3xl text-red-500';
        icon.textContent = 'error';
    }
    document.getElementById('info-message').textContent = message;
    document.getElementById('info-modal').classList.remove('hidden');
}

function closeInfoModal() {
    document.getElementById('info-modal').classList.add('hidden');
}

function showConfirmModal(message, yesLabel, onYes) {
    document.getElementById('confirm-message').textContent = message;
    document.getElementById('confirm-yes-btn').textContent = yesLabel || 'Hapus';
    confirmCallback = onYes;
    document.getElementById('confirm-modal').classList.remove('hidden');
}

function closeConfirmModal() {
    document.getElementById('confirm-modal').classList.add('hidden');
    confirmCallback = null;
}


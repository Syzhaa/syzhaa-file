// Auto-split from room.html inline script. Shared globals via window scope.

function startCountdownTimer() {
    if (countdownInterval) clearInterval(countdownInterval);
    
    updateCountdown();
    countdownInterval = setInterval(updateCountdown, 1000);
}

function updateCountdown() {
    const expiresAt = new Date(currentRoom.expires_at);
    const now = new Date();
    const diff = expiresAt - now;
    
    if (diff <= 0) {
        document.getElementById('countdown-timer').innerHTML = '<span class="text-red-600">EXPIRED</span>';
        clearInterval(countdownInterval);
        setTimeout(() => {
            showErrorModal(
                'Room Telah Kadaluarsa',
                'Waktu room telah habis. Semua file akan dihapus dari server.'
            );
        }, 1000);
        return;
    }
    
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((diff % (1000 * 60)) / 1000);
    
    let displayText = '';
    let colorClass = 'text-gray-800';
    
    if (days > 0) {
        displayText = `${days}<span class="text-sm">D</span> ${hours}<span class="text-sm">H</span>`;
    } else if (hours > 0) {
        displayText = `${hours}<span class="text-sm">H</span> ${minutes}<span class="text-sm">M</span>`;
        if (hours < 1) colorClass = 'text-orange-600';
    } else if (minutes > 0) {
        displayText = `${minutes}<span class="text-sm">M</span> ${seconds}<span class="text-sm">S</span>`;
        colorClass = 'text-orange-600';
    } else {
        displayText = `${seconds}<span class="text-sm">S</span>`;
        colorClass = 'text-red-600';
    }
    
    document.getElementById('countdown-timer').innerHTML = `<span class="${colorClass}">${displayText}</span>`;
}


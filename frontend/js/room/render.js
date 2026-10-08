// Auto-split from room.html inline script. Shared globals via window scope.

function renderRoom() {
    const mainContent = document.getElementById('main-content');
    mainContent.innerHTML = `
        <!-- Room Info -->
        <div class="bg-white border border-line rounded-2xl shadow-card p-5 sm:p-6 mb-6">
            <div class="grid grid-cols-3 gap-4 sm:gap-6">
                <div class="text-center sm:text-left">
                    <p class="text-[11px] sm:text-xs font-semibold text-muted uppercase tracking-wide mb-1.5">PIN Akses</p>
                    <p class="text-2xl sm:text-4xl font-extrabold font-mono tracking-widest text-ink">${currentRoom.pin}</p>
                </div>
                <div class="text-center timer-pulse">
                    <p class="text-[11px] sm:text-xs font-semibold text-muted uppercase tracking-wide mb-1.5">Sisa Waktu</p>
                    <div id="countdown-timer" class="text-xl sm:text-2xl font-bold text-ink">
                        <span class="text-muted text-base">...</span>
                    </div>
                </div>
                <div class="text-center sm:text-right">
                    <p class="text-[11px] sm:text-xs font-semibold text-muted uppercase tracking-wide mb-1.5">File</p>
                    <p class="text-2xl sm:text-4xl font-extrabold text-ink" id="total-files-count">${totalFilesInRoom}</p>
                </div>
            </div>
            <div class="mt-5 pt-5 border-t border-line">
                <p class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Link berbagi</p>
                <div class="flex gap-2">
                    <input type="text" readonly
                           value="${window.location.href}"
                           class="field flex-1 min-w-0 px-3.5 py-2.5 bg-wash border border-line rounded-lg text-[13px] font-mono text-ink"
                           onclick="this.select()">
                    <button onclick="copyLink()"
                            class="shrink-0 px-4 sm:px-6 py-2.5 bg-ink text-white rounded-lg hover:bg-black text-sm font-semibold transition-colors flex items-center gap-1.5">
                        <span class="material-symbols-outlined text-[18px]">content_copy</span>
                        <span class="hidden sm:inline">Salin</span>
                    </button>
                </div>
            </div>
        </div>

        <!-- Upload Area -->
        <div class="bg-white border-2 border-dashed border-line rounded-2xl p-8 sm:p-12 mb-6 text-center hover:border-brand-500 transition-colors cursor-pointer"
             id="upload-area"
             ondrop="handleDrop(event)"
             ondragover="handleDragOver(event)"
             ondragleave="handleDragLeave(event)"
             onclick="document.getElementById('file-input').click()">
            <span class="material-symbols-outlined text-5xl text-muted mb-3 block">cloud_upload</span>
            <h3 class="text-lg font-bold mb-1.5">Tarik & letakkan file di sini</h3>
            <p class="text-muted text-sm mb-4">atau <span class="font-semibold text-ink underline underline-offset-2">klik untuk memilih</span></p>
            <input type="file" id="file-input" multiple class="hidden" onchange="handleFileSelect(event)">
            <p class="text-xs text-muted mt-4" id="quota-label">${currentRoom.quota_unlimited ? 'Tanpa batas kuota' : 'Maksimal ' + (currentRoom.quota_label || '1 GB') + ' total per ruangan'}</p>
            <div class="mt-3 flex items-center justify-center gap-2 text-xs text-muted">
                <span>Upload bersamaan:</span>
                <select id="upload-parallel-select" class="border border-line rounded-lg text-xs px-2 py-1.5 bg-white font-semibold text-ink" title="Jumlah file yang diupload bersamaan">
                    <option value="1">1</option>
                    <option value="2">2</option>
                    <option value="3" selected>3</option>
                    <option value="4">4</option>
                    <option value="5">5</option>
                    <option value="6">6</option>
                </select>
            </div>
        </div>

        <!-- File List -->
        <div class="bg-white rounded-2xl border border-line shadow-card p-4 sm:p-6">
            <!-- Navigation Bar with Back Button -->
            <div id="folder-nav-bar" class="hidden mb-4 pb-4 border-b border-line">
                <div class="flex items-center gap-3">
                    <button onclick="navigateBack()"
                            class="flex items-center gap-1.5 px-3.5 py-2 bg-wash border border-line hover:border-ink rounded-lg transition-colors">
                        <span class="material-symbols-outlined text-[20px]">arrow_back</span>
                        <span class="font-semibold text-sm">Kembali</span>
                    </button>
                    <div id="folder-path-display" class="flex items-center gap-1.5 text-sm font-semibold text-ink min-w-0 overflow-x-auto">
                        <!-- Path will be inserted here -->
                    </div>
                </div>
            </div>

            <div class="flex flex-wrap justify-between items-center gap-3 mb-4">
                <h3 class="text-lg font-extrabold">File (${currentFiles.length})</h3>
                <div class="flex items-center gap-2.5">
                    <!-- View Toggle -->
                    <div class="flex bg-wash border border-line rounded-lg p-1">
                        <button onclick="switchView('list')" id="view-list-btn"
                                class="p-1.5 px-2.5 rounded-md bg-white shadow-card text-sm font-semibold transition-all text-ink">
                            <span class="material-symbols-outlined text-[20px] block">view_list</span>
                        </button>
                        <button onclick="switchView('card')" id="view-card-btn"
                                class="p-1.5 px-2.5 rounded-md text-sm font-semibold transition-all text-muted">
                            <span class="material-symbols-outlined text-[20px] block">grid_view</span>
                        </button>
                    </div>
                    ${currentFiles.length > 0 ? `
                        <button onclick="downloadAllAsZip()"
                                class="flex items-center gap-1.5 px-3.5 py-2 bg-ink text-white rounded-lg hover:bg-black text-sm font-semibold transition-colors">
                            <span class="material-symbols-outlined text-[20px]">folder_zip</span>
                            <span class="hidden sm:inline">Unduh ZIP</span>
                        </button>
                    ` : ''}
                </div>
            </div>
            
            <!-- Upload Progress (Inline) -->
            <div id="upload-progress-inline" class="hidden mb-4 p-4 bg-brand-50 border border-brand-100 rounded-xl">
                <div class="flex justify-between items-center mb-3 gap-2">
                    <div class="flex items-center gap-2 min-w-0">
                        <div class="spinner w-5 h-5 border-[3px] border-brand-200 border-t-brand-500 rounded-full shrink-0"></div>
                        <span class="font-semibold text-ink text-sm truncate">Mengupload <span id="upload-total-text">0/0</span>...</span>
                    </div>
                    <div class="flex items-center gap-3 shrink-0">
                        <button onclick="toggleSelectAllUpload()" class="text-brand-600 hover:text-brand-700 font-semibold text-sm">
                            Pilih semua
                        </button>
                        <button onclick="cancelUpload()" class="text-red-600 hover:text-red-700 font-semibold text-sm">
                            Batal
                        </button>
                    </div>
                </div>
                <div id="upload-queue-list" class="space-y-2 max-h-60 overflow-y-auto">
                    <!-- File items will be inserted here -->
                </div>
            </div>

            <!-- Batch Actions Bar -->
            <div class="flex flex-wrap items-center justify-between gap-3 mb-4 p-3 bg-wash rounded-xl border border-line">
                <div class="flex items-center gap-3.5">
                    <label class="flex items-center gap-2 cursor-pointer">
                        <input type="checkbox" id="select-all"
                               onchange="toggleSelectAll()"
                               class="w-[18px] h-[18px] rounded border-line accent-[#F6821F]">
                        <span class="text-sm font-semibold text-ink">Pilih semua</span>
                    </label>
                    <span id="selected-count" class="text-sm text-muted">0 dipilih</span>
                </div>
                <div class="flex gap-2">
                    <button onclick="openCreateFolderModal()"
                            class="px-3.5 py-2 bg-white border border-line text-ink rounded-lg text-sm font-semibold hover:border-ink transition-colors flex items-center gap-1.5">
                        <span class="material-symbols-outlined text-[20px]">create_new_folder</span>
                        <span class="hidden sm:inline">Folder</span>
                    </button>
                    <button id="download-selected-btn"
                            onclick="downloadSelected()"
                            disabled
                            class="px-3.5 py-2 bg-brand-500 text-white rounded-lg text-sm font-semibold hover:bg-brand-600 disabled:bg-line disabled:text-muted disabled:cursor-not-allowed transition-colors flex items-center gap-1.5">
                        <span class="material-symbols-outlined text-[20px]">download</span>
                        Unduh
                    </button>
                </div>
            </div>

            <div id="file-list" class="space-y-2.5">
                ${currentFiles.length === 0 ? 
                    '<p class="text-center text-muted py-10 text-sm">Belum ada file yang diupload</p>' :
                    currentFiles.map(file => createFileCard(file)).join('')
                }
            </div>
        </div>
        
        <!-- Download Progress (Fixed Bottom) -->
        <div id="download-progress" class="hidden fixed bottom-20 md:bottom-6 right-4 md:right-6 left-4 md:left-auto bg-white rounded-2xl shadow-pop p-5 border border-line md:w-96 max-h-[60vh] overflow-hidden flex flex-col z-50">
            <div class="flex justify-between items-center mb-3 gap-2">
                <h4 class="font-bold text-sm flex items-center gap-2 min-w-0">
                    <div class="spinner w-5 h-5 border-[3px] border-brand-200 border-t-brand-500 rounded-full shrink-0"></div>
                    <span class="truncate">Mengunduh <span id="download-total-text">0/0</span>...</span>
                </h4>
                <div class="flex items-center gap-3 shrink-0">
                    <button onclick="toggleSelectAllDownload()" class="text-brand-600 hover:text-brand-700 font-semibold text-sm">
                        Semua
                    </button>
                    <button onclick="cancelDownload()" class="text-muted hover:text-ink p-1">
                        <span class="material-symbols-outlined text-[20px]">close</span>
                    </button>
                </div>
            </div>
            <div id="download-queue-list" class="space-y-2 overflow-y-auto flex-1">
                <!-- File items will be inserted here -->
            </div>
        </div>

        <!-- Error Modal -->
        <div id="error-modal" class="hidden fixed inset-0 bg-ink/50 z-[60] flex items-center justify-center p-4">
            <div class="bg-white rounded-2xl shadow-pop max-w-md w-full p-8 text-center">
                <div class="w-16 h-16 bg-red-50 border border-red-100 rounded-2xl flex items-center justify-center mx-auto mb-4">
                    <span class="material-symbols-outlined text-4xl text-red-500">error</span>
                </div>
                <h3 id="error-title" class="text-xl font-extrabold text-ink mb-2">
                    Room Tidak Ditemukan
                </h3>
                <p id="error-message" class="text-muted text-sm mb-6">
                    Room mungkin sudah kadaluarsa atau tidak pernah ada.
                </p>
                <button onclick="closeErrorModal()"
                        class="w-full py-3 bg-ink text-white rounded-lg font-semibold hover:bg-black transition-colors">
                    Kembali ke Beranda
                </button>
            </div>
        </div>

        <!-- Generic Confirm Modal (pengganti confirm) -->
        <div id="confirm-modal" class="hidden fixed inset-0 bg-ink/50 z-[70] flex items-center justify-center p-4" onclick="if(event.target===this)closeConfirmModal()">
            <div class="bg-white rounded-2xl shadow-pop max-w-sm w-full p-6 text-center">
                <div class="w-14 h-14 bg-red-50 border border-red-100 rounded-2xl flex items-center justify-center mx-auto mb-4">
                    <span class="material-symbols-outlined text-3xl text-red-500">warning</span>
                </div>
                <p id="confirm-message" class="text-ink text-sm font-medium mb-6" style="white-space:pre-line"></p>
                <div class="flex gap-2">
                    <button onclick="closeConfirmModal()" class="flex-1 py-3 border border-line rounded-lg font-semibold hover:border-ink transition-colors">Batal</button>
                    <button id="confirm-yes-btn" class="flex-1 py-3 bg-red-500 text-white rounded-lg font-semibold hover:bg-red-600 transition-colors">Hapus</button>
                </div>
            </div>
        </div>
        <!-- Generic Info Modal (pengganti alert) -->
        <div id="info-modal" class="hidden fixed inset-0 bg-ink/50 z-[70] flex items-center justify-center p-4" onclick="if(event.target===this)closeInfoModal()">
            <div class="bg-white rounded-2xl shadow-pop max-w-sm w-full p-6 text-center">
                <div id="info-icon-wrap" class="w-14 h-14 rounded-2xl flex items-center justify-center mx-auto mb-4">
                    <span id="info-icon" class="material-symbols-outlined text-3xl">info</span>
                </div>
                <p id="info-message" class="text-ink text-sm font-medium mb-6" style="white-space:pre-line"></p>
                <button onclick="closeInfoModal()"
                        class="w-full py-3 bg-ink text-white rounded-lg font-semibold hover:bg-black transition-colors">
                    OK
                </button>
            </div>
        </div>

        <!-- Share Modal -->
        <div id="share-modal" class="hidden fixed inset-0 bg-ink/50 z-[70] flex items-end sm:items-center justify-center sm:p-4" onclick="if(event.target===this)closeShareModal()">
            <div class="bg-white rounded-t-3xl sm:rounded-2xl shadow-pop w-full sm:max-w-md p-6 max-h-[90vh] overflow-y-auto">
                <div class="flex items-center justify-between mb-5">
                    <h3 class="text-lg font-bold text-ink">Bagikan Room</h3>
                    <button onclick="closeShareModal()" class="p-2 text-muted hover:text-ink rounded-lg">
                        <span class="material-symbols-outlined">close</span>
                    </button>
                </div>
                
                <!-- Link -->
                <p class="text-xs font-semibold text-muted uppercase tracking-wide mb-2">Link Berbagi</p>
                <div class="flex gap-2 mb-5">
                    <input id="share-link-input" readonly class="flex-1 px-3 py-2.5 bg-wash border border-line rounded-lg text-sm text-ink truncate" value="">
                    <button onclick="copyShareLink()" class="px-4 py-2.5 bg-ink text-white rounded-lg text-sm font-semibold hover:bg-black transition-colors shrink-0">Salin</button>
                </div>
                
                <!-- Info -->
                <div id="share-room-info" class="text-sm text-muted mb-5 space-y-1"></div>
                
                <!-- Permission Settings -->
                <p class="text-xs font-semibold text-muted uppercase tracking-wide mb-3">Pengaturan Room</p>
                
                <div class="space-y-3">
                    <div>
                        <p class="text-sm font-medium text-ink mb-2">Izin akses file</p>
                        <div class="flex flex-wrap gap-2" id="share-perm-btns"></div>
                    </div>
                    
                    <div class="flex items-center justify-between py-2">
                        <div>
                            <p class="text-sm font-medium text-ink">Izinkan hapus file</p>
                            <p class="text-xs text-muted">Matikan untuk sembunyikan tombol hapus</p>
                        </div>
                        <button id="share-delete-toggle" onclick="toggleAllowDelete()" class="w-12 h-7 rounded-full transition-colors relative shrink-0">
                            <span id="share-delete-knob" class="absolute top-1 w-5 h-5 bg-white rounded-full shadow transition-all"></span>
                        </button>
                    </div>
                    
                    <div class="flex items-center justify-between py-2">
                        <div>
                            <p class="text-sm font-medium text-ink">Izinkan download</p>
                            <p class="text-xs text-muted">Matikan = hanya bisa lihat</p>
                        </div>
                        <button id="share-download-toggle" onclick="toggleDownloadPermission()" class="w-12 h-7 rounded-full transition-colors relative shrink-0">
                            <span id="share-download-knob" class="absolute top-1 w-5 h-5 bg-white rounded-full shadow transition-all"></span>
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Create Folder Modal -->
        <div id="create-folder-modal" class="hidden fixed inset-0 bg-ink/50 z-[60] flex items-center justify-center p-4">
            <div class="bg-white rounded-2xl shadow-pop max-w-md w-full p-6 sm:p-8">
                <div class="flex justify-between items-center mb-5">
                    <h3 class="text-xl font-extrabold text-ink">Buat Folder</h3>
                    <button onclick="closeCreateFolderModal()" class="text-muted hover:text-ink p-1">
                        <span class="material-symbols-outlined">close</span>
                    </button>
                </div>
                <div class="space-y-4">
                    <div>
                        <label class="block text-sm font-semibold mb-2 text-ink">Nama folder</label>
                        <input type="text"
                               id="folder-name-input"
                               placeholder="cth: Dokumen"
                               class="field w-full px-4 py-3 border border-line rounded-lg text-[15px]">
                    </div>
                    <div class="flex gap-3">
                        <button onclick="closeCreateFolderModal()"
                                class="flex-1 py-3 border border-line text-ink rounded-lg font-semibold hover:border-ink transition-colors">
                            Batal
                        </button>
                        <button onclick="createFolder()"
                                class="flex-1 py-3 bg-brand-500 text-white rounded-lg font-semibold hover:bg-brand-600 transition-colors">
                            Buat
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    // Start countdown timer
    startCountdownTimer();
}

/* ===== Modal Dialog ===== */
function showModal(title, contentHTML, options) {
    options = options || {};
    const overlay = document.getElementById('modal-overlay');
    const modalTitle = document.getElementById('modalTitle');
    const modalBody = document.getElementById('modalBody');
    const modalFooter = document.getElementById('modalFooter');
    const confirmBtn = document.getElementById('modalConfirmBtn');

    if (!overlay || !modalTitle || !modalBody || !modalFooter || !confirmBtn) return;

    modalTitle.textContent = title || '';
    modalBody.innerHTML = contentHTML || '';

    // Build footer buttons
    modalFooter.innerHTML = '';

    const cancelBtn = document.createElement('button');
    cancelBtn.className = 'btn btn-ghost';
    cancelBtn.textContent = options.cancelText || '取消';
    cancelBtn.onclick = function () {
        if (typeof options.onCancel === 'function') options.onCancel();
        closeModal();
    };
    modalFooter.appendChild(cancelBtn);

    if (options.showConfirm !== false) {
        const okBtn = document.createElement('button');
        okBtn.className = options.confirmClass || 'btn btn-primary';
        okBtn.textContent = options.confirmText || '确认';
        okBtn.id = 'modalConfirmBtn';
        okBtn.onclick = async function () {
            if (typeof options.onConfirm === 'function') {
                try {
                    const result = options.onConfirm();
                    if (result && typeof result.then === 'function') {
                        okBtn.disabled = true;
                        okBtn.innerHTML = '<span class="btn-spinner"><svg width="16" height="16" viewBox="0 0 16 16" fill="none"><circle cx="8" cy="8" r="6" stroke="currentColor" stroke-width="2" stroke-dasharray="28" stroke-linecap="round"/></svg></span>';
                        await result;
                        closeModal();
                    } else {
                        closeModal();
                    }
                } catch (e) {
                    // Error handled by caller
                }
            } else {
                closeModal();
            }
        };
        modalFooter.appendChild(okBtn);
    }

    overlay.classList.remove('hidden');
    overlay.style.display = 'flex';

    // Close on overlay click
    overlay.onclick = function (e) {
        if (e.target === overlay) {
            if (typeof options.onCancel === 'function') options.onCancel();
            closeModal();
        }
    };

    // Close on Escape
    const escHandler = function (e) {
        if (e.key === 'Escape') {
            document.removeEventListener('keydown', escHandler);
            if (typeof options.onCancel === 'function') options.onCancel();
            closeModal();
        }
    };
    document.addEventListener('keydown', escHandler);
}

function closeModal() {
    const overlay = document.getElementById('modal-overlay');
    if (overlay) {
        overlay.classList.add('hidden');
        overlay.style.display = 'none';
    }
}

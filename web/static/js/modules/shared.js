;(function() {
  'use strict';

  var toastCounter = 0;

  function showNotification(message, type) {
    type = type || 'info';
    var id = 'toast-' + (++toastCounter);
    var container = document.querySelector('.toast-container');
    if (!container) {
      container = document.createElement('div');
      container.className = 'toast-container';
      document.body.appendChild(container);
    }

    var icons = {
      success: 'fa-check-circle',
      error: 'fa-exclamation-circle',
      warning: 'fa-exclamation-triangle',
      info: 'fa-info-circle',
    };

    var colors = {
      success: 'text-emerald-500',
      error: 'text-rose-500',
      warning: 'text-amber-500',
      info: 'text-sky-500',
    };

    var titles = {
      success: 'Success',
      error: 'Error',
      warning: 'Warning',
      info: 'Notice',
    };

    var toast = document.createElement('div');
    toast.id = id;
    toast.className = 'toast toast-' + type;
    toast.innerHTML =
      '<i class="fas ' + (icons[type] || icons.info) + ' toast-icon ' + (colors[type] || colors.info) + '"></i>' +
      '<div class="toast-content">' +
        '<div class="toast-title">' + (titles[type] || titles.info) + '</div>' +
        '<div class="toast-message">' + escapeHtml(message) + '</div>' +
      '</div>' +
      '<button class="toast-close" onclick="removeToast(\'' + id + '\')" aria-label="Close">' +
        '<i class="fas fa-times text-xs"></i>' +
      '</button>' +
      '<div class="toast-progress" style="width:100%"></div>';

    container.appendChild(toast);
    requestAnimationFrame(function() {
      var progress = toast.querySelector('.toast-progress');
      if (progress) {
        progress.style.width = '0%';
        progress.style.transition = 'width 3s linear';
      }
    });

    setTimeout(function() {
      removeToast(id);
    }, 3200);
  }

  function removeToast(id) {
    var toast = document.getElementById(id);
    if (!toast) return;
    if (toast.classList.contains('toast-removing')) return;
    toast.classList.add('toast-removing');
    setTimeout(function() {
      if (toast.parentNode) toast.parentNode.removeChild(toast);
      var container = document.querySelector('.toast-container');
      if (container && container.children.length === 0) {
        document.body.removeChild(container);
      }
    }, 200);
  }

  function escapeHtml(text) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(text));
    return div.innerHTML;
  }

  function getCookie(name) {
    var value = '; ' + document.cookie;
    var parts = value.split('; ' + name + '=');
    if (parts.length === 2) return parts.pop().split(';').shift();
  }

  function toggleHeaderMenu(btn) {
    var menu = btn.nextElementSibling;
    if (menu) {
      menu.classList.toggle('hidden');
    }
  }

  document.addEventListener('click', function(e) {
    if (e.target.closest && !e.target.closest('.relative.lg\\:hidden')) {
      var menus = document.querySelectorAll('#headerMenu:not(.hidden)');
      for (var i = 0; i < menus.length; i++) {
        menus[i].classList.add('hidden');
      }
    }
  });

  var modalCounter = 0;

  function showConfirmModal(options) {
    return new Promise(function(resolve) {
      var id = 'modal-' + (++modalCounter);
      var overlay = document.createElement('div');
      overlay.id = id;
      overlay.className = 'fixed inset-0 z-[100] flex items-center justify-center';
      overlay.style.background = 'rgba(0,0,0,0.5)';

      var title = options.title || 'Confirm';
      var message = options.message || 'Are you sure?';
      var confirmText = options.confirmText || 'Confirm';
      var cancelText = options.cancelText || 'Cancel';
      var confirmClass = options.confirmClass || 'bg-[var(--color-primary)] text-white';

      overlay.innerHTML =
        '<div class="bg-[var(--color-surface)] rounded-xl shadow-2xl p-5 mx-4 w-full max-w-sm border border-[var(--color-border)] animate-fade-in-down">' +
          '<h3 class="text-base font-semibold text-[var(--color-text)] mb-2">' + escapeHtml(title) + '</h3>' +
          '<p class="text-sm text-[var(--color-text-secondary)] mb-5">' + message + '</p>' +
          '<div class="flex gap-2 justify-end">' +
            '<button class="modal-cancel px-4 py-2 rounded-lg border border-[var(--color-border)] text-[var(--color-text)] hover:bg-[var(--color-surface-secondary)] text-sm font-medium transition">' + escapeHtml(cancelText) + '</button>' +
            '<button class="modal-confirm px-4 py-2 rounded-lg text-sm font-medium transition shadow-sm ' + confirmClass + ' hover:opacity-90">' + escapeHtml(confirmText) + '</button>' +
          '</div>' +
        '</div>';

      overlay.querySelector('.modal-cancel').addEventListener('click', function() {
        overlay.remove();
        resolve(false);
      });
      overlay.querySelector('.modal-confirm').addEventListener('click', function() {
        overlay.remove();
        resolve(true);
      });
      overlay.addEventListener('click', function(e) {
        if (e.target === overlay) {
          overlay.remove();
          resolve(false);
        }
      });

      document.body.appendChild(overlay);
    });
  }

  function showPromptModal(options) {
    return new Promise(function(resolve) {
      var id = 'modal-' + (++modalCounter);
      var overlay = document.createElement('div');
      overlay.id = id;
      overlay.className = 'fixed inset-0 z-[100] flex items-center justify-center';
      overlay.style.background = 'rgba(0,0,0,0.5)';

      var title = options.title || 'Input';
      var message = options.message || '';
      var defaultValue = options.defaultValue || '';
      var placeholder = options.placeholder || '';
      var confirmText = options.confirmText || 'OK';
      var cancelText = options.cancelText || 'Cancel';

      overlay.innerHTML =
        '<div class="bg-[var(--color-surface)] rounded-xl shadow-2xl p-5 mx-4 w-full max-w-sm border border-[var(--color-border)] animate-fade-in-down">' +
          '<h3 class="text-base font-semibold text-[var(--color-text)] mb-2">' + escapeHtml(title) + '</h3>' +
          (message ? '<p class="text-sm text-[var(--color-text-secondary)] mb-3">' + message + '</p>' : '') +
          '<input type="text" class="modal-input w-full px-3 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] text-[var(--color-text)] text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] mb-4" value="' + escapeHtml(defaultValue) + '" placeholder="' + escapeHtml(placeholder) + '">' +
          '<div class="flex gap-2 justify-end">' +
            '<button class="modal-cancel px-4 py-2 rounded-lg border border-[var(--color-border)] text-[var(--color-text)] hover:bg-[var(--color-surface-secondary)] text-sm font-medium transition">' + escapeHtml(cancelText) + '</button>' +
            '<button class="modal-confirm px-4 py-2 rounded-lg bg-[var(--color-primary)] text-white text-sm font-medium transition shadow-sm hover:opacity-90">' + escapeHtml(confirmText) + '</button>' +
          '</div>' +
        '</div>';

      var input = overlay.querySelector('.modal-input');
      input.focus();
      input.select();

      overlay.querySelector('.modal-cancel').addEventListener('click', function() {
        overlay.remove();
        resolve(null);
      });
      overlay.querySelector('.modal-confirm').addEventListener('click', function() {
        var val = input.value;
        overlay.remove();
        resolve(val);
      });
      input.addEventListener('keydown', function(e) {
        if (e.key === 'Enter') {
          var val = input.value;
          overlay.remove();
          resolve(val);
        }
        if (e.key === 'Escape') {
          overlay.remove();
          resolve(null);
        }
      });
      overlay.addEventListener('click', function(e) {
        if (e.target === overlay) {
          overlay.remove();
          resolve(null);
        }
      });

      document.body.appendChild(overlay);
    });
  }

  function showPaymentModal(maxAmount, currencySym) {
    currencySym = currencySym || '$';
    return new Promise(function(resolve) {
      var id = 'modal-pay-' + (++modalCounter);
      var overlay = document.createElement('div');
      overlay.id = id;
      overlay.className = 'fixed inset-0 z-[100] flex items-center justify-center';
      overlay.style.background = 'rgba(0,0,0,0.5)';

      overlay.innerHTML =
        '<div class="bg-[var(--color-surface)] rounded-xl shadow-2xl p-5 mx-4 w-full max-w-sm border border-[var(--color-border)] animate-fade-in-down">' +
          '<h3 class="text-base font-semibold text-[var(--color-text)] mb-4">Submit Payment</h3>' +

          '<label class="text-xs font-medium text-[var(--color-text-secondary)] block mb-1">Amount (' + currencySym + ')</label>' +
          '<input type="number" step="0.01" min="0.01" max="' + maxAmount.toFixed(2) + '" class="modal-amount w-full px-3 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] text-[var(--color-text)] text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] mb-3" value="' + maxAmount.toFixed(2) + '" placeholder="0.00">' +

          '<label class="text-xs font-medium text-[var(--color-text-secondary)] block mb-1">Method</label>' +
          '<select class="modal-method w-full px-3 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] text-[var(--color-text)] text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] mb-3">' +
            '<option value="mobile_money">Mobile Money</option>' +
            '<option value="bank_transfer">Bank Transfer</option>' +
            '<option value="card">Card Payment</option>' +
            '<option value="cash">Cash</option>' +
          '</select>' +

          '<label class="text-xs font-medium text-[var(--color-text-secondary)] block mb-1">Reference (optional)</label>' +
          '<input type="text" class="modal-reference w-full px-3 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] text-[var(--color-text)] text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] mb-4" placeholder="Transaction ID or note">' +

          '<div class="flex gap-2 justify-end">' +
            '<button class="modal-cancel px-4 py-2 rounded-lg border border-[var(--color-border)] text-[var(--color-text)] hover:bg-[var(--color-surface-secondary)] text-sm font-medium transition">Cancel</button>' +
            '<button class="modal-confirm px-4 py-2 rounded-lg bg-[var(--color-success)] text-white text-sm font-medium transition shadow-sm hover:opacity-90">Confirm Payment</button>' +
          '</div>' +
        '</div>';

      var amountInput = overlay.querySelector('.modal-amount');
      var methodSelect = overlay.querySelector('.modal-method');
      var refInput = overlay.querySelector('.modal-reference');

      amountInput.focus();

      function getResult() {
        var amt = parseFloat(amountInput.value);
        if (isNaN(amt) || amt <= 0 || amt > maxAmount) {
          showNotification('Amount must be between ' + currencySym + '0.01 and ' + currencySym + maxAmount.toFixed(2), 'error');
          return null;
        }
        return {
          amount: amt,
          method: methodSelect.value,
          reference: refInput.value
        };
      }

      overlay.querySelector('.modal-cancel').addEventListener('click', function() {
        overlay.remove();
        resolve(null);
      });
      overlay.querySelector('.modal-confirm').addEventListener('click', function() {
        var result = getResult();
        if (result) { overlay.remove(); resolve(result); }
      });
      amountInput.addEventListener('keydown', function(e) {
        if (e.key === 'Enter') {
          var result = getResult();
          if (result) { overlay.remove(); resolve(result); }
        }
        if (e.key === 'Escape') { overlay.remove(); resolve(null); }
      });
      overlay.addEventListener('click', function(e) {
        if (e.target === overlay) { overlay.remove(); resolve(null); }
      });

      document.body.appendChild(overlay);
    });
  }

  window.showConfirmModal = showConfirmModal;
  window.showPromptModal = showPromptModal;
  window.showPaymentModal = showPaymentModal;
  window.showNotification = showNotification;
  window.removeToast = removeToast;
  window.getCookie = getCookie;
  window.toggleHeaderMenu = toggleHeaderMenu;
})();

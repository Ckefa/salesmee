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

  window.showNotification = showNotification;
  window.removeToast = removeToast;
  window.getCookie = getCookie;
  window.toggleHeaderMenu = toggleHeaderMenu;
})();

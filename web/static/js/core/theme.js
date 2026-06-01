;(function() {
  'use strict';

  const THEME_KEY = 'salesmee-theme';
  const DARK_CLASS = 'dark';

  function getPreferredTheme() {
    const stored = localStorage.getItem(THEME_KEY);
    if (stored) return stored;
    if (window.matchMedia('(prefers-color-scheme: dark)').matches) return 'dark';
    return 'light';
  }

  function applyTheme(theme) {
    const root = document.documentElement;
    if (theme === 'dark') {
      root.classList.add(DARK_CLASS);
    } else {
      root.classList.remove(DARK_CLASS);
    }
    localStorage.setItem(THEME_KEY, theme);
    updateToggleIcons(theme);
  }

  function toggleTheme() {
    const root = document.documentElement;
    const isDark = root.classList.contains(DARK_CLASS);
    applyTheme(isDark ? 'light' : 'dark');
  }

  function updateToggleIcons(theme) {
    document.querySelectorAll('.theme-toggle').forEach(function(el) {
      const sun = el.querySelector('.theme-sun');
      const moon = el.querySelector('.theme-moon');
      if (sun && moon) {
        if (theme === 'dark') {
          sun.classList.add('hidden');
          moon.classList.remove('hidden');
        } else {
          sun.classList.remove('hidden');
          moon.classList.add('hidden');
        }
      }
    });
  }

  function initTheme() {
    applyTheme(getPreferredTheme());
    document.querySelectorAll('.theme-toggle').forEach(function(el) {
      el.addEventListener('click', toggleTheme);
    });
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function(e) {
      if (!localStorage.getItem(THEME_KEY)) {
        applyTheme(e.matches ? 'dark' : 'light');
      }
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initTheme);
  } else {
    initTheme();
  }

  window.SalesMeeTheme = {
    toggle: toggleTheme,
    get: function() { return document.documentElement.classList.contains(DARK_CLASS) ? 'dark' : 'light'; },
    set: applyTheme,
  };
})();

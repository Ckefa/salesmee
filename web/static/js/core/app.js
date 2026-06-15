class SalesMeeApp {
    constructor() {
        this.initProgressBar();
        this.initHTMX();
        this.initAlpine();
        this.setupEventListeners();
        this.initClientFeatures();
        this.initPWA();
    }

    initHTMX() {
        if (typeof htmx !== 'undefined') {
            htmx.config.globalViewTransitions = true;
            htmx.config.revalidateOnLoad = false;
            document.addEventListener('htmx:configRequest', function(event) {
                var token = (window.getCookie || function(name) {
                    var value = '; ' + document.cookie, parts = value.split('; ' + name + '=');
                    if (parts.length === 2) return parts.pop().split(';').shift();
                })('csrf_token');
                if (token) event.detail.headers['X-CSRF-Token'] = token;
            });
        }
    }

    initAlpine() {
        if (typeof Alpine !== 'undefined') {
            Alpine.start();
        }
    }

    initProgressBar() {
        var bar = document.getElementById('pageProgress');
        if (!bar) {
            bar = document.createElement('div');
            bar.id = 'pageProgress';
            bar.className = 'progress-bar';
            document.body.appendChild(bar);
        }
        this.progressBar = bar;

        document.addEventListener('htmx:beforeSend', function() {
            this.setProgress(30);
        }.bind(this));

        document.addEventListener('htmx:afterRequest', function() {
            this.setProgress(100);
            setTimeout(function() {
                this.setProgress(0);
            }.bind(this), 300);
        }.bind(this));
    }

    setProgress(pct) {
        var bar = this.progressBar;
        if (!bar) return;
        if (pct > 0) {
            bar.classList.add('active');
            bar.style.width = pct + '%';
        } else {
            bar.style.opacity = '0';
            setTimeout(function() {
                bar.classList.remove('active');
                bar.style.width = '0%';
                bar.style.opacity = '';
            }, 200);
        }
    }

    setupEventListeners() {
        document.addEventListener('htmx:afterRequest', function(event) {
            this.handleHtmxResponse(event);
        }.bind(this));

        document.addEventListener('htmx:responseError', function(event) {
            this.handleError(event);
        }.bind(this));

        document.addEventListener('htmx:beforeSwap', function(event) {
            if (event.detail.target.id === 'messages-container') {
                var container = event.detail.target;
                var isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 150;
                event.detail.delay = function() {
                    if (isNearBottom) {
                        container.scrollTop = container.scrollHeight;
                    }
                };
            }
        });

        var messagesContainer = document.getElementById('messages-container');
        if (messagesContainer) {
            this.autoScrollMessages(messagesContainer);
            this.initScrollToBottom(messagesContainer);
        }

        document.addEventListener('htmx:afterSwap', function(event) {
            var targetId = event.detail.target.id;
            if (targetId === 'messages-container' || targetId === 'chat-area') {
                var btn = document.getElementById('scrollToBottom');
                if (btn) btn.classList.remove('visible');
                var newContainer = document.getElementById('messages-container');
                if (newContainer) {
                    this.autoScrollMessages(newContainer);
                    this.initScrollToBottom(newContainer);
                }
            }
        }.bind(this));
    }

    initScrollToBottom(container) {
        var btn = document.getElementById('scrollToBottom');
        if (!btn) return;
        var toggle = function() {
            var isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 150;
            btn.classList.toggle('visible', !isNearBottom);
            if (isNearBottom && window.clearUnreadBelow) window.clearUnreadBelow();
        };
        container.addEventListener('scroll', toggle);
        setTimeout(toggle, 100);
    }

    initClientFeatures() {
        this.initRealTimeUpdates();
    }

    handleHtmxResponse(event) {
        var target = event.target;
        if (event.detail.successful) {
            var form = target.closest('form');
            if (form) {
                var input = form.querySelector('input[type="text"], input[type="email"]');
                if (input) {
                    input.value = '';
                }
            }
        }
    }

    handleError(event) {
        console.error('HTMX Error:', event);
        if (typeof showNotification === 'function') {
            showNotification('An error occurred. Please try again.', 'error');
        }
    }

    autoScrollMessages(container) {
        container.scrollTop = container.scrollHeight;
    }

    initPWA() {
        this.registerSW();
        this.setupInstallPrompt();
        this.setupOfflineDetection();
        this.showInstallBanner();
    }

    registerSW() {
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/service-worker.js').catch(function(err) {
                console.warn('SW registration failed:', err);
            });
        }
    }

    setupInstallPrompt() {
        window._deferredPrompt = null;
        window.addEventListener('beforeinstallprompt', function(e) {
            e.preventDefault();
            window._deferredPrompt = e;
            var banner = document.getElementById('pwa-install-banner');
            var prompt = document.getElementById('pwa-install-prompt');
            var hint = document.getElementById('pwa-mobile-hint');
            if (banner && prompt && hint) {
                hint.classList.add('hidden');
                prompt.classList.remove('hidden');
                banner.classList.remove('hidden');
            }
        });

        document.addEventListener('click', function(e) {
            var btn = e.target.closest('#pwaInstallBtn');
            if (btn && window._deferredPrompt) {
                window._deferredPrompt.prompt();
                window._deferredPrompt.userChoice.then(function(result) {
                    if (result.outcome === 'accepted') {
                        pwaDismiss('install', 365);
                    } else {
                        pwaDismiss('install', 7);
                    }
                    window._deferredPrompt = null;
                });
            }
        });

        window.addEventListener('appinstalled', function() {
            pwaDismiss('install', 365);
            window._deferredPrompt = null;
        });

        document.addEventListener('click', function(e) {
            if (e.target.closest('#pwaDismissBtn')) {
                pwaDismiss('install', 7);
                var banner = document.getElementById('pwa-install-banner');
                if (banner) banner.classList.add('hidden');
            }
            if (e.target.closest('#pwaMobileDismissBtn')) {
                pwaDismiss('mobile', 30);
                var banner = document.getElementById('pwa-install-banner');
                if (banner) banner.classList.add('hidden');
            }
        });
    }

    setupOfflineDetection() {
        var toast = document.createElement('div');
        toast.id = 'pwaOfflineToast';
        toast.className = 'pwa-offline-toast hidden';
        document.body.appendChild(toast);

        function showOfflineToast(msg, type) {
            toast.textContent = msg;
            toast.className = 'pwa-offline-toast ' + type;
            toast.classList.remove('hidden');
        }

        function hideOfflineToast() {
            toast.classList.add('hidden');
        }

        window.addEventListener('offline', function() {
            showOfflineToast('You are offline. Some features may be unavailable.', 'offline');
        });

        window.addEventListener('online', function() {
            showOfflineToast('Back online!', 'online');
            setTimeout(hideOfflineToast, 3000);
        });

        if (!navigator.onLine) {
            showOfflineToast('You are offline. Some features may be unavailable.', 'offline');
        }
    }

    showInstallBanner() {
        if (window.matchMedia('(display-mode: standalone)').matches || window.navigator.standalone) return;

        var path = window.location.pathname;
        var validPaths = path === '/' || path.startsWith('/business') || path.startsWith('/client');
        if (!validPaths) return;

        if (pwaShouldShow('install') || pwaShouldShow('mobile')) {
            setTimeout(function() {
                var banner = document.getElementById('pwa-install-banner');
                var prompt = document.getElementById('pwa-install-prompt');
                var hint = document.getElementById('pwa-mobile-hint');
                if (!banner || !prompt || !hint) return;

                if (window._deferredPrompt) {
                    hint.classList.add('hidden');
                    prompt.classList.remove('hidden');
                } else if (isMobile()) {
                    if (pwaShouldShow('mobile')) {
                        prompt.classList.add('hidden');
                        hint.classList.remove('hidden');
                        updateMobileHintText();
                    } else {
                        return;
                    }
                } else {
                    return;
                }
                banner.classList.remove('hidden');
            }, 3000);
        }
    }

    initRealTimeUpdates() {
        setInterval(function() {
            this.checkForNewMessages();
        }.bind(this), 5000);
    }

    checkForNewMessages() {
        var currentPath = window.location.pathname;
        if (currentPath.includes('/client/businesses/') && currentPath.includes('/messages')) {
            var businessId = currentPath.split('/')[3];
            this.fetchNewMessages(businessId);
        }
    }

    fetchNewMessages(businessId) {
        fetch('/client/businesses/' + businessId + '/messages')
            .then(function(response) { return response.json(); })
            .then(function(data) {})
            .catch(function() {});
    }
}

document.addEventListener('DOMContentLoaded', function() {
    new SalesMeeApp();
});

window.SalesMeeApp = SalesMeeApp;

function pwaDismiss(type, days) {
    var key = 'pwa_' + type + '_dismissed';
    var data = { dismissed: true, at: Date.now(), ttl: days * 86400000 };
    try { localStorage.setItem(key, JSON.stringify(data)); } catch(e) {}
    var banner = document.getElementById('pwa-install-banner');
    if (banner) banner.classList.add('hidden');
}

function pwaShouldShow(type) {
    var key = 'pwa_' + type + '_dismissed';
    try {
        var raw = localStorage.getItem(key);
        if (!raw) return true;
        var data = JSON.parse(raw);
        if (!data.dismissed) return true;
        return (Date.now() - data.at) > data.ttl;
    } catch(e) {
        return true;
    }
}

function isMobile() {
    return /Android|iPhone|iPad|iPod|webOS|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent)
        || (navigator.maxTouchPoints > 0 && window.innerWidth < 768);
}

function updateMobileHintText() {
    var el = document.getElementById('pwaMobileHintText');
    if (!el) return;
    if (/iPhone|iPad|iPod/i.test(navigator.userAgent)) {
        el.textContent = 'Tap the Share button then "Add to Home Screen" for the best experience.';
    } else if (/Android/i.test(navigator.userAgent)) {
        el.textContent = 'Open the Chrome menu (⋮) and select "Install app" or "Add to Home Screen".';
    } else {
        el.textContent = 'Use your browser\'s menu to add SalesMee to your home screen.';
    }
}

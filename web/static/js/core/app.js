class SalesMeeApp {
    constructor() {
        this.initProgressBar();
        this.initHTMX();
        this.initAlpine();
        this.setupEventListeners();
        this.initClientFeatures();
    }

    initHTMX() {
        if (typeof htmx !== 'undefined') {
            htmx.config.globalViewTransitions = true;
            htmx.config.revalidateOnLoad = false;
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
        }
    }

    initClientFeatures() {
        this.startHeartbeat();
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
        var observer = new MutationObserver(function() {
            container.scrollTop = container.scrollHeight;
        });
        observer.observe(container, { childList: true, subtree: true });
    }

    startHeartbeat() {
        setInterval(function() {
            fetch('/client/heartbeat', { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
                .catch(function() {});
        }, 30000);
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

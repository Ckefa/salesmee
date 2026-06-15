const CACHE = 'salesmee-v1';
const STATIC_CACHE = 'salesmee-static-v1';
const OFFLINE_URL = '/static/pwa/offline.html';

const PRECACHE_URLS = [
  OFFLINE_URL,
  '/static/images/icon-192.png',
  '/static/images/icon-512.png',
  '/static/images/apple-touch-icon.png',
  '/static/images/salesmee.ico',
  '/static/images/salesmeebrand.png',
];

self.addEventListener('install', function(event) {
  event.waitUntil(
    caches.open(CACHE).then(function(cache) {
      return cache.addAll(PRECACHE_URLS);
    }).then(function() {
      return self.skipWaiting();
    })
  );
});

self.addEventListener('activate', function(event) {
  event.waitUntil(
    caches.keys().then(function(keys) {
      return Promise.all(
        keys.filter(function(key) {
          return key !== CACHE && key !== STATIC_CACHE;
        }).map(function(key) {
          return caches.delete(key);
        })
      );
    }).then(function() {
      return self.clients.claim();
    })
  );
});

self.addEventListener('fetch', function(event) {
  var request = event.request;
  var url = new URL(request.url);

  // API/XHR requests — network only, never cache
  if (request.headers.get('Accept') && request.headers.get('Accept').includes('text/html') === false) {
    return;
  }

  // HTMX fragment requests — network only
  if (url.searchParams.has('_') || request.headers.get('HX-Request')) {
    event.respondWith(networkFirst(request));
    return;
  }

  // HTML navigations — network first, fall back to cache, then offline page
  if (request.mode === 'navigate') {
    event.respondWith(networkFirst(request));
    return;
  }

  // Static assets (JS, CSS, images, fonts) — cache first
  if (
    url.pathname.match(/\.(js|css|png|jpg|jpeg|gif|svg|ico|woff2?|ttf|eot)$/) ||
    url.pathname.startsWith('/static/')
  ) {
    event.respondWith(cacheFirst(request));
    return;
  }

  // Everything else — network first
  event.respondWith(networkFirst(request));
});

function cacheFirst(request) {
  return caches.match(request).then(function(response) {
    if (response) return response;
    return fetch(request).then(function(networkResponse) {
      if (!networkResponse || networkResponse.status !== 200) return networkResponse;
      var clone = networkResponse.clone();
      caches.open(STATIC_CACHE).then(function(cache) {
        cache.put(request, clone);
      });
      return networkResponse;
    }).catch(function() {
      return caches.match(OFFLINE_URL);
    });
  });
}

function networkFirst(request) {
  return fetch(request).then(function(response) {
    if (response && response.status === 200) {
      var clone = response.clone();
      caches.open(CACHE).then(function(cache) {
        cache.put(request, clone);
      });
    }
    return response;
  }).catch(function() {
    return caches.match(request).then(function(cached) {
      return cached || caches.match(OFFLINE_URL);
    });
  });
}

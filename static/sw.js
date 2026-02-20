var CACHE_NAME = 'bonusperme-v2';
var PRECACHE = [
    '/',
    '/manifest.json',
    '/icon-192.png',
    '/fonts/fonts.css'
];

self.addEventListener('install', function(e) {
    e.waitUntil(
        caches.open(CACHE_NAME).then(function(cache) {
            return cache.addAll(PRECACHE);
        })
    );
    self.skipWaiting();
});

self.addEventListener('activate', function(e) {
    e.waitUntil(
        caches.keys().then(function(keys) {
            return Promise.all(
                keys.filter(function(k) { return k !== CACHE_NAME; })
                    .map(function(k) { return caches.delete(k); })
            );
        })
    );
    self.clients.claim();
});

self.addEventListener('fetch', function(e) {
    if (e.request.method !== 'GET') return;
    if (e.request.url.includes('/api/')) return;
    if (!e.request.url.startsWith(self.location.origin)) return;

    e.respondWith(
        fetch(e.request)
            .then(function(response) {
                if (response.status === 200) {
                    var clone = response.clone();
                    caches.open(CACHE_NAME).then(function(cache) {
                        cache.put(e.request, clone);
                    });
                }
                return response;
            })
            .catch(function() {
                return caches.match(e.request).then(function(cached) {
                    if (cached) return cached;
                    if (e.request.mode === 'navigate') {
                        return caches.match('/');
                    }
                });
            })
    );
});

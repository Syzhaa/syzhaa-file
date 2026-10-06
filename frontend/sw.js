const CACHE_NAME = 'ambilfile-v1.1.0';
const RUNTIME_CACHE = 'ambilfile-runtime-v1.1.0';

// Assets to cache on install (ONLY static assets, NOT HTML/JS)
const urlsToCache = [
  '/manifest.json',
  '/icon-192x192.png',
  '/icon-512x512.png'
];

// Install event - cache static assets
self.addEventListener('install', event => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then(cache => {
        console.log('Opened cache');
        return cache.addAll(urlsToCache);
      })
      .then(() => self.skipWaiting())
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', event => {
  const currentCaches = [CACHE_NAME, RUNTIME_CACHE];
  event.waitUntil(
    caches.keys().then(cacheNames => {
      return cacheNames.filter(cacheName => !currentCaches.includes(cacheName));
    }).then(cachesToDelete => {
      return Promise.all(cachesToDelete.map(cacheToDelete => {
        console.log('Deleting old cache:', cacheToDelete);
        return caches.delete(cacheToDelete);
      }));
    }).then(() => self.clients.claim())
  );
});

// Fetch event - NETWORK FIRST for HTML/JS/CSS, cache for images only
self.addEventListener('fetch', event => {
  // Skip cross-origin requests
  if (!event.request.url.startsWith(self.location.origin)) {
    return;
  }

  // Skip API requests from caching
  if (event.request.url.includes('/api/')) {
    return;
  }

  const url = new URL(event.request.url);
  const isStaticAsset = /\.(jpg|jpeg|png|gif|svg|webp|ico|woff|woff2|ttf|eot)$/i.test(url.pathname);

  // For images/fonts: cache-first (they don't change)
  if (isStaticAsset) {
    event.respondWith(
      caches.match(event.request).then(cachedResponse => {
        if (cachedResponse) {
          return cachedResponse;
        }
        return fetch(event.request).then(response => {
          return caches.open(RUNTIME_CACHE).then(cache => {
            cache.put(event.request, response.clone());
            return response;
          });
        });
      })
    );
    return;
  }

  // For HTML/JS/CSS: NETWORK FIRST (always get fresh version!)
  event.respondWith(
    fetch(event.request)
      .then(async response => {
        // Cache the fresh response for offline use (clone before returning)
        if (event.request.method === 'GET' && response.status === 200 && response.type === 'basic') {
          try {
            const cache = await caches.open(RUNTIME_CACHE);
            await cache.put(event.request, response.clone());
          } catch (e) {
            // Clone failed, ignore caching
          }
        }
        return response;
      })
      .catch(() => {
        // Network failed, try cache as fallback
        return caches.match(event.request).then(cachedResponse => {
          if (cachedResponse) {
            return cachedResponse;
          }
          // If no cache, show offline page for navigation requests
          if (event.request.mode === 'navigate') {
            return caches.match('/offline.html');
          }
        });
      })
  );
});

// Push notification support
self.addEventListener('push', event => {
  const options = {
    body: event.data ? event.data.text() : 'New notification',
    icon: '/icon-192x192.png',
    badge: '/icon-72x72.png',
    vibrate: [200, 100, 200],
    data: {
      dateOfArrival: Date.now(),
      primaryKey: 1
    }
  };

  event.waitUntil(
    self.registration.showNotification('AmbilFile', options)
  );
});

// Notification click handler
self.addEventListener('notificationclick', event => {
  event.notification.close();
  event.waitUntil(
    clients.openWindow('/')
  );
});

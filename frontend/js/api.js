// api.js — shared fetch helper for AmbilFile frontend.
// - Always sends credentials (cookies) so logged-in sessions work.
// - Parses JSON responses, throws on HTTP errors.
// - Redirects to login on 401.
// - CSRF: slot ready via getCSRFToken() (backend CSRF currently disabled).

(function () {
    // Override this if your backend enables CSRF later.
    function getCSRFToken() {
        const m = document.cookie.match(/(?:^|;)\s*csrf_token=([^;]+)/);
        return m ? decodeURIComponent(m[1]) : '';
    }

    function loginUrl(path) {
        return path.startsWith('/admin') ? '/admin/login.html' : '/user-login.html';
    }

    async function api(path, options) {
        options = options || {};
        const headers = Object.assign(
            { 'Content-Type': 'application/json' },
            options.headers || {}
        );
        const csrf = getCSRFToken();
        if (csrf) headers['X-CSRF-Token'] = csrf;

        const res = await fetch(path, {
            method: options.method || 'GET',
            headers: headers,
            credentials: 'include',
            body: options.body !== undefined ? options.body : undefined,
        });

        if (res.status === 401) {
            // Session checks on public pages (e.g. landing) pass {noRedirect:true}
            // so anonymous visitors aren't bounced to the login page.
            if (!options.noRedirect) {
                window.location.href = loginUrl(path);
            }
            throw new Error('Unauthorized');
        }

        const text = await res.text();
        let data = null;
        try { data = text ? JSON.parse(text) : null; } catch (e) { /* non-JSON */ }

        if (!res.ok) {
            const msg = (data && (data.error || data.message)) || ('HTTP ' + res.status);
            throw new Error(msg);
        }
        return data;
    }

    api.get = function (path, opts) { return api(path, Object.assign({ method: 'GET' }, opts)); };
    api.post = function (path, body, opts) {
        return api(path, Object.assign({ method: 'POST', body: body !== undefined ? JSON.stringify(body) : undefined }, opts));
    };
    api.put = function (path, body, opts) {
        return api(path, Object.assign({ method: 'PUT', body: body !== undefined ? JSON.stringify(body) : undefined }, opts));
    };
    api.del = function (path, opts) { return api(path, Object.assign({ method: 'DELETE' }, opts)); };

    // Uniform error display (replaces scattered alert() calls).
    api.showError = function (msg) {
        // Prefer a toast if the page provides one, else fall back to alert.
        if (typeof window.showToast === 'function') { window.showToast(msg, 'error'); return; }
        alert(msg);
    };

    window.api = api;
})();

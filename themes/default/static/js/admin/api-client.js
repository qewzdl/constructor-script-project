/**
 * Admin API Client Module
 * Handles authenticated API requests, CSRF tokens, and endpoint management
 */
(() => {
    /**
     * Create API client with authentication support
     */
    const createApiClient = ({ auth, endpoints: endpointsConfig = {} }) => {
        const stateChangingMethods = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

        /**
         * Read cookie value by name
         */
        const readCookie = (name) => {
            if (!name || typeof document?.cookie !== 'string') {
                return '';
            }
            const cookies = document.cookie.split('; ');
            for (let index = 0; index < cookies.length; index += 1) {
                const cookie = cookies[index];
                if (!cookie) {
                    continue;
                }
                const [key, ...rest] = cookie.split('=');
                if (key === name) {
                    return decodeURIComponent(rest.join('='));
                }
            }
            return '';
        };

        /**
         * Get CSRF token from cookies
         */
        const getCSRFCookie = () => readCookie('csrf_token');

        /**
         * Build request init object with authentication headers
         */
        const buildAuthenticatedRequestInit = (options = {}) => {
            const init = { ...options };
            const headers = Object.assign({}, options.headers || {});
            const method = (options.method || 'GET').toUpperCase();

            init.method = method;

            if (options.body && !(options.body instanceof FormData)) {
                headers['Content-Type'] =
                    headers['Content-Type'] || 'application/json';
            }

            const token =
                auth && typeof auth.getToken === 'function'
                    ? auth.getToken()
                    : undefined;
            if (token && !headers.Authorization) {
                headers.Authorization = `Bearer ${token}`;
            }

            if (stateChangingMethods.has(method)) {
                const csrfToken = getCSRFCookie();
                if (csrfToken && !headers['X-CSRF-Token']) {
                    headers['X-CSRF-Token'] = csrfToken;
                }
            }

            init.headers = headers;
            init.credentials = 'include';
            return init;
        };

        /**
         * Perform authenticated fetch request
         */
        const authenticatedFetch = (url, options = {}) =>
            fetch(url, buildAuthenticatedRequestInit(options));

        /**
         * Perform API request with error handling
         */
        const apiRequest = async (url, options = {}) => {
            const response = await authenticatedFetch(url, options);

            const contentType = response.headers.get('content-type') || '';
            const isJson = contentType.includes('application/json');
            const payload = isJson
                ? await response.json().catch(() => null)
                : await response.text();

            if (!response.ok) {
                const message =
                    payload && typeof payload === 'object' && payload.error
                        ? payload.error
                        : typeof payload === 'string'
                        ? payload
                        : 'Request failed';
                const error = new Error(message);
                error.status = response.status;
                error.payload = payload;
                throw error;
            }

            return payload;
        };

        /**
         * Check if user is authenticated
         */
        const requireAuth = (redirectUrl = '/admin') => {
            if (!auth || typeof auth.getToken !== 'function') {
                return true;
            }
            if (!auth.getToken()) {
                window.location.href = `/login?redirect=${encodeURIComponent(redirectUrl)}`;
                return false;
            }
            return true;
        };

        return {
            request: apiRequest,
            fetch: authenticatedFetch,
            requireAuth,
            endpoints: endpointsConfig,
        };
    };

    // Export to global namespace
    window.AdminApiClient = {
        create: createApiClient,
    };
})();

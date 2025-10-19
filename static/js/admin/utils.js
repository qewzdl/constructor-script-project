(() => {
    const formatDate = (value) => {
        if (!value) {
            return '—';
        }
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            return value;
        }
        try {
            return new Intl.DateTimeFormat(undefined, {
                dateStyle: 'medium',
                timeStyle: 'short',
            }).format(date);
        } catch (error) {
            return date.toLocaleString();
        }
    };

    const booleanLabel = (value) => (value ? 'Yes' : 'No');

    const createElement = (tag, options = {}) => {
        const element = document.createElement(tag);
        if (options.className) {
            element.className = options.className;
        }
        if (options.textContent !== undefined) {
            element.textContent = options.textContent;
        }
        if (options.html !== undefined) {
            element.innerHTML = options.html;
        }
        return element;
    };

    const buildAbsoluteUrl = (path, site) => {
        if (!path) {
            return '';
        }
        const trimmedPath = path.startsWith('/') ? path : `/${path}`;
        const siteUrl = site?.url || site?.Url;
        if (siteUrl) {
            try {
                return new URL(trimmedPath, siteUrl).toString();
            } catch (error) {
                // Fall back to returning the path below
            }
        }
        if (window.location?.origin) {
            try {
                return new URL(trimmedPath, window.location.origin).toString();
            } catch (error) {
                // If URL construction fails, return the trimmed path
            }
        }
        return trimmedPath;
    };

    const randomId = () => {
        if (window.crypto && typeof window.crypto.randomUUID === 'function') {
            return window.crypto.randomUUID();
        }
        return `id-${Math.random().toString(36).slice(2, 11)}`;
    };

    const normaliseString = (value) => {
        if (typeof value === 'string') {
            return value;
        }
        if (value === null || value === undefined) {
            return '';
        }
        if (typeof value === 'number' || typeof value === 'boolean') {
            return String(value);
        }
        return '';
    };

    const ensureArray = (value) => (Array.isArray(value) ? value : []);

    const createImageState = (image = {}) => ({
        clientId: randomId(),
        url: normaliseString(image.url ?? image.URL ?? ''),
        alt: normaliseString(image.alt ?? image.Alt ?? ''),
        caption: normaliseString(image.caption ?? image.Caption ?? ''),
    });

    const SVG_NS = 'http://www.w3.org/2000/svg';
    const createSvgElement = (tag, attributes = {}) => {
        const element = document.createElementNS(SVG_NS, tag);
        Object.entries(attributes).forEach(([key, value]) => {
            if (value !== undefined && value !== null) {
                element.setAttribute(key, value);
            }
        });
        return element;
    };

    const formatNumber = (value) => {
        const numeric = Number(value);
        if (Number.isNaN(numeric)) {
            return '0';
        }
        try {
            return numeric.toLocaleString();
        } catch (error) {
            return String(numeric);
        }
    };

    const monthFormatter = (() => {
        try {
            return new Intl.DateTimeFormat(undefined, {
                month: 'short',
                year: 'numeric',
            });
        } catch (error) {
            return null;
        }
    })();

    const formatMonthLabel = (value) => {
        if (!value) {
            return '';
        }
        const date = value instanceof Date ? value : new Date(value);
        if (Number.isNaN(date.getTime())) {
            return typeof value === 'string' ? value : '';
        }
        if (monthFormatter) {
            try {
                return monthFormatter.format(date);
            } catch (error) {
                // Ignore and fall back to ISO-like formatting.
            }
        }
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        return `${year}-${month}`;
    };

    window.AdminUtils = {
        formatDate,
        booleanLabel,
        createElement,
        buildAbsoluteUrl,
        randomId,
        normaliseString,
        ensureArray,
        createImageState,
        createSvgElement,
        formatNumber,
        formatMonthLabel,
    };
})();
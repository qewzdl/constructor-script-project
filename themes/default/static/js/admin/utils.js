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

    const parseDateInput = (value) => {
        if (value instanceof Date) {
            const time = value.getTime();
            return Number.isNaN(time) ? null : new Date(time);
        }
        if (typeof value === 'number') {
            const date = new Date(value);
            return Number.isNaN(date.getTime()) ? null : date;
        }
        if (typeof value === 'string') {
            const trimmed = value.trim();
            if (!trimmed) {
                return null;
            }
            const date = new Date(trimmed);
            return Number.isNaN(date.getTime()) ? null : date;
        }
        return null;
    };

    const formatDateTimeInput = (value) => {
        const date = parseDateInput(value);
        if (!date) {
            return '';
        }
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');
        return `${year}-${month}-${day}T${hours}:${minutes}`;
    };

    const booleanLabel = (value) => (value ? 'Yes' : 'No');

    const createElement = (tag, options = {}) => {
        const {
            className,
            textContent,
            html,
            attributes,
            dataset,
            children,
            ...rest
        } = options || {};

        const element = document.createElement(tag);

        if (className) {
            element.className = className;
        }
        if (textContent !== undefined) {
            element.textContent = textContent;
        }
        if (html !== undefined) {
            element.innerHTML = html;
        }

        const assignPropertyOrAttribute = (key, value) => {
            if (value === undefined || value === null) {
                return;
            }
            if (key === 'style' && typeof value === 'object') {
                Object.assign(element.style, value);
                return;
            }
            if (key in element) {
                try {
                    element[key] = value;
                    return;
                } catch (error) {
                    // Fall back to setAttribute below
                }
            }
            element.setAttribute(key, value);
        };

        if (attributes && typeof attributes === 'object') {
            Object.entries(attributes).forEach(([key, value]) => {
                assignPropertyOrAttribute(key, value);
            });
        }

        if (dataset && typeof dataset === 'object') {
            Object.entries(dataset).forEach(([key, value]) => {
                if (value !== undefined && value !== null) {
                    element.dataset[key] = value;
                }
            });
        }

        Object.entries(rest).forEach(([key, value]) => {
            assignPropertyOrAttribute(key, value);
        });

        if (Array.isArray(children)) {
            const appendChild = (child) => {
                if (child instanceof Node) {
                    element.appendChild(child);
                } else if (Array.isArray(child)) {
                    child.forEach(appendChild);
                } else if (child !== undefined && child !== null) {
                    element.append(String(child));
                }
            };
            children.forEach(appendChild);
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

    const CYRILLIC_TO_LATIN = {
        а: 'a',
        б: 'b',
        в: 'v',
        г: 'g',
        д: 'd',
        е: 'e',
        ё: 'e',
        ж: 'zh',
        з: 'z',
        и: 'i',
        й: 'y',
        к: 'k',
        л: 'l',
        м: 'm',
        н: 'n',
        о: 'o',
        п: 'p',
        р: 'r',
        с: 's',
        т: 't',
        у: 'u',
        ф: 'f',
        х: 'h',
        ц: 'ts',
        ч: 'ch',
        ш: 'sh',
        щ: 'sch',
        ъ: '',
        ы: 'y',
        ь: '',
        э: 'e',
        ю: 'yu',
        я: 'ya',
    };

    const slugify = (value, { maxLength = 80 } = {}) => {
        const normalised = normaliseString(value).trim().toLowerCase();
        if (!normalised) {
            return '';
        }
        const transliterated = normalised.replace(
            /[а-яё]/g,
            (char) => CYRILLIC_TO_LATIN[char] ?? ''
        );
        const withoutDiacritics =
            typeof transliterated.normalize === 'function'
                ? transliterated.normalize('NFD').replace(/[\u0300-\u036f]/g, '')
                : transliterated;
        const cleaned = withoutDiacritics
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/-{2,}/g, '-')
            .replace(/^-+|-+$/g, '');
        const limited =
            Number.isFinite(maxLength) && maxLength > 0
                ? cleaned.slice(0, maxLength)
                : cleaned;
        return limited.replace(/^-+|-+$/g, '');
    };

    const createImageState = (image = {}) => ({
        clientId: randomId(),
        url: normaliseString(image.url ?? image.URL ?? ''),
        alt: normaliseString(image.alt ?? image.Alt ?? ''),
        caption: normaliseString(image.caption ?? image.Caption ?? ''),
    });

    const createFileState = (file = {}) => ({
        clientId: randomId(),
        url: normaliseString(file.url ?? file.URL ?? ''),
        label: normaliseString(file.label ?? file.Label ?? file.name ?? file.Name ?? ''),
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

    const periodFormatter = (() => {
        try {
            return new Intl.DateTimeFormat(undefined, {
                day: 'numeric',
                month: 'short',
                year: 'numeric',
            });
        } catch (error) {
            return null;
        }
    })();

    const formatPeriodLabel = (value) => {
        if (!value) {
            return '';
        }
        const date = value instanceof Date ? value : new Date(value);
        if (Number.isNaN(date.getTime())) {
            return typeof value === 'string' ? value : '';
        }
        if (periodFormatter) {
            try {
                return periodFormatter.format(date);
            } catch (error) {
                // Ignore and fall back to ISO-like formatting.
            }
        }
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        return `${year}-${month}-${day}`;
    };

    const parseContentDispositionFilename = (header) => {
        if (!header || typeof header !== 'string') {
            return '';
        }
        const filenameStarMatch = header.match(/filename\*=UTF-8''([^;]+)/i);
        if (filenameStarMatch && filenameStarMatch[1]) {
            try {
                return decodeURIComponent(filenameStarMatch[1]);
            } catch (error) {
                return filenameStarMatch[1];
            }
        }
        const filenameMatch = header.match(/filename\s*=\s*"?([^";]+)"?/i);
        if (filenameMatch && filenameMatch[1]) {
            return filenameMatch[1].trim();
        }
        return '';
    };

    window.AdminUtils = {
        formatDate,
        parseDateInput,
        formatDateTimeInput,
        booleanLabel,
        createElement,
        buildAbsoluteUrl,
        randomId,
        normaliseString,
        ensureArray,
        slugify,
        createImageState,
        createFileState,
        createSvgElement,
        formatNumber,
        formatPeriodLabel,
        parseContentDispositionFilename,
    };
})();

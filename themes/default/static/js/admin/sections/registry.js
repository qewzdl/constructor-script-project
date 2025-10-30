(() => {
    const definitions = new Map();

    const normaliseType = (type) =>
        typeof type === 'string' ? type.trim().toLowerCase() : '';

    const register = (type, definition = {}) => {
        const normalised = normaliseType(type);
        if (!normalised || !definition) {
            return;
        }

        const entry = {
            type: normalised,
            label:
                typeof definition.label === 'string' && definition.label.trim()
                    ? definition.label.trim()
                    : normalised,
            order: Number.isFinite(definition.order) ? definition.order : 0,
            description:
                typeof definition.description === 'string'
                    ? definition.description.trim()
                    : '',
            supportsElements:
                definition.supportsElements === undefined
                    ? true
                    : Boolean(definition.supportsElements),
            settings:
                definition.settings && typeof definition.settings === 'object'
                    ? definition.settings
                    : undefined,
        };

        definitions.set(normalised, entry);
    };

    const get = (type) => definitions.get(normaliseType(type));

    const getDefinitions = () => {
        const entries = Array.from(definitions.entries());
        return entries.reduce((accumulator, [key, value]) => {
            accumulator[key] = value;
            return accumulator;
        }, {});
    };

    const getOrderedTypes = () =>
        Array.from(definitions.entries())
            .sort(([, a], [, b]) => (a.order || 0) - (b.order || 0))
            .map(([type]) => type);

    const getDefaultType = () => {
        const ordered = getOrderedTypes();
        return ordered.length ? ordered[0] : 'standard';
    };

    const parseDefinitionJSON = (elementId) => {
        if (typeof document === 'undefined') {
            return null;
        }
        const node = document.getElementById(elementId);
        if (!node) {
            return null;
        }
        const raw = node.textContent || node.innerText || '';
        if (!raw.trim()) {
            return null;
        }
        try {
            return JSON.parse(raw);
        } catch (error) {
            console.error('Failed to parse section definitions', error);
            return null;
        }
    };

    const initialDefinitions = parseDefinitionJSON('section-definitions-data');
    if (initialDefinitions && typeof initialDefinitions === 'object') {
        Object.entries(initialDefinitions).forEach(([type, definition]) => {
            register(type, definition);
        });
    }

    const ensureRegistered = (type, definition) => {
        if (!definitions.has(normaliseType(type))) {
            register(type, definition);
        }
    };

    ensureRegistered('standard', {
        label: 'Standard section',
        order: 0,
        supportsElements: true,
        description:
            'Flexible content area for combining paragraphs, media, and lists.',
    });

    ensureRegistered('hero', {
        label: 'Hero section',
        order: 10,
        supportsElements: false,
        description:
            'Prominent introduction block without additional content elements.',
    });

    ensureRegistered('grid', {
        label: 'Grid section',
        order: 15,
        supportsElements: true,
        description:
            'Displays content blocks in a responsive grid. Add at least two elements for a balanced layout.',
    });

    ensureRegistered('posts_list', {
        label: 'Posts list',
        order: 20,
        supportsElements: false,
        description: 'Automatically displays the most recent blog posts.',
        settings: {
            limit: {
                label: 'Number of posts to display',
                min: 1,
                max: 24,
                default: 6,
            },
        },
    });

    window.AdminSectionRegistry = {
        register,
        get,
        getDefinitions,
        getOrderedTypes,
        getDefaultType,
    };
})();

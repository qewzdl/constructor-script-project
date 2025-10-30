(() => {
    const definitions = new Map();

    const normaliseType = (type) =>
        typeof type === 'string' ? type.trim().toLowerCase() : '';

    const register = (type, definition) => {
        const normalisedType = normaliseType(type);
        if (!normalisedType || !definition) {
            return;
        }
        definitions.set(normalisedType, definition);
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
            .sort(([, aDefinition], [, bDefinition]) => {
                const aOrder = aDefinition?.order || 0;
                const bOrder = bDefinition?.order || 0;
                return aOrder - bOrder;
            })
            .map(([type]) => type);

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
            console.error('Failed to parse element definitions', error);
            return null;
        }
    };

    const initialDefinitions = parseDefinitionJSON('element-definitions-data');
    if (initialDefinitions && typeof initialDefinitions === 'object') {
        Object.entries(initialDefinitions).forEach(([type, definition]) => {
            register(type, definition);
        });
    }

    window.AdminElementRegistry = {
        register,
        get,
        getDefinitions,
        getOrderedTypes,
    };
})();

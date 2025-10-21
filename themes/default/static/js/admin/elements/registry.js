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

    window.AdminElementRegistry = {
        register,
        get,
        getDefinitions,
        getOrderedTypes,
    };
})();
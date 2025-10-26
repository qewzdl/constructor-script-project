(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    const sectionRegistry = window.AdminSectionRegistry;
    if (!utils || !registry || !sectionRegistry) {
        return;
    }

    const {
        ensureArray,
        normaliseString,
        randomId,
        createImageState,
    } = utils;

    const parseInteger = (value) => {
        if (typeof value === 'number' && Number.isFinite(value)) {
            return Math.trunc(value);
        }
        if (typeof value === 'string' && value.trim()) {
            const parsed = Number.parseInt(value, 10);
            return Number.isFinite(parsed) ? parsed : NaN;
        }
        return NaN;
    };

    const clampLimit = (value, definition) => {
        if (!definition || typeof definition !== 'object') {
            return value > 0 && Number.isFinite(value)
                ? Math.max(1, Math.round(value))
                : 0;
        }
        const rawMin = parseInteger(definition.min);
        const rawMax = parseInteger(definition.max);
        const rawDefault = parseInteger(definition.default);
        const min = Number.isFinite(rawMin) && rawMin > 0 ? rawMin : 1;
        const max = Number.isFinite(rawMax) && rawMax >= min ? rawMax : Infinity;
        const fallback = Number.isFinite(rawDefault) && rawDefault > 0
            ? Math.min(Math.max(rawDefault, min), max)
            : min;
        let limit = Number.isFinite(value) ? Math.round(value) : 0;
        if (limit <= 0) {
            limit = fallback;
        }
        if (limit < min) {
            limit = min;
        }
        if (Number.isFinite(max) && limit > max) {
            limit = max;
        }
        return limit;
    };

    const createElementState = (definitions, element = {}) => {
        const type =
            normaliseString(element.type ?? element.Type ?? '').toLowerCase() ||
            'paragraph';
        const id = normaliseString(element.id ?? element.ID ?? '');
        const rawContent = element.content ?? element.Content ?? {};
        const definition = definitions[type];

        if (definition && typeof definition.fromRaw === 'function') {
            return definition.fromRaw({ id, rawContent });
        }

        return {
            clientId: randomId(),
            id,
            type,
            content: rawContent || {},
            unsupported: true,
        };
    };

    const createSectionState = (
        elementDefinitions,
        sectionDefinitions,
        defaultType,
        section = {}
    ) => {
        const requestedType = normaliseString(
            section.type ?? section.Type ?? defaultType
        );
        const type = sectionDefinitions[requestedType]
            ? requestedType
            : defaultType;
        const supportsElements =
            sectionDefinitions[type]?.supportsElements !== false;
        const elementsSource = supportsElements
            ? ensureArray(section.elements ?? section.Elements)
            : [];
        const limitDefinition = sectionDefinitions[type]?.settings?.limit;
        const rawLimit = section.limit ?? section.Limit;
        const limitValue = limitDefinition
            ? clampLimit(parseInteger(rawLimit), limitDefinition)
            : 0;
        let styleGridItems = true;
        const styleGridItemsSource =
            section.styleGridItems ??
            section.StyleGridItems ??
            section.style_grid_items ??
            section.Style_grid_items;
        if (styleGridItemsSource !== undefined && styleGridItemsSource !== null) {
            if (typeof styleGridItemsSource === 'string') {
                const normalisedValue = styleGridItemsSource.trim().toLowerCase();
                styleGridItems =
                    normalisedValue === 'true' ||
                    normalisedValue === '1' ||
                    normalisedValue === 'yes';
            } else {
                styleGridItems = Boolean(styleGridItemsSource);
            }
        }
        return {
            clientId: randomId(),
            id: normaliseString(section.id ?? section.ID ?? ''),
            type,
            title: normaliseString(section.title ?? section.Title ?? ''),
            image: normaliseString(section.image ?? section.Image ?? ''),
            elements: supportsElements
                ? elementsSource.map((element) =>
                      createElementState(elementDefinitions, element)
                  )
                : [],
            limit: limitValue,
            styleGridItems,
        };
    };

    const elementHasContent = (definitions, element) => {
        if (!element) {
            return false;
        }
        const definition = definitions[element.type];
        if (definition && typeof definition.hasContent === 'function') {
            return definition.hasContent(element);
        }
        return true;
    };

    const sanitiseElement = (definitions, element, index) => {
        if (!elementHasContent(definitions, element)) {
            return null;
        }
        const definition = definitions[element.type];
        if (definition && typeof definition.sanitise === 'function') {
            return definition.sanitise(element, index);
        }
        return {
            id: element.id || '',
            type: element.type,
            order: index + 1,
            content: element.content,
        };
    };

    const createStateManager = (definitions, sectionDefinitions) => {
        let sections = [];
        const listeners = new Set();
        const sectionDefs = sectionDefinitions || {};
        const defaultSectionType = normaliseString(
            sectionRegistry.getDefaultType?.() ?? 'standard'
        );

        const findSection = (sectionClientId) =>
            sections.find((section) => section.clientId === sectionClientId) ||
            null;

        const supportsElements = (type) =>
            sectionDefs[type]?.supportsElements !== false;

        const findElement = (section, elementClientId) =>
            section?.elements?.find((element) => element.clientId === elementClientId) ||
            null;

        const nilSlice = [];

        const getSections = () =>
            sections
                .map((section, index) => {
                    const elements = section.elements
                        .map((element, elementIndex) =>
                            sanitiseElement(definitions, element, elementIndex)
                        )
                        .filter(Boolean);

                    const image = section.image.trim();
                    const title = section.title.trim();
                    const hasContent = supportsElements(section.type)
                        ? Boolean(title || image || elements.length > 0)
                        : Boolean(title || image);

                    if (!hasContent) {
                        return null;
                    }

                    const payload = {
                        id: section.id || '',
                        type: section.type,
                        title,
                        order: index + 1,
                        elements: supportsElements(section.type) ? elements : nilSlice,
                    };

                    if (image) {
                        payload.image = image;
                    }

                    if (section.type === 'grid') {
                        payload.style_grid_items = section.styleGridItems !== false;
                    }

                    const limitDefinition =
                        sectionDefs[section.type]?.settings?.limit;
                    if (limitDefinition) {
                        const limit = clampLimit(
                            parseInteger(section.limit),
                            limitDefinition
                        );
                        section.limit = limit;
                        if (limit > 0) {
                            payload.limit = limit;
                        }
                    }

                    return payload;
                })
                .filter(Boolean);

        const notify = () => {
            const snapshot = getSections();
            listeners.forEach((listener) => {
                try {
                    listener(snapshot);
                } catch (error) {
                    console.error('Section builder listener failed', error);
                }
            });
        };

        const setSections = (nextSections) => {
            sections = ensureArray(nextSections).map((section) =>
                createSectionState(
                    definitions,
                    sectionDefs,
                    defaultSectionType,
                    section
                )
            );
            return sections;
        };

        const reset = () => {
            sections = [];
            return sections;
        };

        const addSection = (type) => {
            const section = createSectionState(
                definitions,
                sectionDefs,
                defaultSectionType,
                { type }
            );
            section.elements = [];
            sections.push(section);
            return section;
        };

        const removeSection = (sectionClientId) => {
            sections = sections.filter(
                (section) => section.clientId !== sectionClientId
            );
            return sections;
        };

        const addElementToSection = (sectionClientId, type) => {
            const section = findSection(sectionClientId);
            if (!section) {
                return null;
            }
            if (!supportsElements(section.type)) {
                return null;
            }
            const definition = definitions[type];
            const element =
                definition && typeof definition.create === 'function'
                    ? definition.create()
                    : {
                          clientId: randomId(),
                          id: '',
                          type,
                          content: {},
                      };
            if (!section.elements) {
                section.elements = [];
            }
            section.elements.push(element);
            return element;
        };

        const removeElementFromSection = (sectionClientId, elementClientId) => {
            const section = findSection(sectionClientId);
            if (!section || !Array.isArray(section.elements)) {
                return;
            }
            section.elements = section.elements.filter(
                (element) => element.clientId !== elementClientId
            );
        };

        const addGroupImage = (sectionClientId, elementClientId) => {
            const section = findSection(sectionClientId);
            const element = findElement(section, elementClientId);
            if (!element || element.type !== 'image_group') {
                return null;
            }
            if (!Array.isArray(element.content.images)) {
                element.content.images = [];
            }
            const image = createImageState({});
            element.content.images.push(image);
            return image;
        };

        const removeGroupImage = (sectionClientId, elementClientId, imageClientId) => {
            const section = findSection(sectionClientId);
            const element = findElement(section, elementClientId);
            if (!element || element.type !== 'image_group') {
                return;
            }
            element.content.images = ensureArray(element.content.images).filter(
                (image) => image.clientId !== imageClientId
            );
        };

        const updateSectionField = (sectionClientId, field, value) => {
            const section = findSection(sectionClientId);
            if (!section) {
                return;
            }
            if (field === 'section-title') {
                section.title = value;
            } else if (field === 'section-image') {
                section.image = value;
            } else if (field === 'section-grid-style') {
                section.styleGridItems = Boolean(value);
            } else if (field === 'section-limit') {
                const limitDefinition = sectionDefs[section.type]?.settings?.limit;
                if (limitDefinition) {
                    section.limit = clampLimit(
                        parseInteger(value),
                        limitDefinition
                    );
                }
            } else if (field === 'section-type') {
                const nextType = normaliseString(value);
                const ensuredType = sectionDefs[nextType]
                    ? nextType
                    : defaultSectionType;
                if (ensuredType === section.type) {
                    return;
                }
                section.type = ensuredType;
                if (section.type === 'grid' && section.styleGridItems === undefined) {
                    section.styleGridItems = true;
                }
                if (!supportsElements(section.type)) {
                    section.elements = [];
                }
                const limitDefinition = sectionDefs[section.type]?.settings?.limit;
                if (limitDefinition) {
                    section.limit = clampLimit(
                        parseInteger(section.limit),
                        limitDefinition
                    );
                } else {
                    section.limit = 0;
                }
            }
        };

        const updateElementField = (
            sectionClientId,
            elementClientId,
            field,
            value,
            imageClientId
        ) => {
            const section = findSection(sectionClientId);
            if (!section) {
                return false;
            }
            const element = findElement(section, elementClientId);
            if (!element) {
                return false;
            }
            const definition = definitions[element.type];
            if (definition && typeof definition.updateField === 'function') {
                return Boolean(
                    definition.updateField(
                        element,
                        field,
                        value,
                        imageClientId
                    )
                );
            }
            return false;
        };

        const subscribe = (listener) => {
            if (typeof listener === 'function') {
                listeners.add(listener);
            }
        };

        const unsubscribe = (listener) => {
            listeners.delete(listener);
        };

        return {
            getSections,
            getState: () => sections,
            notify,
            setSections,
            reset,
            addSection,
            removeSection,
            addElementToSection,
            removeElementFromSection,
            addGroupImage,
            removeGroupImage,
            updateSectionField,
            updateElementField,
            subscribe,
            unsubscribe,
        };
    };

    window.AdminSectionState = {
        createManager: (definitions, sectionDefinitions) =>
            createStateManager(
                definitions || registry.getDefinitions(),
                sectionDefinitions || sectionRegistry.getDefinitions()
            ),
    };
})();
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
            } else if (field === 'section-type') {
                const nextType = normaliseString(value);
                const ensuredType = sectionDefs[nextType]
                    ? nextType
                    : defaultSectionType;
                if (ensuredType === section.type) {
                    return;
                }
                section.type = ensuredType;
                if (!supportsElements(section.type)) {
                    section.elements = [];
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
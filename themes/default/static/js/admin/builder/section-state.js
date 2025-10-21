(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    if (!utils || !registry) {
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

    const createSectionState = (definitions, section = {}) => {
        const elementsSource = ensureArray(section.elements ?? section.Elements);
        return {
            clientId: randomId(),
            id: normaliseString(section.id ?? section.ID ?? ''),
            title: normaliseString(section.title ?? section.Title ?? ''),
            image: normaliseString(section.image ?? section.Image ?? ''),
            elements: elementsSource.map((element) =>
                createElementState(definitions, element)
            ),
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

    const createStateManager = (definitions) => {
        let sections = [];
        const listeners = new Set();

        const findSection = (sectionClientId) =>
            sections.find((section) => section.clientId === sectionClientId) ||
            null;

        const findElement = (section, elementClientId) =>
            section?.elements?.find((element) => element.clientId === elementClientId) ||
            null;

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
                    const hasContent = Boolean(title || image || elements.length > 0);

                    if (!hasContent) {
                        return null;
                    }

                    const payload = {
                        id: section.id || '',
                        title,
                        order: index + 1,
                        elements,
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
                createSectionState(definitions, section)
            );
            return sections;
        };

        const reset = () => {
            sections = [];
            return sections;
        };

        const addSection = () => {
            const section = createSectionState(definitions, {});
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
        createManager: (definitions) =>
            createStateManager(definitions || registry.getDefinitions()),
    };
})();
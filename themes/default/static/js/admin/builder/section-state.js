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
        createFileState,
    } = utils;

    const createAllowedElementsResolver = (sectionDefinitions) => {
        const cache = new Map();
        return (sectionType) => {
            const typeKey = normaliseString(sectionType);
            if (cache.has(typeKey)) {
                return cache.get(typeKey);
            }
            const allowedList = sectionDefinitions?.[typeKey]?.allowedElements;
            if (!Array.isArray(allowedList) || !allowedList.length) {
                cache.set(typeKey, null);
                return null;
            }
            const set = new Set(
                allowedList
                    .map((item) => normaliseString(item).toLowerCase())
                    .filter(Boolean)
            );
            if (set.size === 0) {
                cache.set(typeKey, null);
                return null;
            }
            cache.set(typeKey, set);
            return set;
        };
    };

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

    const parseBoolean = (value, fallback = false) => {
        if (value === true || value === false) {
            return Boolean(value);
        }
        if (typeof value === 'string') {
            const normalised = value.trim().toLowerCase();
            if (['true', '1', 'yes', 'on'].includes(normalised)) {
                return true;
            }
            if (['false', '0', 'no', 'off'].includes(normalised)) {
                return false;
            }
        }
        return fallback;
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

    // Read builder configuration from server
    const getBuilderConfig = () => {
        const configElement = document.getElementById('builder-config-data');
        if (configElement && configElement.textContent) {
            try {
                return JSON.parse(configElement.textContent);
            } catch (error) {
                console.error('Failed to parse builder config:', error);
            }
        }
        return {
            paddingOptions: [0, 4, 8, 16, 32, 64, 128],
            marginOptions: [0, 4, 8, 16, 32, 64, 128],
            defaultSectionPadding: 16,
            defaultSectionMargin: 0,
        };
    };

    const builderConfig = getBuilderConfig();
    const paddingOptions = builderConfig.paddingOptions || [0, 4, 8, 16, 32, 64, 128];
    const marginOptions = builderConfig.marginOptions || [0, 4, 8, 16, 32, 64, 128];
    const defaultPadding = paddingOptions[0] || 0;
    const defaultMargin = marginOptions[0] || 0;
    const newSectionDefaultPadding = builderConfig.defaultSectionPadding || 16;
    const newSectionDefaultMargin = builderConfig.defaultSectionMargin || 0;
    const animationOptions =
        (Array.isArray(builderConfig.sectionAnimations) && builderConfig.sectionAnimations.length
            ? builderConfig.sectionAnimations
            : [
                  { value: 'float-up', label: 'Float up' },
                  { value: 'fade-in', label: 'Fade in' },
                  { value: 'slide-left', label: 'Slide from right' },
                  { value: 'zoom-in', label: 'Zoom in' },
                  { value: 'none', label: 'None' },
              ]);
    const animationValues = new Set(
        animationOptions
            .map((option) => normaliseString(option?.value).toLowerCase())
            .filter(Boolean)
    );
    const defaultAnimation = (() => {
        const candidate = normaliseString(
            builderConfig.defaultSectionAnimation ?? builderConfig.default_animation ?? ''
        ).toLowerCase();
        if (candidate && animationValues.has(candidate)) {
            return candidate;
        }
        const fallback = normaliseString(animationOptions[0]?.value).toLowerCase();
        return fallback || 'float-up';
    })();
    const defaultAnimationBlur = parseBoolean(
        builderConfig.defaultAnimationBlur ?? builderConfig.default_animation_blur,
        true
    );
    const normaliseAnimationValue = (value) => {
        const normalised = normaliseString(value).toLowerCase();
        if (normalised && animationValues.has(normalised)) {
            return normalised;
        }
        if (animationValues.has(defaultAnimation)) {
            return defaultAnimation;
        }
        return animationValues.values().next().value || 'none';
    };
    const normaliseAnimationBlurValue = (value) =>
        parseBoolean(value, defaultAnimationBlur);

    const clampPaddingValue = (value) => {
        if (!paddingOptions.length) {
            return 0;
        }
        if (!Number.isFinite(value)) {
            return defaultPadding;
        }
        if (value <= paddingOptions[0]) {
            return paddingOptions[0];
        }
        const last = paddingOptions[paddingOptions.length - 1];
        if (value >= last) {
            return last;
        }
        let closest = paddingOptions[0];
        let minDiff = Math.abs(value - closest);
        for (let i = 1; i < paddingOptions.length; i += 1) {
            const option = paddingOptions[i];
            const diff = Math.abs(value - option);
            if (diff < minDiff) {
                closest = option;
                minDiff = diff;
            }
        }
        return closest;
    };

    const normalisePaddingValue = (value) => {
        if (typeof value === 'number' && Number.isFinite(value)) {
            return clampPaddingValue(value);
        }
        if (typeof value === 'string' && value.trim()) {
            const parsed = Number.parseInt(value, 10);
            if (Number.isFinite(parsed)) {
                return clampPaddingValue(parsed);
            }
        }
        return newSectionDefaultPadding;
    };

    const clampMarginValue = (value) => {
        if (!marginOptions.length) {
            return 0;
        }
        if (!Number.isFinite(value)) {
            return defaultMargin;
        }
        if (value <= marginOptions[0]) {
            return marginOptions[0];
        }
        const last = marginOptions[marginOptions.length - 1];
        if (value >= last) {
            return last;
        }
        let closest = marginOptions[0];
        let minDiff = Math.abs(value - closest);
        for (let i = 1; i < marginOptions.length; i += 1) {
            const option = marginOptions[i];
            const diff = Math.abs(value - option);
            if (diff < minDiff) {
                closest = option;
                minDiff = diff;
            }
        }
        return closest;
    };

    const normaliseMarginValue = (value) => {
        if (typeof value === 'number' && Number.isFinite(value)) {
            return clampMarginValue(value);
        }
        if (typeof value === 'string' && value.trim()) {
            const parsed = Number.parseInt(value, 10);
            if (Number.isFinite(parsed)) {
                return clampMarginValue(parsed);
            }
        }
        return newSectionDefaultMargin;
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

    const normaliseModeValue = (definition, value) => {
        const normalisedValue = normaliseString(value).toLowerCase();
        const options = Array.isArray(definition?.options)
            ? definition.options
                  .map((option) => normaliseString(option?.value).toLowerCase())
                  .filter(Boolean)
            : [];
        if (!options.length) {
            return normalisedValue;
        }
        if (normalisedValue && options.includes(normalisedValue)) {
            return normalisedValue;
        }
        const defaultValue = normaliseString(
            definition?.defaultValue ?? definition?.default_value ?? ''
        ).toLowerCase();
        if (defaultValue && options.includes(defaultValue)) {
            return defaultValue;
        }
        return options[0];
    };

    const createSectionState = (
        elementDefinitions,
        sectionDefinitions,
        defaultType,
        section = {},
        resolveAllowedElements
    ) => {
        const requestedType = normaliseString(
            section.type ?? section.Type ?? defaultType
        );
        const type = sectionDefinitions[requestedType]
            ? requestedType
            : defaultType;
        const supportsElements =
            sectionDefinitions[type]?.supportsElements !== false;
        const allowedElements =
            typeof resolveAllowedElements === 'function'
                ? resolveAllowedElements(type)
                : null;
        const elementsSource = supportsElements
            ? ensureArray(section.elements ?? section.Elements)
            : [];
        const limitDefinition = sectionDefinitions[type]?.settings?.limit;
        const rawLimit = section.limit ?? section.Limit;
        const limitValue = limitDefinition
            ? clampLimit(parseInteger(rawLimit), limitDefinition)
            : 0;
        const modeDefinition = sectionDefinitions[type]?.settings?.mode;
        const rawMode = section.mode ?? section.Mode ?? '';
        const modeValue = normaliseModeValue(modeDefinition, rawMode);
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
        const paddingSource =
            section.paddingVertical ??
            section.PaddingVertical ??
            section.padding_vertical ??
            section.Padding_vertical;
        const paddingVertical = normalisePaddingValue(paddingSource);
        const marginSource =
            section.marginVertical ??
            section.MarginVertical ??
            section.margin_vertical ??
            section.Margin_vertical;
        const marginVertical = normaliseMarginValue(marginSource);
        const animation = normaliseAnimationValue(
            section.animation ??
                section.Animation ??
                section.animation_type ??
                section.Animation_type
        );
        const animationBlur = normaliseAnimationBlurValue(
            section.animationBlur ??
                section.AnimationBlur ??
                section.animation_blur ??
                section.Animation_blur
        );
        const headerImageSupported =
            sectionDefinitions[type]?.supportsHeaderImage === true;
        const disabled = parseBoolean(section.disabled ?? section.Disabled, false);
        
        // Handle custom section settings (like hero fields)
        const settingsSource = section.settings ?? section.Settings ?? {};
        const settings = typeof settingsSource === 'object' && settingsSource !== null 
            ? { ...settingsSource } 
            : {};
        
        const elements = supportsElements
            ? elementsSource
                  .map((element) => createElementState(elementDefinitions, element))
                  .filter((element) => {
                      if (!allowedElements || allowedElements.size === 0) {
                          return true;
                      }
                      return allowedElements.has(element.type);
                  })
            : [];

        return {
            clientId: randomId(),
            id: normaliseString(section.id ?? section.ID ?? ''),
            type,
            title: normaliseString(section.title ?? section.Title ?? ''),
            description: normaliseString(section.description ?? section.Description ?? ''),
            image: headerImageSupported
                ? normaliseString(section.image ?? section.Image ?? '')
                : '',
            elements,
            limit: limitValue,
            mode: modeValue,
            disabled,
            styleGridItems,
            paddingVertical,
            marginVertical,
            animation,
            animationBlur,
            settings,
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
        const resolveAllowedElements = createAllowedElementsResolver(sectionDefs);
        const isElementAllowed = (sectionType, elementType) => {
            const allowedSet = resolveAllowedElements(sectionType);
            if (!allowedSet || allowedSet.size === 0) {
                return true;
            }
            const normalised = normaliseString(elementType).toLowerCase();
            return normalised ? allowedSet.has(normalised) : false;
        };

        const findSection = (sectionClientId) =>
            sections.find((section) => section.clientId === sectionClientId) ||
            null;

        const supportsElements = (type) =>
            sectionDefs[type]?.supportsElements !== false;
        const supportsHeaderImage = (type) =>
            sectionDefs[type]?.supportsHeaderImage === true;

        const findElement = (section, elementClientId) =>
            section?.elements?.find((element) => element.clientId === elementClientId) ||
            null;

        const nilSlice = [];

        const getSections = () =>
            sections
                .map((section, index) => {
                    const headerImageSupported = supportsHeaderImage(section.type);
                    const image = headerImageSupported
                        ? section.image.trim()
                        : '';
                    const title = section.title.trim();
                    const description = section.description?.trim() || '';
                    const hasSettings = section.settings && Object.keys(section.settings).length > 0;

                    let elements = nilSlice;
                    if (supportsElements(section.type)) {
                        const sanitisedElements = [];
                        const allowedSet = resolveAllowedElements(section.type);
                        section.elements
                            .filter((element) => {
                                if (!allowedSet || allowedSet.size === 0) {
                                    return true;
                                }
                                return allowedSet.has(normaliseString(element.type).toLowerCase());
                            })
                            .forEach((element) => {
                                const sanitised = sanitiseElement(
                                    definitions,
                                    element,
                                    sanitisedElements.length
                                );
                                if (sanitised) {
                                    sanitisedElements.push(sanitised);
                                }
                            });
                        elements = sanitisedElements;
                    }

                    const hasContent = supportsElements(section.type)
                        ? Boolean(
                              title || description || image || elements.length > 0 || hasSettings
                          )
                        : Boolean(title || description || image || hasSettings);

                    if (!hasContent) {
                    return null;
                }

                    const payload = {
                        id: section.id || '',
                        type: section.type,
                        title,
                        description,
                        order: index + 1,
                        elements: supportsElements(section.type) ? elements : nilSlice,
                    };

                    if (parseBoolean(section.disabled, false)) {
                        payload.disabled = true;
                    }

                    if (headerImageSupported && image) {
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

                    const modeDefinition =
                        sectionDefs[section.type]?.settings?.mode;
                    if (modeDefinition) {
                        const modeValue = normaliseModeValue(
                            modeDefinition,
                            section.mode
                        );
                        section.mode = modeValue;
                        if (modeValue) {
                            payload.mode = modeValue;
                        }
                    } else if (section.mode) {
                        payload.mode = normaliseString(section.mode).toLowerCase();
                    }

                    payload.padding_vertical = clampPaddingValue(
                        Number(section.paddingVertical)
                    );
                    payload.margin_vertical = clampMarginValue(
                        Number(section.marginVertical)
                    );
                    payload.animation = normaliseAnimationValue(section.animation);
                    payload.animation_blur = normaliseAnimationBlurValue(
                        section.animationBlur
                    );

                    // Include custom section settings (like hero fields)
                    if (section.settings && Object.keys(section.settings).length > 0) {
                        payload.settings = section.settings;
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
                    section,
                    resolveAllowedElements
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
                { type },
                resolveAllowedElements
            );
            section.elements = [];
            section.paddingVertical = clampPaddingValue(newSectionDefaultPadding);
            section.marginVertical = clampMarginValue(newSectionDefaultMargin);
            section.animation = normaliseAnimationValue(defaultAnimation);
            section.animationBlur = normaliseAnimationBlurValue(defaultAnimationBlur);
            sections.push(section);
            return section;
        };

        const removeSection = (sectionClientId) => {
            sections = sections.filter(
                (section) => section.clientId !== sectionClientId
            );
            return sections;
        };

        const moveSection = (sectionClientId, direction) => {
            const currentIndex = sections.findIndex(
                (section) => section.clientId === sectionClientId
            );
            if (currentIndex < 0) {
                return -1;
            }

            let targetIndex = currentIndex;
            if (direction === 'up') {
                targetIndex -= 1;
            } else if (direction === 'down') {
                targetIndex += 1;
            } else if (typeof direction === 'number' && Number.isFinite(direction)) {
                targetIndex = Math.trunc(direction);
            } else if (typeof direction === 'string' && direction.trim()) {
                const parsed = Number.parseInt(direction, 10);
                if (Number.isFinite(parsed)) {
                    targetIndex = parsed;
                }
            }

            if (targetIndex < 0) {
                targetIndex = 0;
            }
            if (targetIndex >= sections.length) {
                targetIndex = sections.length - 1;
            }

            if (targetIndex === currentIndex || targetIndex < 0) {
                return -1;
            }

            const [section] = sections.splice(currentIndex, 1);
            sections.splice(targetIndex, 0, section);
            return targetIndex;
        };

        const addElementToSection = (sectionClientId, type) => {
            const section = findSection(sectionClientId);
            if (!section) {
                return null;
            }
            if (!supportsElements(section.type)) {
                return null;
            }
            if (!isElementAllowed(section.type, type)) {
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

        const moveElementInSection = (sectionClientId, elementClientId, direction) => {
            const section = findSection(sectionClientId);
            if (!section || !Array.isArray(section.elements)) {
                return -1;
            }
            const currentIndex = section.elements.findIndex(
                (element) => element.clientId === elementClientId
            );
            if (currentIndex < 0) {
                return -1;
            }

            let targetIndex = currentIndex;
            if (direction === 'up') {
                targetIndex -= 1;
            } else if (direction === 'down') {
                targetIndex += 1;
            } else if (typeof direction === 'number' && Number.isFinite(direction)) {
                targetIndex = Math.trunc(direction);
            } else if (typeof direction === 'string' && direction.trim()) {
                const parsed = Number.parseInt(direction, 10);
                if (Number.isFinite(parsed)) {
                    targetIndex = parsed;
                }
            }

            if (targetIndex < 0) {
                targetIndex = 0;
            }
            if (targetIndex >= section.elements.length) {
                targetIndex = section.elements.length - 1;
            }

            if (targetIndex === currentIndex || targetIndex < 0) {
                return -1;
            }

            const [element] = section.elements.splice(currentIndex, 1);
            section.elements.splice(targetIndex, 0, element);
            return targetIndex;
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

        const addGroupFile = (sectionClientId, elementClientId) => {
            const section = findSection(sectionClientId);
            const element = findElement(section, elementClientId);
            if (!element || element.type !== 'file_group') {
                return null;
            }
            if (!Array.isArray(element.content.files)) {
                element.content.files = [];
            }
            const file = createFileState({});
            element.content.files.push(file);
            return file;
        };

        const removeGroupFile = (
            sectionClientId,
            elementClientId,
            fileClientId
        ) => {
            const section = findSection(sectionClientId);
            const element = findElement(section, elementClientId);
            if (!element || element.type !== 'file_group') {
                return;
            }
            element.content.files = ensureArray(element.content.files).filter(
                (file) => file.clientId !== fileClientId
            );
        };

        const updateSectionField = (sectionClientId, field, value) => {
            const section = findSection(sectionClientId);
            if (!section) {
                return;
            }
            if (field === 'section-title') {
                section.title = value;
            } else if (field === 'section-description') {
                section.description = value;
            } else if (field === 'section-disabled') {
                section.disabled = parseBoolean(value, false);
            } else if (field === 'section-image') {
                if (supportsHeaderImage(section.type)) {
                    section.image = value;
                } else {
                    section.image = '';
                }
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
            } else if (field.startsWith('section-setting-')) {
                // Handle custom section settings (like hero fields)
                const settingKey = field.replace('section-setting-', '');
                if (!section.settings) {
                    section.settings = {};
                }
                section.settings[settingKey] = value;
            } else if (field === 'section-padding-vertical') {
                section.paddingVertical = clampPaddingValue(Number(value));
            } else if (field === 'section-margin-vertical') {
                section.marginVertical = clampMarginValue(Number(value));
            } else if (field === 'section-animation') {
                section.animation = normaliseAnimationValue(value);
            } else if (field === 'section-animation-blur') {
                section.animationBlur = normaliseAnimationBlurValue(value);
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
                if (!supportsHeaderImage(section.type)) {
                    section.image = '';
                }
                if (!supportsElements(section.type)) {
                    section.elements = [];
                } else if (Array.isArray(section.elements)) {
                    section.elements = section.elements.filter((element) =>
                        isElementAllowed(section.type, element.type)
                    );
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
            nestedClientId
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
                        nestedClientId
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
            moveSection,
            addElementToSection,
            removeElementFromSection,
            moveElementInSection,
            addGroupImage,
            removeGroupImage,
            addGroupFile,
            removeGroupFile,
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

(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    const sectionRegistry = window.AdminSectionRegistry;
    if (!utils || !registry || !sectionRegistry) {
        return;
    }

    const { createElement, normaliseString, randomId } = utils;

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

    const describeModeValue = (definition, value) => {
        const normalised = normaliseModeValue(definition, value);
        if (!normalised) {
            return '';
        }
        const option = Array.isArray(definition?.options)
            ? definition.options.find(
                  (item) =>
                      normaliseString(item?.value).toLowerCase() === normalised
              )
            : null;
        const label = option && normaliseString(option.label);
        return label || normalised;
    };

    const normaliseVariationValue = (definition, value) => {
        const variations = Array.isArray(definition?.variations)
            ? definition.variations
                  .map((variation) => ({
                      value: normaliseString(variation?.value).toLowerCase(),
                      isDefault:
                          variation?.isDefault === true ||
                          variation?.is_default === true ||
                          variation?.default === true,
                  }))
                  .filter((variation) => Boolean(variation.value))
            : [];
        if (!variations.length) {
            return '';
        }

        let normalisedValue = normaliseString(value).toLowerCase();
        const definitionType = normaliseString(definition?.type).toLowerCase();
        if (definitionType === 'features' && normalisedValue === 'icon-text') {
            normalisedValue = 'glyph';
        }
        if (normalisedValue) {
            const matched = variations.find(
                (variation) => variation.value === normalisedValue
            );
            if (matched) {
                return matched.value;
            }
        }

        const fallback =
            variations.find((variation) => variation.isDefault)?.value ||
            variations[0].value;
        return fallback || '';
    };

    const describeVariationValue = (definition, value) => {
        const normalised = normaliseVariationValue(definition, value);
        if (!normalised) {
            return '';
        }
        const option = Array.isArray(definition?.variations)
            ? definition.variations.find(
                  (variation) =>
                      normaliseString(variation?.value).toLowerCase() ===
                      normalised
              )
            : null;
        const label = option && normaliseString(option.label);
        return label || normalised;
    };

    const normaliseVariationToken = (value) =>
        normaliseString(value)
            .trim()
            .toLowerCase()
            .replace(/[^a-z0-9_-]+/g, '-')
            .replace(/-{2,}/g, '-')
            .replace(/^-+|-+$/g, '');

    const normaliseBackgroundGroupToken = (value) =>
        normaliseString(value)
            .trim()
            .toLowerCase()
            .replace(/[^a-z0-9_-]+/g, '-')
            .replace(/-{2,}/g, '-')
            .replace(/^-+|-+$/g, '');

    const backgroundStyleOptions = [
        { value: '', label: 'Theme default' },
        { value: 'surface', label: 'Surface' },
        { value: 'muted', label: 'Muted' },
        { value: 'primary-soft', label: 'Primary soft' },
        { value: 'accent-soft', label: 'Accent soft' },
        { value: 'primary-gradient', label: 'Primary gradient' },
        { value: 'accent-gradient', label: 'Accent gradient' },
        { value: 'immersive', label: 'Immersive' },
    ];

    const normaliseBackgroundStyleToken = (value) =>
        normaliseString(value)
            .trim()
            .toLowerCase()
            .replace(/[^a-z0-9_-]+/g, '-')
            .replace(/-{2,}/g, '-')
            .replace(/^-+|-+$/g, '');

    const describeBackgroundStyleValue = (value) => {
        const normalised = normaliseBackgroundStyleToken(value);
        const matched = backgroundStyleOptions.find(
            (option) => option.value === normalised
        );
        if (matched?.label) {
            return matched.label;
        }
        return normalised || 'Theme default';
    };

    const collectVariationOptions = (definition) => {
        if (!Array.isArray(definition?.variations)) {
            return [];
        }

        const seen = new Set();
        const options = [];
        const sectionType = normaliseVariationToken(definition?.type) || 'section';
        definition.variations.forEach((variation) => {
            const value = normaliseString(
                variation?.value ?? variation?.id ?? variation?.type
            ).toLowerCase();
            if (!value || seen.has(value)) {
                return;
            }
            seen.add(value);
            const rawId = normaliseVariationToken(variation?.id);
            options.push({
                value,
                id: rawId || `${sectionType}-${value}`,
                label:
                    normaliseString(variation?.label).trim() || value,
                description: normaliseString(variation?.description).trim(),
                isDefault:
                    variation?.isDefault === true ||
                    variation?.is_default === true ||
                    variation?.default === true,
            });
        });
        return options;
    };

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

    const openSectionTypePicker = ({
        orderedSectionTypes,
        sectionDefinitions,
        activeType,
        onSelect,
        title = 'Choose section type',
        onClose,
    } = {}) => {
        const typeOrder = Array.isArray(orderedSectionTypes)
            ? orderedSectionTypes.filter((type) => typeof type === 'string' && type.trim().length)
            : Object.keys(sectionDefinitions || {}).filter(
                  (type) => typeof type === 'string' && type.trim().length
              );

        if (!typeOrder.length) {
            return () => {};
        }

        const previousFocus = document.activeElement;
        const overlay = createElement('div', {
            className: 'admin-type-picker',
        });
        const dialog = createElement('div', {
            className: 'admin-type-picker__dialog',
            attributes: {
                role: 'dialog',
                'aria-modal': 'true',
            },
        });
        const titleId = `admin-type-picker-title-${randomId()}`;
        dialog.setAttribute('aria-labelledby', titleId);

        const header = createElement('header', {
            className: 'admin-type-picker__header',
        });
        const titleNode = createElement('h2', {
            className: 'admin-type-picker__title',
            textContent: title,
        });
        titleNode.id = titleId;
        header.append(titleNode);

        const searchWrapper = createElement('div', {
            className: 'admin-type-picker__search',
        });
        const searchLabelId = `admin-type-picker-search-${randomId()}`;
        const searchLabel = createElement('label', {
            className: 'admin-type-picker__search-label',
            textContent: 'Search section types',
        });
        searchLabel.htmlFor = searchLabelId;
        const searchInput = createElement('input', {
            className: 'admin-type-picker__search-input',
            attributes: {
                id: searchLabelId,
                type: 'search',
                placeholder: 'Search by name or identifier',
                autocomplete: 'off',
            },
        });
        searchWrapper.append(searchLabel, searchInput);

        const body = createElement('div', {
            className: 'admin-type-picker__body',
        });
        const optionsList = createElement('div', {
            className: 'admin-type-picker__options',
        });
        const emptyState = createElement('p', {
            className: 'admin-type-picker__empty',
            textContent: 'No section types match your search.',
        });
        emptyState.hidden = true;
        body.append(optionsList, emptyState);

        const footer = createElement('div', {
            className: 'admin-type-picker__footer',
        });
        const cancelButton = createElement('button', {
            className: 'admin-builder__button admin-builder__button--ghost',
            type: 'button',
            textContent: 'Cancel',
        });
        footer.append(cancelButton);

        dialog.append(header, searchWrapper, body, footer);
        overlay.append(dialog);

        const optionNodes = [];
        const normaliseValue = (value) => normaliseString(value).trim().toLowerCase();
        const updateActiveState = (nextActive) => {
            optionNodes.forEach((node) => {
                const isActive = node.dataset.type === nextActive;
                node.classList.toggle('is-active', isActive);
                if (isActive) {
                    node.setAttribute('aria-current', 'true');
                } else {
                    node.removeAttribute('aria-current');
                }
            });
        };

        const closeModal = () => {
            if (!overlay.isConnected) {
                return;
            }
            document.removeEventListener('keydown', handleKeyDown);
            overlay.remove();
            if (typeof onClose === 'function') {
                onClose();
            }
            if (previousFocus && typeof previousFocus.focus === 'function') {
                previousFocus.focus();
            }
        };

        const selectType = (type) => {
            if (typeof onSelect === 'function') {
                onSelect(type);
            }
            closeModal();
        };

        typeOrder.forEach((type) => {
            const definition = sectionDefinitions?.[type] || {};
            const optionButton = createElement('button', {
                className: 'admin-type-picker__option',
                type: 'button',
                dataset: {
                    type,
                },
            });
            const optionHeader = createElement('div', {
                className: 'admin-type-picker__option-header',
            });
            const optionLabel = createElement('span', {
                className: 'admin-type-picker__option-label',
                textContent: definition.label || type,
            });
            const optionCode = createElement('code', {
                className: 'admin-type-picker__option-code',
                textContent: type,
            });
            optionHeader.append(optionLabel, optionCode);
            optionButton.append(optionHeader);
            const description = normaliseString(definition.description);
            if (description) {
                optionButton.append(
                    createElement('span', {
                        className: 'admin-type-picker__option-description',
                        textContent: description,
                    })
                );
            }
            optionButton.dataset.keywords = [type, optionLabel.textContent || '', description]
                .map(normaliseValue)
                .join(' ');
            if (type === activeType) {
                optionButton.classList.add('is-active');
                optionButton.setAttribute('aria-current', 'true');
            }
            optionButton.addEventListener('click', (event) => {
                event.preventDefault();
                selectType(type);
            });
            optionNodes.push(optionButton);
            optionsList.append(optionButton);
        });

        const filterOptions = (term) => {
            const query = normaliseValue(term);
            let visibleCount = 0;
            optionNodes.forEach((node) => {
                const keywords = node.dataset.keywords || '';
                const matches = !query || keywords.includes(query);
                node.hidden = !matches;
                if (matches) {
                    visibleCount += 1;
                }
            });
            emptyState.hidden = visibleCount > 0;
        };

        const handleKeyDown = (event) => {
            if (event.key === 'Escape') {
                event.preventDefault();
                closeModal();
                return;
            }
            if (event.key === 'Enter' && event.target === searchInput) {
                const firstVisible = optionNodes.find((node) => !node.hidden);
                if (firstVisible) {
                    event.preventDefault();
                    firstVisible.click();
                }
            }
        };

        searchInput.addEventListener('input', (event) => {
            filterOptions(event.target.value || '');
        });

        cancelButton.addEventListener('click', (event) => {
            event.preventDefault();
            closeModal();
        });
        overlay.addEventListener('click', (event) => {
            if (event.target === overlay) {
                closeModal();
            }
        });

        document.addEventListener('keydown', handleKeyDown);

        document.body.append(overlay);
        filterOptions('');
        updateActiveState(activeType);

        if (typeof window.requestAnimationFrame === 'function') {
            window.requestAnimationFrame(() => {
                searchInput.focus();
            });
        } else {
            searchInput.focus();
        }

        return closeModal;
    };

    const openSectionVariationPicker = ({
        sectionType,
        sectionDefinitions,
        activeVariation,
        onSelect,
        onCancel,
        onClose,
        title,
    } = {}) => {
        const type = normaliseString(sectionType).trim().toLowerCase();
        const definition = sectionDefinitions?.[type] || {};
        const options = collectVariationOptions(definition);
        if (options.length <= 1) {
            if (typeof onSelect === 'function') {
                onSelect('');
            }
            if (typeof onClose === 'function') {
                onClose();
            }
            return () => {};
        }

        const currentVariation = normaliseVariationValue(
            definition,
            activeVariation
        );
        const previousFocus = document.activeElement;
        const overlay = createElement('div', {
            className: 'admin-type-picker',
        });
        const dialog = createElement('div', {
            className: 'admin-type-picker__dialog',
            attributes: {
                role: 'dialog',
                'aria-modal': 'true',
            },
        });
        const titleId = `admin-variation-picker-title-${randomId()}`;
        dialog.setAttribute('aria-labelledby', titleId);

        const header = createElement('header', {
            className: 'admin-type-picker__header',
        });
        const heading =
            normaliseString(definition.label).trim() || type || 'section';
        const titleNode = createElement('h2', {
            className: 'admin-type-picker__title',
            textContent:
                normaliseString(title).trim() ||
                `Choose variation for ${heading}`,
        });
        titleNode.id = titleId;
        header.append(titleNode);

        const body = createElement('div', {
            className: 'admin-type-picker__body',
        });
        const optionsList = createElement('div', {
            className: 'admin-type-picker__options',
        });
        body.append(optionsList);

        const footer = createElement('div', {
            className: 'admin-type-picker__footer',
        });
        const cancelButton = createElement('button', {
            className: 'admin-builder__button admin-builder__button--ghost',
            type: 'button',
            textContent: 'Cancel',
        });
        footer.append(cancelButton);

        dialog.append(header, body, footer);
        overlay.append(dialog);

        let selectionCommitted = false;

        const closeModal = (cancelled = false) => {
            if (!overlay.isConnected) {
                return;
            }
            document.removeEventListener('keydown', handleKeyDown);
            overlay.remove();
            if (cancelled && !selectionCommitted && typeof onCancel === 'function') {
                onCancel();
            }
            if (typeof onClose === 'function') {
                onClose();
            }
            if (previousFocus && typeof previousFocus.focus === 'function') {
                previousFocus.focus();
            }
        };

        const selectVariation = (value) => {
            selectionCommitted = true;
            if (typeof onSelect === 'function') {
                onSelect(value);
            }
            closeModal(false);
        };

        options.forEach((option) => {
            const optionButton = createElement('button', {
                className: 'admin-type-picker__option',
                type: 'button',
                dataset: {
                    variation: option.value,
                },
            });
            if (
                option.value === currentVariation ||
                (!currentVariation && option.isDefault)
            ) {
                optionButton.classList.add('is-active');
                optionButton.setAttribute('aria-current', 'true');
            }

            const optionHeader = createElement('div', {
                className: 'admin-type-picker__option-header',
            });
            const optionLabel = createElement('span', {
                className: 'admin-type-picker__option-label',
                textContent: option.label,
            });
            const optionCode = createElement('code', {
                className: 'admin-type-picker__option-code',
                textContent: option.value,
            });
            optionHeader.append(optionLabel, optionCode);
            optionButton.append(optionHeader);

            if (option.description) {
                optionButton.append(
                    createElement('span', {
                        className: 'admin-type-picker__option-description',
                        textContent: option.description,
                    })
                );
            }

            optionButton.addEventListener('click', (event) => {
                event.preventDefault();
                selectVariation(option.value);
            });

            optionsList.append(optionButton);
        });

        const handleKeyDown = (event) => {
            if (event.key === 'Escape') {
                event.preventDefault();
                closeModal(true);
            }
        };

        cancelButton.addEventListener('click', (event) => {
            event.preventDefault();
            closeModal(true);
        });
        overlay.addEventListener('click', (event) => {
            if (event.target === overlay) {
                closeModal(true);
            }
        });
        document.addEventListener('keydown', handleKeyDown);
        document.body.append(overlay);

        const activeNode = optionsList.querySelector('.admin-type-picker__option.is-active');
        const firstNode = optionsList.querySelector('.admin-type-picker__option');
        const focusTarget = activeNode || firstNode;
        if (focusTarget && typeof focusTarget.focus === 'function') {
            if (typeof window.requestAnimationFrame === 'function') {
                window.requestAnimationFrame(() => {
                    focusTarget.focus();
                });
            } else {
                focusTarget.focus();
            }
        }

        return () => closeModal(true);
    };

    window.AdminSectionTypePicker = window.AdminSectionTypePicker || {};
    window.AdminSectionTypePicker.open = openSectionTypePicker;
    window.AdminSectionTypePicker.openVariations = openSectionVariationPicker;

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
            sectionAnimations: [
                { value: 'float-up', label: 'Float up' },
                { value: 'fade-in', label: 'Fade in' },
                { value: 'slide-left', label: 'Slide from right' },
                { value: 'zoom-in', label: 'Zoom in' },
                { value: 'none', label: 'None' },
            ],
            defaultSectionAnimation: 'float-up',
        };
    };

    const builderConfig = getBuilderConfig();
    const paddingOptions = builderConfig.paddingOptions || [0, 4, 8, 16, 32, 64, 128];
    const marginOptions = builderConfig.marginOptions || [0, 4, 8, 16, 32, 64, 128];

    const clampPaddingValue = (value) => {
        if (!paddingOptions.length) {
            return 0;
        }
        const numeric = Number(value);
        if (!Number.isFinite(numeric)) {
            return paddingOptions[0];
        }
        if (numeric <= paddingOptions[0]) {
            return paddingOptions[0];
        }
        const last = paddingOptions[paddingOptions.length - 1];
        if (numeric >= last) {
            return last;
        }
        let closest = paddingOptions[0];
        let minDiff = Math.abs(numeric - closest);
        for (let i = 1; i < paddingOptions.length; i += 1) {
            const option = paddingOptions[i];
            const diff = Math.abs(numeric - option);
            if (diff < minDiff) {
                closest = option;
                minDiff = diff;
            }
        }
        return closest;
    };

    const paddingIndexForValue = (value) => {
        const clamped = clampPaddingValue(value);
        const index = paddingOptions.indexOf(clamped);
        return index >= 0 ? index : 0;
    };

    const paddingValueForIndex = (index) => {
        const numeric = Number.parseInt(index, 10);
        if (Number.isNaN(numeric)) {
            return paddingOptions[0];
        }
        if (numeric <= 0) {
            return paddingOptions[0];
        }
        if (numeric >= paddingOptions.length) {
            return paddingOptions[paddingOptions.length - 1];
        }
        return paddingOptions[numeric] ?? paddingOptions[0];
    };

    const clampMarginValue = (value) => {
        if (!marginOptions.length) {
            return 0;
        }
        const numeric = Number(value);
        if (!Number.isFinite(numeric)) {
            return marginOptions[0];
        }
        if (numeric <= marginOptions[0]) {
            return marginOptions[0];
        }
        const last = marginOptions[marginOptions.length - 1];
        if (numeric >= last) {
            return last;
        }
        let closest = marginOptions[0];
        let minDiff = Math.abs(numeric - closest);
        for (let i = 1; i < marginOptions.length; i += 1) {
            const option = marginOptions[i];
            const diff = Math.abs(numeric - option);
            if (diff < minDiff) {
                closest = option;
                minDiff = diff;
            }
        }
        return closest;
    };

    const marginIndexForValue = (value) => {
        const clamped = clampMarginValue(value);
        const index = marginOptions.indexOf(clamped);
        return index >= 0 ? index : 0;
    };

    const marginValueForIndex = (index) => {
        const numeric = Number.parseInt(index, 10);
        if (Number.isNaN(numeric)) {
            return marginOptions[0];
        }
        if (numeric <= 0) {
            return marginOptions[0];
        }
        if (numeric >= marginOptions.length) {
            return marginOptions[marginOptions.length - 1];
        }
        return marginOptions[numeric] ?? marginOptions[0];
    };

    const SPACING_MODE_LINKED = 'linked';
    const SPACING_MODE_SEPARATE = 'separate';

    const normaliseSpacingModeValue = (value) => {
        const normalised = normaliseString(value).toLowerCase();
        return normalised === SPACING_MODE_SEPARATE
            ? SPACING_MODE_SEPARATE
            : SPACING_MODE_LINKED;
    };

    const resolveSpacingModeValue = (section) => {
        const explicitValue = normaliseString(
            section?.settings?.spacing_mode ?? section?.settings?.spacingMode ?? ''
        ).toLowerCase();
        if (
            explicitValue === SPACING_MODE_LINKED ||
            explicitValue === SPACING_MODE_SEPARATE
        ) {
            return explicitValue;
        }

        const paddingVertical = clampPaddingValue(section?.paddingVertical);
        const paddingTop = clampPaddingValue(
            section?.paddingTop ?? paddingVertical
        );
        const paddingBottom = clampPaddingValue(
            section?.paddingBottom ?? paddingVertical
        );
        const marginVertical = clampMarginValue(section?.marginVertical);
        const marginTop = clampMarginValue(section?.marginTop ?? marginVertical);
        const marginBottom = clampMarginValue(
            section?.marginBottom ?? marginVertical
        );

        if (
            paddingTop !== paddingVertical ||
            paddingBottom !== paddingVertical ||
            marginTop !== marginVertical ||
            marginBottom !== marginVertical
        ) {
            return SPACING_MODE_SEPARATE;
        }
        return SPACING_MODE_LINKED;
    };

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
    const describeAnimationValue = (value) => {
        const normalised = normaliseAnimationValue(value);
        const option = animationOptions.find(
            (entry) => normaliseString(entry?.value).toLowerCase() === normalised
        );
        return option?.label || normalised || 'None';
    };
    const isBlurEnabled = (value) => {
        if (value === true) {
            return true;
        }
        if (value === false) {
            return false;
        }
        const normalised = normaliseString(value).toLowerCase();
        if (['false', '0', 'no', 'off'].includes(normalised)) {
            return false;
        }
        if (['true', '1', 'yes', 'on'].includes(normalised)) {
            return true;
        }
        return true;
    };

    const elementLabel = (definitions, type) => {
        const definition = definitions[type];
        if (definition && definition.label) {
            return definition.label;
        }
        return type || 'Block';
    };

    const renderElementEditor = (
        definitions,
        elementNode,
        element,
        context = {}
    ) => {
        const definition = definitions[element.type];
        if (definition && typeof definition.renderEditor === 'function') {
            definition.renderEditor(elementNode, element, context);
            return;
        }
        if (element.unsupported) {
            elementNode.append(
                createElement('p', {
                    className: 'admin-builder__note',
                    textContent:
                        "This block type isn't supported in the visual builder yet, but it will be kept intact when you save.",
                })
            );
        }
    };

    const createView = ({
        listElement,
        emptyState,
        definitions,
        orderedTypes,
        sectionDefinitions,
        orderedSectionTypes,
        applyPaddingToAllSections,
        applyBackgroundGroupToSections,
        applyBackgroundStyleToGroup,
        isSectionCollapsed,
    }) => {
        const resolveAllowedElements = createAllowedElementsResolver(sectionDefinitions);
        const isElementAllowed = (sectionType, elementType) => {
            const allowedSet = resolveAllowedElements(sectionType);
            if (!allowedSet || allowedSet.size === 0) {
                return true;
            }
            const normalised = normaliseString(elementType).toLowerCase();
            return normalised ? allowedSet.has(normalised) : false;
        };

        const focusField = (selector) => {
            if (!selector) {
                return;
            }
            if (typeof window.requestAnimationFrame === 'function') {
                window.requestAnimationFrame(() => {
                    const field = listElement.querySelector(selector);
                    if (field && typeof field.focus === 'function') {
                        field.focus();
                    }
                });
                return;
            }
            const field = listElement.querySelector(selector);
            if (field && typeof field.focus === 'function') {
                field.focus();
            }
        };

        const formatSettingsSummary = (section, sectionDefinition) => {
            if (!section) {
                return '';
            }
            const countSelectedCourses = (value) => {
                if (!value) {
                    return 0;
                }
                const entries = Array.isArray(value)
                    ? value
                    : String(value).split(/[,;\n\r]/);
                return entries
                    .map((entry) => normaliseString(entry).toLowerCase())
                    .filter(Boolean).length;
            };
            const parts = [];
            parts.push(
                `Vertical padding: ${clampPaddingValue(section.paddingVertical)}px`
            );
            parts.push(
                `Padding top: ${clampPaddingValue(
                    section.paddingTop ?? section.paddingVertical
                )}px`
            );
            parts.push(
                `Padding bottom: ${clampPaddingValue(
                    section.paddingBottom ?? section.paddingVertical
                )}px`
            );
            parts.push(
                `Vertical margin: ${clampMarginValue(section.marginVertical)}px`
            );
            parts.push(
                `Margin top: ${clampMarginValue(
                    section.marginTop ?? section.marginVertical
                )}px`
            );
            parts.push(
                `Margin bottom: ${clampMarginValue(
                    section.marginBottom ?? section.marginVertical
                )}px`
            );
            parts.push(`Animation: ${describeAnimationValue(section.animation)}`);
            parts.push(`Blur: ${isBlurEnabled(section.animationBlur) ? 'on' : 'off'}`);
            if (sectionDefinition?.supportsHeaderImage === true) {
                parts.push(section.image ? 'Side image: added' : 'Side image: none');
            }
            if (section?.type === 'grid') {
                parts.push(
                    section.styleGridItems !== false
                        ? 'Grid styling: on'
                        : 'Grid styling: off'
                );
            }
            const limitDefinition = sectionDefinition?.settings?.limit;
            if (limitDefinition) {
                const limitValue = Number.isFinite(section.limit) && section.limit > 0
                    ? section.limit
                    : 'Automatic';
                parts.push(`Items limit: ${limitValue}`);
            }
            const modeDefinition = sectionDefinition?.settings?.mode;
            if (modeDefinition?.options?.length) {
                const modeLabel = describeModeValue(modeDefinition, section.mode);
                if (modeLabel) {
                    parts.push(`Mode: ${modeLabel}`);
                }
            }
            const variationLabel = describeVariationValue(
                sectionDefinition,
                section.variation
            );
            if (variationLabel) {
                parts.push(`Variation: ${variationLabel}`);
            }
            const backgroundGroup = normaliseBackgroundGroupToken(
                section.settings?.background_group ??
                    section.settings?.backgroundGroup
            );
            if (backgroundGroup) {
                parts.push(`Background group: ${backgroundGroup}`);
                const backgroundStyle = normaliseBackgroundStyleToken(
                    section.settings?.background_style ??
                        section.settings?.backgroundStyle
                );
                parts.push(
                    `Background: ${describeBackgroundStyleValue(backgroundStyle)}`
                );
            }
            const displayModeDefinition = sectionDefinition?.settings?.display_mode;
            if (displayModeDefinition?.options?.length) {
                const displayLabel = describeModeValue(
                    displayModeDefinition,
                    section.settings?.display_mode
                );
                if (displayLabel) {
                    parts.push(`Display: ${displayLabel}`);
                }
                const displayMode = normaliseString(
                    section.settings?.display_mode
                ).toLowerCase();
                const supportsSelection =
                    displayMode === 'selected' || displayMode === 'carousel';
                if (supportsSelection) {
                    const selectedCount = countSelectedCourses(
                        section.settings?.selected_courses
                    );
                    if (selectedCount > 0) {
                        parts.push(`Selected courses: ${selectedCount}`);
                    }
                }
            }
            // Show custom section settings preview (like hero title)
            if (section.settings && Object.keys(section.settings).length > 0) {
                if (section.settings.title) {
                    parts.push(`Title: ${section.settings.title}`);
                }
            }
            return parts.join(' · ');
        };

        const createSectionSettingsModal = ({
            sectionItem,
            section,
            sectionDefinition,
            onChange,
            onClose,
            applyPaddingToAllSections: applyPaddingCallback,
            applyBackgroundGroupToSections: applyBackgroundGroupCallback,
            applyBackgroundStyleToGroup: applyBackgroundStyleCallback,
            renderInline = false,
        }) => {
            if (!sectionItem || !section) {
                return renderInline ? null : () => {};
            }

            if (!renderInline) {
                const existingModal = sectionItem.querySelector(
                    '[data-role="section-settings"]'
                );
                if (existingModal) {
                    const focusTarget = existingModal.querySelector('[data-field]');
                    if (focusTarget && typeof focusTarget.focus === 'function') {
                        focusTarget.focus();
                    }
                    return () => {};
                }
            }

            const scheduleChange = (() => {
                let pending = false;
                const notify = () => {
                    pending = false;
                    if (typeof onChange === 'function') {
                        onChange();
                    }
                };
                return () => {
                    if (pending) {
                        return;
                    }
                    pending = true;
                    if (typeof window.requestAnimationFrame === 'function') {
                        window.requestAnimationFrame(notify);
                    } else {
                        window.setTimeout(notify, 0);
                    }
                };
            })();
            let closeModal = () => {};
            let overlay = null;
            let dialog = null;
            const inlineSettingsRoot = renderInline
                ? createElement('div', {
                      className: 'admin-builder__settings-inline',
                      dataset: {
                          role: 'section-settings-inline',
                      },
                  })
                : null;

            if (!renderInline) {
                overlay = createElement('div', {
                    className: 'admin-builder__settings-overlay',
                    dataset: {
                        role: 'section-settings',
                    },
                });
                dialog = createElement('div', {
                    className: 'admin-builder__settings-dialog',
                    attributes: {
                        role: 'dialog',
                        'aria-modal': 'true',
                    },
                });
                dialog.setAttribute('tabindex', '-1');
                const titleId = `admin-builder-section-settings-${
                    section.id || section.clientId
                }`;
                dialog.setAttribute('aria-labelledby', titleId);

                const header = createElement('header', {
                    className: 'admin-builder__settings-header',
                });
                const titleNode = createElement('h2', {
                    className: 'admin-builder__settings-title',
                    textContent: 'Additional settings',
                });
                titleNode.id = titleId;
                header.append(titleNode);
                dialog.append(header);
            }

            const body = createElement('div', {
                className: renderInline
                    ? 'admin-builder__settings-body admin-builder__settings-body--inline'
                    : 'admin-builder__settings-body',
            });

            const createSettingsGroup = (title) => {
                const group = createElement('section', {
                    className: 'admin-builder__settings-group',
                });
                if (title) {
                    group.append(
                        createElement('h3', {
                            className: 'admin-builder__settings-group-title',
                            textContent: title,
                        })
                    );
                }
                const content = createElement('div', {
                    className: 'admin-builder__settings-group-body',
                });
                group.append(content);
                return { group, content };
            };

            const generalSettings = createSettingsGroup('Content & layout');
            const spacingSettings = createSettingsGroup('Section spacing');
            const animationSettings = createSettingsGroup('Section animation');
            body.append(generalSettings.group, spacingSettings.group, animationSettings.group);

            const appendField = (field) => {
                if (field) {
                    generalSettings.content.append(field);
                }
            };
            const appendSpacingField = (field) => {
                if (field) {
                    spacingSettings.content.append(field);
                }
            };
            const appendAnimationField = (field) => {
                if (field) {
                    animationSettings.content.append(field);
                }
            };

            if (sectionDefinition?.supportsHeaderImage === true) {
                const imageField = createElement('label', {
                    className: 'admin-builder__field',
                });
                imageField.append(
                    createElement('span', {
                        className: 'admin-builder__label',
                        textContent: 'Side image URL',
                    })
                );
                const imageInput = createElement('input', {
                    className: 'admin-builder__input',
                });
                imageInput.type = 'url';
                imageInput.placeholder = 'https://example.com/side-image.jpg';
                imageInput.value = section.image || '';
                imageInput.dataset.field = 'section-image';
                const imageInputId = `admin-builder-section-image-${
                    section.id || section.clientId
                }`;
                imageInput.id = imageInputId;
                imageInput.addEventListener('input', scheduleChange);
                imageField.append(imageInput);

                const actions = createElement('div', {
                    className: 'admin-builder__field-actions',
                });
                const browseButton = createElement('button', {
                    className: 'admin-builder__media-button',
                    textContent: 'Browse uploads',
                });
                browseButton.type = 'button';
                browseButton.dataset.action = 'open-media-library';
                browseButton.dataset.mediaTarget = `#${imageInputId}`;
                browseButton.dataset.mediaAllowedTypes = 'image';
                actions.append(browseButton);
                imageField.append(actions);

                appendField(imageField);
            }

            if (section.type === 'grid') {
                const styleField = createElement('label', {
                    className: 'admin-builder__field admin-builder__field--checkbox',
                });
                const styleInput = createElement('input', {
                    className: 'admin-builder__checkbox checkbox__input',
                });
                styleInput.type = 'checkbox';
                styleInput.dataset.field = 'section-grid-style';
                styleInput.checked = section.styleGridItems !== false;
                styleInput.addEventListener('change', scheduleChange);
                const styleLabel = createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Apply border, background, and padding to grid items',
                });
                styleField.append(styleInput, styleLabel);
                appendField(styleField);
            }

            const displayModeDefinition = sectionDefinition?.settings?.display_mode;
            const displayModeOptions =
                displayModeDefinition?.options?.filter((option) =>
                    Boolean(normaliseString(option?.value))
                ) || [];
            const displayModeDefault =
                (displayModeDefinition?.defaultValue ??
                    displayModeDefinition?.default_value ??
                    displayModeOptions[0]?.value) ||
                'limited';
            const displayModeSelect =
                displayModeOptions.length > 0
                    ? createElement('select', {
                          className: 'admin-builder__select',
                      })
                    : null;
            let showAllCheckbox = null;
            if (displayModeSelect) {
                const displayModeField = createElement('label', {
                    className: 'admin-builder__field',
                });
                displayModeField.append(
                    createElement('span', {
                        className: 'admin-builder__label',
                        textContent: displayModeDefinition?.label || 'Display mode',
                    })
                );
                const currentDisplayMode = normaliseModeValue(
                    {
                        options: displayModeOptions,
                        defaultValue: displayModeDefault,
                    },
                    section.settings?.display_mode
                );
                displayModeOptions.forEach((option) => {
                    const value = normaliseString(option?.value).toLowerCase();
                    if (!value) {
                        return;
                    }
                    const optionNode = createElement('option', {
                        value,
                        textContent:
                            normaliseString(option?.label) ||
                            normaliseString(option?.value),
                    });
                    if (value === currentDisplayMode) {
                        optionNode.selected = true;
                    }
                    displayModeSelect.append(optionNode);
                });
                displayModeField.append(displayModeSelect);
                appendField(displayModeField);
            }

            const conditionalFields = [];
            const registerConditionalField = (node, modes) => {
                if (!node) {
                    return;
                }
                conditionalFields.push({
                    node,
                    modes: Array.isArray(modes)
                        ? modes.map((mode) => normaliseString(mode).toLowerCase())
                        : null,
                });
            };

            const limitDefinition = sectionDefinition?.settings?.limit;
            let limitLabelSpan;
            let limitField;
            const defaultLimitLabel =
                limitDefinition?.label || 'Number of items to display';
            const perPageLimitLabel =
                limitDefinition?.perPageLabel ||
                (defaultLimitLabel
                    ? `${defaultLimitLabel} on a page`
                    : 'Number of items to display on a page');
            if (limitDefinition) {
                limitField = createElement('label', {
                    className: 'admin-builder__field',
                });
                limitField.append(
                    (limitLabelSpan = createElement('span', {
                        className: 'admin-builder__label',
                        textContent: defaultLimitLabel,
                    }))
                );
                const limitInput = createElement('input', {
                    className: 'admin-builder__input',
                });
                limitInput.type = 'number';
                if (Number.isFinite(limitDefinition.min)) {
                    limitInput.min = String(limitDefinition.min);
                }
                if (Number.isFinite(limitDefinition.max)) {
                    limitInput.max = String(limitDefinition.max);
                }
                limitInput.step = '1';
                const defaultLimit = Number.isFinite(limitDefinition.default)
                    ? limitDefinition.default
                    : Number.parseInt(limitDefinition.default, 10);
                const limitValue =
                    Number.isFinite(section.limit) && section.limit > 0
                        ? section.limit
                        : Number.isFinite(defaultLimit) && defaultLimit > 0
                          ? defaultLimit
                          : NaN;
                limitInput.value = Number.isFinite(limitValue)
                    ? String(Math.round(limitValue))
                    : '';
                limitInput.dataset.field = 'section-limit';
                limitInput.addEventListener('input', scheduleChange);
                limitField.append(limitInput);
                registerConditionalField(limitField, ['limited', 'paginated', 'selected']);
                appendField(limitField);
            }

            const modeDefinition = sectionDefinition?.settings?.mode;
            if (modeDefinition?.options?.length) {
                const modeField = createElement('label', {
                    className: 'admin-builder__field',
                });
                modeField.append(
                    createElement('span', {
                        className: 'admin-builder__label',
                        textContent: modeDefinition.label || 'Courses to show',
                    })
                );
                const modeSelect = createElement('select', {
                    className: 'admin-builder__select',
                });
                modeSelect.dataset.field = 'section-mode';
                const currentMode = normaliseModeValue(
                    modeDefinition,
                    section.mode
                );
                modeDefinition.options.forEach((option) => {
                    const value = normaliseString(option?.value).toLowerCase();
                    if (!value) {
                        return;
                    }
                    const optionNode = createElement('option', {
                        value,
                        textContent:
                            normaliseString(option?.label) ||
                            normaliseString(option?.value),
                    });
                    if (value === currentMode) {
                        optionNode.selected = true;
                    }
                    modeSelect.append(optionNode);
                });
                modeSelect.addEventListener('change', () => {
                    section.mode = normaliseString(modeSelect.value).toLowerCase();
                    scheduleChange();
                });
                modeField.append(modeSelect);
                appendField(modeField);
            }

            // Handle custom section settings (like hero section fields)
            if (sectionDefinition?.settings) {
                const activeVariation = normaliseVariationValue(
                    sectionDefinition,
                    section.variation
                );
                const allSettingEntries = Object.entries(sectionDefinition.settings);
                const prioritizedSettingKeys = [
                    'title_position',
                    'description_position',
                ];
                const prioritizedSettings = [];
                prioritizedSettingKeys.forEach((priorityKey) => {
                    const matchedEntry = allSettingEntries.find(
                        ([candidateKey]) => candidateKey === priorityKey
                    );
                    if (matchedEntry) {
                        prioritizedSettings.push(matchedEntry);
                    }
                });
                const regularSettings = allSettingEntries.filter(
                    ([candidateKey]) => !prioritizedSettingKeys.includes(candidateKey)
                );
                const orderedSettings = [...prioritizedSettings, ...regularSettings];

                orderedSettings.forEach(([key, settingDef]) => {
                    // Skip limit, mode, and display_mode as they are handled above
                    if (
                        key === 'limit' ||
                        key === 'mode' ||
                        key === 'display_mode' ||
                        key === 'variation'
                    ) {
                        return;
                    }

                    if (!section.settings) {
                        section.settings = {};
                    }

                    const options = Array.isArray(settingDef.options)
                        ? settingDef.options.filter((option) =>
                              Boolean(normaliseString(option?.value))
                          )
                        : [];
                    let fieldType = normaliseString(settingDef.type).toLowerCase();
                    if (!fieldType) {
                        fieldType = options.length ? 'select' : 'text';
                    }
                    if (fieldType === 'string') {
                        fieldType = 'text';
                    } else if (fieldType === 'integer') {
                        fieldType = 'number';
                    }
                    const fieldLabel = settingDef.label || key;
                    const isRequired = settingDef.required === true;
                    const hiddenForVariationsSource = Array.isArray(
                        settingDef.hiddenForVariations
                    )
                        ? settingDef.hiddenForVariations
                        : Array.isArray(settingDef.hidden_for_variations)
                          ? settingDef.hidden_for_variations
                          : [];
                    const hiddenForVariations = hiddenForVariationsSource
                        .map((value) => normaliseString(value).toLowerCase())
                        .filter(Boolean);
                    if (
                        hiddenForVariations.length > 0 &&
                        hiddenForVariations.includes(activeVariation)
                    ) {
                        if (
                            section.settings &&
                            Object.prototype.hasOwnProperty.call(
                                section.settings,
                                key
                            )
                        ) {
                            delete section.settings[key];
                        }
                        return;
                    }

                    if (fieldType === 'select' && options.length) {
                        const field = createElement('label', {
                            className: 'admin-builder__field',
                        });
                        const labelSpan = createElement('span', {
                            className: 'admin-builder__label',
                            textContent: fieldLabel,
                        });
                        field.append(labelSpan);

                        const select = createElement('select', {
                            className: 'admin-builder__select',
                        });
                        select.dataset.field = `section-setting-${key}`;
                        const currentValue = normaliseModeValue(
                            {
                                options,
                                defaultValue:
                                    settingDef.defaultValue ?? settingDef.default_value ?? '',
                            },
                            section.settings[key]
                        );
                        options.forEach((option) => {
                            const value = normaliseString(option?.value).toLowerCase();
                            if (!value) {
                                return;
                            }
                            const optionNode = createElement('option', {
                                value,
                                textContent:
                                    normaliseString(option?.label) ||
                                    normaliseString(option?.value),
                            });
                            if (value === currentValue) {
                                optionNode.selected = true;
                            }
                            select.append(optionNode);
                        });
                        select.addEventListener('change', () => {
                            section.settings[key] = normaliseString(select.value);
                            scheduleChange();
                        });
                        field.append(select);
                        appendField(field);
                    } else if (fieldType === 'boolean') {
                        const field = createElement('label', {
                            className: 'admin-builder__field admin-builder__field--checkbox',
                        });
                        const input = createElement('input', {
                            className: 'admin-builder__checkbox checkbox__input',
                        });
                        input.type = 'checkbox';
                        input.dataset.field = `section-setting-${key}`;
                        const rawValue = section.settings[key];
                        const normalisedValue =
                            typeof rawValue === 'string'
                                ? rawValue.trim().toLowerCase()
                                : rawValue;
                        input.checked =
                            rawValue === true ||
                            normalisedValue === 'true' ||
                            normalisedValue === '1' ||
                            normalisedValue === 1 ||
                            normalisedValue === 'yes' ||
                            normalisedValue === 'on';
                        input.addEventListener('change', scheduleChange);
                        const labelSpan = createElement('span', {
                            className: 'admin-builder__label',
                            textContent: fieldLabel,
                        });
                        if (isRequired) {
                            labelSpan.append(
                                createElement('em', {
                                    className: 'admin-builder__required',
                                    textContent: ' (required)',
                                })
                            );
                        }
                        field.append(input, labelSpan);
                        if (key === 'show_all_button') {
                            registerConditionalField(field, ['paginated', 'selected']);
                            showAllCheckbox = input;
                        }
                        appendField(field);
                    } else if (fieldType === 'number') {
                        const min = Number.isFinite(settingDef.min)
                            ? settingDef.min
                            : undefined;
                        const max = Number.isFinite(settingDef.max)
                            ? settingDef.max
                            : undefined;
                        const step = Number.isFinite(settingDef.step)
                            ? settingDef.step
                            : 1;
                        const defaultValue = Number.isFinite(settingDef.default)
                            ? settingDef.default
                            : Number.parseInt(settingDef.default, 10);
                        const parsedCurrent = Number.parseInt(
                            section.settings[key],
                            10
                        );
                        const currentValue = Number.isFinite(parsedCurrent)
                            ? parsedCurrent
                            : Number.isFinite(defaultValue)
                              ? defaultValue
                              : Number.isFinite(min)
                                ? min
                                : 0;

                        const field = createElement('label', {
                            className: 'admin-builder__field',
                        });
                        const labelSpan = createElement('span', {
                            className: 'admin-builder__label',
                            textContent: fieldLabel,
                        });
                        if (isRequired) {
                            labelSpan.append(
                                createElement('em', {
                                    className: 'admin-builder__required',
                                    textContent: ' (required)',
                                })
                            );
                        }
                        field.append(labelSpan);

                        const input = createElement('input', {
                            className: 'admin-builder__input',
                        });
                        input.type = 'number';
                        if (Number.isFinite(min)) {
                            input.min = String(min);
                        }
                        if (Number.isFinite(max)) {
                            input.max = String(max);
                        }
                        input.step = String(step);
                        input.value = String(currentValue);
                        input.dataset.field = `section-setting-${key}`;
                        if (isRequired) {
                            input.required = true;
                        }
                        input.addEventListener('input', () => {
                            const parsedValue = Number.parseInt(input.value, 10);
                            if (Number.isFinite(parsedValue)) {
                                section.settings[key] = parsedValue;
                            } else if (input.value.trim()) {
                                section.settings[key] = input.value.trim();
                            } else {
                                delete section.settings[key];
                            }
                            scheduleChange();
                        });
                        field.append(input);
                        appendField(field);
                    } else if (fieldType === 'range') {
                        const min = Number.isFinite(settingDef.min)
                            ? settingDef.min
                            : 0;
                        const max = Number.isFinite(settingDef.max)
                            ? settingDef.max
                            : min + 10;
                        const step = Number.isFinite(settingDef.step)
                            ? settingDef.step
                            : 1;
                        const defaultValue = Number.isFinite(settingDef.default)
                            ? settingDef.default
                            : Number.parseInt(settingDef.default, 10);
                        const currentValue = Number.isFinite(section.settings[key])
                            ? section.settings[key]
                            : Number.isFinite(defaultValue)
                              ? defaultValue
                              : min;

                        const field = createElement('label', {
                            className: 'admin-builder__field',
                        });
                        const labelSpan = createElement('span', {
                            className: 'admin-builder__label',
                            textContent: fieldLabel,
                        });
                        if (isRequired) {
                            labelSpan.append(
                                createElement('em', {
                                    className: 'admin-builder__required',
                                    textContent: ' (required)',
                                })
                            );
                        }
                        field.append(labelSpan);

                        const rangeWrapper = createElement('div', {
                            className: 'admin-builder__range',
                        });
                        const rangeInput = createElement('input', {
                            className: 'admin-builder__range-input',
                        });
                        rangeInput.type = 'range';
                        rangeInput.min = String(min);
                        rangeInput.max = String(max);
                        rangeInput.step = String(step);
                        rangeInput.value = String(currentValue);
                        rangeInput.dataset.field = `section-setting-${key}`;
                        rangeInput.setAttribute('aria-valuemin', String(min));
                        rangeInput.setAttribute('aria-valuemax', String(max));
                        rangeInput.setAttribute('aria-valuenow', String(currentValue));
                        const rangeValue = createElement('span', {
                            className: 'admin-builder__range-value',
                            textContent: String(currentValue),
                        });
                        rangeWrapper.append(rangeInput, rangeValue);
                        rangeInput.addEventListener('input', () => {
                            const value = Number.parseInt(rangeInput.value, 10);
                            rangeValue.textContent = String(value);
                            rangeInput.setAttribute('aria-valuenow', String(value));
                            section.settings[key] = value;
                            scheduleChange();
                        });
                        field.append(rangeWrapper);
                        appendField(field);
                    } else if (fieldType === 'textarea') {
                        const field = createElement('label', {
                            className: 'admin-builder__field',
                        });
                        const labelSpan = createElement('span', {
                            className: 'admin-builder__label',
                            textContent: fieldLabel,
                        });
                        if (isRequired) {
                            labelSpan.append(
                                createElement('em', {
                                    className: 'admin-builder__required',
                                    textContent: ' (required)',
                                })
                            );
                        }
                        field.append(labelSpan);

                        const textarea = createElement('textarea', {
                            className: 'admin-builder__input',
                            attributes: {
                                rows: '3',
                            },
                        });
                        textarea.placeholder = settingDef.placeholder || '';
                        textarea.value = section.settings[key] || '';
                        textarea.dataset.field = `section-setting-${key}`;
                        if (isRequired) {
                            textarea.required = true;
                        }
                        textarea.addEventListener('input', scheduleChange);
                        field.append(textarea);
                        appendField(field);
                    } else {
                        const field = createElement('label', {
                            className: 'admin-builder__field',
                        });
                        const labelSpan = createElement('span', {
                            className: 'admin-builder__label',
                            textContent: fieldLabel,
                        });
                        if (isRequired) {
                            labelSpan.append(
                                createElement('em', {
                                    className: 'admin-builder__required',
                                    textContent: ' (required)',
                                })
                            );
                        }
                        field.append(labelSpan);
                        
                        const input = createElement('input', {
                            className: 'admin-builder__input',
                        });
                        input.type = fieldType === 'url' ? 'url' : 'text';
                        input.placeholder = settingDef.placeholder || '';
                        input.value = section.settings[key] || '';
                        input.dataset.field = `section-setting-${key}`;
                        if (isRequired) {
                            input.required = true;
                        }
                        input.addEventListener('input', scheduleChange);
                        field.append(input);

                        if (key === 'selected_courses' || key === 'selected_posts') {
                            registerConditionalField(field, ['selected', 'carousel']);
                        }

                        // Add media browse button for image/url fields
                        if (
                            settingDef.allowMediaBrowse ||
                            settingDef.allowAnchorPicker ||
                            settingDef.allowCoursePicker ||
                            settingDef.allowPostPicker
                        ) {
                            const inputId = `section-${section.clientId}-setting-${key}`;
                            input.id = inputId;
                            const actions = createElement('div', {
                                className: 'admin-builder__field-actions',
                            });
                            
                            if (settingDef.allowMediaBrowse) {
                                const browseButton = createElement('button', {
                                    className: 'admin-builder__media-button',
                                    textContent: 'Browse uploads',
                                });
                                browseButton.type = 'button';
                                browseButton.dataset.action = 'open-media-library';
                                browseButton.dataset.mediaTarget = `#${inputId}`;
                                browseButton.dataset.mediaAllowedTypes = 'image';
                                actions.append(browseButton);
                            }
                            
                            if (settingDef.allowAnchorPicker) {
                                const anchorButton = createElement('button', {
                                    className: 'admin-builder__anchor-button',
                                    textContent: 'Link to section',
                                });
                                anchorButton.type = 'button';
                                anchorButton.dataset.action = 'open-anchor-picker';
                                anchorButton.dataset.anchorTarget = `#${inputId}`;
                                actions.append(anchorButton);
                            }

                            if (settingDef.allowCoursePicker) {
                                const courseButton = createElement('button', {
                                    className: 'admin-builder__anchor-button',
                                    textContent: 'Select courses',
                                });
                                courseButton.type = 'button';
                                courseButton.dataset.action = 'open-course-picker';
                                courseButton.dataset.courseTarget = `#${inputId}`;
                                actions.append(courseButton);
                            }

                            if (settingDef.allowPostPicker) {
                                const postButton = createElement('button', {
                                    className: 'admin-builder__anchor-button',
                                    textContent: 'Select posts',
                                });
                                postButton.type = 'button';
                                postButton.dataset.action = 'open-post-picker';
                                postButton.dataset.postTarget = `#${inputId}`;
                                actions.append(postButton);
                            }
                            
                            field.append(actions);
                        }

                        if (key === 'all_courses_url' || key === 'all_courses_label') {
                            registerConditionalField(field, ['paginated', 'selected'], true);
                        }
                        appendField(field);
                    }
                });
            }

            const initialBackgroundGroup = normaliseBackgroundGroupToken(
                section.settings?.background_group ??
                    section.settings?.backgroundGroup
            );
            const initialBackgroundStyle = normaliseBackgroundStyleToken(
                section.settings?.background_style ??
                    section.settings?.backgroundStyle
            );
            if (section.settings?.backgroundGroup !== undefined) {
                delete section.settings.backgroundGroup;
            }
            if (section.settings?.backgroundStyle !== undefined) {
                delete section.settings.backgroundStyle;
            }
            const applyCurrentBackgroundGroup = (value) => {
                const nextGroup = normaliseBackgroundGroupToken(value);
                section.settings = section.settings || {};
                if (nextGroup) {
                    section.settings.background_group = nextGroup;
                } else {
                    delete section.settings.background_group;
                }
                if (section.settings.backgroundGroup !== undefined) {
                    delete section.settings.backgroundGroup;
                }
                sectionItem.dataset.sectionBackgroundGroup = nextGroup;
                return nextGroup;
            };
            const applyCurrentBackgroundStyle = (value) => {
                const nextStyle = normaliseBackgroundStyleToken(value);
                section.settings = section.settings || {};
                if (nextStyle) {
                    section.settings.background_style = nextStyle;
                } else {
                    delete section.settings.background_style;
                }
                if (section.settings.backgroundStyle !== undefined) {
                    delete section.settings.backgroundStyle;
                }
                sectionItem.dataset.sectionBackgroundStyle = nextStyle;
                return nextStyle;
            };
            applyCurrentBackgroundGroup(initialBackgroundGroup);
            applyCurrentBackgroundStyle(initialBackgroundStyle);

            const backgroundGroupField = createElement('label', {
                className: 'admin-builder__field',
            });
            backgroundGroupField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Background group',
                })
            );
            const backgroundGroupInput = createElement('input', {
                className: 'admin-builder__input',
            });
            backgroundGroupInput.type = 'text';
            backgroundGroupInput.placeholder = 'e.g. hero-surface';
            backgroundGroupInput.value = initialBackgroundGroup;
            backgroundGroupInput.dataset.field = 'section-setting-background_group';
            backgroundGroupInput.addEventListener('input', () => {
                const nextGroup = applyCurrentBackgroundGroup(
                    backgroundGroupInput.value
                );
                if (backgroundGroupInput.value !== nextGroup) {
                    backgroundGroupInput.value = nextGroup;
                }
                scheduleChange();
            });
            backgroundGroupField.append(backgroundGroupInput);
            const backgroundStyleSelect = createElement('select', {
                className: 'admin-builder__select',
            });
            backgroundStyleSelect.dataset.field = 'section-setting-background_style';
            const knownStyleValues = new Set();
            backgroundStyleOptions.forEach((option) => {
                const optionValue = normaliseBackgroundStyleToken(option.value);
                knownStyleValues.add(optionValue);
                const optionNode = createElement('option', {
                    value: optionValue,
                    textContent: option.label,
                });
                backgroundStyleSelect.append(optionNode);
            });
            if (
                initialBackgroundStyle &&
                !knownStyleValues.has(initialBackgroundStyle)
            ) {
                backgroundStyleSelect.append(
                    createElement('option', {
                        value: initialBackgroundStyle,
                        textContent: `Custom (${initialBackgroundStyle})`,
                    })
                );
            }
            backgroundStyleSelect.value = initialBackgroundStyle;
            backgroundStyleSelect.addEventListener('change', () => {
                let nextStyle = applyCurrentBackgroundStyle(
                    backgroundStyleSelect.value
                );
                const activeGroup = normaliseBackgroundGroupToken(
                    backgroundGroupInput.value
                );
                if (
                    activeGroup &&
                    typeof applyBackgroundStyleCallback === 'function'
                ) {
                    const appliedStyle = applyBackgroundStyleCallback({
                        groupValue: activeGroup,
                        styleValue: nextStyle,
                    });
                    nextStyle = applyCurrentBackgroundStyle(
                        appliedStyle || nextStyle
                    );
                }
                if (backgroundStyleSelect.value !== nextStyle) {
                    backgroundStyleSelect.value = nextStyle;
                }
                scheduleChange();
            });
            backgroundGroupField.append(backgroundStyleSelect);
            const backgroundGroupActions = createElement('div', {
                className: 'admin-builder__field-actions',
            });
            const backgroundGroupPickerButton = createElement('button', {
                className: 'admin-builder__anchor-button',
                textContent: 'Select sections',
            });
            backgroundGroupPickerButton.type = 'button';
            backgroundGroupPickerButton.disabled =
                typeof applyBackgroundGroupCallback !== 'function';
            backgroundGroupActions.append(backgroundGroupPickerButton);
            backgroundGroupField.append(backgroundGroupActions);
            backgroundGroupField.append(
                createElement('p', {
                    className: 'admin-builder__hint',
                    textContent:
                        'Sections in one group share one continuous background and style.',
                })
            );
            appendField(backgroundGroupField);

            const openBackgroundGroupPicker = () => {
                const sectionNodes = Array.from(
                    document.querySelectorAll('[data-section-client]')
                );
                if (!sectionNodes.length) {
                    window.alert('No sections found on this page.');
                    return;
                }

                const previousFocus = document.activeElement;
                const currentGroup = normaliseBackgroundGroupToken(
                    backgroundGroupInput.value
                );
                const overlay = createElement('div', {
                    className: 'anchor-picker-overlay',
                });
                const modal = createElement('div', {
                    className: 'anchor-picker-modal',
                });
                const header = createElement('div', {
                    className: 'anchor-picker-header',
                });
                const titleNode = createElement('h3', {
                    className: 'anchor-picker-title',
                    textContent: 'Select sections for this background',
                });
                header.append(titleNode);
                const list = createElement('div', {
                    className: 'anchor-picker-list',
                });

                sectionNodes.forEach((node, index) => {
                    const sectionClientId = normaliseString(
                        node.dataset.sectionClient
                    );
                    if (!sectionClientId) {
                        return;
                    }

                    const sectionType = normaliseString(
                        node.dataset.sectionType
                    ).trim();
                    const sectionId = normaliseString(
                        node.dataset.sectionId
                    ).trim();
                    const sectionTitleInput = node.querySelector(
                        '[data-field="section-title"]'
                    );
                    const rawTitle = normaliseString(
                        sectionTitleInput?.value
                    ).trim();
                    const displayTitle =
                        rawTitle || `Section ${index + 1}`;
                    const sectionGroup = normaliseBackgroundGroupToken(
                        node.dataset.sectionBackgroundGroup
                    );
                    const shouldSelect = currentGroup
                        ? sectionGroup === currentGroup
                        : sectionClientId === section.clientId;

                    const item = createElement('label', {
                        className:
                            'anchor-picker-item anchor-picker-item--selectable',
                    });
                    const checkbox = createElement('input', {
                        className: 'anchor-picker-checkbox',
                        attributes: {
                            type: 'checkbox',
                        },
                    });
                    checkbox.dataset.sectionClient = sectionClientId;
                    checkbox.checked = shouldSelect;

                    const body = createElement('div');
                    const itemTitle = createElement('div', {
                        className: 'anchor-picker-item-title',
                        textContent: displayTitle,
                    });
                    body.append(itemTitle);

                    const meta = createElement('div', {
                        className: 'anchor-picker-item-meta',
                    });
                    if (sectionType) {
                        meta.append(
                            createElement('span', {
                                textContent: `Type: ${sectionType}`,
                            })
                        );
                    }
                    if (sectionId) {
                        meta.append(
                            createElement('code', {
                                className: 'anchor-picker-item-code',
                                textContent: sectionId,
                            })
                        );
                    }
                    if (sectionGroup) {
                        meta.append(
                            createElement('span', {
                                textContent: `Group: ${sectionGroup}`,
                            })
                        );
                    }
                    if (meta.childElementCount > 0) {
                        body.append(meta);
                    }

                    item.append(checkbox, body);
                    list.append(item);
                });

                if (!list.childElementCount) {
                    window.alert('No sections found on this page.');
                    return;
                }

                const footer = createElement('div', {
                    className: 'anchor-picker-footer',
                });
                const cancelButton = createElement('button', {
                    className: 'admin-builder__button',
                    textContent: 'Cancel',
                });
                cancelButton.type = 'button';
                const applyButton = createElement('button', {
                    className:
                        'admin-builder__button admin-builder__button--primary',
                    textContent: 'Apply',
                });
                applyButton.type = 'button';
                footer.append(cancelButton, applyButton);

                const closePicker = () => {
                    if (!document.body.contains(overlay)) {
                        return;
                    }
                    document.body.removeChild(overlay);
                    if (previousFocus && typeof previousFocus.focus === 'function') {
                        previousFocus.focus();
                    }
                };

                cancelButton.addEventListener('click', closePicker);
                overlay.addEventListener('click', (event) => {
                    if (event.target === overlay) {
                        closePicker();
                    }
                });
                applyButton.addEventListener('click', () => {
                    const selectedSectionClientIds = Array.from(
                        list.querySelectorAll('.anchor-picker-checkbox:checked')
                    )
                        .map((input) => normaliseString(input.dataset.sectionClient))
                        .filter(Boolean);

                    if (!selectedSectionClientIds.length) {
                        window.alert('Select at least one section.');
                        return;
                    }

                    let nextGroup = normaliseBackgroundGroupToken(
                        backgroundGroupInput.value
                    );
                    if (!nextGroup) {
                        nextGroup = normaliseBackgroundGroupToken(
                            section.id || section.title || section.clientId
                        );
                    }
                    if (!nextGroup) {
                        nextGroup = `bg-group-${Date.now().toString(36)}`;
                    }

                    const appliedGroup = applyBackgroundGroupCallback({
                        sectionClientIds: selectedSectionClientIds,
                        groupValue: nextGroup,
                        previousGroup: currentGroup,
                        styleValue: backgroundStyleSelect.value,
                    });
                    const resolvedGroup = applyCurrentBackgroundGroup(
                        appliedGroup || nextGroup
                    );
                    backgroundGroupInput.value = resolvedGroup;
                    scheduleChange();
                    closePicker();
                });

                modal.append(header, list, footer);
                overlay.append(modal);
                document.body.append(overlay);
            };
            backgroundGroupPickerButton.addEventListener('click', (event) => {
                event.preventDefault();
                if (typeof applyBackgroundGroupCallback !== 'function') {
                    return;
                }
                openBackgroundGroupPicker();
            });

            const animationValue = normaliseAnimationValue(section.animation);
            const animationField = createElement('label', {
                className: 'admin-builder__field',
            });
            animationField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Section animation',
                })
            );
            const animationSelect = createElement('select', {
                className: 'admin-builder__input',
                dataset: {
                    field: 'section-animation',
                },
            });
            animationOptions.forEach((option) => {
                const optionNode = createElement('option', {
                    value: option.value,
                    textContent: option.label || option.value,
                });
                if (option.description) {
                    optionNode.title = option.description;
                }
                animationSelect.append(optionNode);
            });
            animationSelect.value =
                animationValues.has(animationValue) && animationValue
                    ? animationValue
                    : defaultAnimation;
            const animationHint = createElement('p', {
                className: 'admin-builder__hint',
            });
            const updateAnimationHint = () => {
                const option = animationOptions.find(
                    (entry) =>
                        normaliseString(entry?.value).toLowerCase() ===
                        normaliseString(animationSelect.value).toLowerCase()
                );
                if (option?.description) {
                    animationHint.textContent = option.description;
                    animationHint.hidden = false;
                } else {
                    animationHint.textContent = '';
                    animationHint.hidden = true;
                }
            };
            updateAnimationHint();
            animationSelect.addEventListener('change', () => {
                if (!animationValues.has(normaliseString(animationSelect.value).toLowerCase())) {
                    animationSelect.value = defaultAnimation;
                }
                updateAnimationHint();
                scheduleChange();
            });
            animationField.append(animationSelect, animationHint);
            appendAnimationField(animationField);
            const animationBlurField = createElement('label', {
                className: 'admin-builder__field admin-builder__field--checkbox',
            });
            const animationBlurInput = createElement('input', {
                className: 'admin-builder__checkbox checkbox__input',
                attributes: { type: 'checkbox' },
                dataset: {
                    field: 'section-animation-blur',
                },
            });
            animationBlurInput.checked = isBlurEnabled(section.animationBlur);
            animationBlurInput.addEventListener('change', scheduleChange);
            const animationBlurLabel = createElement('span', {
                className: 'admin-builder__label',
                textContent: 'Use blur in animation',
            });
            const animationBlurHint = createElement('p', {
                className: 'admin-builder__hint',
                textContent: 'Disable to keep edges crisp while animating.',
            });
            animationBlurField.append(animationBlurInput, animationBlurLabel, animationBlurHint);
            appendAnimationField(animationBlurField);

            const spacingModeField = createElement('label', {
                className: 'admin-builder__field',
            });
            spacingModeField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Spacing controls',
                })
            );
            const spacingModeSelect = createElement('select', {
                className: 'admin-builder__input',
            });
            spacingModeSelect.dataset.field = 'section-setting-spacing_mode';
            [
                {
                    value: SPACING_MODE_LINKED,
                    label: 'Linked (top + bottom together)',
                },
                {
                    value: SPACING_MODE_SEPARATE,
                    label: 'Separate (top and bottom independently)',
                },
            ].forEach((option) => {
                const optionNode = createElement('option', {
                    value: option.value,
                    textContent: option.label,
                });
                spacingModeSelect.append(optionNode);
            });
            spacingModeSelect.value = resolveSpacingModeValue(section);
            spacingModeField.append(spacingModeSelect);
            appendSpacingField(spacingModeField);

            const paddingValue = clampPaddingValue(section.paddingVertical);
            const paddingField = createElement('label', {
                className: 'admin-builder__field',
            });
            paddingField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Vertical padding',
                })
            );
            const paddingRangeWrapper = createElement('div', {
                className: 'admin-builder__range',
            });
            const paddingRangeInput = createElement('input', {
                className: 'admin-builder__range-input',
            });
            paddingRangeInput.type = 'range';
            paddingRangeInput.min = '0';
            paddingRangeInput.max = String(paddingOptions.length - 1);
            paddingRangeInput.step = '1';
            paddingRangeInput.dataset.field = 'section-padding-vertical';
            paddingRangeInput.dataset.options = paddingOptions.join(',');
            const paddingRangeIndex = paddingIndexForValue(paddingValue);
            paddingRangeInput.value = String(paddingRangeIndex);
            paddingRangeInput.setAttribute(
                'aria-valuemin',
                String(paddingOptions[0])
            );
            paddingRangeInput.setAttribute(
                'aria-valuemax',
                String(paddingOptions[paddingOptions.length - 1])
            );
            paddingRangeInput.setAttribute('aria-valuenow', String(paddingValue));
            paddingRangeInput.setAttribute(
                'aria-valuetext',
                `${paddingValue} pixels`
            );
            const paddingRangeValue = createElement('span', {
                className: 'admin-builder__range-value',
                textContent: `${paddingValue}px`,
            });
            paddingRangeValue.dataset.role = 'section-padding-value';
            paddingRangeWrapper.append(paddingRangeInput, paddingRangeValue);
            paddingRangeInput.addEventListener('input', () => {
                const currentValue = paddingValueForIndex(paddingRangeInput.value);
                paddingRangeValue.textContent = `${currentValue}px`;
                paddingRangeInput.setAttribute('aria-valuenow', String(currentValue));
                paddingRangeInput.setAttribute(
                    'aria-valuetext',
                    `${currentValue} pixels`
                );
                if (
                    normaliseSpacingModeValue(spacingModeSelect.value) ===
                    SPACING_MODE_LINKED
                ) {
                    const syncedIndex = paddingIndexForValue(currentValue);
                    paddingTopRangeInput.value = String(syncedIndex);
                    paddingBottomRangeInput.value = String(syncedIndex);
                    paddingTopRangeValue.textContent = `${currentValue}px`;
                    paddingBottomRangeValue.textContent = `${currentValue}px`;
                    paddingTopRangeInput.setAttribute(
                        'aria-valuenow',
                        String(currentValue)
                    );
                    paddingTopRangeInput.setAttribute(
                        'aria-valuetext',
                        `${currentValue} pixels`
                    );
                    paddingBottomRangeInput.setAttribute(
                        'aria-valuenow',
                        String(currentValue)
                    );
                    paddingBottomRangeInput.setAttribute(
                        'aria-valuetext',
                        `${currentValue} pixels`
                    );
                    section.paddingVertical = currentValue;
                    section.paddingTop = currentValue;
                    section.paddingBottom = currentValue;
                }
                scheduleChange();
            });
            paddingField.append(paddingRangeWrapper);

            if (typeof applyPaddingCallback === 'function') {
                const defaultLabel = 'Apply to all page sections';
                const loadingLabel = 'Applying...';

                const bulkActions = createElement('div', {
                    className: 'admin-builder__field-actions',
                });
                const applyAllButton = createElement('button', {
                    className: 'admin-builder__button',
                    type: 'button',
                    textContent: defaultLabel,
                });
                bulkActions.append(applyAllButton);
                paddingField.append(bulkActions);

                paddingField.append(
                    createElement('p', {
                        className: 'admin-builder__hint',
                        textContent:
                            'Updates vertical padding for every section on every page.',
                    })
                );

                applyAllButton.addEventListener('click', async (event) => {
                    event.preventDefault();
                    const targetPadding = clampPaddingValue(
                        Number(section.paddingVertical)
                    );
                    const message =
                        `Apply ${targetPadding}px vertical padding to all sections across every page? ` +
                        'Existing values will be overwritten.';
                    if (!window.confirm(message)) {
                        return;
                    }

                    applyAllButton.disabled = true;
                    applyAllButton.textContent = loadingLabel;

                    try {
                        await applyPaddingCallback(targetPadding);
                        closeModal();
                        scheduleChange();
                    } catch (error) {
                        console.error(
                            'Failed to apply padding to all sections',
                            error
                        );
                    } finally {
                        applyAllButton.disabled = false;
                        applyAllButton.textContent = defaultLabel;
                    }
                });
            }

            appendSpacingField(paddingField);

            const paddingTopValue = clampPaddingValue(
                section.paddingTop ?? section.paddingVertical
            );
            const paddingTopField = createElement('label', {
                className: 'admin-builder__field',
            });
            paddingTopField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Padding top',
                })
            );
            const paddingTopRangeWrapper = createElement('div', {
                className: 'admin-builder__range',
            });
            const paddingTopRangeInput = createElement('input', {
                className: 'admin-builder__range-input',
            });
            paddingTopRangeInput.type = 'range';
            paddingTopRangeInput.min = '0';
            paddingTopRangeInput.max = String(paddingOptions.length - 1);
            paddingTopRangeInput.step = '1';
            paddingTopRangeInput.dataset.field = 'section-padding-top';
            paddingTopRangeInput.dataset.options = paddingOptions.join(',');
            const paddingTopRangeIndex = paddingIndexForValue(paddingTopValue);
            paddingTopRangeInput.value = String(paddingTopRangeIndex);
            paddingTopRangeInput.setAttribute(
                'aria-valuemin',
                String(paddingOptions[0])
            );
            paddingTopRangeInput.setAttribute(
                'aria-valuemax',
                String(paddingOptions[paddingOptions.length - 1])
            );
            paddingTopRangeInput.setAttribute(
                'aria-valuenow',
                String(paddingTopValue)
            );
            paddingTopRangeInput.setAttribute(
                'aria-valuetext',
                `${paddingTopValue} pixels`
            );
            const paddingTopRangeValue = createElement('span', {
                className: 'admin-builder__range-value',
                textContent: `${paddingTopValue}px`,
            });
            paddingTopRangeValue.dataset.role = 'section-padding-top-value';
            paddingTopRangeWrapper.append(paddingTopRangeInput, paddingTopRangeValue);
            paddingTopRangeInput.addEventListener('input', () => {
                const currentValue = paddingValueForIndex(paddingTopRangeInput.value);
                paddingTopRangeValue.textContent = `${currentValue}px`;
                paddingTopRangeInput.setAttribute('aria-valuenow', String(currentValue));
                paddingTopRangeInput.setAttribute(
                    'aria-valuetext',
                    `${currentValue} pixels`
                );
                scheduleChange();
            });
            paddingTopField.append(paddingTopRangeWrapper);
            appendSpacingField(paddingTopField);

            const paddingBottomValue = clampPaddingValue(
                section.paddingBottom ?? section.paddingVertical
            );
            const paddingBottomField = createElement('label', {
                className: 'admin-builder__field',
            });
            paddingBottomField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Padding bottom',
                })
            );
            const paddingBottomRangeWrapper = createElement('div', {
                className: 'admin-builder__range',
            });
            const paddingBottomRangeInput = createElement('input', {
                className: 'admin-builder__range-input',
            });
            paddingBottomRangeInput.type = 'range';
            paddingBottomRangeInput.min = '0';
            paddingBottomRangeInput.max = String(paddingOptions.length - 1);
            paddingBottomRangeInput.step = '1';
            paddingBottomRangeInput.dataset.field = 'section-padding-bottom';
            paddingBottomRangeInput.dataset.options = paddingOptions.join(',');
            const paddingBottomRangeIndex = paddingIndexForValue(paddingBottomValue);
            paddingBottomRangeInput.value = String(paddingBottomRangeIndex);
            paddingBottomRangeInput.setAttribute(
                'aria-valuemin',
                String(paddingOptions[0])
            );
            paddingBottomRangeInput.setAttribute(
                'aria-valuemax',
                String(paddingOptions[paddingOptions.length - 1])
            );
            paddingBottomRangeInput.setAttribute(
                'aria-valuenow',
                String(paddingBottomValue)
            );
            paddingBottomRangeInput.setAttribute(
                'aria-valuetext',
                `${paddingBottomValue} pixels`
            );
            const paddingBottomRangeValue = createElement('span', {
                className: 'admin-builder__range-value',
                textContent: `${paddingBottomValue}px`,
            });
            paddingBottomRangeValue.dataset.role = 'section-padding-bottom-value';
            paddingBottomRangeWrapper.append(
                paddingBottomRangeInput,
                paddingBottomRangeValue
            );
            paddingBottomRangeInput.addEventListener('input', () => {
                const currentValue = paddingValueForIndex(
                    paddingBottomRangeInput.value
                );
                paddingBottomRangeValue.textContent = `${currentValue}px`;
                paddingBottomRangeInput.setAttribute(
                    'aria-valuenow',
                    String(currentValue)
                );
                paddingBottomRangeInput.setAttribute(
                    'aria-valuetext',
                    `${currentValue} pixels`
                );
                scheduleChange();
            });
            paddingBottomField.append(paddingBottomRangeWrapper);
            appendSpacingField(paddingBottomField);

            const marginValue = clampMarginValue(section.marginVertical);
            const marginField = createElement('label', {
                className: 'admin-builder__field',
            });
            marginField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Vertical margin',
                })
            );
            const marginRangeWrapper = createElement('div', {
                className: 'admin-builder__range',
            });
            const marginRangeInput = createElement('input', {
                className: 'admin-builder__range-input',
            });
            marginRangeInput.type = 'range';
            marginRangeInput.min = '0';
            marginRangeInput.max = String(marginOptions.length - 1);
            marginRangeInput.step = '1';
            marginRangeInput.dataset.field = 'section-margin-vertical';
            marginRangeInput.dataset.options = marginOptions.join(',');
            const marginRangeIndex = marginIndexForValue(marginValue);
            marginRangeInput.value = String(marginRangeIndex);
            marginRangeInput.setAttribute(
                'aria-valuemin',
                String(marginOptions[0])
            );
            marginRangeInput.setAttribute(
                'aria-valuemax',
                String(marginOptions[marginOptions.length - 1])
            );
            marginRangeInput.setAttribute('aria-valuenow', String(marginValue));
            marginRangeInput.setAttribute(
                'aria-valuetext',
                `${marginValue} pixels`
            );
            const marginRangeValue = createElement('span', {
                className: 'admin-builder__range-value',
                textContent: `${marginValue}px`,
            });
            marginRangeValue.dataset.role = 'section-margin-value';
            marginRangeWrapper.append(marginRangeInput, marginRangeValue);
            marginRangeInput.addEventListener('input', () => {
                const currentValue = marginValueForIndex(marginRangeInput.value);
                marginRangeValue.textContent = `${currentValue}px`;
                marginRangeInput.setAttribute('aria-valuenow', String(currentValue));
                marginRangeInput.setAttribute(
                    'aria-valuetext',
                    `${currentValue} pixels`
                );
                if (
                    normaliseSpacingModeValue(spacingModeSelect.value) ===
                    SPACING_MODE_LINKED
                ) {
                    const syncedIndex = marginIndexForValue(currentValue);
                    marginTopRangeInput.value = String(syncedIndex);
                    marginBottomRangeInput.value = String(syncedIndex);
                    marginTopRangeValue.textContent = `${currentValue}px`;
                    marginBottomRangeValue.textContent = `${currentValue}px`;
                    marginTopRangeInput.setAttribute(
                        'aria-valuenow',
                        String(currentValue)
                    );
                    marginTopRangeInput.setAttribute(
                        'aria-valuetext',
                        `${currentValue} pixels`
                    );
                    marginBottomRangeInput.setAttribute(
                        'aria-valuenow',
                        String(currentValue)
                    );
                    marginBottomRangeInput.setAttribute(
                        'aria-valuetext',
                        `${currentValue} pixels`
                    );
                    section.marginVertical = currentValue;
                    section.marginTop = currentValue;
                    section.marginBottom = currentValue;
                }
                scheduleChange();
            });
            marginField.append(marginRangeWrapper);
            appendSpacingField(marginField);

            const marginTopValue = clampMarginValue(
                section.marginTop ?? section.marginVertical
            );
            const marginTopField = createElement('label', {
                className: 'admin-builder__field',
            });
            marginTopField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Margin top',
                })
            );
            const marginTopRangeWrapper = createElement('div', {
                className: 'admin-builder__range',
            });
            const marginTopRangeInput = createElement('input', {
                className: 'admin-builder__range-input',
            });
            marginTopRangeInput.type = 'range';
            marginTopRangeInput.min = '0';
            marginTopRangeInput.max = String(marginOptions.length - 1);
            marginTopRangeInput.step = '1';
            marginTopRangeInput.dataset.field = 'section-margin-top';
            marginTopRangeInput.dataset.options = marginOptions.join(',');
            const marginTopRangeIndex = marginIndexForValue(marginTopValue);
            marginTopRangeInput.value = String(marginTopRangeIndex);
            marginTopRangeInput.setAttribute(
                'aria-valuemin',
                String(marginOptions[0])
            );
            marginTopRangeInput.setAttribute(
                'aria-valuemax',
                String(marginOptions[marginOptions.length - 1])
            );
            marginTopRangeInput.setAttribute(
                'aria-valuenow',
                String(marginTopValue)
            );
            marginTopRangeInput.setAttribute(
                'aria-valuetext',
                `${marginTopValue} pixels`
            );
            const marginTopRangeValue = createElement('span', {
                className: 'admin-builder__range-value',
                textContent: `${marginTopValue}px`,
            });
            marginTopRangeValue.dataset.role = 'section-margin-top-value';
            marginTopRangeWrapper.append(marginTopRangeInput, marginTopRangeValue);
            marginTopRangeInput.addEventListener('input', () => {
                const currentValue = marginValueForIndex(marginTopRangeInput.value);
                marginTopRangeValue.textContent = `${currentValue}px`;
                marginTopRangeInput.setAttribute('aria-valuenow', String(currentValue));
                marginTopRangeInput.setAttribute(
                    'aria-valuetext',
                    `${currentValue} pixels`
                );
                scheduleChange();
            });
            marginTopField.append(marginTopRangeWrapper);
            appendSpacingField(marginTopField);

            const marginBottomValue = clampMarginValue(
                section.marginBottom ?? section.marginVertical
            );
            const marginBottomField = createElement('label', {
                className: 'admin-builder__field',
            });
            marginBottomField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Margin bottom',
                })
            );
            const marginBottomRangeWrapper = createElement('div', {
                className: 'admin-builder__range',
            });
            const marginBottomRangeInput = createElement('input', {
                className: 'admin-builder__range-input',
            });
            marginBottomRangeInput.type = 'range';
            marginBottomRangeInput.min = '0';
            marginBottomRangeInput.max = String(marginOptions.length - 1);
            marginBottomRangeInput.step = '1';
            marginBottomRangeInput.dataset.field = 'section-margin-bottom';
            marginBottomRangeInput.dataset.options = marginOptions.join(',');
            const marginBottomRangeIndex = marginIndexForValue(marginBottomValue);
            marginBottomRangeInput.value = String(marginBottomRangeIndex);
            marginBottomRangeInput.setAttribute(
                'aria-valuemin',
                String(marginOptions[0])
            );
            marginBottomRangeInput.setAttribute(
                'aria-valuemax',
                String(marginOptions[marginOptions.length - 1])
            );
            marginBottomRangeInput.setAttribute(
                'aria-valuenow',
                String(marginBottomValue)
            );
            marginBottomRangeInput.setAttribute(
                'aria-valuetext',
                `${marginBottomValue} pixels`
            );
            const marginBottomRangeValue = createElement('span', {
                className: 'admin-builder__range-value',
                textContent: `${marginBottomValue}px`,
            });
            marginBottomRangeValue.dataset.role = 'section-margin-bottom-value';
            marginBottomRangeWrapper.append(
                marginBottomRangeInput,
                marginBottomRangeValue
            );
            marginBottomRangeInput.addEventListener('input', () => {
                const currentValue = marginValueForIndex(
                    marginBottomRangeInput.value
                );
                marginBottomRangeValue.textContent = `${currentValue}px`;
                marginBottomRangeInput.setAttribute(
                    'aria-valuenow',
                    String(currentValue)
                );
                marginBottomRangeInput.setAttribute(
                    'aria-valuetext',
                    `${currentValue} pixels`
                );
                scheduleChange();
            });
            marginBottomField.append(marginBottomRangeWrapper);
            appendSpacingField(marginBottomField);

            const setSpacingFieldVisibility = (fieldNode, visible) => {
                if (!fieldNode) {
                    return;
                }
                fieldNode.hidden = !visible;
                fieldNode.style.display = visible ? '' : 'none';
            };
            const updateSpacingModeVisibility = () => {
                const mode = normaliseSpacingModeValue(spacingModeSelect.value);
                const showLinked = mode === SPACING_MODE_LINKED;
                setSpacingFieldVisibility(paddingField, showLinked);
                setSpacingFieldVisibility(marginField, showLinked);
                setSpacingFieldVisibility(paddingTopField, !showLinked);
                setSpacingFieldVisibility(paddingBottomField, !showLinked);
                setSpacingFieldVisibility(marginTopField, !showLinked);
                setSpacingFieldVisibility(marginBottomField, !showLinked);
            };
            spacingModeSelect.addEventListener('change', () => {
                const mode = normaliseSpacingModeValue(spacingModeSelect.value);
                spacingModeSelect.value = mode;
                section.settings =
                    section.settings && typeof section.settings === 'object'
                        ? section.settings
                        : {};
                section.settings.spacing_mode = mode;
                if (section.settings.spacingMode !== undefined) {
                    delete section.settings.spacingMode;
                }

                if (mode === SPACING_MODE_LINKED) {
                    const linkedPaddingValue = clampPaddingValue(
                        (
                            paddingValueForIndex(paddingTopRangeInput.value) +
                            paddingValueForIndex(paddingBottomRangeInput.value)
                        ) / 2
                    );
                    const linkedPaddingIndex = paddingIndexForValue(linkedPaddingValue);
                    paddingRangeInput.value = String(linkedPaddingIndex);
                    paddingTopRangeInput.value = String(linkedPaddingIndex);
                    paddingBottomRangeInput.value = String(linkedPaddingIndex);
                    paddingRangeValue.textContent = `${linkedPaddingValue}px`;
                    paddingTopRangeValue.textContent = `${linkedPaddingValue}px`;
                    paddingBottomRangeValue.textContent = `${linkedPaddingValue}px`;
                    paddingRangeInput.setAttribute(
                        'aria-valuenow',
                        String(linkedPaddingValue)
                    );
                    paddingRangeInput.setAttribute(
                        'aria-valuetext',
                        `${linkedPaddingValue} pixels`
                    );
                    paddingTopRangeInput.setAttribute(
                        'aria-valuenow',
                        String(linkedPaddingValue)
                    );
                    paddingTopRangeInput.setAttribute(
                        'aria-valuetext',
                        `${linkedPaddingValue} pixels`
                    );
                    paddingBottomRangeInput.setAttribute(
                        'aria-valuenow',
                        String(linkedPaddingValue)
                    );
                    paddingBottomRangeInput.setAttribute(
                        'aria-valuetext',
                        `${linkedPaddingValue} pixels`
                    );
                    section.paddingVertical = linkedPaddingValue;
                    section.paddingTop = linkedPaddingValue;
                    section.paddingBottom = linkedPaddingValue;

                    const linkedMarginValue = clampMarginValue(
                        (
                            marginValueForIndex(marginTopRangeInput.value) +
                            marginValueForIndex(marginBottomRangeInput.value)
                        ) / 2
                    );
                    const linkedMarginIndex = marginIndexForValue(linkedMarginValue);
                    marginRangeInput.value = String(linkedMarginIndex);
                    marginTopRangeInput.value = String(linkedMarginIndex);
                    marginBottomRangeInput.value = String(linkedMarginIndex);
                    marginRangeValue.textContent = `${linkedMarginValue}px`;
                    marginTopRangeValue.textContent = `${linkedMarginValue}px`;
                    marginBottomRangeValue.textContent = `${linkedMarginValue}px`;
                    marginRangeInput.setAttribute(
                        'aria-valuenow',
                        String(linkedMarginValue)
                    );
                    marginRangeInput.setAttribute(
                        'aria-valuetext',
                        `${linkedMarginValue} pixels`
                    );
                    marginTopRangeInput.setAttribute(
                        'aria-valuenow',
                        String(linkedMarginValue)
                    );
                    marginTopRangeInput.setAttribute(
                        'aria-valuetext',
                        `${linkedMarginValue} pixels`
                    );
                    marginBottomRangeInput.setAttribute(
                        'aria-valuenow',
                        String(linkedMarginValue)
                    );
                    marginBottomRangeInput.setAttribute(
                        'aria-valuetext',
                        `${linkedMarginValue} pixels`
                    );
                    section.marginVertical = linkedMarginValue;
                    section.marginTop = linkedMarginValue;
                    section.marginBottom = linkedMarginValue;
                }

                updateSpacingModeVisibility();
                scheduleChange();
            });
            updateSpacingModeVisibility();

            const updateDisplayModeVisibility = () => {
                const showAllValue = (() => {
                    const raw =
                        showAllCheckbox && typeof showAllCheckbox.checked === 'boolean'
                            ? showAllCheckbox.checked
                            : section.settings?.show_all_button;
                    if (raw === true) return true;
                    if (raw === false) return false;
                    const normalized =
                        typeof raw === 'string' ? raw.trim().toLowerCase() : raw;
                    return (
                        normalized === 'true' ||
                        normalized === '1' ||
                        normalized === 1 ||
                        normalized === 'yes' ||
                        normalized === 'on'
                    );
                })();
                const mode = normaliseString(
                    section.settings?.display_mode ??
                        displayModeSelect?.value ??
                        displayModeDefault
                ).toLowerCase();

                conditionalFields.forEach(({ node, modes, requireShowAll }) => {
                    if (!node) {
                        return;
                    }
                    let visible = true;
                    if (modes && modes.length && !modes.includes(mode)) {
                        visible = false;
                    }
                    if (visible && requireShowAll && !showAllValue) {
                        visible = false;
                    }
                    if (visible && requireShowAll) {
                        node.hidden = false;
                        node.style.display = showAllValue ? '' : 'none';
                    } else {
                        node.hidden = !visible;
                        node.style.display = visible ? '' : 'none';
                    }
                });

                if (limitLabelSpan) {
                    if (mode === 'paginated' || mode === 'selected') {
                        limitLabelSpan.textContent = perPageLimitLabel;
                    } else {
                        limitLabelSpan.textContent = defaultLimitLabel;
                    }
                }
            };

            if (displayModeSelect) {
                displayModeSelect.addEventListener('change', () => {
                    section.settings = section.settings || {};
                    section.settings.display_mode = normaliseString(
                        displayModeSelect.value
                    ).toLowerCase();
                    updateDisplayModeVisibility();
                    scheduleChange();
                });
            }

            if (showAllCheckbox) {
                showAllCheckbox.addEventListener('change', () => {
                    section.settings = section.settings || {};
                    section.settings.show_all_button = showAllCheckbox.checked;
                    updateDisplayModeVisibility();
                    scheduleChange();
                });
            }

            updateDisplayModeVisibility();

            if (renderInline) {
                inlineSettingsRoot?.append(body);
                return inlineSettingsRoot;
            }

            const footer = createElement('div', {
                className: 'admin-builder__settings-footer',
            });
            const doneButton = createElement('button', {
                className: 'admin-builder__button',
                type: 'button',
                textContent: 'Done',
            });
            footer.append(doneButton);

            dialog.append(body, footer);
            overlay.append(dialog);
            sectionItem.append(overlay);

            const previousFocus = document.activeElement;

            const handleKeyDown = (event) => {
                if (event.key === 'Escape') {
                    event.preventDefault();
                    closeModal();
                }
            };

            closeModal = () => {
                document.removeEventListener('keydown', handleKeyDown);
                overlay.remove();
                if (typeof onClose === 'function') {
                    onClose();
                }
                if (previousFocus && typeof previousFocus.focus === 'function') {
                    previousFocus.focus();
                }
            };

            doneButton.addEventListener('click', (event) => {
                event.preventDefault();
                closeModal();
            });
            overlay.addEventListener('click', (event) => {
                if (event.target === overlay) {
                    closeModal();
                }
            });

            document.addEventListener('keydown', handleKeyDown);

            const focusInitialField = () => {
                const focusTarget = dialog.querySelector('[data-field]');
                if (focusTarget && typeof focusTarget.focus === 'function') {
                    focusTarget.focus();
                    return;
                }
                if (typeof dialog.focus === 'function') {
                    dialog.focus();
                }
            };

            if (typeof window.requestAnimationFrame === 'function') {
                window.requestAnimationFrame(focusInitialField);
            } else {
                focusInitialField();
            }

            return closeModal;
        };

        const render = (sections) => {
            listElement.innerHTML = '';

            if (!sections.length) {
                if (emptyState) {
                    emptyState.hidden = false;
                    listElement.append(emptyState);
                }
                return;
            }

            if (emptyState) {
                emptyState.hidden = true;
            }

            const sectionTypeOrder = Array.isArray(orderedSectionTypes)
                ? orderedSectionTypes
                : Object.keys(sectionDefinitions || {});

            const totalSections = sections.length;

            sections.forEach((section, index) => {
            const sectionItem = createElement('li', {
                className: 'admin-builder__section',
            });
            sectionItem.dataset.sectionClient = section.clientId;
            sectionItem.dataset.sectionIndex = String(index);
            sectionItem.dataset.sectionType = section.type;
            const sectionBackgroundGroupToken = normaliseBackgroundGroupToken(
                section.settings?.background_group ??
                    section.settings?.backgroundGroup
            );
            sectionItem.dataset.sectionBackgroundGroup =
                sectionBackgroundGroupToken;
            const sectionBackgroundStyleToken = normaliseBackgroundStyleToken(
                section.settings?.background_style ??
                    section.settings?.backgroundStyle
            );
            sectionItem.dataset.sectionBackgroundStyle =
                sectionBackgroundStyleToken;
            const isDisabled = Boolean(section.disabled);
            sectionItem.dataset.sectionDisabled = String(isDisabled);
            if (isDisabled) {
                sectionItem.classList.add('admin-builder__section--disabled');
            }
            if (section.id) {
                sectionItem.dataset.sectionId = section.id;
            }

            const sectionDefinition = sectionDefinitions?.[section.type] || {};
                const allowElements = sectionDefinition.supportsElements !== false;
                const isFirstSection = index === 0;
                const isLastSection = index === totalSections - 1;
                const isCollapsed =
                    typeof isSectionCollapsed === 'function'
                        ? Boolean(isSectionCollapsed(section.clientId))
                        : false;
                sectionItem.dataset.sectionCollapsed = String(isCollapsed);
                if (isCollapsed) {
                    sectionItem.classList.add('admin-builder__section--collapsed');
                }

                const sectionHeader = createElement('div', {
                    className: 'admin-builder__section-header',
                });
                const sectionHeading = createElement('div', {
                    className: 'admin-builder__section-heading',
                });
                const sectionTitle = createElement('h3', {
                    className: 'admin-builder__section-title',
                    textContent: `Section ${index + 1}`,
                });
                sectionHeading.append(sectionTitle);

                if (isDisabled) {
                    const statusBadge = createElement('span', {
                        className: 'admin-builder__section-status admin-builder__section-status--disabled',
                        textContent: 'Disabled',
                    });
                    sectionHeading.append(statusBadge);
                }

                const sectionActions = createElement('div', {
                    className: 'admin-builder__section-header-actions',
                });
                const sectionActionsMenu = createElement('details', {
                    className: 'admin-builder__section-actions-menu',
                });
                const sectionActionsToggle = createElement('summary', {
                    className:
                        'admin-builder__button admin-builder__button--ghost admin-builder__section-actions-toggle',
                    attributes: {
                        'aria-label': 'Section actions',
                        title: 'Section actions',
                    },
                });
                const sectionActionsList = createElement('div', {
                    className: 'admin-builder__section-actions-list',
                });
                const moveUpButton = createElement('button', {
                    className: 'admin-builder__button',
                    textContent: 'Move up',
                });
                moveUpButton.type = 'button';
                moveUpButton.dataset.action = 'section-move';
                moveUpButton.dataset.direction = 'up';
                moveUpButton.dataset.role = 'section-move-up';
                moveUpButton.disabled = isFirstSection;
                const moveDownButton = createElement('button', {
                    className: 'admin-builder__button',
                    textContent: 'Move down',
                });
                moveDownButton.type = 'button';
                moveDownButton.dataset.action = 'section-move';
                moveDownButton.dataset.direction = 'down';
                moveDownButton.dataset.role = 'section-move-down';
                moveDownButton.disabled = isLastSection;
                const duplicateButton = createElement('button', {
                    className: 'admin-builder__button admin-builder__button--ghost',
                    textContent: 'Duplicate section',
                });
                duplicateButton.type = 'button';
                duplicateButton.dataset.action = 'section-duplicate';
                const collapseButton = createElement('button', {
                    className: 'admin-builder__button admin-builder__button--ghost',
                    textContent: isCollapsed ? 'Expand' : 'Collapse',
                });
                collapseButton.type = 'button';
                collapseButton.dataset.action = 'section-collapse';
                collapseButton.dataset.state = isCollapsed ? 'expand' : 'collapse';
                const toggleButton = createElement('button', {
                    className: 'admin-builder__button admin-builder__button--ghost',
                    textContent: isDisabled ? 'Enable section' : 'Disable section',
                });
                toggleButton.type = 'button';
                toggleButton.dataset.action = 'section-toggle';
                toggleButton.dataset.state = isDisabled ? 'enable' : 'disable';
                const removeButton = createElement('button', {
                    className: 'admin-builder__remove',
                    textContent: 'Remove section',
                });
                removeButton.type = 'button';
                removeButton.dataset.action = 'section-remove';
                sectionActionsList.append(
                    moveUpButton,
                    moveDownButton,
                    duplicateButton,
                    collapseButton,
                    toggleButton,
                    removeButton
                );
                sectionActionsMenu.append(
                    sectionActionsToggle,
                    sectionActionsList
                );
                sectionActions.append(sectionActionsMenu);
                sectionHeader.append(sectionHeading, sectionActions);
                sectionItem.append(sectionHeader);

                const visibleElementCount = Array.isArray(section.elements)
                    ? section.elements.filter((element) =>
                          isElementAllowed(section.type, element?.type)
                      ).length
                    : 0;
                const collapsedSummaryParts = [];
                const sectionTypeLabel = normaliseString(sectionDefinition.label).trim();
                collapsedSummaryParts.push(sectionTypeLabel || section.type || 'Section');
                const sectionTitleText = normaliseString(section.title).trim();
                if (sectionTitleText) {
                    collapsedSummaryParts.push(sectionTitleText);
                }
                collapsedSummaryParts.push(
                    visibleElementCount === 1 ? '1 block' : `${visibleElementCount} blocks`
                );
                const collapsedSummary = createElement('p', {
                    className: 'admin-builder__section-collapsed-summary',
                    textContent: collapsedSummaryParts.join(' | '),
                });
                collapsedSummary.hidden = !isCollapsed;
                sectionItem.append(collapsedSummary);

                const sectionBody = createElement('div', {
                    className: 'admin-builder__section-body',
                });
                sectionBody.hidden = isCollapsed;
                const tabsIdSuffix = section.id || section.clientId || randomId();
                const contentTabId = `admin-builder-section-tab-content-${tabsIdSuffix}`;
                const settingsTabId = `admin-builder-section-tab-settings-${tabsIdSuffix}`;
                const contentPanelId = `admin-builder-section-panel-content-${tabsIdSuffix}`;
                const settingsPanelId = `admin-builder-section-panel-settings-${tabsIdSuffix}`;
                const tabList = createElement('div', {
                    className: 'admin-builder__section-tabs',
                    attributes: {
                        role: 'tablist',
                        'aria-label': `Section ${index + 1} editor tabs`,
                    },
                });
                const contentTabButton = createElement('button', {
                    className: 'admin-builder__section-tab',
                    type: 'button',
                    textContent: 'Content',
                    attributes: {
                        id: contentTabId,
                        role: 'tab',
                        'aria-selected': 'true',
                        'aria-controls': contentPanelId,
                    },
                });
                const settingsTabButton = createElement('button', {
                    className: 'admin-builder__section-tab',
                    type: 'button',
                    textContent: 'Settings',
                    attributes: {
                        id: settingsTabId,
                        role: 'tab',
                        'aria-selected': 'false',
                        'aria-controls': settingsPanelId,
                    },
                });
                tabList.append(contentTabButton, settingsTabButton);

                const contentPanel = createElement('div', {
                    className: 'admin-builder__section-tab-panel',
                    attributes: {
                        id: contentPanelId,
                        role: 'tabpanel',
                        'aria-labelledby': contentTabId,
                    },
                });
                const settingsPanel = createElement('div', {
                    className: 'admin-builder__section-tab-panel',
                    attributes: {
                        id: settingsPanelId,
                        role: 'tabpanel',
                        'aria-labelledby': settingsTabId,
                    },
                });
                settingsPanel.hidden = true;

                const setActiveTab = (tabName) => {
                    const isSettings = tabName === 'settings';
                    contentTabButton.setAttribute(
                        'aria-selected',
                        isSettings ? 'false' : 'true'
                    );
                    settingsTabButton.setAttribute(
                        'aria-selected',
                        isSettings ? 'true' : 'false'
                    );
                    contentPanel.hidden = isSettings;
                    settingsPanel.hidden = !isSettings;
                };
                contentTabButton.addEventListener('click', () =>
                    setActiveTab('content')
                );
                settingsTabButton.addEventListener('click', () =>
                    setActiveTab('settings')
                );
                sectionBody.append(tabList, contentPanel, settingsPanel);

                const typeField = createElement('div', {
                    className: 'admin-builder__field admin-builder__field--type',
                });
                const typeSelectId = `admin-builder-section-type-${section.id || section.clientId}`;
                const typeLabel = createElement('label', {
                    className: 'admin-builder__label',
                    textContent: 'Section type',
                });
                typeLabel.htmlFor = typeSelectId;
                typeField.append(typeLabel);
                const typeSelect = createElement('select', {
                    className: 'admin-builder__input admin-builder__input--hidden',
                    attributes: {
                        id: typeSelectId,
                        'aria-hidden': 'true',
                    },
                    dataset: {
                        field: 'section-type',
                    },
                    tabIndex: -1,
                });
                sectionTypeOrder.forEach((type) => {
                    const definition = sectionDefinitions?.[type] || {};
                    const option = createElement('option', {
                        textContent: definition.label || type,
                    });
                    option.value = type;
                    if (type === section.type) {
                        option.selected = true;
                    }
                    typeSelect.append(option);
                });
                typeField.append(typeSelect);
                const variationInput = createElement('input', {
                    type: 'hidden',
                    attributes: {
                        'aria-hidden': 'true',
                    },
                    dataset: {
                        field: 'section-variation',
                    },
                    tabIndex: -1,
                });
                typeField.append(variationInput);
                const metaStack = createElement('div', {
                    className: 'admin-builder__meta-stack',
                });
                const typeMeta = createElement('div', {
                    className: 'admin-builder__meta-group',
                });
                const typeControl = createElement('div', {
                    className: 'admin-builder__meta-row',
                });
                const typeSummary = createElement('div', {
                    className: 'admin-builder__meta-summary',
                });
                const typeSummaryLabel = createElement('span', {
                    className: 'admin-builder__meta-summary-label',
                });
                const typeSummaryCode = createElement('code', {
                    className: 'admin-builder__meta-summary-code',
                });
                typeSummary.append(typeSummaryLabel, typeSummaryCode);
                const changeTypeButton = createElement('button', {
                    className:
                        'admin-builder__button admin-builder__button--ghost admin-builder__type-button',
                    type: 'button',
                    textContent: 'Change type',
                });
                typeControl.append(typeSummary, changeTypeButton);
                typeMeta.append(typeControl);
                const typeHint = createElement('span', {
                    className: 'admin-builder__hint admin-builder__meta-hint',
                });
                typeHint.hidden = true;
                typeMeta.append(typeHint);
                metaStack.append(typeMeta);

                const variationMeta = createElement('div', {
                    className: 'admin-builder__meta-group',
                });
                const variationControl = createElement('div', {
                    className: 'admin-builder__meta-row',
                });
                const variationSummary = createElement('div', {
                    className: 'admin-builder__meta-summary',
                });
                const variationSummaryLabel = createElement('span', {
                    className: 'admin-builder__meta-summary-label',
                });
                const variationSummaryCode = createElement('code', {
                    className: 'admin-builder__meta-summary-code',
                });
                variationSummary.append(
                    variationSummaryLabel,
                    variationSummaryCode
                );
                const changeVariationButton = createElement('button', {
                    className:
                        'admin-builder__button admin-builder__button--ghost admin-builder__type-button',
                    type: 'button',
                    textContent: 'Change variation',
                });
                variationControl.append(variationSummary, changeVariationButton);
                variationMeta.append(variationControl);
                const variationHint = createElement('span', {
                    className: 'admin-builder__hint admin-builder__meta-hint',
                });
                variationHint.hidden = true;
                variationMeta.append(variationHint);
                variationMeta.hidden = true;
                metaStack.append(variationMeta);
                typeField.append(metaStack);

                const resolveSectionVariation = (type, value) => {
                    const definition = sectionDefinitions?.[type] || {};
                    return normaliseVariationValue(definition, value);
                };
                const updateVariationMetadata = (typeValue = typeSelect.value) => {
                    const definition = sectionDefinitions?.[typeValue] || {};
                    const variationOptions = collectVariationOptions(definition);
                    if (!variationOptions.length) {
                        variationInput.value = '';
                        changeVariationButton.hidden = true;
                        variationMeta.hidden = true;
                        variationSummaryLabel.textContent = '';
                        variationSummaryCode.textContent = '';
                        variationHint.textContent = '';
                        variationHint.hidden = true;
                        return;
                    }

                    const currentVariation = resolveSectionVariation(
                        typeValue,
                        section.variation || variationInput.value
                    );
                    section.variation = currentVariation;
                    variationInput.value = currentVariation;

                    const hasVariationChoices = variationOptions.length > 1;
                    if (!hasVariationChoices) {
                        changeVariationButton.hidden = true;
                        variationMeta.hidden = true;
                        return;
                    }

                    changeVariationButton.hidden = false;
                    const selectedVariation =
                        variationOptions.find(
                            (variation) => variation.value === currentVariation
                        ) ||
                        variationOptions.find((variation) => variation.isDefault) ||
                        variationOptions[0];
                    variationSummaryLabel.textContent =
                        selectedVariation?.label || currentVariation;
                    variationSummaryCode.textContent =
                        selectedVariation?.id || selectedVariation?.value || currentVariation;
                    const description = normaliseString(
                        selectedVariation?.description
                    ).trim();
                    variationHint.textContent = description;
                    variationHint.hidden = !description;
                    variationMeta.hidden = false;
                };

                const updateTypeMetadata = (nextType) => {
                    const definition = sectionDefinitions?.[nextType] || {};
                    const safeType = typeof nextType === 'string' ? nextType : '';
                    const labelText = normaliseString(definition.label).trim();
                    const typeValue = safeType || 'unknown';
                    typeSummaryLabel.textContent = labelText || typeValue;
                    typeSummaryCode.textContent = typeValue;
                    const description = normaliseString(definition.description).trim();
                    if (description) {
                        typeHint.textContent = description;
                        typeHint.hidden = false;
                    } else {
                        typeHint.textContent = '';
                        typeHint.hidden = true;
                    }
                    updateVariationMetadata(typeValue);
                };

                updateTypeMetadata(section.type);

                typeSelect.addEventListener('change', () => {
                    updateTypeMetadata(typeSelect.value);
                });

                const openTypePicker = () => {
                    const typePickerModule = window.AdminSectionTypePicker;
                    if (typePickerModule?.open) {
                        typePickerModule.open({
                            orderedSectionTypes: sectionTypeOrder,
                            sectionDefinitions,
                            activeType: typeSelect.value,
                            onSelect: (nextType) => {
                                if (!nextType || nextType === typeSelect.value) {
                                    return;
                                }
                                typeSelect.value = nextType;
                                updateTypeMetadata(nextType);
                                typeSelect.dispatchEvent(new Event('change', { bubbles: true }));
                            },
                        });
                        return;
                    }
                    if (typeof typeSelect.focus === 'function') {
                        typeSelect.focus();
                    }
                };

                changeTypeButton.addEventListener('click', (event) => {
                    event.preventDefault();
                    openTypePicker();
                });

                const openVariationPicker = () => {
                    const typePickerModule = window.AdminSectionTypePicker;
                    const activeType = normaliseString(typeSelect.value).trim().toLowerCase();
                    if (!activeType) {
                        return;
                    }
                    const definition = sectionDefinitions?.[activeType] || {};
                    const variationOptions = collectVariationOptions(definition);
                    if (variationOptions.length <= 1) {
                        return;
                    }
                    if (typeof typePickerModule?.openVariations === 'function') {
                        typePickerModule.openVariations({
                            sectionType: activeType,
                            sectionDefinitions,
                            activeVariation: resolveSectionVariation(
                                activeType,
                                section.variation || variationInput.value
                            ),
                            onSelect: (nextVariation) => {
                                const resolvedVariation = resolveSectionVariation(
                                    activeType,
                                    nextVariation
                                );
                                section.variation = resolvedVariation;
                                variationInput.value = resolvedVariation;
                                updateVariationMetadata(activeType);
                                variationInput.dispatchEvent(
                                    new Event('change', { bubbles: true })
                                );
                            },
                        });
                        return;
                    }
                    if (typeof variationInput.focus === 'function') {
                        variationInput.focus();
                    }
                };

                changeVariationButton.addEventListener('click', (event) => {
                    event.preventDefault();
                    openVariationPicker();
                });

                contentPanel.append(typeField);

                const titleField = createElement('label', {
                    className: 'admin-builder__field',
                });
                titleField.append(
                    createElement('span', {
                        className: 'admin-builder__label',
                        textContent: 'Section title',
                    })
                );
                const titleInput = createElement('input', {
                    className: 'admin-builder__input',
                });
                titleInput.type = 'text';
                titleInput.placeholder = 'e.g. Getting started';
                titleInput.value = section.title;
                titleInput.dataset.field = 'section-title';
                titleField.append(titleInput);
                contentPanel.append(titleField);

                const descriptionField = createElement('label', {
                    className: 'admin-builder__field',
                });
                descriptionField.append(
                    createElement('span', {
                        className: 'admin-builder__label',
                        textContent: 'Section description',
                    })
                );
                const descriptionInput = createElement('textarea', {
                    className: 'admin-builder__input',
                    attributes: {
                        rows: '2',
                    },
                });
                descriptionInput.placeholder = 'Add a short intro for this section';
                descriptionInput.value = section.description || '';
                descriptionInput.dataset.field = 'section-description';
                descriptionField.append(descriptionInput);
                contentPanel.append(descriptionField);

                const settingsSummary = createElement('p', {
                    className:
                        'admin-builder__settings-summary admin-builder__settings-summary--inline',
                });
                const updateSettingsSummary = () => {
                    settingsSummary.textContent = formatSettingsSummary(
                        section,
                        sectionDefinition
                    );
                };
                updateSettingsSummary();
                settingsPanel.append(settingsSummary);
                const inlineSettings = createSectionSettingsModal({
                    sectionItem,
                    section,
                    sectionDefinition,
                    onChange: updateSettingsSummary,
                    applyPaddingToAllSections,
                    applyBackgroundGroupToSections,
                    applyBackgroundStyleToGroup,
                    renderInline: true,
                });
                if (inlineSettings) {
                    settingsPanel.append(inlineSettings);
                }

                const elementsContainer = createElement('div', {
                    className: 'admin-builder__section-elements',
                });

                if (!allowElements) {
                    elementsContainer.append(
                        createElement('p', {
                            className: 'admin-builder__element-empty',
                            textContent:
                                'This section type does not support additional content blocks.',
                        })
                    );
                } else if (!section.elements.length) {
                    elementsContainer.append(
                        createElement('p', {
                            className: 'admin-builder__element-empty',
                            textContent:
                                'No content blocks yet. Add one below.',
                        })
                    );
                } else {
                    const filteredElements = section.elements.filter((element) =>
                        isElementAllowed(section.type, element.type)
                    );
                    let renderedIndex = 0;
                    filteredElements.forEach((element, elementIndex) => {
                        renderedIndex += 1;
                        const elementNode = createElement('div', {
                            className: 'admin-builder__element',
                        });
                        elementNode.dataset.elementClient = element.clientId;
                        elementNode.dataset.elementType = element.type;
                        elementNode.dataset.elementIndex = String(renderedIndex - 1);

                        const elementHeader = createElement('div', {
                            className: 'admin-builder__element-header',
                        });
                        const elementTitle = createElement('span', {
                            className: 'admin-builder__element-title',
                            textContent: `${elementLabel(definitions, element.type)} ${renderedIndex}`,
                        });
                        const elementActions = createElement('div', {
                            className: 'admin-builder__element-actions',
                        });
                        const moveUpButton = createElement('button', {
                            className: 'admin-builder__element-move',
                            textContent: 'Move up',
                        });
                        moveUpButton.type = 'button';
                        moveUpButton.dataset.action = 'element-move';
                        moveUpButton.dataset.direction = 'up';
                        moveUpButton.dataset.role = 'element-move-up';
                        moveUpButton.disabled = renderedIndex <= 1;
                        const moveDownButton = createElement('button', {
                            className: 'admin-builder__element-move',
                            textContent: 'Move down',
                        });
                        moveDownButton.type = 'button';
                        moveDownButton.dataset.action = 'element-move';
                        moveDownButton.dataset.direction = 'down';
                        moveDownButton.dataset.role = 'element-move-down';
                        moveDownButton.disabled = renderedIndex >= filteredElements.length;
                        const removeElementButton = createElement('button', {
                            className: 'admin-builder__element-remove',
                            textContent: 'Remove',
                        });
                        removeElementButton.type = 'button';
                        removeElementButton.dataset.action = 'element-remove';
                        elementActions.append(moveUpButton, moveDownButton, removeElementButton);
                        elementHeader.append(elementTitle, elementActions);
                        elementNode.append(elementHeader);

                        renderElementEditor(definitions, elementNode, element, {
                            section,
                            sectionDefinition,
                        });

                        elementsContainer.append(elementNode);
                    });
                    if (renderedIndex === 0) {
                        elementsContainer.append(
                            createElement('p', {
                                className: 'admin-builder__element-empty',
                                textContent:
                                    'No content blocks yet. Add one below.',
                            })
                        );
                    }
                }

                contentPanel.append(elementsContainer);

                if (allowElements) {
                    const sectionActions = createElement('div', {
                        className: 'admin-builder__section-actions',
                    });

                    orderedTypes.forEach((type) => {
                        if (!isElementAllowed(section.type, type)) {
                            return;
                        }
                        const elementDefinition = definitions[type];
                        if (!elementDefinition || !elementDefinition.addLabel) {
                            return;
                        }
                        const button = createElement('button', {
                            className:
                                'admin-builder__button admin-builder__button--ghost',
                            textContent: elementDefinition.addLabel,
                        });
                        button.type = 'button';
                        button.dataset.action = 'element-add';
                        button.dataset.elementType = type;
                        sectionActions.append(button);
                    });

                    contentPanel.append(sectionActions);
                }

                sectionItem.append(sectionBody);
                listElement.append(sectionItem);
            });
        };

        return {
            render,
            focusField,
        };
    };

    window.AdminSectionView = {
        createView,
    };
})();

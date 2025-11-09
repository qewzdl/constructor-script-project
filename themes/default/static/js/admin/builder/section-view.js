(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    const sectionRegistry = window.AdminSectionRegistry;
    if (!utils || !registry || !sectionRegistry) {
        return;
    }

    const { createElement, normaliseString, randomId } = utils;

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
        const closeButton = createElement('button', {
            className: 'admin-type-picker__close',
            type: 'button',
            textContent: 'Close',
        });
        closeButton.setAttribute('aria-label', 'Close section type picker');
        header.append(titleNode, closeButton);

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

        closeButton.addEventListener('click', (event) => {
            event.preventDefault();
            closeModal();
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

    window.AdminSectionTypePicker = window.AdminSectionTypePicker || {};
    window.AdminSectionTypePicker.open = openSectionTypePicker;

    const paddingOptions = [0, 4, 8, 16, 32, 64, 128];
    const marginOptions = [0, 4, 8, 16, 32, 64, 128];

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

    const elementLabel = (definitions, type) => {
        const definition = definitions[type];
        if (definition && definition.label) {
            return definition.label;
        }
        return type || 'Block';
    };

    const renderElementEditor = (definitions, elementNode, element) => {
        const definition = definitions[element.type];
        if (definition && typeof definition.renderEditor === 'function') {
            definition.renderEditor(elementNode, element);
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
    }) => {
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

                const sectionDefinition = sectionDefinitions?.[section.type] || {};
                const allowElements = sectionDefinition.supportsElements !== false;
                const supportsHeaderImage =
                    sectionDefinition.supportsHeaderImage === true;
                const isFirstSection = index === 0;
                const isLastSection = index === totalSections - 1;

                const sectionHeader = createElement('div', {
                    className: 'admin-builder__section-header',
                });
                const sectionTitle = createElement('h3', {
                    className: 'admin-builder__section-title',
                    textContent: `Section ${index + 1}`,
                });
                const sectionActions = createElement('div', {
                    className: 'admin-builder__section-actions',
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
                const removeButton = createElement('button', {
                    className: 'admin-builder__remove',
                    textContent: 'Remove section',
                });
                removeButton.type = 'button';
                removeButton.dataset.action = 'section-remove';
                sectionActions.append(moveUpButton, moveDownButton, removeButton);
                sectionHeader.append(sectionTitle, sectionActions);
                sectionItem.append(sectionHeader);

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
                const typeMeta = createElement('div', {
                    className: 'admin-builder__type',
                });
                const typeControl = createElement('div', {
                    className: 'admin-builder__type-control',
                });
                const typeSummary = createElement('div', {
                    className: 'admin-builder__type-summary',
                });
                const typeSummaryLabel = createElement('span', {
                    className: 'admin-builder__type-summary-label',
                });
                const typeSummaryCode = createElement('code', {
                    className: 'admin-builder__type-summary-code',
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
                    className: 'admin-builder__hint',
                });
                typeHint.hidden = true;
                typeMeta.append(typeHint);
                typeField.append(typeMeta);

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

                sectionItem.append(typeField);

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
                sectionItem.append(titleField);

                if (supportsHeaderImage) {
                    const imageField = createElement('label', {
                        className: 'admin-builder__field',
                    });
                    imageField.append(
                        createElement('span', {
                            className: 'admin-builder__label',
                            textContent: 'Optional header image URL',
                        })
                    );
                    const imageInput = createElement('input', {
                        className: 'admin-builder__input',
                    });
                    imageInput.type = 'url';
                    imageInput.placeholder = 'https://example.com/cover.jpg';
                    imageInput.value = section.image;
                    imageInput.dataset.field = 'section-image';
                    const imageInputId = `admin-builder-section-image-${section.id}`;
                    imageInput.id = imageInputId;
                    imageField.append(imageInput);

                    if (section.type === 'hero') {
                        const imageActions = createElement('div', {
                            className: 'admin-builder__field-actions',
                        });
                        const browseButton = createElement('button', {
                            className: 'admin-builder__media-button',
                            textContent: 'Browse uploads',
                            type: 'button',
                            dataset: {
                                action: 'open-media-library',
                                mediaTarget: `#${imageInputId}`,
                                mediaAllowedTypes: 'image',
                            },
                        });
                        imageActions.append(browseButton);
                        imageField.append(imageActions);
                    }

                    sectionItem.append(imageField);
                }

                if (section.type === 'grid') {
                    const styleField = createElement('label', {
                        className:
                            'admin-builder__field admin-builder__field--checkbox',
                    });
                    const styleInput = createElement('input', {
                        className: 'admin-builder__checkbox checkbox__input',
                    });
                    styleInput.type = 'checkbox';
                    styleInput.dataset.field = 'section-grid-style';
                    styleInput.checked = section.styleGridItems !== false;
                    const styleLabel = createElement('span', {
                        className: 'admin-builder__label',
                        textContent:
                            'Apply border, background, and padding to grid items',
                    });
                    styleField.append(styleInput, styleLabel);
                    sectionItem.append(styleField);
                }

                const limitDefinition = sectionDefinition.settings?.limit;
                if (limitDefinition) {
                    const limitField = createElement('label', {
                        className: 'admin-builder__field',
                    });
                    limitField.append(
                        createElement('span', {
                            className: 'admin-builder__label',
                            textContent:
                                limitDefinition.label ||
                                'Number of posts to display',
                        })
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
                    limitField.append(limitInput);
                    sectionItem.append(limitField);
                }

                const paddingValue = clampPaddingValue(section.paddingVertical);
                const marginValue = clampMarginValue(section.marginVertical);
                const paddingField = createElement('label', {
                    className: 'admin-builder__field',
                });
                paddingField.append(
                    createElement('span', {
                        className: 'admin-builder__label',
                        textContent: 'Vertical padding',
                    })
                );
                const rangeWrapper = createElement('div', {
                    className: 'admin-builder__range',
                });
                const rangeInput = createElement('input', {
                    className: 'admin-builder__range-input',
                });
                rangeInput.type = 'range';
                rangeInput.min = '0';
                rangeInput.max = String(paddingOptions.length - 1);
                rangeInput.step = '1';
                rangeInput.dataset.field = 'section-padding-vertical';
                rangeInput.dataset.options = paddingOptions.join(',');
                const rangeIndex = paddingIndexForValue(paddingValue);
                rangeInput.value = String(rangeIndex);
                rangeInput.setAttribute('aria-valuemin', String(paddingOptions[0]));
                rangeInput.setAttribute(
                    'aria-valuemax',
                    String(paddingOptions[paddingOptions.length - 1])
                );
                rangeInput.setAttribute('aria-valuenow', String(paddingValue));
                rangeInput.setAttribute(
                    'aria-valuetext',
                    `${paddingValue} pixels`
                );
                const rangeValue = createElement('span', {
                    className: 'admin-builder__range-value',
                    textContent: `${paddingValue}px`,
                });
                rangeValue.dataset.role = 'section-padding-value';
                rangeWrapper.append(rangeInput, rangeValue);
                rangeInput.addEventListener('input', () => {
                    const currentValue = paddingValueForIndex(rangeInput.value);
                    rangeValue.textContent = `${currentValue}px`;
                    rangeInput.setAttribute('aria-valuenow', String(currentValue));
                    rangeInput.setAttribute(
                        'aria-valuetext',
                        `${currentValue} pixels`
                    );
                });
                paddingField.append(rangeWrapper);
                sectionItem.append(paddingField);

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
                marginRangeInput.setAttribute('aria-valuemin', String(marginOptions[0]));
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
                });
                marginField.append(marginRangeWrapper);
                sectionItem.append(marginField);

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
                    section.elements.forEach((element, elementIndex) => {
                        const elementNode = createElement('div', {
                            className: 'admin-builder__element',
                        });
                        elementNode.dataset.elementClient = element.clientId;
                        elementNode.dataset.elementType = element.type;
                        elementNode.dataset.elementIndex = String(elementIndex);

                        const elementHeader = createElement('div', {
                            className: 'admin-builder__element-header',
                        });
                        const elementTitle = createElement('span', {
                            className: 'admin-builder__element-title',
                            textContent: `${elementLabel(definitions, element.type)} ${
                                elementIndex + 1
                            }`,
                        });
                        const elementActions = createElement('div', {
                            className: 'admin-builder__element-actions',
                        });
                        const removeElementButton = createElement('button', {
                            className: 'admin-builder__element-remove',
                            textContent: 'Remove',
                        });
                        removeElementButton.type = 'button';
                        removeElementButton.dataset.action = 'element-remove';
                        elementActions.append(removeElementButton);
                        elementHeader.append(elementTitle, elementActions);
                        elementNode.append(elementHeader);

                        renderElementEditor(definitions, elementNode, element);

                        elementsContainer.append(elementNode);
                    });
                }

                sectionItem.append(elementsContainer);

                if (allowElements) {
                    const sectionActions = createElement('div', {
                        className: 'admin-builder__section-actions',
                    });

                    orderedTypes.forEach((type) => {
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

                    sectionItem.append(sectionActions);
                }

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

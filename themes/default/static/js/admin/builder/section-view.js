(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    const sectionRegistry = window.AdminSectionRegistry;
    if (!utils || !registry || !sectionRegistry) {
        return;
    }

    const { createElement } = utils;

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
                }
                return;
            }

            if (emptyState) {
                emptyState.hidden = true;
            }

            const sectionTypeOrder = Array.isArray(orderedSectionTypes)
                ? orderedSectionTypes
                : Object.keys(sectionDefinitions || {});

            sections.forEach((section, index) => {
                const sectionItem = createElement('li', {
                    className: 'admin-builder__section',
                });
                sectionItem.dataset.sectionClient = section.clientId;
                sectionItem.dataset.sectionIndex = String(index);

                const sectionDefinition = sectionDefinitions?.[section.type] || {};
                const allowElements = sectionDefinition.supportsElements !== false;

                const sectionHeader = createElement('div', {
                    className: 'admin-builder__section-header',
                });
                const sectionTitle = createElement('h3', {
                    className: 'admin-builder__section-title',
                    textContent: `Section ${index + 1}`,
                });
                const removeButton = createElement('button', {
                    className: 'admin-builder__remove',
                    textContent: 'Remove section',
                });
                removeButton.type = 'button';
                removeButton.dataset.action = 'section-remove';
                sectionHeader.append(sectionTitle, removeButton);
                sectionItem.append(sectionHeader);

                const typeField = createElement('label', {
                    className: 'admin-builder__field',
                });
                typeField.append(
                    createElement('span', {
                        className: 'admin-builder__label',
                        textContent: 'Section type',
                    })
                );
                const typeSelect = createElement('select', {
                    className: 'admin-builder__input',
                });
                typeSelect.dataset.field = 'section-type';
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
                if (sectionDefinition.description) {
                    typeField.append(
                        createElement('span', {
                            className: 'admin-builder__hint',
                            textContent: sectionDefinition.description,
                        })
                    );
                }
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
                imageField.append(imageInput);
                sectionItem.append(imageField);

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

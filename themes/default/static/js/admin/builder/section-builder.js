(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    const sectionRegistry = window.AdminSectionRegistry;
    const stateModule = window.AdminSectionState;
    const viewModule = window.AdminSectionView;
    const eventsModule = window.AdminSectionEvents;

    if (
        !utils ||
        !registry ||
        !sectionRegistry ||
        !stateModule ||
        !viewModule ||
        !eventsModule
    ) {
        return;
    }
    const createSectionBuilder = (form, options = {}) => {
        if (!form) {
            return null;
        }

        const builderRoot = form.querySelector('[data-section-builder]');
        if (!builderRoot) {
            return null;
        }

        const queryWithFallback = (root, selectors) => {
            if (!root) {
                return null;
            }
            for (const selector of selectors) {
                if (!selector) {
                    continue;
                }
                const node = root.querySelector(selector);
                if (node) {
                    return node;
                }
            }
            return null;
        };

        const sectionList = queryWithFallback(builderRoot, [
            '[data-section-list]',
            '[data-role="section-list"]',
        ]);
        const emptyState = queryWithFallback(builderRoot, [
            '[data-section-empty]',
            '[data-role="section-empty"]',
        ]);
        const addSectionButton = queryWithFallback(builderRoot, [
            '[data-action="section-add"]',
            '[data-role="section-add"]',
        ]);

        if (!sectionList || !addSectionButton) {
            return null;
        }

        const definitions = registry.getDefinitions();
        const orderedTypes = registry.getOrderedTypes();
        const sectionDefinitions = sectionRegistry.getDefinitions();
        const orderedSectionTypes = sectionRegistry.getOrderedTypes();
        const sectionTypeOrder = Array.isArray(orderedSectionTypes)
            ? orderedSectionTypes
            : Object.keys(sectionDefinitions || {});
        let selectedSectionType = sectionRegistry.getDefaultType?.()
            || sectionTypeOrder?.[0]
            || 'standard';
        if (addSectionButton.parentElement) {
            const typePickerModule = window.AdminSectionTypePicker;
            if (typePickerModule?.open && sectionTypeOrder.length) {
                const typeSelector = utils.createElement('div', {
                    className: 'section-builder__type-selector',
                });
                const typeMeta = utils.createElement('div', {
                    className: 'admin-builder__type',
                });
                const typeControl = utils.createElement('div', {
                    className: 'admin-builder__type-control',
                });
                const typeSummary = utils.createElement('div', {
                    className: 'admin-builder__type-summary',
                });
                const typeSummaryLabel = utils.createElement('span', {
                    className: 'admin-builder__type-summary-label',
                });
                const typeSummaryCode = utils.createElement('code', {
                    className: 'admin-builder__type-summary-code',
                });
                typeSummary.append(typeSummaryLabel, typeSummaryCode);
                const changeTypeButton = utils.createElement('button', {
                    className:
                        'admin-builder__button admin-builder__button--ghost admin-builder__type-button',
                    type: 'button',
                    textContent: 'Change type',
                });
                typeControl.append(typeSummary, changeTypeButton);
                typeMeta.append(typeControl);
                const typeHint = utils.createElement('span', {
                    className: 'admin-builder__hint section-builder__type-hint',
                });
                typeHint.hidden = true;
                typeMeta.append(typeHint);
                typeSelector.append(typeMeta);

                const updateSelector = () => {
                    const definition = sectionDefinitions?.[selectedSectionType] || {};
                    const safeType =
                        typeof selectedSectionType === 'string' ? selectedSectionType : '';
                    const typeValue = safeType || 'unknown';
                    const labelText = utils.normaliseString(definition.label).trim();
                    typeSummaryLabel.textContent = labelText || typeValue;
                    typeSummaryCode.textContent = typeValue;
                    const description = utils.normaliseString(definition.description).trim();
                    if (description) {
                        typeHint.textContent = description;
                        typeHint.hidden = false;
                    } else {
                        typeHint.textContent = '';
                        typeHint.hidden = true;
                    }
                };

                updateSelector();

                changeTypeButton.addEventListener('click', (event) => {
                    event.preventDefault();
                    typePickerModule.open({
                        orderedSectionTypes: sectionTypeOrder,
                        sectionDefinitions,
                        activeType: selectedSectionType,
                        onSelect: (nextType) => {
                            if (!nextType || nextType === selectedSectionType) {
                                return;
                            }
                            selectedSectionType = nextType;
                            updateSelector();
                        },
                    });
                });

                addSectionButton.parentElement.insertBefore(
                    typeSelector,
                    addSectionButton
                );
            } else {
                const typePicker = utils.createElement('select', {
                    className: 'admin-builder__type-picker',
                });
                typePicker.setAttribute('aria-label', 'Section type');
                sectionTypeOrder.forEach((type) => {
                    const definition = sectionDefinitions?.[type] || {};
                    const option = utils.createElement('option', {
                        textContent: definition.label || type,
                    });
                    option.value = type;
                    if (type === selectedSectionType) {
                        option.selected = true;
                    }
                    typePicker.append(option);
                });
                typePicker.addEventListener('change', (event) => {
                    if (event.target && event.target.value) {
                        selectedSectionType = event.target.value;
                    }
                });
                addSectionButton.parentElement.insertBefore(typePicker, addSectionButton);
            }
        }
        const state = stateModule.createManager(definitions, sectionDefinitions);
        const { onApplyPaddingToAllSections } = options || {};

        const view = viewModule.createView({
            listElement: sectionList,
            emptyState,
            definitions,
            orderedTypes,
            sectionDefinitions,
            orderedSectionTypes,
            applyPaddingToAllSections: onApplyPaddingToAllSections,
        });

        const render = () => {
            view.render(state.getState());
        };

        const emitChange = () => {
            state.notify();
        };

        let pageId = null;

        const setPageId = (id) => {
            pageId = id;
        };

        const setSections = (nextSections) => {
            state.setSections(nextSections);
            render();
            emitChange();
        };

        const reset = () => {
            state.reset();
            render();
            emitChange();
        };

        const addSection = () => {
            const section = state.addSection(selectedSectionType);
            render();
            emitChange();
            view.focusField(
                `[data-section-client="${section.clientId}"] [data-field="section-title"]`
            );
        };

        const removeSection = (sectionClientId) => {
            state.removeSection(sectionClientId);
            render();
            emitChange();
        };

        const moveSection = (sectionClientId, direction) => {
            const nextIndex = state.moveSection(sectionClientId, direction);
            if (nextIndex < 0) {
                return;
            }
            render();
            emitChange();
            let focusRole = '';
            if (direction === 'up') {
                focusRole = nextIndex <= 0 ? 'section-move-down' : 'section-move-up';
            } else if (direction === 'down') {
                const lastIndex = state.getState().length - 1;
                focusRole = nextIndex >= lastIndex ? 'section-move-up' : 'section-move-down';
            }
            if (focusRole) {
                view.focusField(
                    `[data-section-client="${sectionClientId}"] [data-role="${focusRole}"]`
                );
            } else {
                view.focusField(
                    `[data-section-client="${sectionClientId}"] [data-field="section-title"]`
                );
            }
        };

        const addElementToSection = (sectionClientId, type) => {
            const element = state.addElementToSection(sectionClientId, type);
            if (!element) {
                return;
            }
            render();
            emitChange();
            const definition = definitions[type];
            const focusSelector = definition?.initialFocusSelector
                ? ` ${definition.initialFocusSelector}`
                : type === 'paragraph'
                ? ' textarea'
                : ' [data-field]';
            view.focusField(
                `[data-section-client="${sectionClientId}"] [data-element-client="${element.clientId}"]${focusSelector}`
            );
        };

        const removeElementFromSection = (sectionClientId, elementClientId) => {
            state.removeElementFromSection(sectionClientId, elementClientId);
            render();
            emitChange();
        };

        const moveElementInSection = (sectionClientId, elementClientId, direction) => {
            const nextIndex = state.moveElementInSection(sectionClientId, elementClientId, direction);
            if (nextIndex < 0) {
                return;
            }
            render();
            emitChange();
            const focusRole = direction === 'up' ? 'element-move-up' : 'element-move-down';
            view.focusField(
                `[data-section-client="${sectionClientId}"] [data-element-client="${elementClientId}"] [data-role="${focusRole}"]`
            );
        };

        const addGroupImage = (sectionClientId, elementClientId) => {
            const image = state.addGroupImage(sectionClientId, elementClientId);
            if (!image) {
                return;
            }
            render();
            emitChange();
            view.focusField(
                `[data-section-client="${sectionClientId}"] [data-element-client="${elementClientId}"] [data-group-image-client="${image.clientId}"] [data-field="group-image-url"]`
            );
        };

        const removeGroupImage = (
            sectionClientId,
            elementClientId,
            imageClientId
        ) => {
            state.removeGroupImage(sectionClientId, elementClientId, imageClientId);
            render();
            emitChange();
        };

        const addGroupFile = (sectionClientId, elementClientId) => {
            const file = state.addGroupFile(sectionClientId, elementClientId);
            if (!file) {
                return;
            }
            render();
            emitChange();
            view.focusField(
                `[data-section-client="${sectionClientId}"] [data-element-client="${elementClientId}"] [data-group-file-client="${file.clientId}"] [data-field="group-file-url"]`
            );
        };

        const removeGroupFile = (
            sectionClientId,
            elementClientId,
            fileClientId
        ) => {
            state.removeGroupFile(sectionClientId, elementClientId, fileClientId);
            render();
            emitChange();
        };

        const updateSectionField = (sectionClientId, field, value) => {
            state.updateSectionField(sectionClientId, field, value);
            if (field === 'section-type') {
                render();
                emitChange();
            }
        };

        const updateElementField = (
            sectionClientId,
            elementClientId,
            field,
            value,
            nestedClientId
        ) => {
            state.updateElementField(
                sectionClientId,
                elementClientId,
                field,
                value,
                nestedClientId
            );
        };

        addSectionButton.addEventListener('click', () => {
            addSection();
        });

        const events = eventsModule.bind({
            listElement: sectionList,
            onSectionRemove: removeSection,
            onSectionMove: moveSection,
            onElementRemove: removeElementFromSection,
            onElementMove: moveElementInSection,
            onElementAdd: addElementToSection,
            onGroupImageAdd: addGroupImage,
            onGroupImageRemove: removeGroupImage,
            onGroupFileAdd: addGroupFile,
            onGroupFileRemove: removeGroupFile,
            onSectionFieldChange: (sectionClientId, field, value) => {
                updateSectionField(sectionClientId, field, value);
                emitChange();
            },
            onElementFieldChange: (
                sectionClientId,
                elementClientId,
                field,
                value,
                imageClientId
            ) => {
                updateElementField(
                    sectionClientId,
                    elementClientId,
                    field,
                    value,
                    imageClientId
                );
                emitChange();
            },
        });

        const onChange = (listener) => {
            if (typeof listener !== 'function') {
                return () => {};
            }
            state.subscribe(listener);
            return () => state.unsubscribe(listener);
        };

        render();

        return {
            setSections,
            reset,
            getSections: () => state.getSections(),
            setPageId,
            onChange,
            destroy: () => {
                events.destroy();
            },
        };
    };

    window.AdminSectionBuilder = {
        create: createSectionBuilder,
    };
})();

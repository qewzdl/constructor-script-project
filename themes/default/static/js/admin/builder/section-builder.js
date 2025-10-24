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
    const createSectionBuilder = (form) => {
        if (!form) {
            return null;
        }

        const builderRoot = form.querySelector('[data-section-builder]');
        if (!builderRoot) {
            return null;
        }

        const sectionList = builderRoot.querySelector('[data-section-list]');
        const emptyState = builderRoot.querySelector('[data-section-empty]');
        const addSectionButton = builderRoot.querySelector(
            '[data-action="section-add"]'
        );

        if (!sectionList || !addSectionButton) {
            return null;
        }

        const definitions = registry.getDefinitions();
        const orderedTypes = registry.getOrderedTypes();
        const sectionDefinitions = sectionRegistry.getDefinitions();
        const orderedSectionTypes = sectionRegistry.getOrderedTypes();
        let selectedSectionType = sectionRegistry.getDefaultType?.()
            || orderedSectionTypes?.[0]
            || 'standard';
        if (addSectionButton.parentElement) {
            const typePicker = utils.createElement('select', {
                className: 'admin-builder__type-picker',
            });
            typePicker.setAttribute('aria-label', 'Section type');
            const sectionTypeOrder = Array.isArray(orderedSectionTypes)
                ? orderedSectionTypes
                : Object.keys(sectionDefinitions || {});
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
        const state = stateModule.createManager(definitions, sectionDefinitions);
        const view = viewModule.createView({
            listElement: sectionList,
            emptyState,
            definitions,
            orderedTypes,
            sectionDefinitions,
            orderedSectionTypes,
        });

        const render = () => {
            view.render(state.getState());
        };

        const emitChange = () => {
            state.notify();
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
            imageClientId
        ) => {
            state.updateElementField(
                sectionClientId,
                elementClientId,
                field,
                value,
                imageClientId
            );
        };

        addSectionButton.addEventListener('click', () => {
            addSection();
        });

        const events = eventsModule.bind({
            listElement: sectionList,
            onSectionRemove: removeSection,
            onElementRemove: removeElementFromSection,
            onElementAdd: addElementToSection,
            onGroupImageAdd: addGroupImage,
            onGroupImageRemove: removeGroupImage,
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
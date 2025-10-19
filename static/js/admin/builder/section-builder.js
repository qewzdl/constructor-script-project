(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    const stateModule = window.AdminSectionState;
    const viewModule = window.AdminSectionView;
    const eventsModule = window.AdminSectionEvents;

    if (!utils || !registry || !stateModule || !viewModule || !eventsModule) {
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
        const state = stateModule.createManager(definitions);
        const view = viewModule.createView({
            listElement: sectionList,
            emptyState,
            definitions,
            orderedTypes,
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
            const section = state.addSection();
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
            onSectionFieldChange: updateSectionField,
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
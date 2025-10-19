
(() => {
    const createEvents = ({
        listElement,
        onSectionRemove,
        onElementRemove,
        onElementAdd,
        onGroupImageAdd,
        onGroupImageRemove,
        onSectionFieldChange,
        onElementFieldChange,
    }) => {
        if (!listElement) {
            return { destroy: () => {} };
        }

        const handleClick = (event) => {
            const target = event.target;
            if (!(target instanceof HTMLElement)) {
                return;
            }
            const sectionNode = target.closest('[data-section-client]');
            if (!sectionNode) {
                return;
            }
            const sectionClientId = sectionNode.dataset.sectionClient;
            if (!sectionClientId) {
                return;
            }

            if (target.matches('[data-action="section-remove"]')) {
                event.preventDefault();
                onSectionRemove?.(sectionClientId);
                return;
            }

            if (target.matches('[data-action="element-remove"]')) {
                event.preventDefault();
                const elementNode = target.closest('[data-element-client]');
                if (!elementNode) {
                    return;
                }
                onElementRemove?.(sectionClientId, elementNode.dataset.elementClient);
                return;
            }

            if (target.matches('[data-action="element-add"]')) {
                event.preventDefault();
                const type = target.dataset.elementType || 'paragraph';
                onElementAdd?.(sectionClientId, type);
                return;
            }

            if (target.matches('[data-action="group-image-add"]')) {
                event.preventDefault();
                const elementNode = target.closest('[data-element-client]');
                if (!elementNode) {
                    return;
                }
                onGroupImageAdd?.(sectionClientId, elementNode.dataset.elementClient);
                return;
            }

            if (target.matches('[data-action="group-image-remove"]')) {
                event.preventDefault();
                const elementNode = target.closest('[data-element-client]');
                const imageNode = target.closest('[data-group-image-client]');
                if (!elementNode || !imageNode) {
                    return;
                }
                onGroupImageRemove?.(
                    sectionClientId,
                    elementNode.dataset.elementClient,
                    imageNode.dataset.groupImageClient
                );
            }
        };

        const handleInput = (event) => {
            const target = event.target;
            if (!(target instanceof HTMLElement)) {
                return;
            }
            const sectionNode = target.closest('[data-section-client]');
            if (!sectionNode) {
                return;
            }
            const sectionClientId = sectionNode.dataset.sectionClient;
            if (!sectionClientId) {
                return;
            }

            const field = target.dataset.field;
            if (!field) {
                return;
            }

            const value = target.type === 'checkbox' ? target.checked : target.value;
            const elementNode = target.closest('[data-element-client]');
            if (elementNode) {
                const elementClientId = elementNode.dataset.elementClient;
                const imageNode = target.closest('[data-group-image-client]');
                const imageClientId = imageNode
                    ? imageNode.dataset.groupImageClient
                    : undefined;
                onElementFieldChange?.(
                    sectionClientId,
                    elementClientId,
                    field,
                    value,
                    imageClientId
                );
                return;
            }

            onSectionFieldChange?.(sectionClientId, field, value);
        };

        listElement.addEventListener('click', handleClick);
        listElement.addEventListener('input', handleInput);

        const destroy = () => {
            listElement.removeEventListener('click', handleClick);
            listElement.removeEventListener('input', handleInput);
        };

        return { destroy };
    };

    window.AdminSectionEvents = {
        bind: createEvents,
    };
})();
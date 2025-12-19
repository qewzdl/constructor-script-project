
(() => {
    const utils = window.AdminUtils;

    const openAnchorPicker = (inputElement) => {
        if (!inputElement || !utils) {
            return;
        }

        // Get all sections on the page
        const sections = Array.from(document.querySelectorAll('[data-section-client]'));
        if (sections.length === 0) {
            alert('No sections found on this page');
            return;
        }

        // Create modal overlay
        const overlay = utils.createElement('div', {
            className: 'anchor-picker-overlay'
        });

        // Create modal content
        const modal = utils.createElement('div', {
            className: 'anchor-picker-modal'
        });

        // Header
        const header = utils.createElement('div', {
            className: 'anchor-picker-header'
        });
        const title = utils.createElement('h3', {
            className: 'anchor-picker-title',
            textContent: 'Select a section to link'
        });
        header.appendChild(title);

        // Section list
        const sectionList = utils.createElement('div', {
            className: 'anchor-picker-list'
        });

        sections.forEach((section, index) => {
            const sectionClientId = section.dataset.sectionClient;
            const sectionId = section.dataset.sectionId;
            const titleInput = section.querySelector('[data-field="section-title"]');
            const sectionTitle = titleInput ? titleInput.value.trim() : '';
            const sectionType = section.dataset.sectionType || 'unknown';
            
            const displayTitle = sectionTitle || `Section ${index + 1}`;
            const anchorId = sectionId || sectionClientId;

            const button = utils.createElement('button', {
                className: 'anchor-picker-item',
                type: 'button'
            });
            
            const buttonTitle = utils.createElement('div', {
                className: 'anchor-picker-item-title',
                textContent: displayTitle
            });
            
            const buttonMeta = utils.createElement('div', {
                className: 'anchor-picker-item-meta'
            });
            
            const typeSpan = utils.createElement('span', {
                textContent: `Type: ${sectionType}`
            });
            
            const anchorSpan = utils.createElement('code', {
                className: 'anchor-picker-item-code',
                textContent: `#section-${anchorId}`
            });
            
            buttonMeta.append(typeSpan, anchorSpan);
            button.append(buttonTitle, buttonMeta);

            button.addEventListener('click', () => {
                inputElement.value = `#section-${anchorId}`;
                inputElement.dispatchEvent(new Event('input', { bubbles: true }));
                inputElement.dispatchEvent(new Event('change', { bubbles: true }));
                document.body.removeChild(overlay);
            });

            sectionList.appendChild(button);
        });

        // Footer
        const footer = utils.createElement('div', {
            className: 'anchor-picker-footer'
        });
        const cancelButton = utils.createElement('button', {
            className: 'admin-builder__button',
            textContent: 'Cancel',
            type: 'button'
        });
        footer.appendChild(cancelButton);

        modal.append(header, sectionList, footer);
        overlay.appendChild(modal);

        // Close handlers
        const closeModal = () => {
            if (document.body.contains(overlay)) {
                document.body.removeChild(overlay);
            }
        };

        cancelButton.addEventListener('click', closeModal);
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                closeModal();
            }
        });

        document.body.appendChild(overlay);
    };

    const createEvents = ({
        listElement,
        onSectionRemove,
        onSectionMove,
        onElementRemove,
        onElementMove,
        onElementAdd,
        onGroupImageAdd,
        onGroupImageRemove,
        onGroupFileAdd,
        onGroupFileRemove,
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

            if (target.matches('[data-action="section-move"]')) {
                event.preventDefault();
                const direction = target.dataset.direction || '';
                onSectionMove?.(sectionClientId, direction);
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

            if (target.matches('[data-action="element-move"]')) {
                event.preventDefault();
                const elementNode = target.closest('[data-element-client]');
                if (!elementNode) {
                    return;
                }
                const direction = target.dataset.direction || '';
                onElementMove?.(sectionClientId, elementNode.dataset.elementClient, direction);
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
                return;
            }

            if (target.matches('[data-action="group-file-add"]')) {
                event.preventDefault();
                const elementNode = target.closest('[data-element-client]');
                if (!elementNode) {
                    return;
                }
                onGroupFileAdd?.(sectionClientId, elementNode.dataset.elementClient);
                return;
            }

            if (target.matches('[data-action="group-file-remove"]')) {
                event.preventDefault();
                const elementNode = target.closest('[data-element-client]');
                const fileNode = target.closest('[data-group-file-client]');
                if (!elementNode || !fileNode) {
                    return;
                }
                onGroupFileRemove?.(
                    sectionClientId,
                    elementNode.dataset.elementClient,
                    fileNode.dataset.groupFileClient
                );
                return;
            }

            if (target.matches('[data-action="open-anchor-picker"]')) {
                event.preventDefault();
                const targetInputId = target.dataset.anchorTarget;
                if (!targetInputId || !targetInputId.startsWith('#')) {
                    return;
                }
                const inputElement = document.querySelector(targetInputId);
                if (!inputElement) {
                    return;
                }
                openAnchorPicker(inputElement);
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

            let value = target.type === 'checkbox' ? target.checked : target.value;
            if (
                field === 'section-padding-vertical' ||
                field === 'section-margin-vertical'
            ) {
                const options = target.dataset.options
                    ? target.dataset.options
                          .split(',')
                          .map((option) => Number.parseInt(option.trim(), 10))
                          .filter((option) => Number.isFinite(option))
                    : [];
                const rawIndex = Number.parseInt(String(value), 10);
                const maxIndex = options.length - 1;
                const clampedIndex = Number.isFinite(rawIndex)
                    ? Math.min(Math.max(rawIndex, 0), maxIndex >= 0 ? maxIndex : 0)
                    : 0;
                if (Number.isFinite(rawIndex) && clampedIndex !== rawIndex) {
                    target.value = String(clampedIndex);
                }
                const actualValue = options[clampedIndex] ?? 0;
                value = actualValue;
                if (options.length) {
                    target.setAttribute('aria-valuenow', String(actualValue));
                    target.setAttribute('aria-valuetext', `${actualValue} pixels`);
                }
                const displayRole =
                    field === 'section-padding-vertical'
                        ? 'section-padding-value'
                        : 'section-margin-value';
                const displayNode = target.parentElement?.querySelector(
                    `[data-role="${displayRole}"]`
                );
                if (displayNode) {
                    displayNode.textContent = `${actualValue}px`;
                }
            }
            const elementNode = target.closest('[data-element-client]');
            if (elementNode) {
                const elementClientId = elementNode.dataset.elementClient;
                const imageNode = target.closest('[data-group-image-client]');
                const fileNode = target.closest('[data-group-file-client]');
                const nestedClientId = imageNode
                    ? imageNode.dataset.groupImageClient
                    : fileNode?.dataset.groupFileClient;
                onElementFieldChange?.(
                    sectionClientId,
                    elementClientId,
                    field,
                    value,
                    nestedClientId
                );
                return;
            }

            onSectionFieldChange?.(sectionClientId, field, value);
        };

        listElement.addEventListener('click', handleClick);
        listElement.addEventListener('input', handleInput);
        listElement.addEventListener('change', handleInput);

        const destroy = () => {
            listElement.removeEventListener('click', handleClick);
            listElement.removeEventListener('input', handleInput);
            listElement.removeEventListener('change', handleInput);
        };

        return { destroy };
    };

    window.AdminSectionEvents = {
        bind: createEvents,
    };
})();

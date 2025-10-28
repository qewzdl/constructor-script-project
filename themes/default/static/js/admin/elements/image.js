(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    if (!utils || !registry) {
        return;
    }

    const { createElement, normaliseString, randomId } = utils;

    registry.register('image', {
        label: 'Image',
        addLabel: 'Add image',
        order: 20,
        initialFocusSelector: '[data-field="image-url"]',
        create: () => ({
            clientId: randomId(),
            id: '',
            type: 'image',
            content: {
                url: '',
                alt: '',
                caption: '',
            },
        }),
        fromRaw: ({ id, rawContent }) => ({
            clientId: randomId(),
            id,
            type: 'image',
            content: {
                url: normaliseString(rawContent.url ?? rawContent.URL ?? ''),
                alt: normaliseString(rawContent.alt ?? rawContent.Alt ?? ''),
                caption: normaliseString(
                    rawContent.caption ?? rawContent.Caption ?? ''
                ),
            },
        }),
        renderEditor: (elementNode, element) => {
            const urlField = createElement('label', {
                className: 'admin-builder__field',
            });
            urlField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Image URL',
                })
            );
            const urlInput = createElement('input', {
                className: 'admin-builder__input',
            });
            urlInput.type = 'url';
            urlInput.placeholder = 'https://example.com/visual.png';
            urlInput.value = element.content?.url || '';
            urlInput.dataset.field = 'image-url';
            const inputId = `admin-builder-image-${element.clientId}`;
            urlInput.id = inputId;
            urlField.append(urlInput);

            const urlActions = createElement('div', {
                className: 'admin-builder__field-actions',
            });
            const browseButton = createElement('button', {
                className: 'admin-builder__media-button',
                textContent: 'Browse uploads',
            });
            browseButton.type = 'button';
            browseButton.dataset.action = 'open-media-library';
            browseButton.dataset.mediaTarget = `#${inputId}`;
            urlActions.append(browseButton);
            urlField.append(urlActions);
            elementNode.append(urlField);

            const altField = createElement('label', {
                className: 'admin-builder__field',
            });
            altField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Alt text',
                })
            );
            const altInput = createElement('input', {
                className: 'admin-builder__input',
            });
            altInput.type = 'text';
            altInput.placeholder = 'Describe the image';
            altInput.value = element.content?.alt || '';
            altInput.dataset.field = 'image-alt';
            altField.append(altInput);
            elementNode.append(altField);

            const captionField = createElement('label', {
                className: 'admin-builder__field',
            });
            captionField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Caption',
                })
            );
            const captionInput = createElement('textarea', {
                className: 'admin-builder__textarea',
            });
            captionInput.placeholder = 'Add context that appears under the image';
            captionInput.value = element.content?.caption || '';
            captionInput.dataset.field = 'image-caption';
            captionField.append(captionInput);
            elementNode.append(captionField);
        },
        updateField: (element, field, value) => {
            if (field === 'image-url') {
                element.content.url = value;
                return true;
            }
            if (field === 'image-alt') {
                element.content.alt = value;
                return true;
            }
            if (field === 'image-caption') {
                element.content.caption = value;
                return true;
            }
            return false;
        },
        hasContent: (element) => Boolean(element.content?.url?.trim()),
        sanitise: (element, index) => {
            const payload = {
                url: element.content.url.trim(),
            };
            if (element.content.alt && element.content.alt.trim()) {
                payload.alt = element.content.alt.trim();
            }
            if (element.content.caption && element.content.caption.trim()) {
                payload.caption = element.content.caption.trim();
            }
            return {
                id: element.id || '',
                type: 'image',
                order: index + 1,
                content: payload,
            };
        },
    });
})();
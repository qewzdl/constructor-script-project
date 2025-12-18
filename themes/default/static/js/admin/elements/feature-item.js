(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    if (!utils || !registry) {
        return;
    }

    const { createElement, normaliseString, randomId } = utils;

    registry.register('feature_item', {
        label: 'Feature item',
        addLabel: 'Add feature',
        order: 25,
        initialFocusSelector: '[data-field="feature-text"]',
        create: () => ({
            clientId: randomId(),
            id: '',
            type: 'feature_item',
            content: {
                text: '',
                image_url: '',
                image_alt: '',
            },
        }),
        fromRaw: ({ id, rawContent }) => ({
            clientId: randomId(),
            id,
            type: 'feature_item',
            content: {
                text: normaliseString(rawContent.text ?? rawContent.Text ?? ''),
                image_url: normaliseString(
                    rawContent.image_url ??
                        rawContent.imageUrl ??
                        rawContent.Image_url ??
                        rawContent.ImageUrl ??
                        ''
                ),
                image_alt: normaliseString(
                    rawContent.image_alt ??
                        rawContent.imageAlt ??
                        rawContent.Image_alt ??
                        rawContent.ImageAlt ??
                        ''
                ),
            },
        }),
        renderEditor: (elementNode, element) => {
            const textField = createElement('label', {
                className: 'admin-builder__field',
            });
            textField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Feature text',
                })
            );
            const textInput = createElement('textarea', {
                className: 'admin-builder__textarea',
            });
            textInput.placeholder = 'Describe the feature benefit';
            textInput.value = element.content?.text || '';
            textInput.dataset.field = 'feature-text';
            textField.append(textInput);
            elementNode.append(textField);

            const imageField = createElement('label', {
                className: 'admin-builder__field',
            });
            imageField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Image URL',
                })
            );
            const imageInput = createElement('input', {
                className: 'admin-builder__input',
            });
            imageInput.type = 'url';
            imageInput.placeholder = 'https://example.com/feature.jpg';
            imageInput.value = element.content?.image_url || '';
            imageInput.dataset.field = 'feature-image-url';
            const imageInputId = `admin-builder-feature-${element.clientId}`;
            imageInput.id = imageInputId;
            imageField.append(imageInput);

            const imageActions = createElement('div', {
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
            imageActions.append(browseButton);
            imageField.append(imageActions);
            elementNode.append(imageField);

            const altField = createElement('label', {
                className: 'admin-builder__field',
            });
            altField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Image alt text',
                })
            );
            const altInput = createElement('input', {
                className: 'admin-builder__input',
            });
            altInput.type = 'text';
            altInput.placeholder = 'Describe the image content';
            altInput.value = element.content?.image_alt || '';
            altInput.dataset.field = 'feature-image-alt';
            altField.append(altInput);
            elementNode.append(altField);
        },
        updateField: (element, field, value) => {
            if (field === 'feature-text') {
                element.content.text = value;
                return true;
            }
            if (field === 'feature-image-url') {
                element.content.image_url = value;
                return true;
            }
            if (field === 'feature-image-alt') {
                element.content.image_alt = value;
                return true;
            }
            return false;
        },
        hasContent: (element) =>
            Boolean(
                element.content?.text?.trim() ||
                    element.content?.image_url?.trim()
            ),
        sanitise: (element, index) => {
            const payload = {};
            if (element.content.text && element.content.text.trim()) {
                payload.text = element.content.text.trim();
            }
            if (element.content.image_url && element.content.image_url.trim()) {
                payload.image_url = element.content.image_url.trim();
            }
            if (element.content.image_alt && element.content.image_alt.trim()) {
                payload.image_alt = element.content.image_alt.trim();
            }
            return {
                id: element.id || '',
                type: 'feature_item',
                order: index + 1,
                content: payload,
            };
        },
        preview: (element, parts) => {
            if (element.content?.text) {
                parts.push(element.content.text);
            }
            if (element.content?.image_url) {
                parts.push(element.content.image_url);
            }
        },
    });
})();

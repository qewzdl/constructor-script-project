(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    if (!utils || !registry) {
        return;
    }

    const { createElement, normaliseString, randomId } = utils;
    const isGlyphVariation = (context = {}) => {
        const section = context?.section;
        if (!section) {
            return false;
        }
        const sectionType = normaliseString(section.type).toLowerCase();
        const sectionVariation = normaliseString(section.variation).toLowerCase();
        return (
            sectionType === 'features' &&
            (sectionVariation === 'glyph' || sectionVariation === 'icon-text')
        );
    };
    const isConstellationVariation = (context = {}) => {
        const section = context?.section;
        if (!section) {
            return false;
        }
        const sectionType = normaliseString(section.type).toLowerCase();
        const sectionVariation = normaliseString(section.variation).toLowerCase();
        return sectionType === 'features' && sectionVariation === 'constellation';
    };

    registry.register('feature_item', {
        label: 'Feature item',
        addLabel: 'Add feature',
        order: 25,
        initialFocusSelector: '[data-field="feature-title"]',
        create: () => ({
            clientId: randomId(),
            id: '',
            type: 'feature_item',
            content: {
                title: '',
                subtitle: '',
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
                title: normaliseString(rawContent.title ?? rawContent.Title ?? ''),
                subtitle: normaliseString(
                    rawContent.subtitle ?? rawContent.Subtitle ?? ''
                ),
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
        renderEditor: (elementNode, element, context = {}) => {
            const glyphVariation = isGlyphVariation(context);
            const constellationVariation = isConstellationVariation(context);
            const iconLedVariation = glyphVariation || constellationVariation;
            const titleField = createElement('label', {
                className: 'admin-builder__field',
            });
            titleField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Feature title',
                })
            );
            const titleInput = createElement('input', {
                className: 'admin-builder__input',
            });
            titleInput.type = 'text';
            titleInput.placeholder = 'Name the feature highlight';
            titleInput.value = element.content?.title || '';
            titleInput.dataset.field = 'feature-title';
            titleField.append(titleInput);
            elementNode.append(titleField);

            if (constellationVariation) {
                const subtitleField = createElement('label', {
                    className: 'admin-builder__field',
                });
                subtitleField.append(
                    createElement('span', {
                        className: 'admin-builder__label',
                        textContent: 'Feature subtitle',
                    })
                );
                const subtitleInput = createElement('input', {
                    className: 'admin-builder__input',
                });
                subtitleInput.type = 'text';
                subtitleInput.placeholder = 'Add a concise supporting line';
                subtitleInput.value = element.content?.subtitle || '';
                subtitleInput.dataset.field = 'feature-subtitle';
                subtitleField.append(subtitleInput);
                elementNode.append(subtitleField);
            }

            if (!glyphVariation) {
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
            }

            const imageField = createElement('label', {
                className: 'admin-builder__field',
            });
            imageField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: iconLedVariation ? 'Icon URL' : 'Image URL',
                })
            );
            const imageInput = createElement('input', {
                className: 'admin-builder__input',
            });
            imageInput.type = 'url';
            imageInput.placeholder = iconLedVariation
                ? 'https://example.com/icon.svg'
                : 'https://example.com/feature.jpg';
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
                    textContent: iconLedVariation
                        ? 'Icon alt text'
                        : 'Image alt text',
                })
            );
            const altInput = createElement('input', {
                className: 'admin-builder__input',
            });
            altInput.type = 'text';
            altInput.placeholder = iconLedVariation
                ? 'Describe the icon'
                : 'Describe the image content';
            altInput.value = element.content?.image_alt || '';
            altInput.dataset.field = 'feature-image-alt';
            altField.append(altInput);
            elementNode.append(altField);
        },
        updateField: (element, field, value) => {
            if (field === 'feature-title') {
                element.content.title = value;
                return true;
            }
            if (field === 'feature-subtitle') {
                element.content.subtitle = value;
                return true;
            }
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
        hasContent: (element, context = {}) => {
            const hasTitle = Boolean(element.content?.title?.trim());
            const hasSubtitle = Boolean(element.content?.subtitle?.trim());
            const hasText = Boolean(element.content?.text?.trim());
            const hasImage = Boolean(element.content?.image_url?.trim());
            if (isGlyphVariation(context)) {
                return hasTitle || hasImage;
            }
            if (isConstellationVariation(context)) {
                return hasTitle || hasSubtitle || hasText || hasImage;
            }
            return hasTitle || hasText || hasImage;
        },
        sanitise: (element, index, context = {}) => {
            const payload = {};
            const glyphVariation = isGlyphVariation(context);
            const constellationVariation = isConstellationVariation(context);
            if (element.content.title && element.content.title.trim()) {
                payload.title = element.content.title.trim();
            }
            if (element.content.subtitle && element.content.subtitle.trim()) {
                payload.subtitle = element.content.subtitle.trim();
            }
            if (
                !glyphVariation &&
                element.content.text &&
                element.content.text.trim()
            ) {
                payload.text = element.content.text.trim();
            }
            if (element.content.image_url && element.content.image_url.trim()) {
                payload.image_url = element.content.image_url.trim();
            }
            if (element.content.image_alt && element.content.image_alt.trim()) {
                payload.image_alt = element.content.image_alt.trim();
            }
            if (
                glyphVariation &&
                (!payload.title || !payload.image_url)
            ) {
                return null;
            }
            if (
                constellationVariation &&
                (!payload.title ||
                    !payload.subtitle ||
                    !payload.text ||
                    !payload.image_url)
            ) {
                return null;
            }
            return {
                id: element.id || '',
                type: 'feature_item',
                order: index + 1,
                content: payload,
            };
        },
        preview: (element, parts, context = {}) => {
            const glyphVariation = isGlyphVariation(context);
            const constellationVariation = isConstellationVariation(context);
            if (element.content?.title) {
                parts.push(element.content.title);
            }
            if (constellationVariation && element.content?.subtitle) {
                parts.push(element.content.subtitle);
            }
            if (!glyphVariation && element.content?.text) {
                parts.push(element.content.text);
            }
            if (element.content?.image_url) {
                parts.push(element.content.image_url);
            }
        },
    });
})();

(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    if (!utils || !registry) {
        return;
    }

    const { createElement, ensureArray, createImageState, normaliseString, randomId } =
        utils;

    registry.register('image_group', {
        label: 'Image group',
        addLabel: 'Add image group',
        order: 30,
        initialFocusSelector: '[data-field="image-group-layout"]',
        create: () => ({
            clientId: randomId(),
            id: '',
            type: 'image_group',
            content: {
                layout: 'grid',
                images: [createImageState({})],
            },
        }),
        fromRaw: ({ id, rawContent }) => {
            const images = ensureArray(rawContent.images ?? rawContent.Images).map(
                createImageState
            );
            return {
                clientId: randomId(),
                id,
                type: 'image_group',
                content: {
                    layout: normaliseString(
                        rawContent.layout ?? rawContent.Layout ?? 'grid'
                    ),
                    images,
                },
            };
        },
        renderEditor: (elementNode, element) => {
            const groupContainer = createElement('div', {
                className: 'admin-builder__group',
            });

            const layoutField = createElement('label', {
                className: 'admin-builder__field',
            });
            layoutField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Layout preset',
                })
            );
            const layoutInput = createElement('input', {
                className: 'admin-builder__input',
            });
            layoutInput.type = 'text';
            layoutInput.placeholder = 'e.g. grid, carousel, mosaic';
            layoutInput.value = element.content?.layout || '';
            layoutInput.dataset.field = 'image-group-layout';
            layoutField.append(layoutInput);
            groupContainer.append(layoutField);

            const groupList = createElement('div', {
                className: 'admin-builder__group-list',
            });

            if (!element.content?.images?.length) {
                groupList.append(
                    createElement('p', {
                        className: 'admin-builder__element-empty',
                        textContent: 'No images in this group yet.',
                    })
                );
            } else {
                element.content.images.forEach((image) => {
                    const groupItem = createElement('div', {
                        className: 'admin-builder__group-item',
                    });
                    groupItem.dataset.groupImageClient = image.clientId;

                    const groupUrlField = createElement('label', {
                        className: 'admin-builder__field',
                    });
                    groupUrlField.append(
                        createElement('span', {
                            className: 'admin-builder__label',
                            textContent: 'Image URL',
                        })
                    );
                    const groupUrlInput = createElement('input', {
                        className: 'admin-builder__input',
                    });
                    groupUrlInput.type = 'url';
                    groupUrlInput.placeholder = 'https://example.com/gallery-image.jpg';
                    groupUrlInput.value = image.url || '';
                    groupUrlInput.dataset.field = 'group-image-url';
                    const inputId = `admin-builder-group-image-${image.clientId}`;
                    groupUrlInput.id = inputId;
                    groupUrlField.append(groupUrlInput);

                    const groupUrlActions = createElement('div', {
                        className: 'admin-builder__field-actions',
                    });
                    const browseButton = createElement('button', {
                        className: 'admin-builder__media-button',
                        textContent: 'Browse uploads',
                    });
                    browseButton.type = 'button';
                    browseButton.dataset.action = 'open-media-library';
                    browseButton.dataset.mediaTarget = `#${inputId}`;
                    groupUrlActions.append(browseButton);
                    groupUrlField.append(groupUrlActions);
                    groupItem.append(groupUrlField);

                    const groupAltField = createElement('label', {
                        className: 'admin-builder__field',
                    });
                    groupAltField.append(
                        createElement('span', {
                            className: 'admin-builder__label',
                            textContent: 'Alt text',
                        })
                    );
                    const groupAltInput = createElement('input', {
                        className: 'admin-builder__input',
                    });
                    groupAltInput.type = 'text';
                    groupAltInput.placeholder = 'Describe this image';
                    groupAltInput.value = image.alt || '';
                    groupAltInput.dataset.field = 'group-image-alt';
                    groupAltField.append(groupAltInput);
                    groupItem.append(groupAltField);

                    const groupCaptionField = createElement('label', {
                        className: 'admin-builder__field',
                    });
                    groupCaptionField.append(
                        createElement('span', {
                            className: 'admin-builder__label',
                            textContent: 'Caption',
                        })
                    );
                    const groupCaptionInput = createElement('input', {
                        className: 'admin-builder__input',
                    });
                    groupCaptionInput.type = 'text';
                    groupCaptionInput.placeholder = 'Optional caption';
                    groupCaptionInput.value = image.caption || '';
                    groupCaptionInput.dataset.field = 'group-image-caption';
                    groupCaptionField.append(groupCaptionInput);
                    groupItem.append(groupCaptionField);

                    const groupActions = createElement('div', {
                        className: 'admin-builder__group-actions',
                    });
                    const removeImageButton = createElement('button', {
                        className: 'admin-builder__element-remove',
                        textContent: 'Remove image',
                    });
                    removeImageButton.type = 'button';
                    removeImageButton.dataset.action = 'group-image-remove';
                    groupActions.append(removeImageButton);
                    groupItem.append(groupActions);

                    groupList.append(groupItem);
                });
            }

            groupContainer.append(groupList);

            const addGroupImageButton = createElement('button', {
                className: 'admin-builder__button admin-builder__button--ghost',
                textContent: 'Add image to group',
            });
            addGroupImageButton.type = 'button';
            addGroupImageButton.dataset.action = 'group-image-add';
            groupContainer.append(addGroupImageButton);

            elementNode.append(groupContainer);
        },
        updateField: (element, field, value, imageClientId) => {
            if (field === 'image-group-layout') {
                element.content.layout = value;
                return true;
            }
            if (
                field === 'group-image-url' ||
                field === 'group-image-alt' ||
                field === 'group-image-caption'
            ) {
                if (!element.content.images) {
                    element.content.images = [];
                }
                const image = element.content.images.find(
                    (img) => img.clientId === imageClientId
                );
                if (!image) {
                    return false;
                }
                if (field === 'group-image-url') {
                    image.url = value;
                    return true;
                }
                if (field === 'group-image-alt') {
                    image.alt = value;
                    return true;
                }
                image.caption = value;
                return true;
            }
            return false;
        },
        hasContent: (element) =>
            Array.isArray(element.content?.images) &&
            element.content.images.some((image) => image.url && image.url.trim()),
        sanitise: (element, index) => {
            const images = (element.content.images || [])
                .map((image) => {
                    const url = (image.url || '').trim();
                    if (!url) {
                        return null;
                    }
                    const payload = { url };
                    if (image.alt && image.alt.trim()) {
                        payload.alt = image.alt.trim();
                    }
                    if (image.caption && image.caption.trim()) {
                        payload.caption = image.caption.trim();
                    }
                    return payload;
                })
                .filter(Boolean);

            if (!images.length) {
                return null;
            }

            const payload = { images };
            if (element.content.layout && element.content.layout.trim()) {
                payload.layout = element.content.layout.trim();
            }

            return {
                id: element.id || '',
                type: 'image_group',
                order: index + 1,
                content: payload,
            };
        },
    });
})();
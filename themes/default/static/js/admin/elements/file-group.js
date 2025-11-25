(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    if (!utils || !registry) {
        return;
    }

    const { createElement, ensureArray, normaliseString, randomId, createFileState } =
        utils;

    registry.register('file_group', {
        label: 'File group',
        addLabel: 'Add file group',
        order: 45,
        initialFocusSelector: '[data-field="file-group-title"]',
        create: () => ({
            clientId: randomId(),
            id: '',
            type: 'file_group',
            content: {
                title: '',
                description: '',
                files: [createFileState({})],
            },
        }),
        fromRaw: ({ id, rawContent }) => ({
            clientId: randomId(),
            id,
            type: 'file_group',
            content: {
                title: normaliseString(rawContent.title ?? rawContent.Title ?? ''),
                description: normaliseString(
                    rawContent.description ?? rawContent.Description ?? ''
                ),
                files: ensureArray(rawContent.files ?? rawContent.Files).map(
                    createFileState
                ),
            },
        }),
        renderEditor: (elementNode, element) => {
            const groupContainer = createElement('div', {
                className: 'admin-builder__group',
            });

            const titleField = createElement('label', {
                className: 'admin-builder__field',
            });
            titleField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Group title',
                })
            );
            const titleInput = createElement('input', {
                className: 'admin-builder__input',
            });
            titleInput.type = 'text';
            titleInput.placeholder = 'Optional heading for this group';
            titleInput.value = element.content?.title || '';
            titleInput.dataset.field = 'file-group-title';
            titleField.append(titleInput);
            groupContainer.append(titleField);

            const descriptionField = createElement('label', {
                className: 'admin-builder__field',
            });
            descriptionField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Description',
                })
            );
            const descriptionInput = createElement('textarea', {
                className: 'admin-builder__textarea',
            });
            descriptionInput.placeholder = 'Explain what these files contain';
            descriptionInput.value = element.content?.description || '';
            descriptionInput.dataset.field = 'file-group-description';
            descriptionField.append(descriptionInput);
            groupContainer.append(descriptionField);

            const filesWrapper = createElement('div', {
                className: 'admin-builder__group-list',
            });

            const files = ensureArray(element.content.files).length
                ? ensureArray(element.content.files)
                : [createFileState({})];
            element.content.files = files;

            const renderFileItem = (file) => {
                const item = createElement('div', {
                    className: 'admin-builder__group-item',
                });
                item.dataset.groupFileClient = file.clientId;

                const nameField = createElement('label', {
                    className: 'admin-builder__field',
                });
                nameField.append(
                    createElement('span', {
                        className: 'admin-builder__label',
                        textContent: 'File name',
                    })
                );
                const nameInput = createElement('input', {
                    className: 'admin-builder__input',
                });
                nameInput.type = 'text';
                nameInput.placeholder = 'Display name for the file';
                nameInput.value = file.label || '';
                nameInput.dataset.field = 'group-file-label';
                nameField.append(nameInput);
                item.append(nameField);

                const urlField = createElement('label', {
                    className: 'admin-builder__field',
                });
                urlField.append(
                    createElement('span', {
                        className: 'admin-builder__label',
                        textContent: 'File URL',
                    })
                );
                const urlInput = createElement('input', {
                    className: 'admin-builder__input',
                });
                urlInput.type = 'url';
                urlInput.placeholder = 'https://example.com/file.pdf';
                urlInput.value = file.url || '';
                urlInput.dataset.field = 'group-file-url';
                const inputId = `admin-builder-file-${file.clientId}`;
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
                item.append(urlField);

                const itemActions = createElement('div', {
                    className: 'admin-builder__group-actions',
                });
                const removeButton = createElement('button', {
                    className: 'admin-builder__element-remove',
                    textContent: 'Remove file',
                });
                removeButton.type = 'button';
                removeButton.dataset.action = 'group-file-remove';
                itemActions.append(removeButton);
                item.append(itemActions);

                return item;
            };

            const listContainer = createElement('div', {
                className: 'admin-builder__group-items',
            });

            if (!files.length) {
                listContainer.append(
                    createElement('p', {
                        className: 'admin-builder__element-empty',
                        textContent: 'No files in this group yet.',
                    })
                );
            } else {
                files.forEach((file) => {
                    listContainer.append(renderFileItem(file));
                });
            }

            filesWrapper.append(listContainer);
            groupContainer.append(filesWrapper);

            const addFileButton = createElement('button', {
                className: 'admin-builder__button admin-builder__button--ghost',
                textContent: 'Add file',
            });
            addFileButton.type = 'button';
            addFileButton.dataset.action = 'group-file-add';
            groupContainer.append(addFileButton);

            elementNode.append(groupContainer);
        },
        updateField: (element, field, value, fileClientId) => {
            if (field === 'file-group-title') {
                element.content.title = value;
                return true;
            }
            if (field === 'file-group-description') {
                element.content.description = value;
                return true;
            }
            if (field === 'group-file-label' || field === 'group-file-url') {
                if (!Array.isArray(element.content.files)) {
                    element.content.files = [];
                }
                const file = element.content.files.find(
                    (item) => item.clientId === fileClientId
                );
                if (!file) {
                    return false;
                }
                if (field === 'group-file-label') {
                    file.label = value;
                    return true;
                }
                file.url = value;
                return true;
            }
            return false;
        },
        hasContent: (element) =>
            Array.isArray(element.content?.files) &&
            element.content.files.some((file) => file.url && file.url.trim()),
        sanitise: (element, index) => {
            const files = (element.content.files || [])
                .map((file) => {
                    const url = (file.url || '').trim();
                    if (!url) {
                        return null;
                    }
                    const payload = { url };
                    if (file.label && file.label.trim()) {
                        payload.label = file.label.trim();
                    }
                    return payload;
                })
                .filter(Boolean);

            if (!files.length) {
                return null;
            }

            const payload = { files };
            if (element.content.title && element.content.title.trim()) {
                payload.title = element.content.title.trim();
            }
            if (element.content.description && element.content.description.trim()) {
                payload.description = element.content.description.trim();
            }

            return {
                id: element.id || '',
                type: 'file_group',
                order: index + 1,
                content: payload,
            };
        },
    });
})();

(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    if (!utils || !registry) {
        return;
    }

    const { createElement, ensureArray, normaliseString, randomId } = utils;

    registry.register('list', {
        label: 'List',
        addLabel: 'Add list',
        order: 40,
        initialFocusSelector: '.list-item__input',
        create: () => ({
            clientId: randomId(),
            id: '',
            type: 'list',
            content: {
                ordered: false,
                items: [''],
            },
        }),
        fromRaw: ({ id, rawContent }) => ({
            clientId: randomId(),
            id,
            type: 'list',
            content: {
                ordered: Boolean(
                    rawContent.ordered ?? rawContent.Ordered ?? false
                ),
                items: ensureArray(rawContent.items ?? rawContent.Items).map(
                    normaliseString
                ),
            },
        }),
        renderEditor: (elementNode, element) => {
            if (!element.content || typeof element.content !== 'object') {
                element.content = {};
            }

            const items = ensureArray(element.content.items).map((item) =>
                normaliseString(item)
            );
            element.content.items = items.length ? items : [''];

            const orderedField = createElement('label', {
                className: 'admin-builder__field admin-builder__field--inline',
            });
            const orderedCheckbox = createElement('input', {
                className: 'admin-builder__checkbox checkbox__input',
            });
            orderedCheckbox.type = 'checkbox';
            orderedCheckbox.checked = Boolean(element.content?.ordered);
            orderedCheckbox.dataset.field = 'list-ordered';
            orderedField.append(orderedCheckbox);
            orderedField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Display as numbered list',
                })
            );
            elementNode.append(orderedField);

            const itemsField = createElement('div', {
                className:
                    'admin-builder__field admin-builder__field--list-items',
            });
            itemsField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'List items',
                })
            );

            const listItemsWrapper = createElement('div', {
                className: 'list-items',
            });
            const listItemsList = createElement('div', {
                className: 'list-items__list',
            });
            const listItemsActions = createElement('div', {
                className: 'list-items__actions',
            });
            const addListItem = createElement('button', {
                className: 'list-items__add',
                textContent: 'Add item',
                type: 'button',
            });
            listItemsActions.append(addListItem);
            listItemsWrapper.append(listItemsList, listItemsActions);

            const itemsStateField = createElement('textarea', {
                className: 'admin-builder__textarea',
                dataset: { field: 'list-items' },
                style: { display: 'none' },
                value: `json:${JSON.stringify(element.content.items)}`,
                attributes: {
                    'aria-hidden': 'true',
                    tabindex: '-1',
                },
            });

            itemsField.append(listItemsWrapper, itemsStateField);
            elementNode.append(itemsField);

            let listItemDraggingIndex = null;

            const syncStateField = () => {
                itemsStateField.value = `json:${JSON.stringify(
                    element.content.items
                )}`;
            };

            const dispatchChange = () => {
                itemsStateField.dispatchEvent(
                    new Event('input', { bubbles: true })
                );
            };

            const clearListItemDropIndicators = () => {
                listItemsList
                    .querySelectorAll(
                        '.list-item--drop-before, .list-item--drop-after'
                    )
                    .forEach((item) => {
                        item.classList.remove(
                            'list-item--drop-before',
                            'list-item--drop-after'
                        );
                        delete item.dataset.dropPosition;
                    });
            };

            const endListItemDrag = () => {
                listItemDraggingIndex = null;
                listItemsWrapper.classList.remove('list-items--dragging');
                clearListItemDropIndicators();
                listItemsList.querySelectorAll('.list-item').forEach((item) => {
                    item.classList.remove('list-item--dragging');
                    item.draggable = false;
                    delete item.dataset.dropPosition;
                });
            };

            const startListItemDrag = (index, item) => {
                listItemDraggingIndex = index;
                listItemsWrapper.classList.add('list-items--dragging');
                item.classList.add('list-item--dragging');
            };

            const renderListItems = (focusIndex = null) => {
                listItemsList.innerHTML = '';

                const currentItems = element.content.items;
                if (!currentItems.length) {
                    listItemsList.append(
                        createElement('p', {
                            className: 'list-items__empty',
                            textContent: 'No list items yet.',
                        })
                    );
                    return;
                }

                currentItems.forEach((value, itemIndex) => {
                    const row = createElement('div', {
                        className: 'list-item',
                    });
                    row.dataset.index = String(itemIndex);
                    row.draggable = false;

                    const dragHandle = createElement('button', {
                        className: 'list-item__drag-handle',
                        type: 'button',
                        attributes: {
                            'aria-label': 'Reorder list item',
                            title: 'Drag to reorder list item',
                        },
                        html: '<span aria-hidden="true">⋮⋮</span>',
                    });

                    dragHandle.addEventListener('pointerdown', () => {
                        row.draggable = true;
                    });

                    const resetDraggable = () => {
                        if (!row.classList.contains('list-item--dragging')) {
                            row.draggable = false;
                        }
                    };

                    dragHandle.addEventListener('pointerup', resetDraggable);
                    dragHandle.addEventListener('pointercancel', resetDraggable);

                    row.append(dragHandle);

                    const input = createElement('input', {
                        className: 'list-item__input',
                        type: 'text',
                        value,
                    });
                    input.placeholder = 'List item';
                    input.addEventListener('input', (event) => {
                        element.content.items[itemIndex] = event.target.value;
                        syncStateField();
                        dispatchChange();
                    });
                    row.append(input);

                    const controls = createElement('div', {
                        className: 'list-item__controls',
                    });

                    const moveUp = createElement('button', {
                        className: 'list-item__control',
                        textContent: 'Up',
                        type: 'button',
                    });
                    moveUp.disabled = itemIndex === 0;
                    moveUp.addEventListener('click', () => {
                        moveListItem(itemIndex, itemIndex - 1);
                    });
                    controls.append(moveUp);

                    const moveDown = createElement('button', {
                        className: 'list-item__control',
                        textContent: 'Down',
                        type: 'button',
                    });
                    moveDown.disabled = itemIndex === currentItems.length - 1;
                    moveDown.addEventListener('click', () => {
                        moveListItem(itemIndex, itemIndex + 1);
                    });
                    controls.append(moveDown);

                    const removeItem = createElement('button', {
                        className: 'list-item__control',
                        textContent: 'Remove',
                        type: 'button',
                    });
                    removeItem.addEventListener('click', () => {
                        element.content.items.splice(itemIndex, 1);
                        if (!element.content.items.length) {
                            element.content.items = [''];
                        }
                        syncStateField();
                        dispatchChange();
                        renderListItems(
                            Math.max(
                                0,
                                Math.min(
                                    itemIndex,
                                    element.content.items.length - 1
                                )
                            )
                        );
                    });
                    controls.append(removeItem);

                    row.append(controls);

                    row.addEventListener('dragstart', (event) => {
                        startListItemDrag(itemIndex, row);
                        try {
                            event.dataTransfer.effectAllowed = 'move';
                            event.dataTransfer.setData(
                                'text/plain',
                                String(itemIndex)
                            );
                        } catch (error) {
                            /* ignore dataTransfer issues */
                        }
                    });

                    row.addEventListener('dragover', (event) => {
                        if (
                            listItemDraggingIndex === null ||
                            listItemDraggingIndex === itemIndex
                        ) {
                            return;
                        }
                        event.preventDefault();
                        clearListItemDropIndicators();
                        const rect = row.getBoundingClientRect();
                        const offset = event.clientY - rect.top;
                        const insertBefore = offset < rect.height / 2;
                        row.dataset.dropPosition = insertBefore
                            ? 'before'
                            : 'after';
                        row.classList.add(
                            insertBefore
                                ? 'list-item--drop-before'
                                : 'list-item--drop-after'
                        );
                        try {
                            event.dataTransfer.dropEffect = 'move';
                        } catch (error) {
                            /* ignore dataTransfer issues */
                        }
                    });

                    row.addEventListener('dragleave', (event) => {
                        if (!row.contains(event.relatedTarget)) {
                            row.classList.remove(
                                'list-item--drop-before',
                                'list-item--drop-after'
                            );
                            delete row.dataset.dropPosition;
                        }
                    });

                    row.addEventListener('drop', (event) => {
                        if (listItemDraggingIndex === null) {
                            return;
                        }
                        event.preventDefault();
                        event.stopPropagation();
                        const fromIndex = listItemDraggingIndex;
                        const dropPosition =
                            row.dataset.dropPosition === 'before'
                                ? 'before'
                                : 'after';
                        let destination =
                            dropPosition === 'before' ? itemIndex : itemIndex + 1;
                        if (fromIndex < destination) {
                            destination -= 1;
                        }
                        endListItemDrag();
                        moveListItem(fromIndex, destination);
                    });

                    row.addEventListener('dragend', () => {
                        endListItemDrag();
                    });

                    listItemsList.append(row);
                });

                if (focusIndex !== null) {
                    const target = listItemsList.querySelector(
                        `[data-index="${focusIndex}"] .list-item__input`
                    );
                    if (target) {
                        target.focus();
                        target.select();
                    }
                }
            };

            const moveListItem = (fromIndex, toIndex) => {
                if (fromIndex === toIndex) {
                    return;
                }
                const items = element.content.items;
                if (
                    fromIndex < 0 ||
                    fromIndex >= items.length ||
                    toIndex < 0 ||
                    toIndex > items.length
                ) {
                    return;
                }
                const [moved] = items.splice(fromIndex, 1);
                items.splice(toIndex, 0, moved);
                syncStateField();
                dispatchChange();
                renderListItems(toIndex);
            };

            listItemsList.addEventListener('dragover', (event) => {
                if (listItemDraggingIndex === null) {
                    return;
                }
                if (event.target !== listItemsList) {
                    return;
                }
                event.preventDefault();
                clearListItemDropIndicators();
                try {
                    event.dataTransfer.dropEffect = 'move';
                } catch (error) {
                    /* ignore dataTransfer issues */
                }
            });

            listItemsList.addEventListener('drop', (event) => {
                if (listItemDraggingIndex === null) {
                    return;
                }
                if (event.target !== listItemsList) {
                    return;
                }
                event.preventDefault();
                const fromIndex = listItemDraggingIndex;
                endListItemDrag();
                moveListItem(fromIndex, element.content.items.length);
            });

            addListItem.addEventListener('click', () => {
                element.content.items.push('');
                syncStateField();
                dispatchChange();
                renderListItems(element.content.items.length - 1);
            });

            renderListItems(0);
        },
        updateField: (element, field, value) => {
            if (field === 'list-ordered') {
                element.content.ordered = Boolean(value);
                return true;
            }
            if (field === 'list-items') {
                const assignItems = (items) => {
                    element.content.items = ensureArray(items).map(
                        (item) => normaliseString(item)
                    );
                };
                if (Array.isArray(value)) {
                    assignItems(value);
                    return true;
                }
                if (typeof value === 'string') {
                    if (value.startsWith('json:')) {
                        try {
                            const parsed = JSON.parse(value.slice(5));
                            assignItems(parsed);
                            return true;
                        } catch (error) {
                            // fall back to plain text parsing
                        }
                    }
                    const parts = value.replace(/\r/g, '').split('\n');
                    assignItems(parts);
                    return true;
                }
            }
            return false;
        },
        hasContent: (element) => {
            if (!Array.isArray(element.content?.items)) {
                return false;
            }
            return element.content.items.some(
                (item) => item && item.toString().trim()
            );
        },
        sanitise: (element, index) => {
            const sourceItems = Array.isArray(element.content.items)
                ? element.content.items
                : [];
            const items = sourceItems
                .map((item) => {
                    if (typeof item === 'string') {
                        return item.trim();
                    }
                    if (item === null || item === undefined) {
                        return '';
                    }
                    return String(item).trim();
                })
                .filter(Boolean);

            if (!items.length) {
                return null;
            }

            const payload = { items };
            if (element.content.ordered) {
                payload.ordered = true;
            }

            return {
                id: element.id || '',
                type: 'list',
                order: index + 1,
                content: payload,
            };
        },
        preview: (element, parts) => {
            if (!Array.isArray(element.content?.items)) {
                return;
            }
            element.content.items
                .filter((item) => item && item.toString().trim())
                .forEach((item) => {
                    parts.push(item.toString());
                });
        },
    });
})();
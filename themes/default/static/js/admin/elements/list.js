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
        initialFocusSelector: '[data-field="list-items"]',
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

            const itemsField = createElement('label', {
                className: 'admin-builder__field',
            });
            itemsField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'List items',
                })
            );
            const itemsTextarea = createElement('textarea', {
                className: 'admin-builder__textarea',
            });
            const items = ensureArray(element.content?.items);
            itemsTextarea.value = items.join('\n');
            itemsTextarea.placeholder = 'Enter each item on a new line';
            itemsTextarea.dataset.field = 'list-items';
            itemsField.append(itemsTextarea);
            elementNode.append(itemsField);
        },
        updateField: (element, field, value) => {
            if (field === 'list-ordered') {
                element.content.ordered = Boolean(value);
                return true;
            }
            if (field === 'list-items') {
                if (Array.isArray(value)) {
                    element.content.items = value;
                    return true;
                }
                if (typeof value === 'string') {
                    element.content.items = value.replace(/\r/g, '').split('\n');
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
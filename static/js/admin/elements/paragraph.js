(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    if (!utils || !registry) {
        return;
    }

    const { createElement, normaliseString, randomId } = utils;

    registry.register('paragraph', {
        label: 'Paragraph',
        addLabel: 'Add paragraph',
        order: 10,
        initialFocusSelector: 'textarea',
        create: () => ({
            clientId: randomId(),
            id: '',
            type: 'paragraph',
            content: {
                text: '',
            },
        }),
        fromRaw: ({ id, rawContent }) => ({
            clientId: randomId(),
            id,
            type: 'paragraph',
            content: {
                text: normaliseString(rawContent.text ?? rawContent.Text ?? ''),
            },
        }),
        renderEditor: (elementNode, element) => {
            const paragraphField = createElement('label', {
                className: 'admin-builder__field',
            });
            paragraphField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Paragraph text',
                })
            );
            const paragraphTextarea = createElement('textarea', {
                className: 'admin-builder__textarea',
            });
            paragraphTextarea.placeholder =
                'Write the narrative for this part of the section…';
            paragraphTextarea.value = element.content?.text || '';
            paragraphTextarea.dataset.field = 'paragraph-text';
            paragraphField.append(paragraphTextarea);
            elementNode.append(paragraphField);
        },
        updateField: (element, field, value) => {
            if (field === 'paragraph-text') {
                element.content.text = value;
                return true;
            }
            return false;
        },
        hasContent: (element) => Boolean(element.content?.text?.trim()),
        sanitise: (element, index) => ({
            id: element.id || '',
            type: 'paragraph',
            order: index + 1,
            content: {
                text: element.content.text.trim(),
            },
        }),
        preview: (element, parts) => {
            if (element.content?.text) {
                parts.push(element.content.text);
            }
        },
    });
})();
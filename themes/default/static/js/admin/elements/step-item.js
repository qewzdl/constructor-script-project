(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    if (!utils || !registry) {
        return;
    }

    const { createElement, normaliseString, randomId } = utils;

    registry.register('step_item', {
        label: 'Step item',
        addLabel: 'Add step',
        order: 26,
        initialFocusSelector: '[data-field="step-title"]',
        create: () => ({
            clientId: randomId(),
            id: '',
            type: 'step_item',
            content: {
                number: '',
                title: '',
                text: '',
            },
        }),
        fromRaw: ({ id, rawContent }) => ({
            clientId: randomId(),
            id,
            type: 'step_item',
            content: {
                number: normaliseString(rawContent.number ?? rawContent.Number ?? ''),
                title: normaliseString(rawContent.title ?? rawContent.Title ?? ''),
                text: normaliseString(rawContent.text ?? rawContent.Text ?? ''),
            },
        }),
        renderEditor: (elementNode, element) => {
            const numberField = createElement('label', {
                className: 'admin-builder__field',
            });
            numberField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Step number',
                })
            );
            const numberInput = createElement('input', {
                className: 'admin-builder__input',
            });
            numberInput.type = 'text';
            numberInput.placeholder = '01';
            numberInput.value = element.content?.number || '';
            numberInput.dataset.field = 'step-number';
            numberField.append(numberInput);
            elementNode.append(numberField);

            const titleField = createElement('label', {
                className: 'admin-builder__field',
            });
            titleField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Step title',
                })
            );
            const titleInput = createElement('input', {
                className: 'admin-builder__input',
            });
            titleInput.type = 'text';
            titleInput.placeholder = 'Name this step';
            titleInput.value = element.content?.title || '';
            titleInput.dataset.field = 'step-title';
            titleField.append(titleInput);
            elementNode.append(titleField);

            const textField = createElement('label', {
                className: 'admin-builder__field',
            });
            textField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Step text',
                })
            );
            const textInput = createElement('textarea', {
                className: 'admin-builder__textarea',
            });
            textInput.placeholder = 'Describe what happens in this step';
            textInput.value = element.content?.text || '';
            textInput.dataset.field = 'step-text';
            textField.append(textInput);
            elementNode.append(textField);
        },
        updateField: (element, field, value) => {
            if (field === 'step-number') {
                element.content.number = value;
                return true;
            }
            if (field === 'step-title') {
                element.content.title = value;
                return true;
            }
            if (field === 'step-text') {
                element.content.text = value;
                return true;
            }
            return false;
        },
        hasContent: (element) =>
            Boolean(
                element.content?.number?.trim() ||
                    element.content?.title?.trim() ||
                    element.content?.text?.trim()
            ),
        sanitise: (element, index) => {
            const payload = {};
            if (element.content.number && element.content.number.trim()) {
                payload.number = element.content.number.trim();
            }
            if (element.content.title && element.content.title.trim()) {
                payload.title = element.content.title.trim();
            }
            if (element.content.text && element.content.text.trim()) {
                payload.text = element.content.text.trim();
            }
            if (!payload.title || !payload.text) {
                return null;
            }
            return {
                id: element.id || '',
                type: 'step_item',
                order: index + 1,
                content: payload,
            };
        },
        preview: (element, parts) => {
            if (element.content?.number) {
                parts.push(element.content.number);
            }
            if (element.content?.title) {
                parts.push(element.content.title);
            }
            if (element.content?.text) {
                parts.push(element.content.text);
            }
        },
    });
})();

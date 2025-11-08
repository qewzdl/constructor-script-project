(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    if (!utils || !registry) {
        return;
    }

    const { createElement, normaliseString, randomId } = utils;

    const defaultTitle = 'Security';
    const defaultDescription =
        'Change your password regularly and review connected devices.';
    const defaultButton = 'Update password';

    const normaliseContent = (content = {}) => ({
        title: normaliseString(
            content.title ?? content.Title ?? defaultTitle
        ) || defaultTitle,
        description: normaliseString(
            content.description ?? content.Description ?? defaultDescription
        ) || defaultDescription,
        buttonLabel:
            normaliseString(
                content.button_label ??
                    content.buttonLabel ??
                    content.ButtonLabel ??
                    defaultButton
            ) || defaultButton,
    });

    registry.register('profile_security', {
        label: 'Security form',
        addLabel: 'Add security form',
        order: 70,
        create: () => ({
            clientId: randomId(),
            id: '',
            type: 'profile_security',
            content: {
                title: defaultTitle,
                description: defaultDescription,
                button_label: defaultButton,
            },
        }),
        fromRaw: ({ id, rawContent }) => ({
            clientId: randomId(),
            id,
            type: 'profile_security',
            content: {
                title: normaliseString(
                    rawContent?.title ?? rawContent?.Title ?? defaultTitle
                ) || defaultTitle,
                description: normaliseString(
                    rawContent?.description ??
                        rawContent?.Description ??
                        defaultDescription
                ) || defaultDescription,
                button_label:
                    normaliseString(
                        rawContent?.button_label ??
                            rawContent?.buttonLabel ??
                            rawContent?.ButtonLabel ??
                            defaultButton
                    ) || defaultButton,
            },
        }),
        renderEditor: (elementNode, element) => {
            const titleField = createElement('label', {
                className: 'admin-builder__field',
            });
            titleField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Heading',
                })
            );
            const titleInput = createElement('input', {
                className: 'admin-builder__input',
                type: 'text',
                value: element.content?.title || defaultTitle,
            });
            titleInput.dataset.field = 'profile-security-title';
            titleField.append(titleInput);

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
            descriptionInput.dataset.field = 'profile-security-description';
            descriptionInput.placeholder =
                'Describe your security guidance for users…';
            descriptionInput.value =
                element.content?.description || defaultDescription;
            descriptionField.append(descriptionInput);

            const buttonField = createElement('label', {
                className: 'admin-builder__field',
            });
            buttonField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Button label',
                })
            );
            const buttonInput = createElement('input', {
                className: 'admin-builder__input',
                type: 'text',
            });
            buttonInput.dataset.field = 'profile-security-button';
            buttonInput.value =
                element.content?.button_label || defaultButton;
            buttonField.append(buttonInput);

            elementNode.append(titleField, descriptionField, buttonField);
        },
        updateField: (element, field, value) => {
            if (!element.content) {
                element.content = {};
            }
            switch (field) {
                case 'profile-security-title':
                    element.content.title = value;
                    return true;
                case 'profile-security-description':
                    element.content.description = value;
                    return true;
                case 'profile-security-button':
                    element.content.button_label = value;
                    return true;
                default:
                    return false;
            }
        },
        hasContent: () => true,
        sanitise: (element, index) => {
            const content = normaliseContent(element.content);
            return {
                id: element.id || '',
                type: 'profile_security',
                order: index + 1,
                content: {
                    title: content.title,
                    description: content.description,
                    button_label: content.buttonLabel,
                },
            };
        },
        preview: (element, parts) => {
            const content = normaliseContent(element.content);
            parts.push(`${content.title} – ${content.buttonLabel}`);
        },
    });
})();

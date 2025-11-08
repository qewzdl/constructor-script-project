(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    if (!utils || !registry) {
        return;
    }

    const { createElement, normaliseString, randomId } = utils;

    const defaultTitle = 'Account details';
    const defaultDescription =
        'The information below appears in comments and author bylines.';
    const defaultButton = 'Save changes';

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

    registry.register('profile_account_details', {
        label: 'Account details form',
        addLabel: 'Add account details form',
        order: 60,
        create: () => ({
            clientId: randomId(),
            id: '',
            type: 'profile_account_details',
            content: {
                title: defaultTitle,
                description: defaultDescription,
                button_label: defaultButton,
            },
        }),
        fromRaw: ({ id, rawContent }) => ({
            clientId: randomId(),
            id,
            type: 'profile_account_details',
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
            titleInput.dataset.field = 'profile-account-title';
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
            descriptionInput.dataset.field = 'profile-account-description';
            descriptionInput.placeholder =
                'Explain how profile information is displayed…';
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
            buttonInput.dataset.field = 'profile-account-button';
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
                case 'profile-account-title':
                    element.content.title = value;
                    return true;
                case 'profile-account-description':
                    element.content.description = value;
                    return true;
                case 'profile-account-button':
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
                type: 'profile_account_details',
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

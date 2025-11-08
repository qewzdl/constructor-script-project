(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    if (!utils || !registry) {
        return;
    }

    const { createElement, normaliseString, randomId } = utils;

    const defaultTitle = 'Courses';
    const defaultDescription =
        "Review the learning packages currently available to your account.";
    const defaultEmptyMessage = "You don't have any courses yet.";

    const normaliseContent = (content = {}) => ({
        title:
            normaliseString(content.title ?? content.Title ?? defaultTitle) ||
            defaultTitle,
        description:
            normaliseString(
                content.description ??
                    content.Description ??
                    defaultDescription
            ) || defaultDescription,
        emptyMessage:
            normaliseString(
                content.empty_message ??
                    content.emptyMessage ??
                    content.EmptyMessage ??
                    defaultEmptyMessage
            ) || defaultEmptyMessage,
    });

    registry.register('profile_courses', {
        label: 'Courses list',
        addLabel: 'Add courses list',
        order: 80,
        create: () => ({
            clientId: randomId(),
            id: '',
            type: 'profile_courses',
            content: {
                title: defaultTitle,
                description: defaultDescription,
                empty_message: defaultEmptyMessage,
            },
        }),
        fromRaw: ({ id, rawContent }) => ({
            clientId: randomId(),
            id,
            type: 'profile_courses',
            content: {
                title:
                    normaliseString(
                        rawContent?.title ?? rawContent?.Title ?? defaultTitle
                    ) || defaultTitle,
                description:
                    normaliseString(
                        rawContent?.description ??
                            rawContent?.Description ??
                            defaultDescription
                    ) || defaultDescription,
                empty_message:
                    normaliseString(
                        rawContent?.empty_message ??
                            rawContent?.emptyMessage ??
                            rawContent?.EmptyMessage ??
                            defaultEmptyMessage
                    ) || defaultEmptyMessage,
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
            titleInput.dataset.field = 'profile-courses-title';
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
            descriptionInput.dataset.field = 'profile-courses-description';
            descriptionInput.placeholder =
                'Explain what learners will find in this list…';
            descriptionInput.value =
                element.content?.description || defaultDescription;
            descriptionField.append(descriptionInput);

            const emptyField = createElement('label', {
                className: 'admin-builder__field',
            });
            emptyField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Empty state message',
                })
            );
            const emptyInput = createElement('input', {
                className: 'admin-builder__input',
                type: 'text',
                value: element.content?.empty_message || defaultEmptyMessage,
            });
            emptyInput.dataset.field = 'profile-courses-empty';
            emptyField.append(emptyInput);

            elementNode.append(titleField, descriptionField, emptyField);
        },
        updateField: (element, field, value) => {
            if (!element.content) {
                element.content = {};
            }
            switch (field) {
                case 'profile-courses-title':
                    element.content.title = value;
                    return true;
                case 'profile-courses-description':
                    element.content.description = value;
                    return true;
                case 'profile-courses-empty':
                    element.content.empty_message = value;
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
                type: 'profile_courses',
                order: index + 1,
                content: {
                    title: content.title,
                    description: content.description,
                    empty_message: content.emptyMessage,
                },
            };
        },
        preview: (element, parts) => {
            const content = normaliseContent(element.content);
            parts.push(`${content.title} – ${content.emptyMessage}`);
        },
    });
})();

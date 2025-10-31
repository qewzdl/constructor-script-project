(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    if (!utils || !registry) {
        return;
    }

    const { createElement, normaliseString, randomId } = utils;

    const normaliseHeading = (value) => {
        const heading = normaliseString(value).toLowerCase();
        switch (heading) {
            case 'h1':
            case 'h2':
            case 'h3':
            case 'h4':
            case 'h5':
            case 'h6':
                return heading;
            default:
                return 'h2';
        }
    };

    const normaliseSearchType = (value) => {
        const type = normaliseString(value).toLowerCase();
        switch (type) {
            case 'title':
            case 'content':
            case 'tag':
            case 'author':
                return type;
            default:
                return 'all';
        }
    };

    const toBoolean = (value, fallback = false) => {
        if (typeof value === 'boolean') {
            return value;
        }
        if (typeof value === 'string') {
            const trimmed = value.trim().toLowerCase();
            if (!trimmed) {
                return fallback;
            }
            if (['true', '1', 'yes', 'on'].includes(trimmed)) {
                return true;
            }
            if (['false', '0', 'no', 'off'].includes(trimmed)) {
                return false;
            }
            return fallback;
        }
        if (typeof value === 'number') {
            return value !== 0;
        }
        return fallback;
    };

    registry.register('search', {
        label: 'Search block',
        addLabel: 'Add search block',
        order: 45,
        initialFocusSelector: '[data-field="search-title"]',
        create: () => ({
            clientId: randomId(),
            id: '',
            type: 'search',
            content: {
                title: 'Search',
                description: '',
                placeholder: 'Start typing to search',
                submitLabel: 'Search',
                filterLabel: 'Filter by',
                action: '/search',
                showFilters: true,
                heading: 'h2',
                hint: 'Use the search form above to explore the knowledge base.',
                default_type: 'all',
            },
        }),
        fromRaw: ({ id, rawContent }) => {
            const content = rawContent || {};
            return {
                clientId: randomId(),
                id,
                type: 'search',
                content: {
                    title: normaliseString(content.title ?? content.Title ?? 'Search'),
                    description: normaliseString(
                        content.description ?? content.Description ?? ''
                    ),
                    placeholder: normaliseString(
                        content.placeholder ?? content.Placeholder ?? 'Start typing to search'
                    ),
                    submitLabel: normaliseString(
                        content.submit_label ?? content.SubmitLabel ?? 'Search'
                    ),
                    filterLabel: normaliseString(
                        content.filter_label ?? content.FilterLabel ?? 'Filter by'
                    ),
                    action: normaliseString(content.action ?? content.Action ?? '/search'),
                    showFilters: toBoolean(
                        content.show_filters ?? content.ShowFilters ?? true,
                        true
                    ),
                    heading: normaliseHeading(content.heading ?? content.Heading ?? 'h2'),
                    hint: normaliseString(
                        content.hint ?? content.Hint ?? 'Use the search form above to explore the knowledge base.'
                    ),
                    default_type: normaliseSearchType(
                        content.default_type ?? content.DefaultType ?? 'all'
                    ),
                },
            };
        },
        renderEditor: (elementNode, element) => {
            const ensureContent = () => {
                if (!element.content || typeof element.content !== 'object') {
                    element.content = {};
                }
            };
            ensureContent();

            const fields = [
                {
                    label: 'Block title',
                    type: 'input',
                    dataset: 'search-title',
                    placeholder: 'Give this search block a title',
                    value: element.content.title || '',
                    handler: (value) => {
                        ensureContent();
                        element.content.title = value;
                    },
                },
                {
                    label: 'Description (optional)',
                    type: 'textarea',
                    dataset: 'search-description',
                    placeholder: 'Describe what people can find with this search.',
                    value: element.content.description || '',
                    handler: (value) => {
                        ensureContent();
                        element.content.description = value;
                    },
                },
                {
                    label: 'Placeholder text',
                    type: 'input',
                    dataset: 'search-placeholder',
                    placeholder: 'Start typing to search',
                    value: element.content.placeholder || '',
                    handler: (value) => {
                        ensureContent();
                        element.content.placeholder = value;
                    },
                },
                {
                    label: 'Submit button label',
                    type: 'input',
                    dataset: 'search-submit-label',
                    placeholder: 'Search',
                    value: element.content.submitLabel || '',
                    handler: (value) => {
                        ensureContent();
                        element.content.submitLabel = value;
                    },
                },
                {
                    label: 'Filter label',
                    type: 'input',
                    dataset: 'search-filter-label',
                    placeholder: 'Filter by',
                    value: element.content.filterLabel || '',
                    handler: (value) => {
                        ensureContent();
                        element.content.filterLabel = value;
                    },
                },
                {
                    label: 'Form action URL',
                    type: 'input',
                    dataset: 'search-action',
                    placeholder: '/search',
                    value: element.content.action || '',
                    handler: (value) => {
                        ensureContent();
                        element.content.action = value;
                    },
                },
            ];

            fields.forEach(({ label, type, dataset, placeholder, value, handler }) => {
                const field = createElement('label', {
                    className: 'admin-builder__field',
                });
                field.append(
                    createElement('span', {
                        className: 'admin-builder__label',
                        textContent: label,
                    })
                );
                const control =
                    type === 'textarea'
                        ? createElement('textarea', {
                              className: 'admin-builder__textarea',
                          })
                        : createElement('input', {
                              className: 'admin-builder__input',
                              type: 'text',
                          });
                control.dataset.field = dataset;
                if (placeholder) {
                    control.placeholder = placeholder;
                }
                if (typeof value === 'string') {
                    control.value = value;
                }
                field.append(control);
                elementNode.append(field);

                control.addEventListener('input', (event) => {
                    handler(event.target.value);
                });
            });

            const showFiltersField = createElement('label', {
                className: 'admin-builder__field admin-builder__field--inline',
            });
            const showFiltersCheckbox = createElement('input', {
                className: 'admin-builder__checkbox checkbox__input',
            });
            showFiltersCheckbox.type = 'checkbox';
            showFiltersCheckbox.checked = Boolean(element.content.showFilters);
            showFiltersCheckbox.dataset.field = 'search-show-filters';
            showFiltersField.append(showFiltersCheckbox);
            showFiltersField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Allow filtering by title, content, tag, or author',
                })
            );
            elementNode.append(showFiltersField);

            const headingField = createElement('label', {
                className: 'admin-builder__field',
            });
            headingField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Heading level',
                })
            );
            const headingSelect = createElement('select', {
                className: 'admin-builder__input',
            });
            headingSelect.dataset.field = 'search-heading';
            const headingOptions = ['h1', 'h2', 'h3', 'h4', 'h5', 'h6'];
            headingOptions.forEach((option) => {
                const optionElement = createElement('option', {
                    textContent: option.toUpperCase(),
                });
                optionElement.value = option;
                if (option === (element.content.heading || 'h2')) {
                    optionElement.selected = true;
                }
                headingSelect.append(optionElement);
            });
            headingField.append(headingSelect);
            elementNode.append(headingField);

            const hintField = createElement('label', {
                className: 'admin-builder__field',
            });
            hintField.append(
                createElement('span', {
                    className: 'admin-builder__label',
                    textContent: 'Hint text (displayed when no query is provided)',
                })
            );
            const hintTextarea = createElement('textarea', {
                className: 'admin-builder__textarea',
            });
            hintTextarea.dataset.field = 'search-hint';
            hintTextarea.placeholder = 'Use the search form above to explore the knowledge base.';
            hintTextarea.value = element.content.hint || '';
            hintField.append(hintTextarea);
            elementNode.append(hintField);

            showFiltersCheckbox.addEventListener('input', (event) => {
                ensureContent();
                element.content.showFilters = event.target.checked;
            });
            headingSelect.addEventListener('change', (event) => {
                ensureContent();
                element.content.heading = event.target.value;
            });
            hintTextarea.addEventListener('input', (event) => {
                ensureContent();
                element.content.hint = event.target.value;
            });
        },
        updateField: (element, field, value) => {
            if (!element.content || typeof element.content !== 'object') {
                element.content = {};
            }
            switch (field) {
                case 'search-title':
                    element.content.title = normaliseString(value);
                    return true;
                case 'search-description':
                    element.content.description = normaliseString(value);
                    return true;
                case 'search-placeholder':
                    element.content.placeholder = normaliseString(value);
                    return true;
                case 'search-submit-label':
                    element.content.submitLabel = normaliseString(value);
                    return true;
                case 'search-filter-label':
                    element.content.filterLabel = normaliseString(value);
                    return true;
                case 'search-action':
                    element.content.action = normaliseString(value);
                    return true;
                case 'search-show-filters':
                    element.content.showFilters = Boolean(value);
                    return true;
                case 'search-heading':
                    element.content.heading = normaliseHeading(value);
                    return true;
                case 'search-hint':
                    element.content.hint = normaliseString(value);
                    return true;
                default:
                    return false;
            }
        },
        hasContent: () => true,
        sanitise: (element, index) => {
            const content = element.content || {};
            const payload = {};

            const assignIfValue = (key, sourceKey) => {
                const raw = content[sourceKey];
                if (typeof raw === 'string') {
                    const trimmed = raw.trim();
                    if (trimmed) {
                        payload[key] = trimmed;
                    }
                }
            };

            assignIfValue('title', 'title');
            assignIfValue('description', 'description');
            assignIfValue('placeholder', 'placeholder');
            assignIfValue('submit_label', 'submitLabel');
            assignIfValue('filter_label', 'filterLabel');
            assignIfValue('action', 'action');
            assignIfValue('hint', 'hint');

            payload.show_filters = Boolean(content.showFilters);

            const heading = normaliseHeading(content.heading);
            if (heading) {
                payload.heading = heading;
            }

            const defaultType = normaliseSearchType(content.default_type);
            if (defaultType && defaultType !== 'all') {
                payload.default_type = defaultType;
            }

            if (!payload.placeholder) {
                payload.placeholder = 'Start typing to search';
            }

            if (!payload.submit_label) {
                payload.submit_label = 'Search';
            }

            if (!payload.filter_label) {
                payload.filter_label = 'Filter by';
            }

            if (!payload.action) {
                payload.action = '/search';
            }

            if (!payload.hint) {
                payload.hint = 'Use the search form above to explore the knowledge base.';
            }

            if (!payload.title) {
                payload.title = 'Search';
            }

            if (!payload.default_type) {
                payload.default_type = 'all';
            }

            return {
                id: element.id || '',
                type: 'search',
                order: index + 1,
                content: payload,
            };
        },
        preview: (element, parts) => {
            if (element.content?.title) {
                parts.push(element.content.title);
            }
        },
    });
})();

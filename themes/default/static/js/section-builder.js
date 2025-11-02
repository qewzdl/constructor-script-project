(() => {
    const builders = new WeakMap();

    const generateId = () =>
        window.crypto && typeof window.crypto.randomUUID === 'function'
            ? window.crypto.randomUUID()
            : `section-${Date.now().toString(36)}-${Math.random()
                  .toString(36)
                  .slice(2, 10)}`;

    const cloneDeep = (value) => {
        if (typeof window.structuredClone === 'function') {
            try {
                return window.structuredClone(value);
            } catch (error) {
                // fall back to JSON clone
            }
        }
        try {
            return JSON.parse(JSON.stringify(value));
        } catch (error) {
            if (Array.isArray(value)) {
                return value.slice();
            }
            if (value && typeof value === 'object') {
                return Object.assign({}, value);
            }
            return value;
        }
    };

    const capitalise = (value) => {
        if (typeof value !== 'string' || !value.length) {
            return '';
        }
        return value.charAt(0).toUpperCase() + value.slice(1);
    };

    const clampListLimit = (value, defaults = {}) => {
        const parsed = Number.parseInt(value, 10);
        let limit = Number.isFinite(parsed) ? parsed : 0;
        const defaultValue = Number.isFinite(defaults.defaultValue)
            ? defaults.defaultValue
            : Number.parseInt(defaults.defaultValue, 10);
        const minValue = Number.isFinite(defaults.min)
            ? defaults.min
            : Number.parseInt(defaults.min, 10);
        const maxValue = Number.isFinite(defaults.max)
            ? defaults.max
            : Number.parseInt(defaults.max, 10);

        if (limit <= 0 && Number.isFinite(defaultValue)) {
            limit = defaultValue;
        }
        if (Number.isFinite(minValue) && limit < minValue) {
            limit = minValue;
        }
        if (Number.isFinite(maxValue) && limit > maxValue) {
            limit = maxValue;
        }
        if (!Number.isFinite(limit) || limit <= 0) {
            limit = 1;
        }
        return limit;
    };

    const clampPostListLimit = (value) =>
        clampListLimit(value, { defaultValue: 6, min: 1, max: 24 });

    const clampCategoriesListLimit = (value) =>
        clampListLimit(value, { defaultValue: 10, min: 1, max: 30 });

    const clampSectionLimitByType = (type, value) => {
        if (type === 'posts_list') {
            return clampPostListLimit(value);
        }
        if (type === 'categories_list') {
            return clampCategoriesListLimit(value);
        }
        const parsed = Number.parseInt(value, 10);
        return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
    };

    const normaliseLimitDefault = (limitDefinition) => {
        if (!limitDefinition) {
            return 0;
        }
        if (Number.isFinite(limitDefinition.default)) {
            return limitDefinition.default;
        }
        const parsed = Number.parseInt(limitDefinition.default, 10);
        return Number.isFinite(parsed) ? parsed : 0;
    };

    const createSectionTypeRegistry = () => {
        const definitions = new Map();
        let defaultType = '';

        const normaliseType = (type) =>
            typeof type === 'string' ? type.trim().toLowerCase() : '';

        const register = (type, definition = {}) => {
            const normalised = normaliseType(type);
            if (!normalised) {
                return;
            }

            const entry = {
                type: normalised,
                label:
                    typeof definition.label === 'string' && definition.label.trim()
                        ? definition.label.trim()
                        : capitalise(normalised),
                order: Number.isFinite(definition.order) ? definition.order : 0,
                description:
                    typeof definition.description === 'string'
                        ? definition.description.trim()
                        : '',
                supportsElements:
                    definition.supportsElements === undefined
                        ? true
                        : Boolean(definition.supportsElements),
                validate:
                    typeof definition.validate === 'function'
                        ? definition.validate
                        : null,
                settings:
                    definition.settings && typeof definition.settings === 'object'
                        ? definition.settings
                        : undefined,
            };

            definitions.set(normalised, entry);

            if (!defaultType) {
                defaultType = normalised;
            }
        };

        const ensure = (type) => {
            const normalised = normaliseType(type);
            if (definitions.has(normalised)) {
                return normalised;
            }
            return defaultType || 'standard';
        };

        const get = (type) => definitions.get(normaliseType(type));

        const list = () =>
            Array.from(definitions.values()).sort((a, b) => a.order - b.order);

        return {
            register,
            ensure,
            get,
            list,
            getDefault: () => defaultType || 'standard',
        };
    };

    const sectionTypeRegistry = createSectionTypeRegistry();

    const parseDefinitionJSON = (elementId) => {
        if (typeof document === 'undefined') {
            return null;
        }
        const node = document.getElementById(elementId);
        if (!node) {
            return null;
        }
        const raw = node.textContent || node.innerText || '';
        if (!raw.trim()) {
            return null;
        }
        try {
            return JSON.parse(raw);
        } catch (error) {
            console.error('Failed to parse section definitions', error);
            return null;
        }
    };

    const initialSectionDefinitions = parseDefinitionJSON(
        'section-definitions-data'
    );
    if (initialSectionDefinitions && typeof initialSectionDefinitions === 'object') {
        Object.entries(initialSectionDefinitions).forEach(
            ([type, definition]) => {
                sectionTypeRegistry.register(type, definition);
            }
        );
    }

    sectionTypeRegistry.register('standard', {
        label: 'Standard section',
        order: 0,
        supportsElements: true,
        description:
            'Flexible content area that supports paragraphs, images, lists, and more.',
        validate: (section) => {
            if (!Array.isArray(section.elements) || !section.elements.length) {
                return 'must include at least one content block.';
            }
            return null;
        },
    });

    sectionTypeRegistry.register('hero', {
        label: 'Hero section',
        order: 10,
        supportsElements: false,
        description:
            'Large introductory section with a headline and optional image. Does not allow additional content blocks.',
    });

    sectionTypeRegistry.register('grid', {
        label: 'Grid section',
        order: 15,
        supportsElements: true,
        description:
            'Displays content blocks in a responsive grid, ideal for highlighting features or resources side by side.',
        validate: (section) => {
            if (!Array.isArray(section?.elements) || section.elements.length < 2) {
                return 'must include at least two content blocks to form a grid.';
            }
            return null;
        },
    });

    sectionTypeRegistry.register('categories_list', {
        label: 'Categories list',
        order: 18,
        supportsElements: false,
        description:
            'Displays a curated list of blog categories to help readers explore relevant topics.',
        settings: {
            limit: { default: 10, min: 1, max: 30 },
        },
        validate: (section) => {
            const limit = clampSectionLimitByType(
                'categories_list',
                section?.limit ?? section?.Limit
            );
            if (!limit) {
                return 'requires a valid category limit.';
            }
            return null;
        },
    });

    sectionTypeRegistry.register('posts_list', {
        label: 'Posts list',
        order: 20,
        supportsElements: false,
        description:
            'Automatically displays the most recent blog posts without adding manual elements.',
        settings: {
            limit: { default: 6, min: 1, max: 24 },
        },
        validate: (section) => {
            const limit = clampSectionLimitByType(
                'posts_list',
                section?.limit ?? section?.Limit
            );
            if (!limit) {
                return 'requires a valid post limit.';
            }
            return null;
        },
    });

    const createElement = (tag, options = {}) => {
        const element = document.createElement(tag);
        if (options.className) {
            element.className = options.className;
        }
        if (options.textContent !== undefined) {
            element.textContent = options.textContent;
        }
        if (options.html !== undefined) {
            element.innerHTML = options.html;
        }
        if (options.type) {
            element.type = options.type;
        }
        if (options.value !== undefined) {
            element.value = options.value;
        }
        if (options.attrs) {
            Object.entries(options.attrs).forEach(([key, value]) => {
                if (value !== undefined && value !== null) {
                    element.setAttribute(key, value);
                }
            });
        }
        return element;
    };

    const createMediaBrowseActions = (input) => {
        if (!(input instanceof HTMLElement)) {
            return null;
        }

        const actions = createElement('div', {
            className: 'section-field__actions',
        });
        const button = createElement('button', {
            className: 'section-field__media-button',
            type: 'button',
            textContent: 'Browse uploads',
        });
        button.dataset.action = 'open-media-library';
        if (input.id) {
            button.dataset.mediaTarget = `#${input.id}`;
        }
        actions.appendChild(button);
        return actions;
    };

    const normaliseImageContent = (content = {}) => ({
        url: content.url || content.URL || '',
        alt: content.alt || content.Alt || '',
        caption: content.caption || content.Caption || '',
    });

    const normaliseImageGroupContent = (content = {}) => {
        const imagesSource = content.images || content.Images;
        const images = Array.isArray(imagesSource) ? imagesSource : [];
        return {
            layout: content.layout || content.Layout || 'grid',
            images: images.map((image) => normaliseImageContent(image)),
        };
    };

    const normaliseListContent = (content = {}) => {
        const itemsSource = content.items || content.Items;
        const items = Array.isArray(itemsSource)
            ? itemsSource.map((item) => {
                  if (typeof item === 'string') {
                      return item;
                  }
                  if (item === null || item === undefined) {
                      return '';
                  }
                  return String(item);
              })
            : [];
        const orderedValue = content.ordered ?? content.Ordered ?? false;
        const ordered =
            typeof orderedValue === 'string'
                ? orderedValue.toLowerCase() === 'true'
                : Boolean(orderedValue);
        return {
            ordered,
            items,
        };
    };

    const normaliseElement = (element) => {
        if (!element || typeof element !== 'object') {
            return {
                id: generateId(),
                type: 'paragraph',
                content: { text: '' },
            };
        }
        const type = element.type || element.Type || 'paragraph';
        let content = element.content || element.Content || {};
        if (type === 'paragraph') {
            content = { text: content.text || content.Text || '' };
        } else if (type === 'image') {
            content = normaliseImageContent(content);
        } else if (type === 'image_group') {
            content = normaliseImageGroupContent(content);
        } else if (type === 'list') {
            content = normaliseListContent(content);
        } else {
            content = cloneDeep(content);
        }
        return {
            id: element.id || element.ID || generateId(),
            type,
            content,
        };
    };

    const normaliseSection = (section) => {
        const defaultType = sectionTypeRegistry.getDefault();
        if (!section || typeof section !== 'object') {
            const ensuredType = sectionTypeRegistry.ensure(defaultType);
            return {
                id: generateId(),
                type: ensuredType,
                title: '',
                image: '',
                elements: [],
            };
        }

        const type = sectionTypeRegistry.ensure(section.type || section.Type || defaultType);
        const definition = sectionTypeRegistry.get(type);
        const elementsSource = section.elements || section.Elements || [];
        const allowElements = definition?.supportsElements !== false;
        const elements = allowElements && Array.isArray(elementsSource)
            ? elementsSource.map((item) => normaliseElement(item))
            : [];

        const limitDefinition = definition?.settings?.limit;
        const hasLimit =
            Boolean(limitDefinition) || type === 'posts_list' || type === 'categories_list';
        const limitSource =
            section.limit ?? section.Limit ?? normaliseLimitDefault(limitDefinition);
        const limit = hasLimit
            ? clampSectionLimitByType(type, limitSource)
            : Number.isFinite(limitSource)
              ? limitSource
              : 0;

        const normalised = {
            id: section.id || section.ID || generateId(),
            type,
            title: section.title || section.Title || '',
            image: section.image || section.Image || '',
            elements,
            limit,
        };

        if (type === 'grid') {
            const styleGridItemsSource =
                section.styleGridItems ??
                section.StyleGridItems ??
                section.style_grid_items ??
                section.Style_grid_items;

            let styleGridItems = true;
            if (styleGridItemsSource !== undefined && styleGridItemsSource !== null) {
                if (typeof styleGridItemsSource === 'string') {
                    const normalisedValue = styleGridItemsSource.trim().toLowerCase();
                    styleGridItems =
                        normalisedValue === 'true' ||
                        normalisedValue === '1' ||
                        normalisedValue === 'yes';
                } else {
                    styleGridItems = Boolean(styleGridItemsSource);
                }
            }

            normalised.styleGridItems = styleGridItems;
        }

        return normalised;
    };

    const createEmptySection = (type = sectionTypeRegistry.getDefault()) => {
        const ensuredType = sectionTypeRegistry.ensure(type);
        const definition = sectionTypeRegistry.get(ensuredType);
        const limitDefinition = definition?.settings?.limit;
        let limit = 0;
        if (limitDefinition) {
            limit = clampSectionLimitByType(
                ensuredType,
                normaliseLimitDefault(limitDefinition)
            );
        } else if (ensuredType === 'posts_list') {
            limit = clampPostListLimit(6);
        } else if (ensuredType === 'categories_list') {
            limit = clampCategoriesListLimit(10);
        }
        const section = {
            id: generateId(),
            type: ensuredType,
            title: '',
            image: '',
            elements: [],
            limit,
        };
        if (ensuredType === 'grid') {
            section.styleGridItems = true;
        }
        return section;
    };

    const createElementByType = (type) => {
        if (type === 'image') {
            return {
                id: generateId(),
                type: 'image',
                content: { url: '', alt: '', caption: '' },
            };
        }
        if (type === 'image_group') {
            return {
                id: generateId(),
                type: 'image_group',
                content: { layout: 'grid', images: [] },
            };
        }
        if (type === 'list') {
            return {
                id: generateId(),
                type: 'list',
                content: { ordered: false, items: [''] },
            };
        }
        if (type === 'search') {
            return {
                id: generateId(),
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
            };
        }
        return { id: generateId(), type: 'paragraph', content: { text: '' } };
    };

    const serialiseElementContent = (element) => {
        if (!element || typeof element !== 'object') {
            return null;
        }
        if (element.type === 'paragraph') {
            const text = element.content?.text || '';
            return text.trim() ? { text: text } : null;
        }
        if (element.type === 'image') {
            const url = element.content?.url || '';
            if (!url.trim()) {
                return null;
            }
            const payload = { url: url.trim() };
            const alt = element.content?.alt || '';
            if (alt.trim()) {
                payload.alt = alt.trim();
            }
            const caption = element.content?.caption || '';
            if (caption.trim()) {
                payload.caption = caption.trim();
            }
            return payload;
        }
        if (element.type === 'image_group') {
            const layout = element.content?.layout || '';
            const imagesSource = element.content?.images;
            const images = Array.isArray(imagesSource) ? imagesSource : [];
            const serialisedImages = images
                .map((image) => {
                    if (!image) {
                        return null;
                    }
                    const url = image.url || image.URL || '';
                    if (!url.trim()) {
                        return null;
                    }
                    const payload = { url: url.trim() };
                    const alt = image.alt || image.Alt || '';
                    if (alt.trim()) {
                        payload.alt = alt.trim();
                    }
                    const caption = image.caption || image.Caption || '';
                    if (caption.trim()) {
                        payload.caption = caption.trim();
                    }
                    return payload;
                })
                .filter(Boolean);
            if (!serialisedImages.length) {
                return null;
            }
            return {
                layout: (layout || 'grid').trim() || 'grid',
                images: serialisedImages,
            };
        }
        if (element.type === 'list') {
            const itemsSource = Array.isArray(element.content?.items)
                ? element.content.items
                : [];
            const items = itemsSource
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
            if (element.content?.ordered) {
                payload.ordered = true;
            }
            return payload;
        }
        if (element.type === 'search') {
            const content = element.content || {};
            const payload = {};

            const assignString = (key, value) => {
                if (typeof value === 'string') {
                    const trimmed = value.trim();
                    if (trimmed) {
                        payload[key] = trimmed;
                    }
                }
            };

            assignString('title', content.title);
            assignString('description', content.description);
            assignString('placeholder', content.placeholder);
            assignString('submit_label', content.submitLabel);
            assignString('filter_label', content.filterLabel);
            assignString('action', content.action);
            assignString('hint', content.hint);

            payload.show_filters = Boolean(content.showFilters);

            const heading = (content.heading || '').toString().trim().toLowerCase();
            if (['h1', 'h2', 'h3', 'h4', 'h5', 'h6'].includes(heading)) {
                payload.heading = heading;
            }

            const defaultType = (content.default_type || '').toString().trim().toLowerCase();
            if (['title', 'content', 'tag', 'author'].includes(defaultType)) {
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

            return payload;
        }
        return cloneDeep(element.content);
    };

    const buildSubmitSections = (sections) => {
        if (!Array.isArray(sections)) {
            return [];
        }
        return sections.map((section, index) => {
            const type = sectionTypeRegistry.ensure(section.type);
            const definition = sectionTypeRegistry.get(type);
            const allowElements = definition?.supportsElements !== false;
            const elements = allowElements && Array.isArray(section.elements)
                ? section.elements
                      .map((element, elementIndex) => {
                          const content = serialiseElementContent(element);
                          if (!content) {
                              return null;
                          }
                          return {
                              id: element.id || generateId(),
                              type: element.type,
                              order: elementIndex + 1,
                              content,
                          };
                      })
                      .filter(Boolean)
                : [];
            const payload = {
                id: section.id || generateId(),
                type,
                title: (section.title || '').trim(),
                image: (section.image || '').trim(),
                order: index + 1,
                elements,
            };
            if (type === 'grid') {
                payload.style_grid_items = section.styleGridItems !== false;
            }
            const hasLimit =
                Boolean(definition?.settings?.limit) ||
                type === 'posts_list' ||
                type === 'categories_list';
            if (hasLimit) {
                payload.limit = clampSectionLimitByType(
                    type,
                    section.limit ?? section.Limit ?? payload.limit
                );
            }
            return payload;
        });
    };

    const validateSections = (sections) => {
        if (!Array.isArray(sections) || !sections.length) {
            return null;
        }

        for (let i = 0; i < sections.length; i += 1) {
            const section = sections[i] || {};
            const type = sectionTypeRegistry.ensure(section.type);
            const definition = sectionTypeRegistry.get(type);

            if (definition?.validate) {
                const message = definition.validate(section, i);
                if (typeof message === 'string' && message.trim()) {
                    return `Section ${i + 1} ${message.trim()}`;
                }
            }

            const allowElements = definition?.supportsElements !== false;
            if (!allowElements) {
                continue;
            }

            if (!Array.isArray(section.elements) || !section.elements.length) {
                return `Section ${i + 1} must include at least one element.`;
            }

            for (let j = 0; j < section.elements.length; j += 1) {
                const element = section.elements[j];
                if (!element || !element.type) {
                    return `Section ${i + 1}, element ${j + 1} is invalid.`;
                }

                if (element.type === 'paragraph') {
                    const text = element.content?.text || '';
                    if (!text.trim()) {
                        return `Section ${i + 1}, paragraph ${j + 1} must include text.`;
                    }
                } else if (element.type === 'image') {
                    const url = element.content?.url || '';
                    if (!url.trim()) {
                        return `Section ${i + 1}, image ${j + 1} requires an image URL.`;
                    }
                } else if (element.type === 'image_group') {
                    const images = Array.isArray(element.content?.images)
                        ? element.content.images
                        : [];
                    if (!images.length) {
                        return `Section ${i + 1}, image group ${j + 1} must include at least one image.`;
                    }
                    const missingIndex = images.findIndex(
                        (image) => !(image?.url || '').trim()
                    );
                    if (missingIndex !== -1) {
                        return `Section ${i + 1}, image group ${j + 1} image ${
                            missingIndex + 1
                        } requires an image URL.`;
                    }
                } else if (element.type === 'list') {
                    const items = Array.isArray(element.content?.items)
                        ? element.content.items
                        : [];
                    const hasItems = items.some((item) =>
                        (item || '').toString().trim()
                    );
                    if (!hasItems) {
                        return `Section ${i + 1}, list ${j + 1} must include at least one item.`;
                    }
                }
            }
        }

        return null;
    };

    const elementTypeLabel = (type) =>
        capitalise(String(type || 'Paragraph').replace(/_/g, ' '));

    const createBuilder = (root) => {
        if (!root) {
            return null;
        }
        const list = root.querySelector('[data-role="section-list"]');
        const empty = root.querySelector('[data-role="section-empty"]');
        const addButton = root.querySelector('[data-role="section-add"]');
        if (!list || !empty) {
            return null;
        }

        const state = {
            sections: [],
        };

        let selectedSectionType = sectionTypeRegistry.getDefault();
        if (addButton && addButton.parentElement) {
            const typePicker = createElement('select', {
                className: 'section-builder__type-picker',
                attrs: { 'aria-label': 'Section type' },
            });
            sectionTypeRegistry.list().forEach((typeDefinition) => {
                const option = createElement('option', {
                    textContent: typeDefinition.label,
                    value: typeDefinition.type,
                });
                if (typeDefinition.type === selectedSectionType) {
                    option.selected = true;
                }
                typePicker.appendChild(option);
            });
            typePicker.addEventListener('change', (event) => {
                selectedSectionType = sectionTypeRegistry.ensure(event.target.value);
            });
            addButton.parentElement.insertBefore(typePicker, addButton);
        }

        let draggingIndex = null;

        const insertSectionAt = (index, type = selectedSectionType) => {
            const boundedIndex = Math.max(
                0,
                Math.min(index, state.sections.length)
            );
            const section = createEmptySection(type);
            state.sections.splice(boundedIndex, 0, section);
            render();
            window.requestAnimationFrame(() => {
                const target = list.querySelector(
                    `.section-card[data-section-id="${section.id}"]`
                );
                if (target) {
                    target.scrollIntoView({
                        behavior: 'smooth',
                        block: 'center',
                    });
                    const focusable = target.querySelector(
                        '.section-card__drag-handle'
                    );
                    focusable?.focus();
                }
            });
        };

        const clearDropIndicators = () => {
            list.querySelectorAll(
                '.section-card--drop-before, .section-card--drop-after'
            ).forEach((card) => {
                card.classList.remove(
                    'section-card--drop-before',
                    'section-card--drop-after'
                );
                delete card.dataset.dropPosition;
            });
            list.querySelectorAll('.section-builder__insertion').forEach(
                (control) => {
                    control.classList.remove('section-builder__insertion--active');
                }
            );
        };

        const endDrag = () => {
            list.classList.remove('section-builder__list--dragging');
            clearDropIndicators();
            draggingIndex = null;
            list.querySelectorAll('.section-card').forEach((card) => {
                card.classList.remove('section-card--dragging');
                card.draggable = false;
            });
        };

        const moveSection = (fromIndex, toIndex) => {
            if (fromIndex === toIndex) {
                return;
            }
            if (fromIndex < 0 || fromIndex >= state.sections.length) {
                return;
            }
            const [removed] = state.sections.splice(fromIndex, 1);
            const boundedIndex = Math.max(
                0,
                Math.min(toIndex, state.sections.length)
            );
            state.sections.splice(boundedIndex, 0, removed);
            render();
            endDrag();
        };

        const removeSection = (index) => {
            state.sections.splice(index, 1);
            render();
        };

        const moveElement = (sectionIndex, fromIndex, toIndex) => {
            const section = state.sections[sectionIndex];
            if (!section || !Array.isArray(section.elements)) {
                return;
            }
            if (fromIndex < 0 || fromIndex >= section.elements.length) {
                return;
            }
            if (toIndex < 0 || toIndex > section.elements.length) {
                return;
            }
            if (fromIndex === toIndex) {
                return;
            }
            const [removed] = section.elements.splice(fromIndex, 1);
            const boundedIndex = Math.max(
                0,
                Math.min(toIndex, section.elements.length)
            );
            section.elements.splice(boundedIndex, 0, removed);
            render();
        };

        const removeElement = (sectionIndex, elementIndex) => {
            const section = state.sections[sectionIndex];
            if (!section || !Array.isArray(section.elements)) {
                return;
            }
            section.elements.splice(elementIndex, 1);
            render();
        };

        const moveGroupImage = (
            sectionIndex,
            elementIndex,
            fromIndex,
            toIndex
        ) => {
            const section = state.sections[sectionIndex];
            const element = section?.elements?.[elementIndex];
            const images = Array.isArray(element?.content?.images)
                ? element.content.images
                : null;
            if (!images || toIndex < 0 || toIndex >= images.length) {
                return;
            }
            const [removed] = images.splice(fromIndex, 1);
            images.splice(toIndex, 0, removed);
            render();
        };

        const removeGroupImage = (sectionIndex, elementIndex, imageIndex) => {
            const section = state.sections[sectionIndex];
            const element = section?.elements?.[elementIndex];
            const images = Array.isArray(element?.content?.images)
                ? element.content.images
                : null;
            if (!images) {
                return;
            }
            images.splice(imageIndex, 1);
            render();
        };

        const createInsertionControl = (index) => {
            const insertion = createElement('li', {
                className: 'section-builder__insertion',
            });
            insertion.dataset.insertIndex = index;

            const total = state.sections.length;
            let label = 'Add section';
            if (total) {
                if (index === 0) {
                    label = 'Add section to beginning';
                } else if (index >= total) {
                    label = 'Add section to end';
                } else {
                    label = `Add section between sections ${index} and ${index + 1}`;
                }
            }

            const button = createElement('button', {
                className: 'section-builder__insert-button',
                type: 'button',
                textContent: '+',
                attrs: {
                    'aria-label': label,
                    title: label,
                },
            });

            button.addEventListener('click', () => {
                insertSectionAt(index);
            });

            insertion.addEventListener('dragover', (event) => {
                if (draggingIndex === null) {
                    return;
                }
                event.preventDefault();
                insertion.classList.add('section-builder__insertion--active');
                try {
                    event.dataTransfer.dropEffect = 'move';
                } catch (error) {
                    // ignore when dataTransfer is unavailable
                }
            });

            insertion.addEventListener('dragleave', () => {
                insertion.classList.remove('section-builder__insertion--active');
            });

            insertion.addEventListener('drop', (event) => {
                if (draggingIndex === null) {
                    return;
                }
                event.preventDefault();
                event.stopPropagation();
                insertion.classList.remove('section-builder__insertion--active');
                clearDropIndicators();
                const fromIndex = draggingIndex;
                let destination = index;
                if (fromIndex < destination) {
                    destination -= 1;
                }
                moveSection(fromIndex, destination);
            });

            insertion.appendChild(button);
            return insertion;
        };

        const createSectionCard = (section, index) => {
            const item = createElement('li', { className: 'section-card' });
            item.dataset.sectionId = section.id;
            item.dataset.index = index;
            item.draggable = false;

            const header = createElement('div', {
                className: 'section-card__header',
            });

            const dragHandle = createElement('button', {
                className: 'section-card__drag-handle',
                type: 'button',
                attrs: {
                    'aria-label': 'Reorder section',
                    title: 'Drag to reorder section',
                },
                html: '<span aria-hidden="true">⋮⋮</span>',
            });

            dragHandle.addEventListener('pointerdown', () => {
                item.draggable = true;
            });

            const resetDraggable = () => {
                if (!item.classList.contains('section-card--dragging')) {
                    item.draggable = false;
                }
            };

            dragHandle.addEventListener('pointerup', resetDraggable);
            dragHandle.addEventListener('pointercancel', resetDraggable);

            header.appendChild(dragHandle);
            header.appendChild(
                createElement('span', {
                    className: 'section-card__title',
                    textContent: `Section ${index + 1}`,
                })
            );

            const controls = createElement('div', {
                className: 'section-card__controls',
            });
            const moveUp = createElement('button', {
                className: 'section-card__control',
                textContent: 'Move up',
                type: 'button',
            });
            moveUp.disabled = index === 0;
            moveUp.addEventListener('click', () =>
                moveSection(index, index - 1)
            );
            controls.appendChild(moveUp);

            const moveDown = createElement('button', {
                className: 'section-card__control',
                textContent: 'Move down',
                type: 'button',
            });
            moveDown.disabled = index === state.sections.length - 1;
            moveDown.addEventListener('click', () =>
                moveSection(index, index + 1)
            );
            controls.appendChild(moveDown);

            const remove = createElement('button', {
                className: 'section-card__control',
                textContent: 'Remove',
                type: 'button',
            });
            remove.addEventListener('click', () => removeSection(index));
            controls.appendChild(remove);

            header.appendChild(controls);
            item.appendChild(header);

            item.addEventListener('dragstart', (event) => {
                draggingIndex = index;
                list.classList.add('section-builder__list--dragging');
                item.classList.add('section-card--dragging');
                try {
                    event.dataTransfer.effectAllowed = 'move';
                    event.dataTransfer.setData('text/plain', String(index));
                } catch (error) {
                    // ignore when dataTransfer is unavailable
                }
            });

            item.addEventListener('dragover', (event) => {
                if (draggingIndex === null || draggingIndex === index) {
                    return;
                }
                event.preventDefault();
                clearDropIndicators();
                const rect = item.getBoundingClientRect();
                const offset = event.clientY - rect.top;
                const insertBefore = offset < rect.height / 2;
                item.dataset.dropPosition = insertBefore ? 'before' : 'after';
                item.classList.add(
                    insertBefore
                        ? 'section-card--drop-before'
                        : 'section-card--drop-after'
                );
                try {
                    event.dataTransfer.dropEffect = 'move';
                } catch (error) {
                    // ignore when dataTransfer is unavailable
                }
            });

            item.addEventListener('drop', (event) => {
                if (draggingIndex === null) {
                    return;
                }
                event.preventDefault();
                event.stopPropagation();
                const fromIndex = draggingIndex;
                const dropPosition =
                    item.dataset.dropPosition === 'before' ? 'before' : 'after';
                let destination = dropPosition === 'before' ? index : index + 1;
                if (fromIndex < destination) {
                    destination -= 1;
                }
                clearDropIndicators();
                if (fromIndex === destination) {
                    endDrag();
                    return;
                }
                moveSection(fromIndex, destination);
            });

            item.addEventListener('dragend', () => {
                endDrag();
            });

        const body = createElement('div', {
            className: 'section-card__body',
        });
        const definition = sectionTypeRegistry.get(section.type);
        const allowElements = definition?.supportsElements !== false;

        const typeId = `${section.id}-type`;
        const typeField = createElement('div', {
            className: 'section-field',
        });
        const typeLabel = createElement('label', {
            textContent: 'Section type',
            attrs: { for: typeId },
        });
        const typeSelect = createElement('select', {
            attrs: { id: typeId, name: typeId },
        });
        sectionTypeRegistry.list().forEach((typeDefinition) => {
            const option = createElement('option', {
                textContent: typeDefinition.label,
                value: typeDefinition.type,
            });
            if (typeDefinition.type === section.type) {
                option.selected = true;
            }
            typeSelect.appendChild(option);
        });
        typeSelect.addEventListener('change', (event) => {
            const nextType = sectionTypeRegistry.ensure(event.target.value);
            if (nextType === section.type) {
                return;
            }
            section.type = nextType;
            if (nextType === 'grid') {
                if (section.styleGridItems === undefined) {
                    section.styleGridItems = true;
                }
            } else if (section.styleGridItems !== undefined) {
                delete section.styleGridItems;
            }
            const nextDefinition = sectionTypeRegistry.get(nextType);
            if (nextDefinition?.supportsElements === false) {
                section.elements = [];
            }
            render();
        });
        typeField.appendChild(typeLabel);
        typeField.appendChild(typeSelect);
        body.appendChild(typeField);
        if (definition?.description) {
            body.appendChild(
                createElement('p', {
                    className: 'section-field__description',
                    textContent: definition.description,
                })
            );
        }
            const titleId = `${section.id}-title`;
            const titleField = createElement('div', {
                className: 'section-field',
            });
            const titleLabel = createElement('label', {
                textContent: 'Title',
                attrs: { for: titleId },
            });
            const titleInput = createElement('input', { type: 'text' });
            titleInput.id = titleId;
            titleInput.value = section.title || '';
            titleInput.placeholder = 'Section heading';
            titleInput.addEventListener('input', (event) => {
                section.title = event.target.value;
            });
            titleField.appendChild(titleLabel);
            titleField.appendChild(titleInput);
            body.appendChild(titleField);

            const imageId = `${section.id}-image`;
            const imageField = createElement('div', {
                className: 'section-field',
            });
            const imageLabel = createElement('label', {
                textContent: 'Featured image (optional)',
                attrs: { for: imageId },
            });
            const imageInput = createElement('input', { type: 'url' });
            imageInput.id = imageId;
            imageInput.placeholder = 'https://example.com/image.jpg';
            imageInput.value = section.image || '';
            imageInput.addEventListener('input', (event) => {
                section.image = event.target.value;
            });
            imageField.appendChild(imageLabel);
            imageField.appendChild(imageInput);
            const imageActions = createMediaBrowseActions(imageInput);
            if (imageActions) {
                imageField.appendChild(imageActions);
            }
            body.appendChild(imageField);

            if (section.type === 'grid') {
                const styleId = `${section.id}-style-grid-items`;
                const styleField = createElement('div', {
                    className: 'section-field section-field--checkbox',
                });
                const styleLabel = createElement('label', {
                    textContent: 'Apply border, background, and padding to grid items',
                    attrs: { for: styleId },
                });
                const styleInput = createElement('input', {
                    type: 'checkbox',
                    className: 'checkbox__input',
                });
                styleInput.id = styleId;
                styleInput.checked = section.styleGridItems !== false;
                styleInput.addEventListener('input', (event) => {
                    section.styleGridItems = event.target.checked;
                });
                styleField.appendChild(styleInput);
                styleField.appendChild(styleLabel);
                body.appendChild(styleField);
            }

            const elementsWrapper = createElement('div', {
                className: 'section-elements',
            });

            if (!allowElements) {
                elementsWrapper.appendChild(
                    createElement('p', {
                        className: 'section-elements__empty',
                        textContent:
                            'This section type does not support additional content blocks.',
                    })
                );
            } else {
                const elementList = createElement('div', {
                    className: 'section-element-list',
                });

                let elementDraggingIndex = null;

                const clearElementDropIndicators = () => {
                    elementList
                        .querySelectorAll(
                            '.section-element--drop-before, .section-element--drop-after'
                        )
                        .forEach((card) => {
                            card.classList.remove(
                                'section-element--drop-before',
                                'section-element--drop-after'
                            );
                            delete card.dataset.dropPosition;
                        });
                };

                const endElementDrag = () => {
                    elementList.classList.remove('section-element-list--dragging');
                    clearElementDropIndicators();
                    elementDraggingIndex = null;
                    elementList
                        .querySelectorAll('.section-element')
                        .forEach((card) => {
                            card.classList.remove('section-element--dragging');
                            card.draggable = false;
                        });
                };

                const elementHelpers = {
                    clearDropIndicators: clearElementDropIndicators,
                    endDrag: endElementDrag,
                    getDraggingIndex: () => elementDraggingIndex,
                    startDrag: (dragIndex, card) => {
                        elementDraggingIndex = dragIndex;
                        elementList.classList.add('section-element-list--dragging');
                        card.classList.add('section-element--dragging');
                    },
                };

                if (!section.elements.length) {
                    elementList.appendChild(
                        createElement('p', {
                            className: 'section-elements__empty',
                            textContent: 'No elements added yet.',
                        })
                    );
                } else {
                    section.elements.forEach((element, elementIndex) => {
                        elementList.appendChild(
                            createElementCard(
                                section,
                                index,
                                element,
                                elementIndex,
                                elementHelpers
                            )
                        );
                    });
                }
                elementsWrapper.appendChild(elementList);

                const elementActions = createElement('div', {
                    className: 'section-elements__actions',
                });
                const addParagraph = createElement('button', {
                    className: 'section-elements__button',
                    textContent: 'Add paragraph',
                    type: 'button',
                });
                addParagraph.addEventListener('click', () => {
                    section.elements.push(createElementByType('paragraph'));
                    render();
                });
                elementActions.appendChild(addParagraph);

                const addImage = createElement('button', {
                    className: 'section-elements__button',
                    textContent: 'Add image',
                    type: 'button',
                });
                addImage.addEventListener('click', () => {
                    section.elements.push(createElementByType('image'));
                    render();
                });
                elementActions.appendChild(addImage);

                const addGallery = createElement('button', {
                    className: 'section-elements__button',
                    textContent: 'Add image group',
                    type: 'button',
                });
                addGallery.addEventListener('click', () => {
                    section.elements.push(createElementByType('image_group'));
                    render();
                });
                elementActions.appendChild(addGallery);

                const addList = createElement('button', {
                    className: 'section-elements__button',
                    textContent: 'Add list',
                    type: 'button',
                });
                addList.addEventListener('click', () => {
                    section.elements.push(createElementByType('list'));
                    render();
                });
                elementActions.appendChild(addList);

                const addSearch = createElement('button', {
                    className: 'section-elements__button',
                    textContent: 'Add search block',
                    type: 'button',
                });
                addSearch.addEventListener('click', () => {
                    section.elements.push(createElementByType('search'));
                    render();
                });
                elementActions.appendChild(addSearch);

                elementsWrapper.appendChild(elementActions);
            }

            body.appendChild(elementsWrapper);

            item.appendChild(body);
            return item;
        };

        const createElementCard = (
            section,
            sectionIndex,
            element,
            elementIndex,
            elementDragHelpers
        ) => {
            const wrapper = createElement('div', {
                className: 'section-element',
            });
            wrapper.dataset.elementId = element.id;
            wrapper.dataset.index = elementIndex;
            wrapper.draggable = false;

            const header = createElement('div', {
                className: 'section-element__header',
            });

            const dragHandle = createElement('button', {
                className: 'section-element__drag-handle',
                type: 'button',
                attrs: {
                    'aria-label': 'Reorder element',
                    title: 'Drag to reorder element',
                },
                html: '<span aria-hidden="true">⋮⋮</span>',
            });

            dragHandle.addEventListener('pointerdown', () => {
                wrapper.draggable = true;
            });

            const resetElementDraggable = () => {
                if (!wrapper.classList.contains('section-element--dragging')) {
                    wrapper.draggable = false;
                }
            };

            dragHandle.addEventListener('pointerup', resetElementDraggable);
            dragHandle.addEventListener('pointercancel', resetElementDraggable);

            header.appendChild(dragHandle);
            header.appendChild(
                createElement('span', {
                    className: 'section-element__title',
                    textContent: `${elementTypeLabel(element.type)} ${
                        elementIndex + 1
                    }`,
                })
            );
            const controls = createElement('div', {
                className: 'section-element__controls',
            });
            const moveUp = createElement('button', {
                className: 'section-element__control',
                textContent: 'Up',
                type: 'button',
            });
            moveUp.disabled = elementIndex === 0;
            moveUp.addEventListener('click', () =>
                moveElement(sectionIndex, elementIndex, elementIndex - 1)
            );
            controls.appendChild(moveUp);

            const moveDown = createElement('button', {
                className: 'section-element__control',
                textContent: 'Down',
                type: 'button',
            });
            moveDown.disabled = elementIndex === section.elements.length - 1;
            moveDown.addEventListener('click', () =>
                moveElement(sectionIndex, elementIndex, elementIndex + 1)
            );
            controls.appendChild(moveDown);

            const remove = createElement('button', {
                className: 'section-element__control',
                textContent: 'Remove',
                type: 'button',
            });
            remove.addEventListener('click', () =>
                removeElement(sectionIndex, elementIndex)
            );
            controls.appendChild(remove);

            header.appendChild(controls);
            wrapper.appendChild(header);

            wrapper.addEventListener('dragstart', (event) => {
                elementDragHelpers.startDrag(elementIndex, wrapper);
                try {
                    event.dataTransfer.effectAllowed = 'move';
                    event.dataTransfer.setData(
                        'text/plain',
                        String(elementIndex)
                    );
                } catch (error) {
                    // ignore when dataTransfer is unavailable
                }
            });

            wrapper.addEventListener('dragover', (event) => {
                const draggingIndex = elementDragHelpers.getDraggingIndex();
                if (draggingIndex === null || draggingIndex === elementIndex) {
                    return;
                }
                event.preventDefault();
                elementDragHelpers.clearDropIndicators();
                const rect = wrapper.getBoundingClientRect();
                const offset = event.clientY - rect.top;
                const insertBefore = offset < rect.height / 2;
                wrapper.dataset.dropPosition = insertBefore
                    ? 'before'
                    : 'after';
                wrapper.classList.add(
                    insertBefore
                        ? 'section-element--drop-before'
                        : 'section-element--drop-after'
                );
                try {
                    event.dataTransfer.dropEffect = 'move';
                } catch (error) {
                    // ignore when dataTransfer is unavailable
                }
            });

            wrapper.addEventListener('drop', (event) => {
                const draggingIndex = elementDragHelpers.getDraggingIndex();
                if (draggingIndex === null) {
                    return;
                }
                event.preventDefault();
                event.stopPropagation();
                const fromIndex = draggingIndex;
                const dropPosition =
                    wrapper.dataset.dropPosition === 'before'
                        ? 'before'
                        : 'after';
                let destination =
                    dropPosition === 'before' ? elementIndex : elementIndex + 1;
                if (fromIndex < destination) {
                    destination -= 1;
                }
                elementDragHelpers.clearDropIndicators();
                if (fromIndex === destination) {
                    elementDragHelpers.endDrag();
                    return;
                }
                elementDragHelpers.endDrag();
                moveElement(sectionIndex, fromIndex, destination);
            });

            wrapper.addEventListener('dragend', () => {
                elementDragHelpers.endDrag();
            });

            const body = createElement('div', {
                className: 'section-element__body',
            });
            if (element.type === 'paragraph') {
                const contentId = `${section.id}-${element.id}-content`;
                const field = createElement('div', {
                    className: 'section-field',
                });
                const label = createElement('label', {
                    textContent: 'Text content',
                    attrs: { for: contentId },
                });
                const textarea = document.createElement('textarea');
                textarea.id = contentId;
                textarea.placeholder = 'Describe this section…';
                textarea.value = element.content?.text || '';
                textarea.addEventListener('input', (event) => {
                    element.content.text = event.target.value;
                });
                field.appendChild(label);
                field.appendChild(textarea);
                body.appendChild(field);
            } else if (element.type === 'image') {
                const urlId = `${section.id}-${element.id}-url`;
                const urlField = createElement('div', {
                    className: 'section-field',
                });
                const urlLabel = createElement('label', {
                    textContent: 'Image URL',
                    attrs: { for: urlId },
                });
                const urlInput = createElement('input', { type: 'url' });
                urlInput.id = urlId;
                urlInput.placeholder = 'https://example.com/image.jpg';
                urlInput.value = element.content?.url || '';
                urlInput.addEventListener('input', (event) => {
                    element.content.url = event.target.value;
                });
                urlField.appendChild(urlLabel);
                urlField.appendChild(urlInput);
                const urlActions = createMediaBrowseActions(urlInput);
                if (urlActions) {
                    urlField.appendChild(urlActions);
                }
                body.appendChild(urlField);

                const altId = `${section.id}-${element.id}-alt`;
                const altField = createElement('div', {
                    className: 'section-field',
                });
                const altLabel = createElement('label', {
                    textContent: 'Alt text',
                    attrs: { for: altId },
                });
                const altInput = createElement('input', { type: 'text' });
                altInput.id = altId;
                altInput.placeholder = 'Describe the image for accessibility';
                altInput.value = element.content?.alt || '';
                altInput.addEventListener('input', (event) => {
                    element.content.alt = event.target.value;
                });
                altField.appendChild(altLabel);
                altField.appendChild(altInput);
                body.appendChild(altField);

                const captionId = `${section.id}-${element.id}-caption`;
                const captionField = createElement('div', {
                    className: 'section-field',
                });
                const captionLabel = createElement('label', {
                    textContent: 'Caption (optional)',
                    attrs: { for: captionId },
                });
                const captionInput = createElement('input', { type: 'text' });
                captionInput.id = captionId;
                captionInput.placeholder = 'Add a supporting caption';
                captionInput.value = element.content?.caption || '';
                captionInput.addEventListener('input', (event) => {
                    element.content.caption = event.target.value;
                });
                captionField.appendChild(captionLabel);
                captionField.appendChild(captionInput);
                body.appendChild(captionField);
            } else if (element.type === 'list') {
                if (!Array.isArray(element.content?.items)) {
                    element.content.items = [''];
                } else {
                    element.content.items = element.content.items.map((item) => {
                        if (typeof item === 'string') {
                            return item;
                        }
                        if (item === null || item === undefined) {
                            return '';
                        }
                        return String(item);
                    });
                }

                const orderedId = `${section.id}-${element.id}-ordered`;
                const orderedField = createElement('div', {
                    className: 'section-field section-field--checkbox',
                });
                const orderedLabel = createElement('label', {
                    textContent: 'Numbered list',
                    attrs: { for: orderedId },
                });
                const orderedInput = createElement('input', {
                    type: 'checkbox',
                    className: 'checkbox__input',
                });
                orderedInput.id = orderedId;
                orderedInput.checked = Boolean(element.content?.ordered);
                orderedInput.addEventListener('input', (event) => {
                    element.content.ordered = event.target.checked;
                });
                orderedField.appendChild(orderedInput);
                orderedField.appendChild(orderedLabel);
                body.appendChild(orderedField);

                const itemsId = `${section.id}-${element.id}-items`;
                const itemsField = createElement('div', {
                    className: 'section-field section-field--list-items',
                });
                const itemsLabel = createElement('label', {
                    textContent: 'List items',
                    attrs: { for: itemsId },
                });
                itemsField.appendChild(itemsLabel);

                const listItemsWrapper = createElement('div', {
                    className: 'list-items',
                    attrs: { id: itemsId },
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
                listItemsActions.appendChild(addListItem);
                listItemsWrapper.appendChild(listItemsList);
                listItemsWrapper.appendChild(listItemsActions);
                itemsField.appendChild(listItemsWrapper);
                body.appendChild(itemsField);

                let listItemDraggingIndex = null;

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

                    const items = element.content.items;
                    if (!items.length) {
                        listItemsList.appendChild(
                            createElement('p', {
                                className: 'list-items__empty',
                                textContent: 'No list items yet.',
                            })
                        );
                        return;
                    }

                    items.forEach((value, itemIndex) => {
                        const row = createElement('div', {
                            className: 'list-item',
                        });
                        row.dataset.index = String(itemIndex);
                        row.draggable = false;

                        const dragHandle = createElement('button', {
                            className: 'list-item__drag-handle',
                            type: 'button',
                            attrs: {
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

                        row.appendChild(dragHandle);

                        const input = createElement('input', {
                            className: 'list-item__input',
                            type: 'text',
                            value: value,
                        });
                        input.placeholder = 'List item';
                        input.addEventListener('input', (event) => {
                            element.content.items[itemIndex] = event.target.value;
                        });
                        row.appendChild(input);

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
                        controls.appendChild(moveUp);

                        const moveDown = createElement('button', {
                            className: 'list-item__control',
                            textContent: 'Down',
                            type: 'button',
                        });
                        moveDown.disabled = itemIndex === items.length - 1;
                        moveDown.addEventListener('click', () => {
                            moveListItem(itemIndex, itemIndex + 1);
                        });
                        controls.appendChild(moveDown);

                        const removeItem = createElement('button', {
                            className: 'list-item__control',
                            textContent: 'Remove',
                            type: 'button',
                        });
                        removeItem.addEventListener('click', () => {
                            element.content.items.splice(itemIndex, 1);
                            const nextFocus = Math.max(
                                0,
                                Math.min(itemIndex, element.content.items.length - 1)
                            );
                            renderListItems(
                                element.content.items.length ? nextFocus : null
                            );
                        });
                        controls.appendChild(removeItem);

                        row.appendChild(controls);

                        row.addEventListener('dragstart', (event) => {
                            startListItemDrag(itemIndex, row);
                            try {
                                event.dataTransfer.effectAllowed = 'move';
                                event.dataTransfer.setData(
                                    'text/plain',
                                    String(itemIndex)
                                );
                            } catch (error) {
                                // ignore if dataTransfer is unavailable
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
                                // ignore if dataTransfer is unavailable
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
                                dropPosition === 'before'
                                    ? itemIndex
                                    : itemIndex + 1;
                            if (fromIndex < destination) {
                                destination -= 1;
                            }
                            endListItemDrag();
                            moveListItem(fromIndex, destination);
                        });

                        row.addEventListener('dragend', () => {
                            endListItemDrag();
                        });

                        listItemsList.appendChild(row);
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
                        // ignore if dataTransfer is unavailable
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
                    renderListItems(element.content.items.length - 1);
                });

                renderListItems();
            } else if (element.type === 'search') {
                if (!element.content || typeof element.content !== 'object') {
                    element.content = {};
                }

                element.content.title = element.content.title || 'Search';
                element.content.placeholder = element.content.placeholder || 'Start typing to search';
                element.content.submitLabel = element.content.submitLabel || 'Search';
                element.content.filterLabel = element.content.filterLabel || 'Filter by';
                element.content.action = element.content.action || '/search';
                if (element.content.showFilters === undefined) {
                    element.content.showFilters = true;
                }
                element.content.heading = (element.content.heading || 'h2').toLowerCase();
                element.content.hint = element.content.hint || 'Use the search form above to explore the knowledge base.';
                element.content.default_type = (element.content.default_type || 'all').toLowerCase();

                const fieldId = (name) => `${section.id}-${element.id}-${name}`;

                const createTextField = (labelText, key, placeholder = '', isTextarea = false) => {
                    const id = fieldId(key);
                    const field = createElement('div', {
                        className: 'section-field',
                    });
                    const label = createElement('label', {
                        textContent: labelText,
                        attrs: { for: id },
                    });
                    const control = isTextarea
                        ? document.createElement('textarea')
                        : createElement('input', { type: 'text' });
                    control.id = id;
                    if (placeholder) {
                        control.placeholder = placeholder;
                    }
                    control.value = element.content[key] || '';
                    control.addEventListener('input', (event) => {
                        element.content[key] = event.target.value;
                    });
                    field.appendChild(label);
                    field.appendChild(control);
                    body.appendChild(field);
                };

                createTextField('Block title', 'title', 'Search');
                createTextField('Description (optional)', 'description', 'Describe what visitors can search for.', true);
                createTextField('Placeholder text', 'placeholder', 'Start typing to search');
                createTextField('Submit button label', 'submitLabel', 'Search');
                createTextField('Filter label', 'filterLabel', 'Filter by');
                createTextField('Form action URL', 'action', '/search');

                const showFiltersId = fieldId('show-filters');
                const showFiltersField = createElement('div', {
                    className: 'section-field section-field--checkbox',
                });
                const showFiltersLabel = createElement('label', {
                    textContent: 'Allow filtering by title, content, tag, or author',
                    attrs: { for: showFiltersId },
                });
                const showFiltersInput = createElement('input', {
                    type: 'checkbox',
                    className: 'checkbox__input',
                });
                showFiltersInput.id = showFiltersId;
                showFiltersInput.checked = Boolean(element.content.showFilters);
                showFiltersInput.addEventListener('input', (event) => {
                    element.content.showFilters = event.target.checked;
                });
                showFiltersField.appendChild(showFiltersInput);
                showFiltersField.appendChild(showFiltersLabel);
                body.appendChild(showFiltersField);

                const headingId = fieldId('heading');
                const headingField = createElement('div', {
                    className: 'section-field',
                });
                const headingLabel = createElement('label', {
                    textContent: 'Heading level',
                    attrs: { for: headingId },
                });
                const headingSelect = document.createElement('select');
                headingSelect.id = headingId;
                ['h1', 'h2', 'h3', 'h4', 'h5', 'h6'].forEach((option) => {
                    const optionElement = document.createElement('option');
                    optionElement.value = option;
                    optionElement.textContent = option.toUpperCase();
                    if (option === element.content.heading) {
                        optionElement.selected = true;
                    }
                    headingSelect.appendChild(optionElement);
                });
                headingSelect.addEventListener('change', (event) => {
                    element.content.heading = event.target.value;
                });
                headingField.appendChild(headingLabel);
                headingField.appendChild(headingSelect);
                body.appendChild(headingField);

                const defaultTypeId = fieldId('default-type');
                const defaultTypeField = createElement('div', {
                    className: 'section-field',
                });
                const defaultTypeLabel = createElement('label', {
                    textContent: 'Default filter option',
                    attrs: { for: defaultTypeId },
                });
                const defaultTypeSelect = document.createElement('select');
                defaultTypeSelect.id = defaultTypeId;
                [
                    { value: 'all', label: 'Title & content' },
                    { value: 'title', label: 'Title' },
                    { value: 'content', label: 'Content' },
                    { value: 'tag', label: 'Tag' },
                    { value: 'author', label: 'Author' },
                ].forEach((option) => {
                    const optionElement = document.createElement('option');
                    optionElement.value = option.value;
                    optionElement.textContent = option.label;
                    if (option.value === element.content.default_type) {
                        optionElement.selected = true;
                    }
                    defaultTypeSelect.appendChild(optionElement);
                });
                defaultTypeSelect.addEventListener('change', (event) => {
                    element.content.default_type = event.target.value;
                });
                defaultTypeField.appendChild(defaultTypeLabel);
                defaultTypeField.appendChild(defaultTypeSelect);
                body.appendChild(defaultTypeField);

                const hintId = fieldId('hint');
                const hintField = createElement('div', {
                    className: 'section-field',
                });
                const hintLabel = createElement('label', {
                    textContent: 'Hint text (shown when no search query is provided)',
                    attrs: { for: hintId },
                });
                const hintTextarea = document.createElement('textarea');
                hintTextarea.id = hintId;
                hintTextarea.placeholder = 'Use the search form above to explore the knowledge base.';
                hintTextarea.value = element.content.hint || '';
                hintTextarea.addEventListener('input', (event) => {
                    element.content.hint = event.target.value;
                });
                hintField.appendChild(hintLabel);
                hintField.appendChild(hintTextarea);
                body.appendChild(hintField);
            } else if (element.type === 'image_group') {
                if (!Array.isArray(element.content?.images)) {
                    element.content.images = [];
                }
                const layoutId = `${section.id}-${element.id}-layout`;
                const layoutField = createElement('div', {
                    className: 'section-field',
                });
                const layoutLabel = createElement('label', {
                    textContent: 'Layout',
                    attrs: { for: layoutId },
                });
                const layoutInput = createElement('input', { type: 'text' });
                layoutInput.id = layoutId;
                layoutInput.placeholder = 'grid | columns | masonry';
                layoutInput.value = element.content?.layout || 'grid';
                layoutInput.addEventListener('input', (event) => {
                    element.content.layout = event.target.value;
                });
                layoutField.appendChild(layoutLabel);
                layoutField.appendChild(layoutInput);
                body.appendChild(layoutField);

                const imageList = createElement('div', {
                    className: 'image-group-list',
                });
                if (!element.content.images.length) {
                    imageList.appendChild(
                        createElement('p', {
                            className: 'section-elements__empty',
                            textContent: 'No images added to this group.',
                        })
                    );
                } else {
                    element.content.images.forEach((image, imageIndex) => {
                        const item = createElement('div', {
                            className: 'image-group-item',
                        });
                        const urlId = `${section.id}-${element.id}-image-${imageIndex}-url`;
                        const urlField = createElement('div', {
                            className: 'section-field',
                        });
                        const urlLabel = createElement('label', {
                            textContent: `Image ${imageIndex + 1} URL`,
                            attrs: { for: urlId },
                        });
                        const urlInput = createElement('input', {
                            type: 'url',
                        });
                        urlInput.id = urlId;
                        urlInput.placeholder = 'https://example.com/image.jpg';
                        urlInput.value = image.url || image.URL || '';
                        urlInput.addEventListener('input', (event) => {
                            image.url = event.target.value;
                        });
                        urlField.appendChild(urlLabel);
                        urlField.appendChild(urlInput);
                        const urlActions = createMediaBrowseActions(urlInput);
                        if (urlActions) {
                            urlField.appendChild(urlActions);
                        }
                        item.appendChild(urlField);

                        const altId = `${section.id}-${element.id}-image-${imageIndex}-alt`;
                        const altField = createElement('div', {
                            className: 'section-field',
                        });
                        const altLabel = createElement('label', {
                            textContent: 'Alt text',
                            attrs: { for: altId },
                        });
                        const altInput = createElement('input', {
                            type: 'text',
                        });
                        altInput.id = altId;
                        altInput.placeholder = 'Describe the image';
                        altInput.value = image.alt || image.Alt || '';
                        altInput.addEventListener('input', (event) => {
                            image.alt = event.target.value;
                        });
                        altField.appendChild(altLabel);
                        altField.appendChild(altInput);
                        item.appendChild(altField);

                        const captionId = `${section.id}-${element.id}-image-${imageIndex}-caption`;
                        const captionField = createElement('div', {
                            className: 'section-field',
                        });
                        const captionLabel = createElement('label', {
                            textContent: 'Caption',
                            attrs: { for: captionId },
                        });
                        const captionInput = createElement('input', {
                            type: 'text',
                        });
                        captionInput.id = captionId;
                        captionInput.placeholder = 'Optional caption';
                        captionInput.value =
                            image.caption || image.Caption || '';
                        captionInput.addEventListener('input', (event) => {
                            image.caption = event.target.value;
                        });
                        captionField.appendChild(captionLabel);
                        captionField.appendChild(captionInput);
                        item.appendChild(captionField);

                        const imageControls = createElement('div', {
                            className: 'image-group-item__controls',
                        });
                        const imageUp = createElement('button', {
                            className: 'image-group-item__control',
                            textContent: 'Up',
                            type: 'button',
                        });
                        imageUp.disabled = imageIndex === 0;
                        imageUp.addEventListener('click', () =>
                            moveGroupImage(
                                sectionIndex,
                                elementIndex,
                                imageIndex,
                                imageIndex - 1
                            )
                        );
                        imageControls.appendChild(imageUp);

                        const imageDown = createElement('button', {
                            className: 'image-group-item__control',
                            textContent: 'Down',
                            type: 'button',
                        });
                        imageDown.disabled =
                            imageIndex === element.content.images.length - 1;
                        imageDown.addEventListener('click', () =>
                            moveGroupImage(
                                sectionIndex,
                                elementIndex,
                                imageIndex,
                                imageIndex + 1
                            )
                        );
                        imageControls.appendChild(imageDown);

                        const imageRemove = createElement('button', {
                            className: 'image-group-item__control',
                            textContent: 'Remove',
                            type: 'button',
                        });
                        imageRemove.addEventListener('click', () =>
                            removeGroupImage(
                                sectionIndex,
                                elementIndex,
                                imageIndex
                            )
                        );
                        imageControls.appendChild(imageRemove);

                        item.appendChild(imageControls);
                        imageList.appendChild(item);
                    });
                }

                const addImageButton = createElement('button', {
                    className: 'section-elements__button',
                    textContent: 'Add image to group',
                    type: 'button',
                });
                addImageButton.addEventListener('click', () => {
                    element.content.images.push({
                        url: '',
                        alt: '',
                        caption: '',
                    });
                    render();
                });
                imageList.appendChild(addImageButton);
                body.appendChild(imageList);
            }

            wrapper.appendChild(body);
            return wrapper;
        };

        list.addEventListener('dragover', (event) => {
            if (draggingIndex === null) {
                return;
            }
            event.preventDefault();
            if (event.target === list) {
                clearDropIndicators();
            }
        });

        list.addEventListener('drop', (event) => {
            if (draggingIndex === null) {
                return;
            }
            event.preventDefault();
            event.stopPropagation();
            clearDropIndicators();
            const fromIndex = draggingIndex;
            const destination = state.sections.length - 1;
            if (fromIndex === destination) {
                endDrag();
                return;
            }
            moveSection(fromIndex, destination);
        });

        const render = () => {
            list.innerHTML = '';
            if (!state.sections.length) {
                empty.hidden = false;
                return;
            }
            empty.hidden = true;
            state.sections.forEach((section, index) => {
                if (index === 0) {
                    list.appendChild(createInsertionControl(0));
                }
                list.appendChild(createSectionCard(section, index));
                list.appendChild(createInsertionControl(index + 1));
            });
        };

        addButton?.addEventListener('click', () => {
            insertSectionAt(state.sections.length);
        });

        render();

        return {
            root,
            getSections: () => state.sections,
            setSections: (sections) => {
                if (Array.isArray(sections)) {
                    state.sections = sections.map((section) =>
                        normaliseSection(section)
                    );
                } else {
                    state.sections = [];
                }
                render();
            },
            reset: () => {
                state.sections = [];
                render();
            },
            validate: () => validateSections(state.sections),
            serialize: () => buildSubmitSections(state.sections),
        };
    };

    const init = (root) => {
        if (!root) {
            return null;
        }
        if (builders.has(root)) {
            return builders.get(root);
        }
        const builder = createBuilder(root);
        if (builder) {
            builders.set(root, builder);
        }
        return builder;
    };

    window.SectionBuilder = {
        init,
    };
})();

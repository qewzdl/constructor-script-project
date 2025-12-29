(() => {
    const definitions = new Map();

    const getAdminRoot = () => {
        if (typeof document === 'undefined') {
            return null;
        }
        return document.querySelector('.admin[data-page="admin"]');
    };

    const adminRoot = getAdminRoot();
    const blogEnabled = !adminRoot || adminRoot.dataset.blogEnabled !== 'false';
    const coursesEnabled = !adminRoot || adminRoot.dataset.coursesEnabled !== 'false';

    const normaliseType = (type) =>
        typeof type === 'string' ? type.trim().toLowerCase() : '';

    const normaliseTypeList = (value) => {
        if (!Array.isArray(value)) {
            return [];
        }
        const set = new Set();
        value.forEach((item) => {
            const normalised = normaliseType(item);
            if (normalised) {
                set.add(normalised);
            }
        });
        return Array.from(set);
    };

    const register = (type, definition = {}) => {
        const normalised = normaliseType(type);
        if (!normalised || !definition) {
            return;
        }

        const supportsElementsSource =
            definition.supportsElements !== undefined
                ? definition.supportsElements
                : definition.supports_elements;
        const supportsHeaderImageSource =
            definition.supportsHeaderImage !== undefined
                ? definition.supportsHeaderImage
                : definition.supports_header_image;

        const allowedElementsSource =
            definition.allowedElements !== undefined
                ? definition.allowedElements
                : definition.allowed_elements;
        const allowedElements = normaliseTypeList(allowedElementsSource);

        const entry = {
            type: normalised,
            label:
                typeof definition.label === 'string' && definition.label.trim()
                    ? definition.label.trim()
                    : normalised,
            order: Number.isFinite(definition.order) ? definition.order : 0,
            description:
                typeof definition.description === 'string'
                    ? definition.description.trim()
                    : '',
            supportsElements:
                supportsElementsSource === undefined
                    ? true
                    : Boolean(
                          typeof supportsElementsSource === 'string'
                              ? supportsElementsSource.toLowerCase() !== 'false'
                              : supportsElementsSource
                      ),
            supportsHeaderImage:
                supportsHeaderImageSource === undefined
                    ? false
                    : Boolean(
                          typeof supportsHeaderImageSource === 'string'
                              ? supportsHeaderImageSource.toLowerCase() !== 'false'
                              : supportsHeaderImageSource
                      ),
            allowedElements: allowedElements.length ? allowedElements : undefined,
            settings:
                definition.settings && typeof definition.settings === 'object'
                    ? definition.settings
                    : undefined,
        };

        definitions.set(normalised, entry);
    };

    const get = (type) => definitions.get(normaliseType(type));

    const getDefinitions = () => {
        const entries = Array.from(definitions.entries());
        return entries.reduce((accumulator, [key, value]) => {
            accumulator[key] = value;
            return accumulator;
        }, {});
    };

    const getOrderedTypes = () =>
        Array.from(definitions.entries())
            .sort(([, a], [, b]) => (a.order || 0) - (b.order || 0))
            .map(([type]) => type);

    const getDefaultType = () => {
        const ordered = getOrderedTypes();
        return ordered.length ? ordered[0] : 'standard';
    };

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

    const initialDefinitions = parseDefinitionJSON('section-definitions-data');
    if (initialDefinitions && typeof initialDefinitions === 'object') {
        Object.entries(initialDefinitions).forEach(([type, definition]) => {
            register(type, definition);
        });
    }

    const ensureRegistered = (type, definition) => {
        if (!definitions.has(normaliseType(type))) {
            register(type, definition);
        }
    };

    ensureRegistered('standard', {
        label: 'Standard section',
        order: 0,
        supportsElements: true,
        supportsHeaderImage: true,
        description:
            'Flexible content area for combining paragraphs, media, and lists.',
        allowedElements: ['paragraph', 'image', 'image_group', 'list', 'file_group', 'search'],
        settings: {
            image_alt: {
                label: 'Side image alt text',
                placeholder: 'Describe the right-side image',
            },
        },
    });

    ensureRegistered('grid', {
        label: 'Grid section',
        order: 15,
        supportsElements: true,
        description:
            'Displays content blocks in a responsive grid. Add at least two elements for a balanced layout.',
        allowedElements: ['paragraph', 'image', 'image_group', 'list', 'file_group'],
    });
    ensureRegistered('features', {
        label: 'Features',
        order: 16,
        supportsElements: true,
        description:
            'Showcase key features with image and text pairs laid out in a grid.',
        allowedElements: ['feature_item'],
    });

    ensureRegistered('file_list', {
        label: 'File list',
        order: 17,
        supportsElements: true,
        description: 'Display downloadable files with optional grouping.',
        allowedElements: ['file_group'],
    });

    if (blogEnabled) {
        ensureRegistered('categories_list', {
            label: 'Categories list',
            order: 18,
            supportsElements: false,
            description:
                'Displays a list of blog categories for quick navigation.',
            settings: {
                limit: {
                    label: 'Number of categories to display',
                    min: 1,
                    max: 30,
                    default: 10,
                },
            },
        });
        register('posts_list', {
            label: 'Posts list',
            order: 20,
            supportsElements: false,
            description: 'Automatically displays the most recent blog posts.',
            settings: {
                display_mode: {
                    label: 'Display mode',
                    type: 'select',
                    options: [
                        { value: 'limited', label: 'Limited (latest posts)' },
                        { value: 'carousel', label: 'Carousel' },
                        { value: 'paginated', label: 'Paginated (all posts)' },
                        { value: 'selected', label: 'Selected posts' },
                    ],
                    defaultValue: 'limited',
                },
                carousel_columns: {
                    label: 'Columns in carousel',
                    type: 'range',
                    min: 1,
                    max: 4,
                    default: 3,
                },
                limit: {
                    label: 'Number of posts to display',
                    perPageLabel: 'Number of posts to display on a page',
                    min: 1,
                    max: 24,
                    default: 6,
                },
                selected_posts: {
                    label: 'Selected posts',
                    type: 'text',
                    placeholder: 'Choose posts to feature',
                    allowPostPicker: true,
                },
            },
        });
    }

    if (coursesEnabled) {
        register('courses_list', {
            label: 'Courses list',
            order: 22,
            supportsElements: false,
            description: 'Highlights available course packages with pricing and topics.',
            settings: {
                display_mode: {
                    label: 'Display mode',
                    type: 'select',
                    options: [
                        { value: 'limited', label: 'Limited (latest courses)' },
                        { value: 'carousel', label: 'Carousel' },
                        { value: 'paginated', label: 'Paginated (all courses)' },
                        { value: 'selected', label: 'Selected courses' },
                    ],
                    defaultValue: 'limited',
                },
                carousel_columns: {
                    label: 'Columns in carousel',
                    type: 'range',
                    min: 1,
                    max: 4,
                    default: 3,
                },
                limit: {
                    label: 'Number of courses to display',
                    perPageLabel: 'Number of courses to display on a page',
                    min: 1,
                    max: 12,
                    default: 3,
                },
                selected_courses: {
                    label: 'Selected courses',
                    type: 'text',
                    placeholder: 'Choose courses to feature',
                    allowCoursePicker: true,
                },
                show_all_button: {
                    label: 'Show link to all courses',
                    type: 'boolean',
                    defaultValue: 'false',
                },
                all_courses_url: {
                    label: 'All courses link',
                    type: 'url',
                    placeholder: '/courses',
                    allowAnchorPicker: true,
                },
                all_courses_label: {
                    label: 'All courses link label',
                    placeholder: 'View all courses',
                },
            },
        });
    }

    window.AdminSectionRegistry = {
        register,
        get,
        getDefinitions,
        getOrderedTypes,
        getDefaultType,
    };
})();

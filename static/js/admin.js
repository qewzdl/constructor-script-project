(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    const builderModule = window.AdminSectionBuilder;

    if (!utils || !registry || !builderModule) {
        console.error('Admin dashboard dependencies are missing.');
        return;
    }

    const {
        formatDate,
        booleanLabel,
        createElement,
        buildAbsoluteUrl,
        createSvgElement,
        formatNumber,
        formatMonthLabel,
        normaliseString,
    } = utils;

    const elementDefinitions = registry.getDefinitions();
    const createSectionBuilder = builderModule.create;
    
    const generateContentPreview = (sections) => {
        if (!Array.isArray(sections) || sections.length === 0) {
            return '';
        }
        const parts = [];
        sections.forEach((section) => {
            if (section.title) {
                parts.push(section.title);
            }
            if (Array.isArray(section.elements)) {
                section.elements.forEach((element) => {
                    const definition = elementDefinitions[element.type];
                    if (definition && typeof definition.preview === 'function') {
                        definition.preview(element, parts);
                    }
                });
            }
        });
        return parts.join('\n\n');
    };

    const initialiseAdminDashboard = () => {
        const root = document.querySelector('[data-page="admin"]');
        if (!root) {
            return;
        }

        const app = window.App || {};
        const auth = app.auth;
        const fallbackApiRequest = async (url, options = {}) => {
            const headers = Object.assign({}, options.headers || {});
            const token =
                auth && typeof auth.getToken === 'function'
                    ? auth.getToken()
                    : undefined;

            if (options.body && !(options.body instanceof FormData)) {
                headers['Content-Type'] =
                    headers['Content-Type'] || 'application/json';
            }

            if (token) {
                headers.Authorization =
                    headers.Authorization || `Bearer ${token}`;
            }

            const response = await fetch(url, {
                credentials: 'include',
                ...options,
                headers,
            });

            const contentType = response.headers.get('content-type') || '';
            const isJson = contentType.includes('application/json');
            const payload = isJson
                ? await response.json().catch(() => null)
                : await response.text();

            if (!response.ok) {
                const message =
                    payload && typeof payload === 'object' && payload.error
                        ? payload.error
                        : typeof payload === 'string'
                        ? payload
                        : 'Request failed';
                const error = new Error(message);
                error.status = response.status;
                error.payload = payload;
                throw error;
            }

            return payload;
        };

        const apiRequest =
            typeof app.apiRequest === 'function'
                ? app.apiRequest
                : fallbackApiRequest;
        if (typeof app.apiRequest !== 'function') {
            console.warn(
                'Admin dashboard is using fallback API client because App.apiRequest is unavailable.'
            );
        }
        const setAlert =
            typeof app.setAlert === 'function' ? app.setAlert : null;
        const toggleFormDisabled =
            typeof app.toggleFormDisabled === 'function'
                ? app.toggleFormDisabled
                : null;

        const requireAuth = () => {
            if (!auth || typeof auth.getToken !== 'function') {
                return true;
            }
            if (!auth.getToken()) {
                window.location.href = '/login?redirect=/admin';
                return false;
            }
            return true;
        };

        if (!requireAuth()) {
            return;
        }

        const endpoints = {
            stats: root.dataset.endpointStats,
            posts: root.dataset.endpointPosts,
            pages: root.dataset.endpointPages,
            categories: root.dataset.endpointCategories,
            categoriesIndex: root.dataset.endpointCategoriesIndex,
            comments: root.dataset.endpointComments,
            tags: root.dataset.endpointTags,
            tagsAdmin: root.dataset.endpointTagsAdmin,
            siteSettings: root.dataset.endpointSiteSettings,
            faviconUpload: root.dataset.endpointFaviconUpload,
            logoUpload: root.dataset.endpointLogoUpload,
            socialLinks: root.dataset.endpointSocialLinks,
            menuItems: root.dataset.endpointMenuItems,
        };

        const alertElement = document.getElementById('admin-alert');
        const showAlert = (message, type = 'info') => {
            if (!alertElement) {
                return;
            }
            if (typeof setAlert === 'function') {
                setAlert(alertElement, message, type);
                return;
            }
            alertElement.textContent = message || '';
            alertElement.hidden = !message;
        };

        const clearAlert = () => showAlert('');

        const handleRequestError = (error) => {
            if (!error) {
                return;
            }
            if (error.status === 401) {
                if (auth && typeof auth.clearToken === 'function') {
                    auth.clearToken();
                }
                window.location.href = '/login?redirect=/admin';
                return;
            }
            if (error.status === 403) {
                showAlert(
                    'You do not have permission to perform this action.',
                    'error'
                );
                return;
            }
            const message =
                error.message || 'Request failed. Please try again.';
            showAlert(message, 'error');
            console.error('Admin dashboard request failed', error);
        };

        const disableForm = (form, disabled) => {
            if (!form) {
                return;
            }
            if (typeof toggleFormDisabled === 'function') {
                toggleFormDisabled(form, disabled);
                return;
            }
            form.querySelectorAll('input, select, textarea, button').forEach(
                (field) => {
                    field.disabled = disabled;
                }
            );
        };

        const focusFirstField = (form) => {
            if (!form) {
                return null;
            }
            const selector = [
                'input:not([type="hidden"]):not([disabled])',
                'textarea:not([disabled])',
                'select:not([disabled])',
            ].join(', ');
            const field = form.querySelector(selector);
            if (field && typeof field.focus === 'function') {
                field.focus();
                return field;
            }
            if (typeof form.focus === 'function') {
                form.focus();
            }
            return field || null;
        };

        const bringFormIntoView = (form) => {
            if (!form) {
                return;
            }
            if (typeof form.scrollIntoView === 'function') {
                try {
                    form.scrollIntoView({ behavior: 'smooth', block: 'start' });
                } catch (error) {
                    form.scrollIntoView();
                }
            }
            const scheduleFocus = () => focusFirstField(form);
            if (typeof window.requestAnimationFrame === 'function') {
                window.requestAnimationFrame(scheduleFocus);
            } else {
                scheduleFocus();
            }
        };

        const metricElements = new Map();
        root.querySelectorAll('.admin__metric').forEach((card) => {
            const key = card.dataset.metric;
            const valueElement = card.querySelector('.admin__metric-value');
            if (key && valueElement) {
                metricElements.set(key, valueElement);
            }
        });

        const chartContainer = root.querySelector(
            '[data-role="metrics-chart"]'
        );
        const chartSvg = chartContainer?.querySelector('svg');
        const chartLegend = chartContainer?.querySelector(
            '[data-role="chart-legend"]'
        );
        const chartSummary = chartContainer?.querySelector(
            '[data-role="chart-summary"]'
        );
        const chartEmpty = chartContainer?.querySelector(
            '[data-role="chart-empty"]'
        );
        const chartSeries = [
            { key: 'posts', label: 'Posts', color: 'var(--admin-chart-posts)' },
            {
                key: 'comments',
                label: 'Comments',
                color: 'var(--admin-chart-comments)',
            },
        ];

        const tables = {
            posts: root.querySelector('#admin-posts-table'),
            pages: root.querySelector('#admin-pages-table'),
            categories: root.querySelector('#admin-categories-table'),
        };
        const postSearchInput = root.querySelector('[data-role="post-search"]');
        const pageSearchInput = root.querySelector('[data-role="page-search"]');
        const categorySearchInput = root.querySelector('[data-role="category-search"]');
        const commentsList = root.querySelector('#admin-comments-list');
        const postForm = root.querySelector('#admin-post-form');
        const pageForm = root.querySelector('#admin-page-form');
        const categoryForm = root.querySelector('#admin-category-form');
        const settingsForm = root.querySelector('#admin-settings-form');
        const socialList = root.querySelector('[data-role="social-list"]');
        const socialEmpty = root.querySelector('[data-role="social-empty"]');
        const socialForm = document.getElementById('admin-social-form');
        const menuList = root.querySelector('[data-role="menu-list"]');
        const menuEmpty = root.querySelector('[data-role="menu-empty"]');
        const menuForm = document.getElementById('admin-menu-form');
        const faviconUrlInput = settingsForm?.querySelector('input[name="favicon"]');
        const faviconUploadInput = settingsForm?.querySelector('[data-role="favicon-file"]');
        const faviconUploadButton = settingsForm?.querySelector('[data-role="favicon-upload"]');
        const faviconPreviewContainer = settingsForm?.querySelector('[data-role="favicon-preview"]');
        const faviconPreviewImage = settingsForm?.querySelector('[data-role="favicon-preview-image"]');
        const logoUrlInput = settingsForm?.querySelector('input[name="logo"]');
        const logoUploadInput = settingsForm?.querySelector('[data-role="logo-file"]');
        const logoUploadButton = settingsForm?.querySelector('[data-role="logo-upload"]');
        const logoPreviewContainer = settingsForm?.querySelector('[data-role="logo-preview"]');
        const logoPreviewImage = settingsForm?.querySelector('[data-role="logo-preview-image"]');
        const socialSubmitButton = socialForm?.querySelector('[data-role="social-submit"]');
        const socialCancelButton = socialForm?.querySelector('[data-role="social-cancel"]');
        const menuSubmitButton = menuForm?.querySelector('[data-role="menu-submit"]');
        const menuCancelButton = menuForm?.querySelector('[data-role="menu-cancel"]');
        const menuLocationField = menuForm?.querySelector('[data-role="menu-location"]');
        const postDeleteButton = postForm?.querySelector(
            '[data-role="post-delete"]'
        );
        const postSubmitButton = postForm?.querySelector(
            '[data-role="post-submit"]'
        );
        const pageDeleteButton = pageForm?.querySelector(
            '[data-role="page-delete"]'
        );
        const pageSubmitButton = pageForm?.querySelector(
            '[data-role="page-submit"]'
        );
        const categoryDeleteButton = categoryForm?.querySelector(
            '[data-role="category-delete"]'
        );
        const categorySubmitButton = categoryForm?.querySelector(
            '[data-role="category-submit"]'
        );
        const postCategorySelect = postForm?.querySelector(
            '#admin-post-category'
        );
        const postTagsInput = postForm?.querySelector('#admin-post-tags');
        const postFeaturedImageInput = postForm?.querySelector(
            'input[name="featured_img"]'
        );
        const tagList = document.getElementById('admin-tags-list');
        const postTagsList = document.getElementById('admin-post-tags-list');
        const DEFAULT_CATEGORY_SLUG = 'uncategorized';
        const pageSlugInput = pageForm?.querySelector('input[name="slug"]');
        const postSectionBuilder = postForm
            ? window.SectionBuilder?.init(
                  postForm.querySelector('[data-section-builder="post"]')
              )
            : null;
        const pageSectionBuilder = pageForm
            ? window.SectionBuilder?.init(
                  pageForm.querySelector('[data-section-builder="page"]')
              )
            : null;
        const pageContentField = pageForm?.querySelector('[name="content"]');
        const postContentField = postForm?.querySelector('[name="content"]');

        if (faviconUploadButton && !endpoints.faviconUpload) {
            faviconUploadButton.disabled = true;
            faviconUploadButton.title = 'Favicon uploads are not available.';
        }

        if (logoUploadButton && !endpoints.logoUpload) {
            logoUploadButton.disabled = true;
            logoUploadButton.title = 'Logo uploads are not available.';
        }

        const sectionBuilder = createSectionBuilder(postForm);
        if (sectionBuilder) {
            sectionBuilder.onChange((sections) => {
                if (!postContentField) {
                    return;
                }
                postContentField.value = generateContentPreview(sections);
            });
        }

        const state = {
            metrics: {},
            activityTrend: [],
            posts: [],
            pages: [],
            categories: [],
            comments: [],
            tags: [],
            socialLinks: [],
            menuItems: [],
            activeMenuLocation: 'header',
            isReorderingMenu: false,
            editingSocialLinkId: '',
            editingMenuItemId: '',
            defaultCategoryId: '',
            site: null,
            postSearchQuery: '',
            pageSearchQuery: '',
            categorySearchQuery: '',
            hasLoadedPosts: false,
            hasLoadedPages: false,
            hasLoadedCategories: false,
        };

        const updateFaviconPreview = (url) => {
            if (!faviconPreviewContainer || !faviconPreviewImage) {
                return;
            }

            const value = typeof url === 'string' ? url.trim() : '';
            if (!value) {
                faviconPreviewImage.src = '';
                faviconPreviewContainer.hidden = true;
                return;
            }

            const absoluteUrl =
                typeof buildAbsoluteUrl === 'function'
                    ? buildAbsoluteUrl(value, state.site)
                    : value;
            faviconPreviewImage.src = absoluteUrl || value;
            faviconPreviewContainer.hidden = false;
        };

        const updateLogoPreview = (url) => {
            if (!logoPreviewContainer || !logoPreviewImage) {
                return;
            }

            const value = typeof url === 'string' ? url.trim() : '';
            if (!value) {
                logoPreviewImage.src = '';
                logoPreviewContainer.hidden = true;
                return;
            }

            const absoluteUrl =
                typeof buildAbsoluteUrl === 'function'
                    ? buildAbsoluteUrl(value, state.site)
                    : value;
            logoPreviewImage.src = absoluteUrl || value;
            logoPreviewContainer.hidden = false;
        };

        const getPostPublicPath = (post) => {
            if (!post) {
                return '';
            }
            const slug = normaliseString(post.slug ?? post.Slug ?? '').trim();
            if (slug) {
                return `/blog/post/${encodeURIComponent(slug)}`;
            }
            const id = post.id ?? post.ID;
            if (id) {
                return `/blog/post/${encodeURIComponent(String(id))}`;
            }
            return '';
        };

        const getPagePublicPath = (page) => {
            if (!page) {
                return '';
            }
            const slug = normaliseString(page.slug ?? page.Slug ?? '').trim();
            if (slug) {
                return `/page/${encodeURIComponent(slug)}`;
            }
            const id = page.id ?? page.ID;
            if (id) {
                return `/page/${encodeURIComponent(String(id))}`;
            }
            return '';
        };

        const createLinkedCell = (label, path) => {
            const cell = createElement('td');
            const text = label?.toString().trim() || 'Untitled';
            if (path) {
                const link = createElement('a', { textContent: text });
                link.href = buildAbsoluteUrl(path, state.site);
                link.target = '_blank';
                link.rel = 'noopener noreferrer';
                link.addEventListener('click', (event) => {
                    event.stopPropagation();
                });
                cell.appendChild(link);
            } else {
                cell.textContent = text;
            }
            return cell;
        };

        const validateSections = (sections) => {
            if (!Array.isArray(sections)) {
                return '';
            }
            for (let index = 0; index < sections.length; index += 1) {
                const section = sections[index];
                if (!section) {
                    continue;
                }
                if (!section.title) {
                    return `Section ${index + 1} needs a title.`;
                }
                if (!Array.isArray(section.elements)) {
                    continue;
                }
                for (
                    let elementIndex = 0;
                    elementIndex < section.elements.length;
                    elementIndex += 1
                ) {
                    const element = section.elements[elementIndex];
                    if (!element) {
                        continue;
                    }
                    if (
                        element.type === 'paragraph' &&
                        !element.content?.text
                    ) {
                        return `Paragraph ${elementIndex + 1} in section "${
                            section.title
                        }" is empty.`;
                    }
                    if (element.type === 'image' && !element.content?.url) {
                        return `Image ${elementIndex + 1} in section "${
                            section.title
                        }" is missing a URL.`;
                    }
                    if (element.type === 'image_group') {
                        const images = Array.isArray(element.content?.images)
                            ? element.content.images
                            : [];
                        if (!images.length) {
                            return `The image group in section "${section.title}" needs at least one image.`;
                        }
                        const missing = images.findIndex((img) => !img?.url);
                        if (missing !== -1) {
                            return `Image ${
                                missing + 1
                            } in the group for section "${
                                section.title
                            }" is missing a URL.`;
                        }
                    }
                    if (element.type === 'list') {
                        const items = Array.isArray(element.content?.items)
                            ? element.content.items
                            : [];
                        const hasItems = items.some(
                            (item) => item && item.toString().trim()
                        );
                        if (!hasItems) {
                            return `List ${elementIndex + 1} in section "${
                                section.title
                            }" needs at least one item.`;
                        }
                    }
                }
            }
            return '';
        };

        const normaliseSlug = (value) =>
            typeof value === 'string' ? value.toLowerCase() : '';

        const extractCategorySlug = (category) => {
            if (!category) {
                return '';
            }
            const candidates = [category.slug, category.Slug];
            for (const candidate of candidates) {
                const normalised = normaliseSlug(candidate);
                if (normalised) {
                    return normalised;
                }
                if (candidate && typeof candidate.value === 'string') {
                    const nested = normaliseSlug(candidate.value);
                    if (nested) {
                        return nested;
                    }
                }
            }
            return normaliseSlug(category.name || category.Name || '');
        };

        const extractCategoryId = (category) => {
            if (!category) {
                return '';
            }
            const candidates = [category.id, category.ID];
            for (const candidate of candidates) {
                if (candidate === undefined || candidate === null) {
                    continue;
                }
                if (typeof candidate === 'object' && candidate !== null) {
                    const value = candidate.value ?? candidate.Value;
                    if (value !== undefined && value !== null) {
                        const normalised = String(value).trim();
                        if (normalised) {
                            return normalised;
                        }
                    }
                    continue;
                }
                const normalised = String(candidate).trim();
                if (normalised) {
                    return normalised;
                }
            }
            return '';
        };

        const extractSectionsFromEntry = (entry) => {
            const sections = entry?.sections || entry?.Sections;
            if (!Array.isArray(sections)) {
                return [];
            }
            return sections.slice().sort((a, b) => {
                const orderA = Number(a?.order ?? a?.Order ?? 0);
                const orderB = Number(b?.order ?? b?.Order ?? 0);
                return orderA - orderB;
            });
        };

        const refreshDefaultCategoryId = () => {
            const defaultSlug = normaliseSlug(DEFAULT_CATEGORY_SLUG);
            const matchBySlug = state.categories.find(
                (category) => extractCategorySlug(category) === defaultSlug
            );
            if (matchBySlug) {
                state.defaultCategoryId = extractCategoryId(matchBySlug);
                return;
            }
            const fallback = state.categories.find((category) =>
                extractCategoryId(category)
            );
            state.defaultCategoryId = fallback
                ? extractCategoryId(fallback)
                : '';
        };

        const ensureDefaultCategorySelection = () => {
            if (!postCategorySelect) {
                return;
            }
            if (!state.defaultCategoryId) {
                refreshDefaultCategoryId();
            }
            if (state.defaultCategoryId) {
                postCategorySelect.value = state.defaultCategoryId;
            }
            if (
                !postCategorySelect.value &&
                postCategorySelect.options.length
            ) {
                const firstUsable = Array.from(postCategorySelect.options).find(
                    (option) => option.value
                );
                if (firstUsable) {
                    postCategorySelect.value = firstUsable.value;
                }
            }
            if (
                !postCategorySelect.value &&
                postCategorySelect.options.length
            ) {
                postCategorySelect.selectedIndex = 0;
            }
            if (postCategorySelect.value) {
                state.defaultCategoryId = postCategorySelect.value;
            }
        };

        const normaliseTagName = (value) =>
            typeof value === 'string' ? value.trim() : '';

        const parseTags = (value) => {
            if (typeof value !== 'string' || !value.trim()) {
                return [];
            }
            const unique = new Map();
            value
                .split(',')
                .map((entry) => normaliseTagName(entry))
                .filter(Boolean)
                .forEach((name) => {
                    const key = name.toLowerCase();
                    if (!unique.has(key)) {
                        unique.set(key, name);
                    }
                });
            return Array.from(unique.values());
        };

        const extractTagNames = (entry) => {
            const tags = entry?.tags || entry?.Tags;
            if (!Array.isArray(tags)) {
                return [];
            }
            const unique = new Map();
            tags.forEach((tag) => {
                const name = normaliseTagName(tag?.name || tag?.Name);
                if (!name) {
                    return;
                }
                const key = name.toLowerCase();
                if (!unique.has(key)) {
                    unique.set(key, name);
                }
            });
            return Array.from(unique.values());
        };

        const normaliseSearchQuery = (value) =>
            typeof value === 'string' ? value.trim().toLowerCase() : '';

        const matchesSearchQuery = (fields, query) => {
            if (!query) {
                return true;
            }
            for (const field of fields) {
                const text = normaliseString(field).toLowerCase();
                if (text && text.includes(query)) {
                    return true;
                }
            }
            return false;
        };

        const getPostSearchFields = (post) => {
            const category = post?.category || post?.Category || {};
            return [
                post?.id,
                post?.ID,
                post?.title,
                post?.Title,
                post?.slug,
                post?.Slug,
                post?.description,
                post?.Description,
                post?.excerpt,
                post?.Excerpt,
                post?.category_name,
                post?.CategoryName,
                category?.name,
                category?.Name,
                category?.slug,
                category?.Slug,
                ...extractTagNames(post),
            ];
        };

        const getPageSearchFields = (page) => [
            page?.id,
            page?.ID,
            page?.title,
            page?.Title,
            page?.slug,
            page?.Slug,
            page?.description,
            page?.Description,
            page?.content,
            page?.Content,
        ];

        const getCategorySearchFields = (category) => [
            extractCategoryId(category),
            category?.name,
            category?.Name,
            extractCategorySlug(category),
            category?.slug,
            category?.Slug,
            category?.description,
            category?.Description,
        ];

        const renderTagSuggestions = () => {
            if (!postTagsList) {
                return;
            }
            const suggestions = new Map();
            const addSuggestion = (name) => {
                const cleaned = normaliseTagName(name);
                if (!cleaned) {
                    return;
                }
                const key = cleaned.toLowerCase();
                if (!suggestions.has(key)) {
                    suggestions.set(key, cleaned);
                }
            };

            state.tags.forEach((tag) => addSuggestion(tag?.name || tag?.Name));
            state.posts.forEach((post) => {
                extractTagNames(post).forEach(addSuggestion);
            });
            if (postTagsInput && postTagsInput.value) {
                parseTags(postTagsInput.value).forEach(addSuggestion);
            }

            const ordered = Array.from(suggestions.values()).sort((a, b) =>
                a.localeCompare(b, undefined, { sensitivity: 'base' })
            );

            postTagsList.innerHTML = '';
            ordered.forEach((name) => {
                const option = document.createElement('option');
                option.value = name;
                postTagsList.appendChild(option);
            });
        };

        const extractTagId = (tag) => {
            if (!tag) {
                return '';
            }
            if (typeof tag.id !== 'undefined' && tag.id !== null) {
                return String(tag.id);
            }
            if (typeof tag.ID !== 'undefined' && tag.ID !== null) {
                return String(tag.ID);
            }
            return '';
        };

        const extractTagSlug = (tag) => {
            if (!tag) {
                return '';
            }
            return normaliseSlug(
                tag.slug || tag.Slug || tag.name || tag.Name || ''
            );
        };

        const handleTagDelete = async (tag, button, usageCount = 0) => {
            if (!endpoints.tagsAdmin) {
                return;
            }
            const id = extractTagId(tag);
            if (!id) {
                return;
            }
            const name = normaliseTagName(tag?.name || tag?.Name);
            const label = name ? `"${name}"` : 'this tag';
            const usageText =
                usageCount === 1 ? '1 post' : `${usageCount} posts`;
            const confirmMessage =
                usageCount > 0
                    ? `The tag ${label} is used by ${usageText}. Deleting it will remove the tag from those posts. Continue?`
                    : `Delete the tag ${label}?`;
            if (!window.confirm(confirmMessage)) {
                return;
            }
            if (button) {
                button.disabled = true;
            }
            clearAlert();
            try {
                await apiRequest(`${endpoints.tagsAdmin}/${id}`, {
                    method: 'DELETE',
                });
                showAlert('Tag deleted successfully.', 'success');
                await loadTags();
                await loadPosts();
            } catch (error) {
                handleRequestError(error);
            } finally {
                if (button) {
                    button.disabled = false;
                }
            }
        };

        const renderTagList = () => {
            if (!tagList) {
                return;
            }
            tagList.innerHTML = '';
            if (!state.tags.length) {
                const empty = createElement('li', {
                    className: 'admin-tags__item admin-tags__item--empty',
                    textContent: 'No tags available.',
                });
                tagList.appendChild(empty);
                return;
            }

            const usage = new Map();
            state.posts.forEach((post) => {
                const tags = post?.tags || post?.Tags;
                if (!Array.isArray(tags)) {
                    return;
                }
                tags.forEach((entry) => {
                    const slug = extractTagSlug(entry);
                    if (!slug) {
                        return;
                    }
                    usage.set(slug, (usage.get(slug) || 0) + 1);
                });
            });

            const sorted = state.tags.slice().sort((a, b) => {
                const nameA = normaliseTagName(a?.name || a?.Name);
                const nameB = normaliseTagName(b?.name || b?.Name);
                return nameA.localeCompare(nameB, undefined, {
                    sensitivity: 'base',
                });
            });

            sorted.forEach((tag) => {
                const id = extractTagId(tag);
                const slug = extractTagSlug(tag);
                const name = normaliseTagName(tag?.name || tag?.Name);
                const count = usage.get(slug) || 0;

                const item = createElement('li', {
                    className: 'admin-tags__item',
                });
                item.dataset.id = id;

                const info = createElement('div', {
                    className: 'admin-tags__info',
                });
                info.appendChild(
                    createElement('span', {
                        className: 'admin-tags__name',
                        textContent: name ? `#${name}` : '(untitled)',
                    })
                );
                info.appendChild(
                    createElement('span', {
                        className: 'admin-tags__meta',
                        textContent: count === 1 ? '1 post' : `${count} posts`,
                    })
                );
                item.appendChild(info);

                const actions = createElement('div', {
                    className: 'admin-tags__actions',
                });
                const button = createElement('button', {
                    className: 'admin-tags__delete',
                    textContent: 'Delete',
                });
                button.type = 'button';
                button.addEventListener('click', () =>
                    handleTagDelete(tag, button, count)
                );
                actions.appendChild(button);
                item.appendChild(actions);

                tagList.appendChild(item);
            });
        };

        const highlightRow = (table, id) => {
            if (!table) {
                return;
            }
            table.querySelectorAll('tr').forEach((row) => {
                row.classList.toggle(
                    'is-selected',
                    id && String(row.dataset.id) === String(id)
                );
            });
        };

        const renderMetrics = (metrics = {}) => {
            Object.entries(metrics).forEach(([key, value]) => {
                const target = metricElements.get(key);
                if (target) {
                    target.textContent = Number.isFinite(Number(value))
                        ? Number(value).toLocaleString()
                        : String(value ?? '—');
                }
            });
        };

        const renderMetricsChart = (trend = []) => {
            if (
                !chartContainer ||
                !chartSvg ||
                !chartLegend ||
                !chartSummary ||
                !chartEmpty
            ) {
                return;
            }

            const normalised = Array.isArray(trend)
                ? trend
                      .map((entry) => {
                          const period =
                              entry?.period ||
                              entry?.Period ||
                              entry?.date ||
                              entry?.Date ||
                              '';
                          const postsValue = Number(
                              entry?.posts ?? entry?.Posts ?? 0
                          );
                          const commentsValue = Number(
                              entry?.comments ?? entry?.Comments ?? 0
                          );
                          return {
                              period,
                              posts: Number.isFinite(postsValue)
                                  ? Math.max(0, postsValue)
                                  : 0,
                              comments: Number.isFinite(commentsValue)
                                  ? Math.max(0, commentsValue)
                                  : 0,
                          };
                      })
                      .filter((entry) => entry.period)
                : [];

            const values = normalised.flatMap((point) =>
                chartSeries.map((series) => {
                    const numeric = Number(point[series.key]);
                    return Number.isFinite(numeric) ? Math.max(0, numeric) : 0;
                })
            );
            const maxValue = values.length ? Math.max(...values, 0) : 0;

            chartLegend.innerHTML = '';
            chartSummary.innerHTML = '';

            if (!normalised.length || maxValue <= 0) {
                chartSvg.innerHTML = '';
                chartEmpty.hidden = false;
                chartLegend.hidden = true;
                chartSummary.hidden = true;
                chartContainer.dataset.state = 'empty';
                return;
            }

            chartEmpty.hidden = true;
            chartLegend.hidden = false;
            chartSummary.hidden = false;
            chartContainer.dataset.state = 'ready';

            const width = 600;
            const height = 260;
            const topPadding = 16;
            const bottomPadding = 32;
            const chartHeight = height - topPadding - bottomPadding;
            const stepX =
                normalised.length > 1 ? width / (normalised.length - 1) : 0;

            chartSvg.setAttribute('viewBox', `0 0 ${width} ${height}`);
            chartSvg.innerHTML = '';

            const gridLines = 4;
            for (let index = 0; index <= gridLines; index += 1) {
                const y = topPadding + (chartHeight / gridLines) * index;
                const line = createSvgElement('line', {
                    x1: 0,
                    x2: width,
                    y1: y.toFixed(2),
                    y2: y.toFixed(2),
                    class: 'admin-chart__grid-line',
                });
                chartSvg.appendChild(line);
            }

            chartSeries.forEach((series) => {
                const pathData = normalised
                    .map((point, index) => {
                        const value = Number(point[series.key]);
                        const safeValue = Number.isFinite(value)
                            ? Math.max(0, value)
                            : 0;
                        const x =
                            normalised.length > 1 ? index * stepX : width / 2;
                        const y =
                            topPadding +
                            chartHeight -
                            (safeValue / maxValue) * chartHeight;
                        return `${index === 0 ? 'M' : 'L'}${x.toFixed(
                            2
                        )} ${y.toFixed(2)}`;
                    })
                    .join(' ');

                const path = createSvgElement('path', {
                    d: pathData,
                    class: 'admin-chart__line',
                    stroke: series.color,
                });
                path.dataset.series = series.key;
                chartSvg.appendChild(path);

                normalised.forEach((point, index) => {
                    const value = Number(point[series.key]);
                    const safeValue = Number.isFinite(value)
                        ? Math.max(0, value)
                        : 0;
                    const x = normalised.length > 1 ? index * stepX : width / 2;
                    const y =
                        topPadding +
                        chartHeight -
                        (safeValue / maxValue) * chartHeight;
                    const circle = createSvgElement('circle', {
                        cx: x.toFixed(2),
                        cy: y.toFixed(2),
                        r: 4,
                        class: 'admin-chart__point',
                        stroke: series.color,
                    });
                    circle.dataset.series = series.key;
                    chartSvg.appendChild(circle);
                });
            });

            chartSeries.forEach((series) => {
                const legendItem = document.createElement('li');
                legendItem.className = 'admin-chart__legend-item';
                legendItem.dataset.series = series.key;
                const swatch = document.createElement('span');
                swatch.className = 'admin-chart__legend-swatch';
                const label = document.createElement('span');
                label.className = 'admin-chart__legend-label';
                label.textContent = series.label;
                legendItem.appendChild(swatch);
                legendItem.appendChild(label);
                chartLegend.appendChild(legendItem);
            });

            normalised.forEach((point) => {
                const item = document.createElement('li');
                item.className = 'admin-chart__summary-item';

                const period = document.createElement('span');
                period.className = 'admin-chart__summary-period';
                period.textContent = formatMonthLabel(point.period) || '—';
                item.appendChild(period);

                chartSeries.forEach((series) => {
                    const value = Number(point[series.key]);
                    const safeValue = Number.isFinite(value)
                        ? Math.max(0, value)
                        : 0;
                    const valueElement = document.createElement('span');
                    valueElement.className = 'admin-chart__summary-value';
                    valueElement.dataset.series = series.key;
                    valueElement.textContent = `${formatNumber(
                        safeValue
                    )} ${series.label.toLowerCase()}`;
                    item.appendChild(valueElement);
                });

                chartSummary.appendChild(item);
            });
        };

        const renderPosts = () => {
            const table = tables.posts;
            if (!table) {
                return;
            }
            table.innerHTML = '';
            const posts = state.posts.filter((post) =>
                matchesSearchQuery(
                    getPostSearchFields(post),
                    state.postSearchQuery
                )
            );
            if (!posts.length) {
                const row = createElement('tr', {
                    className: 'admin-table__placeholder',
                });
                const cell = createElement('td', {
                    textContent: state.postSearchQuery
                        ? 'No posts match your search'
                        : 'No posts found',
                });
                cell.colSpan = 5;
                row.appendChild(cell);
                table.appendChild(row);
                renderTagList();
                return;
            }
            posts.forEach((post) => {
                const row = createElement('tr');
                row.dataset.id = post.id;
                row.appendChild(
                    createLinkedCell(
                        post.title || post.Title || 'Untitled',
                        getPostPublicPath(post)
                    )
                );
                const categoryName =
                    post.category?.name ||
                    post.category?.Name ||
                    post.category_name ||
                    post.CategoryName ||
                    '—';
                row.appendChild(
                    createElement('td', { textContent: categoryName || '—' })
                );
                const tagNames = extractTagNames(post).join(', ');
                row.appendChild(
                    createElement('td', { textContent: tagNames || '—' })
                );
                row.appendChild(
                    createElement('td', {
                        textContent: booleanLabel(post.published),
                    })
                );
                const updated =
                    post.updated_at || post.updatedAt || post.UpdatedAt;
                row.appendChild(
                    createElement('td', { textContent: formatDate(updated) })
                );
                row.addEventListener('click', () => selectPost(post.id));
                table.appendChild(row);
            });
            highlightRow(table, postForm?.dataset.id);
            renderTagList();
        };

        const renderPages = () => {
            const table = tables.pages;
            if (!table) {
                return;
            }
            table.innerHTML = '';
            const pages = state.pages.filter((page) =>
                matchesSearchQuery(
                    getPageSearchFields(page),
                    state.pageSearchQuery
                )
            );
            if (!pages.length) {
                const row = createElement('tr', {
                    className: 'admin-table__placeholder',
                });
                const cell = createElement('td', {
                    textContent: state.pageSearchQuery
                        ? 'No pages match your search'
                        : 'No pages found',
                });
                cell.colSpan = 4;
                row.appendChild(cell);
                table.appendChild(row);
                return;
            }
            pages.forEach((page) => {
                const row = createElement('tr');
                row.dataset.id = page.id;
                row.appendChild(
                    createLinkedCell(
                        page.title || page.Title || 'Untitled',
                        getPagePublicPath(page)
                    )
                );
                row.appendChild(
                    createElement('td', {
                        textContent: page.slug || page.Slug || '—',
                    })
                );
                row.appendChild(
                    createElement('td', {
                        textContent: booleanLabel(page.published),
                    })
                );
                const updated =
                    page.updated_at || page.updatedAt || page.UpdatedAt;
                row.appendChild(
                    createElement('td', { textContent: formatDate(updated) })
                );
                row.addEventListener('click', () => selectPage(page.id));
                table.appendChild(row);
            });
            highlightRow(table, pageForm?.dataset.id);
        };

        const setPostSearchQuery = (value) => {
            const next = normaliseSearchQuery(value);
            if (state.postSearchQuery === next) {
                return;
            }
            state.postSearchQuery = next;
            if (state.hasLoadedPosts) {
                renderPosts();
            }
        };

        const setPageSearchQuery = (value) => {
            const next = normaliseSearchQuery(value);
            if (state.pageSearchQuery === next) {
                return;
            }
            state.pageSearchQuery = next;
            if (state.hasLoadedPages) {
                renderPages();
            }
        };

        const setCategorySearchQuery = (value) => {
            const next = normaliseSearchQuery(value);
            if (state.categorySearchQuery === next) {
                return;
            }
            state.categorySearchQuery = next;
            if (state.hasLoadedCategories) {
                renderCategories();
            }
        };

        const renderCategories = () => {
            const table = tables.categories;
            if (!table) {
                return;
            }
            table.innerHTML = '';
            const categories = state.categories.filter((category) =>
                matchesSearchQuery(
                    getCategorySearchFields(category),
                    state.categorySearchQuery
                )
            );
            if (!categories.length) {
                const row = createElement('tr', {
                    className: 'admin-table__placeholder',
                });
                const cell = createElement('td', {
                    textContent: state.categorySearchQuery
                        ? 'No categories match your search'
                        : 'No categories found',
                });
                cell.colSpan = 3;
                row.appendChild(cell);
                table.appendChild(row);
                return;
            }
            categories.forEach((category) => {
                const id = extractCategoryId(category);
                if (!id) {
                    return;
                }
                const row = createElement('tr');
                row.dataset.id = id;
                row.appendChild(
                    createElement('td', {
                        textContent:
                            category.name ||
                            category.Name ||
                            'Untitled',
                    })
                );
                row.appendChild(
                    createElement('td', {
                        textContent:
                            category.slug ||
                            category.Slug ||
                            extractCategorySlug(category) ||
                            '—',
                    })
                );
                const updated =
                    category.updated_at ||
                    category.updatedAt ||
                    category.UpdatedAt;
                row.appendChild(
                    createElement('td', { textContent: formatDate(updated) })
                );
                row.addEventListener('click', () => selectCategory(id));
                table.appendChild(row);
            });
            highlightRow(table, categoryForm?.dataset.id);
        };

        const renderCategoryOptions = () => {
            if (!postCategorySelect) {
                return;
            }
            const currentValue = postCategorySelect.value;
            postCategorySelect.innerHTML = '';

            const seen = new Set();
            state.categories.forEach((category) => {
                const id = extractCategoryId(category);
                if (!id) {
                    return;
                }
                if (seen.has(id)) {
                    return;
                }
                seen.add(id);
                const option = createElement('option', {
                    textContent: category.name || 'Untitled',
                });
                option.value = id;
                postCategorySelect.appendChild(option);
            });

            if (
                currentValue &&
                state.categories.some(
                    (category) => extractCategoryId(category) === currentValue
                )
            ) {
                postCategorySelect.value = currentValue;
            } else {
                ensureDefaultCategorySelection();
            }
        };

        const renderComments = () => {
            if (!commentsList) {
                return;
            }
            commentsList.innerHTML = '';
            if (!state.comments.length) {
                const item = createElement('li', {
                    className:
                        'admin-comment-list__item admin-comment-list__item--empty',
                    textContent: 'No comments available',
                });
                commentsList.appendChild(item);
                return;
            }
            state.comments.forEach((comment) => {
                const item = createElement('li', {
                    className: 'admin-comment-list__item',
                });
                const meta = createElement('div', {
                    className: 'admin-comment-list__meta',
                });
                const pieces = [];
                if (comment.author?.username) {
                    pieces.push(`by ${comment.author.username}`);
                }
                if (comment.post?.title) {
                    pieces.push(`on "${comment.post.title}"`);
                }
                pieces.push(comment.approved ? 'approved' : 'pending approval');
                const created =
                    comment.created_at ||
                    comment.createdAt ||
                    comment.CreatedAt;
                pieces.push(formatDate(created));
                meta.textContent = pieces.join(' · ');
                const content = createElement('p', {
                    className: 'admin-comment-list__content',
                    textContent: comment.content || '(no content)',
                });
                const actions = createElement('div', {
                    className: 'admin-comment-list__actions',
                });
                if (!comment.approved) {
                    const approveButton = createElement('button', {
                        className: 'admin-comment-button',
                        textContent: 'Approve',
                    });
                    approveButton.dataset.action = 'approve';
                    approveButton.addEventListener('click', () =>
                        approveComment(comment.id, approveButton)
                    );
                    actions.appendChild(approveButton);
                } else {
                    const rejectButton = createElement('button', {
                        className: 'admin-comment-button',
                        textContent: 'Reject',
                    });
                    rejectButton.dataset.action = 'reject';
                    rejectButton.addEventListener('click', () =>
                        rejectComment(comment.id, rejectButton)
                    );
                    actions.appendChild(rejectButton);
                }
                const deleteButton = createElement('button', {
                    className: 'admin-comment-button',
                    textContent: 'Delete',
                });
                deleteButton.dataset.action = 'delete';
                deleteButton.addEventListener('click', () =>
                    deleteComment(comment.id, deleteButton)
                );
                actions.appendChild(deleteButton);
                item.appendChild(meta);
                item.appendChild(content);
                item.appendChild(actions);
                commentsList.appendChild(item);
            });
        };

        const selectPost = (id) => {
            if (!postForm) {
                return;
            }
            const post = state.posts.find(
                (entry) => String(entry.id) === String(id)
            );
            if (!post) {
                return;
            }
            postForm.dataset.id = post.id;
            postForm.title.value = post.title || '';
            postForm.description.value = post.description || '';
            if (postFeaturedImageInput) {
                const featured =
                    post.featured_img ||
                    post.featuredImg ||
                    post.FeaturedImg ||
                    '';
                postFeaturedImageInput.value = featured;
            }
            postForm.content.value = post.content || '';
            if (postContentField) {
                postContentField.value = post.content || '';
            }
            if (sectionBuilder) {
                const postSections = post.sections || post.Sections || [];
                sectionBuilder.setSections(postSections);
            }
            const categoryId =
                post.category?.id ||
                post.category?.ID ||
                post.category_id ||
                post.CategoryID;
            if (postCategorySelect) {
                if (categoryId) {
                    postCategorySelect.value = String(categoryId);
                } else {
                    ensureDefaultCategorySelection();
                }
            }
            if (postTagsInput) {
                postTagsInput.value = extractTagNames(post).join(', ');
            }
            const publishedField = postForm.querySelector(
                'input[name="published"]'
            );
            if (publishedField) {
                publishedField.checked = Boolean(post.published);
            }
            if (postSubmitButton) {
                postSubmitButton.textContent = 'Update post';
            }
            if (postDeleteButton) {
                postDeleteButton.hidden = false;
            }
            postSectionBuilder?.setSections(extractSectionsFromEntry(post));
            renderTagSuggestions();
            highlightRow(tables.posts, post.id);
        };

        const resetPostForm = () => {
            if (!postForm) {
                return;
            }
            postForm.reset();
            delete postForm.dataset.id;
            if (sectionBuilder) {
                sectionBuilder.reset();
            }
            if (postFeaturedImageInput) {
                postFeaturedImageInput.value = '';
            }
            ensureDefaultCategorySelection();
            if (postTagsInput) {
                postTagsInput.value = '';
            }
            if (postContentField) {
                postContentField.value = '';
            }
            if (postSubmitButton) {
                postSubmitButton.textContent = 'Create post';
            }
            if (postDeleteButton) {
                postDeleteButton.hidden = true;
            }
            postSectionBuilder?.reset();
            renderTagSuggestions();
            highlightRow(tables.posts);
            bringFormIntoView(postForm);
        };

        const selectPage = (id) => {
            if (!pageForm) {
                return;
            }
            const page = state.pages.find(
                (entry) => String(entry.id) === String(id)
            );
            if (!page) {
                return;
            }
            pageForm.dataset.id = page.id;
            pageForm.title.value = page.title || '';
            if (pageSlugInput) {
                pageSlugInput.value = page.slug || '';
                pageSlugInput.disabled = true;
                pageSlugInput.title =
                    'The slug is generated from the title when updating';
            }
            pageForm.description.value = page.description || '';
            if (pageContentField) {
                pageContentField.value = page.content || page.Content || '';
            }
            const orderInput = pageForm.querySelector('input[name="order"]');
            if (orderInput) {
                orderInput.value = page.order ?? 0;
            }
            const publishedField = pageForm.querySelector(
                'input[name="published"]'
            );
            if (publishedField) {
                publishedField.checked = Boolean(page.published);
            }
            if (pageSubmitButton) {
                pageSubmitButton.textContent = 'Update page';
            }
            if (pageDeleteButton) {
                pageDeleteButton.hidden = false;
            }
            pageSectionBuilder?.setSections(extractSectionsFromEntry(page));
            highlightRow(tables.pages, page.id);
        };

        const resetPageForm = () => {
            if (!pageForm) {
                return;
            }
            pageForm.reset();
            delete pageForm.dataset.id;
            if (pageSubmitButton) {
                pageSubmitButton.textContent = 'Create page';
            }
            if (pageDeleteButton) {
                pageDeleteButton.hidden = true;
            }
            if (pageSlugInput) {
                pageSlugInput.disabled = false;
                pageSlugInput.title = 'Optional custom slug';
            }
            const orderInput = pageForm.querySelector('input[name="order"]');
            if (orderInput) {
                orderInput.value = 0;
            }
            if (pageContentField) {
                pageContentField.value = '';
            }
            pageSectionBuilder?.reset();
            highlightRow(tables.pages);
            bringFormIntoView(pageForm);
        };

        const selectCategory = (id) => {
            if (!categoryForm) {
                return;
            }
            const category = state.categories.find(
                (entry) => extractCategoryId(entry) === String(id)
            );
            if (!category) {
                return;
            }
            const categoryId = extractCategoryId(category);
            if (categoryId) {
                categoryForm.dataset.id = categoryId;
            } else {
                delete categoryForm.dataset.id;
            }
            categoryForm.name.value = category.name || '';
            categoryForm.description.value = category.description || '';
            if (categorySubmitButton) {
                categorySubmitButton.textContent = 'Update category';
            }
            if (categoryDeleteButton) {
                categoryDeleteButton.hidden = false;
            }
            highlightRow(tables.categories, categoryId);
        };

        const resetCategoryForm = () => {
            if (!categoryForm) {
                return;
            }
            categoryForm.reset();
            delete categoryForm.dataset.id;
            if (categorySubmitButton) {
                categorySubmitButton.textContent = 'Create category';
            }
            if (categoryDeleteButton) {
                categoryDeleteButton.hidden = true;
            }
            highlightRow(tables.categories);
            bringFormIntoView(pageForm);
        };

        const loadStats = async () => {
            if (!endpoints.stats) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.stats);
                const metrics = payload?.statistics || {};
                state.metrics = metrics;
                renderMetrics(metrics);
                const trend = Array.isArray(payload?.activity_trend)
                    ? payload.activity_trend
                    : [];
                state.activityTrend = trend;
                renderMetricsChart(trend);
            } catch (error) {
                handleRequestError(error);
            }
        };

        const loadPosts = async () => {
            if (!endpoints.posts) {
                return;
            }
            try {
                const payload = await apiRequest(`${endpoints.posts}?limit=50`);
                state.posts = payload?.posts || [];
                state.hasLoadedPosts = true;
                renderPosts();
                renderTagSuggestions();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const loadPages = async () => {
            if (!endpoints.pages) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.pages);
                state.pages = payload?.pages || [];
                state.hasLoadedPages = true;
                renderPages();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const loadCategories = async () => {
            if (!endpoints.categoriesIndex) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.categoriesIndex);
                state.categories = payload?.categories || [];
                state.hasLoadedCategories = true;
                refreshDefaultCategoryId();
                renderCategories();
                renderCategoryOptions();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const loadComments = async () => {
            if (!endpoints.comments) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.comments);
                const comments = payload?.comments || [];
                comments.sort((a, b) => {
                    const aDate = new Date(
                        a.created_at || a.createdAt || a.CreatedAt || 0
                    ).getTime();
                    const bDate = new Date(
                        b.created_at || b.createdAt || b.CreatedAt || 0
                    ).getTime();
                    return bDate - aDate;
                });
                state.comments = comments.slice(0, 15);
                renderComments();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const loadTags = async () => {
            if (!endpoints.tags) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.tags);
                state.tags = payload?.tags || [];
                renderTagSuggestions();
                renderTagList();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const populateSiteSettingsForm = (site) => {
            if (!settingsForm) {
                return;
            }

            const entries = [
                ['name', site?.name],
                ['description', site?.description],
                ['url', site?.url],
                ['favicon', site?.favicon],
                ['logo', site?.logo],
                ['unused_tag_retention_hours', site?.unused_tag_retention_hours],
            ];

            entries.forEach(([key, value]) => {
                const field = settingsForm.querySelector(`[name="${key}"]`);
                if (!field) {
                    return;
                }
                field.value = value || '';
            });

            updateFaviconPreview(site?.favicon || site?.Favicon || '');
            updateLogoPreview(site?.logo || site?.Logo || '');
        };

        const loadSiteSettings = async () => {
            if (!endpoints.siteSettings) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.siteSettings);
                state.site = payload?.site || null;
                populateSiteSettingsForm(state.site);
            } catch (error) {
                handleRequestError(error);
            }
        };

        const handleFaviconUploadClick = () => {
            if (!faviconUploadInput) {
                return;
            }
            if (!endpoints.faviconUpload) {
                showAlert('Favicon uploads are not available in this environment.', 'error');
                return;
            }
            faviconUploadInput.click();
        };

        const handleFaviconFileChange = async (event) => {
            const input = event?.target;
            if (!input || !input.files || !input.files.length) {
                return;
            }

            if (!endpoints.faviconUpload) {
                showAlert('Favicon uploads are not available in this environment.', 'error');
                input.value = '';
                return;
            }

            const file = input.files[0];
            const button = faviconUploadButton;
            const originalLabel = button?.textContent;

            if (button) {
                button.disabled = true;
                button.setAttribute('data-loading', 'true');
                if (typeof originalLabel === 'string') {
                    button.textContent = 'Uploading…';
                }
            }

            clearAlert();

            const formData = new FormData();
            formData.append('favicon', file);

            try {
                const response = await apiRequest(endpoints.faviconUpload, {
                    method: 'POST',
                    body: formData,
                });

                const site = response?.site;
                if (site) {
                    state.site = site;
                } else if (!state.site) {
                    state.site = {};
                }

                const faviconUrl =
                    response?.favicon ||
                    site?.favicon ||
                    state.site?.favicon ||
                    '';

                const faviconType =
                    response?.favicon_type ||
                    response?.faviconType ||
                    site?.favicon_type ||
                    state.site?.favicon_type ||
                    '';

                if (state.site) {
                    state.site.favicon = faviconUrl;
                    if (faviconType) {
                        state.site.favicon_type = faviconType;
                    }
                }

                if (faviconUrlInput) {
                    faviconUrlInput.value = faviconUrl || '';
                }

                updateFaviconPreview(faviconUrl);
                showAlert('Favicon uploaded successfully.', 'success');
            } catch (error) {
                handleRequestError(error);
            } finally {
                if (button) {
                    button.disabled = false;
                    button.removeAttribute('data-loading');
                    if (typeof originalLabel === 'string') {
                        button.textContent = originalLabel;
                    }
                }
                if (input) {
                    input.value = '';
                }
            }
        };

        const handleLogoUploadClick = () => {
            if (!logoUploadInput) {
                return;
            }
            if (!endpoints.logoUpload) {
                showAlert('Logo uploads are not available in this environment.', 'error');
                return;
            }
            logoUploadInput.click();
        };

        const handleLogoFileChange = async (event) => {
            const input = event?.target;
            if (!input || !input.files || !input.files.length) {
                return;
            }

            if (!endpoints.logoUpload) {
                showAlert('Logo uploads are not available in this environment.', 'error');
                input.value = '';
                return;
            }

            const file = input.files[0];
            const button = logoUploadButton;
            const originalLabel = button?.textContent;

            if (button) {
                button.disabled = true;
                button.setAttribute('data-loading', 'true');
                if (typeof originalLabel === 'string') {
                    button.textContent = 'Uploading…';
                }
            }

            clearAlert();

            const formData = new FormData();
            formData.append('logo', file);

            try {
                const response = await apiRequest(endpoints.logoUpload, {
                    method: 'POST',
                    body: formData,
                });

                const site = response?.site;
                if (site) {
                    state.site = site;
                } else if (!state.site) {
                    state.site = {};
                }

                const logoUrl =
                    response?.logo ||
                    site?.logo ||
                    state.site?.logo ||
                    '';

                if (state.site) {
                    state.site.logo = logoUrl;
                }

                if (logoUrlInput) {
                    logoUrlInput.value = logoUrl || '';
                }

                updateLogoPreview(logoUrl);
                showAlert('Logo uploaded successfully.', 'success');
            } catch (error) {
                handleRequestError(error);
            } finally {
                if (button) {
                    button.disabled = false;
                    button.removeAttribute('data-loading');
                    if (typeof originalLabel === 'string') {
                        button.textContent = originalLabel;
                    }
                }
                if (input) {
                    input.value = '';
                }
            }
        };

        const renderSocialLinks = () => {
            if (!socialList) {
                return;
            }
            const links = Array.isArray(state.socialLinks)
                ? state.socialLinks
                : [];
            socialList
                .querySelectorAll('[data-role="social-item"]')
                .forEach((item) => item.remove());
            if (!links.length) {
                if (socialEmpty) {
                    socialEmpty.hidden = false;
                }
                return;
            }
            if (socialEmpty) {
                socialEmpty.hidden = true;
            }
            links.forEach((link) => {
                if (!link) {
                    return;
                }
                const li = document.createElement('li');
                li.className = 'admin-social__item';
                li.dataset.role = 'social-item';
                const idValue = link.id || link.ID || link.Id;
                if (idValue !== undefined) {
                    li.dataset.id = String(idValue);
                }

                const details = document.createElement('div');
                details.className = 'admin-social__details';

                const name = document.createElement('span');
                name.className = 'admin-social__name';
                name.textContent = link.name || link.Name || 'Social link';
                details.appendChild(name);

                const url = document.createElement('a');
                url.className = 'admin-social__url';
                url.href = link.url || link.URL || '#';
                url.target = '_blank';
                url.rel = 'noopener noreferrer';
                url.textContent = link.url || link.URL || '';
                details.appendChild(url);

                const actions = document.createElement('div');
                actions.className = 'admin-social__actions';

                const editButton = document.createElement('button');
                editButton.type = 'button';
                editButton.className = 'admin-social__button';
                editButton.dataset.action = 'edit';
                editButton.textContent = 'Edit';
                actions.appendChild(editButton);

                const deleteButton = document.createElement('button');
                deleteButton.type = 'button';
                deleteButton.className = 'admin-social__button admin-social__button--danger';
                deleteButton.dataset.action = 'delete';
                deleteButton.textContent = 'Delete';
                actions.appendChild(deleteButton);

                li.appendChild(details);
                li.appendChild(actions);
                socialList.appendChild(li);
            });
        };

        const resetSocialForm = () => {
            if (!socialForm) {
                return;
            }
            socialForm.reset();
            const idField = socialForm.querySelector('input[name="id"]');
            if (idField) {
                idField.value = '';
            }
            state.editingSocialLinkId = '';
            if (socialSubmitButton) {
                socialSubmitButton.textContent = 'Save social link';
            }
            if (socialCancelButton) {
                socialCancelButton.hidden = true;
                socialCancelButton.disabled = false;
            }
        };

        const startEditSocialLink = (link) => {
            if (!socialForm || !link) {
                return;
            }
            const idField = socialForm.querySelector('input[name="id"]');
            const nameField = socialForm.querySelector('input[name="name"]');
            const urlField = socialForm.querySelector('input[name="url"]');
            const iconField = socialForm.querySelector('input[name="icon"]');

            const idValue = link.id || link.ID || link.Id;
            if (idField) {
                idField.value = idValue ? String(idValue) : '';
            }
            if (nameField) {
                nameField.value = link.name || link.Name || '';
            }
            if (urlField) {
                urlField.value = link.url || link.URL || '';
            }
            if (iconField) {
                iconField.value = link.icon || link.Icon || '';
            }
            state.editingSocialLinkId = idField?.value || '';
            if (socialSubmitButton) {
                socialSubmitButton.textContent = 'Update social link';
            }
            if (socialCancelButton) {
                socialCancelButton.hidden = false;
            }
            bringFormIntoView(socialForm);
        };

        const loadSocialLinks = async () => {
            if (!endpoints.socialLinks) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.socialLinks);
                state.socialLinks = payload?.social_links || [];
                renderSocialLinks();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const handleSocialFormSubmit = async (event) => {
            event.preventDefault();
            if (!socialForm || !endpoints.socialLinks) {
                return;
            }

            const nameField = socialForm.querySelector('input[name="name"]');
            const urlField = socialForm.querySelector('input[name="url"]');
            const iconField = socialForm.querySelector('input[name="icon"]');

            const name = nameField ? nameField.value.trim() : '';
            const url = urlField ? urlField.value.trim() : '';
            const icon = iconField ? iconField.value.trim() : '';

            if (!name) {
                showAlert('Please provide the social network name.', 'error');
                focusFirstField(socialForm);
                return;
            }

            if (!url) {
                showAlert('Please provide the URL for the social profile.', 'error');
                focusFirstField(socialForm);
                return;
            }

            const payload = { name, url, icon };
            const isEditing = Boolean(state.editingSocialLinkId);
            const endpoint = isEditing
                ? `${endpoints.socialLinks}/${state.editingSocialLinkId}`
                : endpoints.socialLinks;
            const method = isEditing ? 'PUT' : 'POST';

            disableForm(socialForm, true);
            clearAlert();

            try {
                await apiRequest(endpoint, {
                    method,
                    body: JSON.stringify(payload),
                });
                await loadSocialLinks();
                showAlert(
                    isEditing
                        ? 'Social link updated successfully.'
                        : 'Social link created successfully.',
                    'success'
                );
                resetSocialForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(socialForm, false);
            }
        };

        const handleSocialCancelEdit = () => {
            resetSocialForm();
        };

        const handleSocialListClick = async (event) => {
            const button = event.target?.closest('[data-action]');
            if (!button || !socialList || !endpoints.socialLinks) {
                return;
            }

            const listItem = button.closest('[data-role="social-item"]');
            if (!listItem) {
                return;
            }

            const id = listItem.dataset.id;
            if (!id) {
                return;
            }

            if (button.dataset.action === 'edit') {
                const link = state.socialLinks.find(
                    (item) => String(item?.id || item?.ID || item?.Id) === id
                );
                if (link) {
                    startEditSocialLink(link);
                }
                return;
            }

            if (button.dataset.action === 'delete') {
                if (!window.confirm('Delete this social link?')) {
                    return;
                }
                disableForm(socialForm, true);
                clearAlert();
                try {
                    await apiRequest(`${endpoints.socialLinks}/${id}`, {
                        method: 'DELETE',
                    });
                    showAlert('Social link deleted.', 'success');
                    if (state.editingSocialLinkId === id) {
                        resetSocialForm();
                    }
                    await loadSocialLinks();
                } catch (error) {
                    handleRequestError(error);
                } finally {
                    disableForm(socialForm, false);
                }
            }
        };

        const getMenuItemId = (item) => {
            if (!item) {
                return NaN;
            }
            const idValue =
                item.id ?? item.ID ?? item.Id ?? item.menu_item_id ?? item.MenuItemId;
            const numericId = Number(idValue);
            return Number.isFinite(numericId) ? numericId : NaN;
        };

        const getMenuItemOrder = (item) => {
            if (!item) {
                return 0;
            }
            const orderValue = item.order ?? item.Order ?? 0;
            const numericOrder = Number(orderValue);
            return Number.isFinite(numericOrder) ? numericOrder : 0;
        };

        const normaliseMenuLocation = (value) => {
            const location =
                typeof value === 'string' ? value.trim().toLowerCase() : '';
            return location || 'header';
        };

        const getMenuItemLocation = (item) => {
            if (!item || typeof item !== 'object') {
                return 'header';
            }
            const locationValue = item.location ?? item.Location ?? '';
            return normaliseMenuLocation(locationValue);
        };

        const getActiveMenuLocation = () =>
            normaliseMenuLocation(state.activeMenuLocation);

        const formatMenuLocationLabel = (value) => {
            const location = normaliseMenuLocation(value);
            if (location === 'footer') {
                return 'Footer';
            }
            return 'Header';
        };

        const renderMenuItems = () => {
            if (!menuList) {
                return;
            }

            const activeLocation = getActiveMenuLocation();
            const items = Array.isArray(state.menuItems)
                ? state.menuItems.filter(
                      (item) => getMenuItemLocation(item) === activeLocation
                  )
                : [];

            const sortedItems = items.slice().sort((a, b) => {
                const orderDiff = getMenuItemOrder(a) - getMenuItemOrder(b);
                if (orderDiff !== 0 && Number.isFinite(orderDiff)) {
                    return orderDiff;
                }
                const idDiff = getMenuItemId(a) - getMenuItemId(b);
                if (Number.isFinite(idDiff)) {
                    return idDiff;
                }
                return 0;
            });

            menuList
                .querySelectorAll('[data-role="menu-item"]')
                .forEach((item) => item.remove());

            if (menuEmpty) {
                const label = formatMenuLocationLabel(activeLocation);
                menuEmpty.textContent = `No menu items added for the ${label.toLowerCase()} menu yet.`;
            }

            if (!sortedItems.length) {
                if (menuEmpty) {
                    menuEmpty.hidden = false;
                }
                if (menuLocationField) {
                    menuLocationField.value = activeLocation;
                }
                return;
            }

            if (menuEmpty) {
                menuEmpty.hidden = true;
            }
            if (menuLocationField) {
                menuLocationField.value = activeLocation;
            }

            sortedItems.forEach((item, index) => {
                if (!item) {
                    return;
                }
                const li = document.createElement('li');
                li.className = 'admin-navigation__item';
                li.dataset.role = 'menu-item';
                const idValue = item.id || item.ID || item.Id;
                if (idValue !== undefined) {
                    li.dataset.id = String(idValue);
                }

                const orderValue = getMenuItemOrder(item);
                const displayOrder = orderValue > 0 ? orderValue : index + 1;
                li.dataset.order = String(displayOrder);

                const orderControls = document.createElement('div');
                orderControls.className = 'admin-navigation__order';

                const orderNumber = document.createElement('span');
                orderNumber.className = 'admin-navigation__order-number';
                orderNumber.textContent = String(displayOrder);
                const orderLabel = `Position ${displayOrder}`;
                orderNumber.title = orderLabel;
                orderNumber.setAttribute('aria-label', orderLabel);
                orderControls.appendChild(orderNumber);

                const orderButtons = document.createElement('div');
                orderButtons.className = 'admin-navigation__order-buttons';

                const moveUpButton = document.createElement('button');
                moveUpButton.type = 'button';
                moveUpButton.className = 'admin-navigation__reorder-button';
                moveUpButton.dataset.action = 'move-up';
                moveUpButton.title = 'Move item up';
                moveUpButton.setAttribute('aria-label', 'Move menu item up');
                moveUpButton.textContent = '▲';

                const moveDownButton = document.createElement('button');
                moveDownButton.type = 'button';
                moveDownButton.className = 'admin-navigation__reorder-button';
                moveDownButton.dataset.action = 'move-down';
                moveDownButton.title = 'Move item down';
                moveDownButton.setAttribute('aria-label', 'Move menu item down');
                moveDownButton.textContent = '▼';

                const isFirst = index === 0;
                const isLast = index === sortedItems.length - 1;
                moveUpButton.disabled = isFirst || state.isReorderingMenu;
                moveDownButton.disabled = isLast || state.isReorderingMenu;
                if (state.isReorderingMenu) {
                    moveUpButton.setAttribute('aria-disabled', 'true');
                    moveDownButton.setAttribute('aria-disabled', 'true');
                }

                orderButtons.appendChild(moveUpButton);
                orderButtons.appendChild(moveDownButton);
                orderControls.appendChild(orderButtons);

                const details = document.createElement('div');
                details.className = 'admin-navigation__details';

                const label = document.createElement('span');
                label.className = 'admin-navigation__label';
                label.textContent = item.title || item.Title || 'Menu item';
                details.appendChild(label);

                const itemLocation = getMenuItemLocation(item);
                const locationMeta = document.createElement('span');
                locationMeta.className = 'admin-navigation__meta';
                locationMeta.textContent = `${formatMenuLocationLabel(itemLocation)} menu`;
                details.appendChild(locationMeta);

                const link = document.createElement('a');
                link.className = 'admin-navigation__url';
                const href = item.url || item.URL || '#';
                const resolvedHref =
                    typeof buildAbsoluteUrl === 'function'
                        ? buildAbsoluteUrl(href, state.site)
                        : href;
                link.href = resolvedHref || href;
                link.target = '_blank';
                link.rel = 'noopener noreferrer';
                link.textContent = href;
                details.appendChild(link);

                const actions = document.createElement('div');
                actions.className = 'admin-navigation__actions';

                const editButton = document.createElement('button');
                editButton.type = 'button';
                editButton.className = 'admin-navigation__button';
                editButton.dataset.action = 'edit';
                editButton.textContent = 'Edit';
                actions.appendChild(editButton);

                const deleteButton = document.createElement('button');
                deleteButton.type = 'button';
                deleteButton.className =
                    'admin-navigation__button admin-navigation__button--danger';
                deleteButton.dataset.action = 'delete';
                deleteButton.textContent = 'Delete';
                actions.appendChild(deleteButton);

                li.appendChild(orderControls);
                li.appendChild(details);
                li.appendChild(actions);
                menuList.appendChild(li);
            });
        };

        const resetMenuForm = () => {
            if (!menuForm) {
                return;
            }
            menuForm.reset();
            if (menuLocationField) {
                menuLocationField.value = getActiveMenuLocation();
            }
            const idField = menuForm.querySelector('input[name="id"]');
            if (idField) {
                idField.value = '';
            }
            state.editingMenuItemId = '';
            if (menuSubmitButton) {
                menuSubmitButton.textContent = 'Save menu item';
            }
            if (menuCancelButton) {
                menuCancelButton.hidden = true;
                menuCancelButton.disabled = false;
            }
        };

        const startEditMenuItem = (item) => {
            if (!menuForm || !item) {
                return;
            }
            const idField = menuForm.querySelector('input[name="id"]');
            const titleField = menuForm.querySelector('input[name="title"]');
            const urlField = menuForm.querySelector('input[name="url"]');
            const location = getMenuItemLocation(item);

            const idValue = item.id || item.ID || item.Id;
            if (idField) {
                idField.value = idValue ? String(idValue) : '';
            }
            if (titleField) {
                titleField.value = item.title || item.Title || '';
            }
            if (urlField) {
                urlField.value = item.url || item.URL || '';
            }
            if (menuLocationField) {
                menuLocationField.value = location;
            }

            state.activeMenuLocation = location;
            renderMenuItems();

            state.editingMenuItemId = idField?.value || '';
            if (menuSubmitButton) {
                menuSubmitButton.textContent = 'Update menu item';
            }
            if (menuCancelButton) {
                menuCancelButton.hidden = false;
            }
            bringFormIntoView(menuForm);
        };

        const loadMenuItems = async () => {
            if (!endpoints.menuItems) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.menuItems);
                const items = Array.isArray(payload?.menu_items)
                    ? payload.menu_items.slice()
                    : [];
                items.sort((a, b) => {
                    const orderDiff = getMenuItemOrder(a) - getMenuItemOrder(b);
                    if (orderDiff !== 0 && Number.isFinite(orderDiff)) {
                        return orderDiff;
                    }
                    const idDiff = getMenuItemId(a) - getMenuItemId(b);
                    if (Number.isFinite(idDiff)) {
                        return idDiff;
                    }
                    return 0;
                });
                const availableLocations = new Set(
                    items.map((entry) => getMenuItemLocation(entry))
                );
                if (availableLocations.size > 0) {
                    const currentLocation = getActiveMenuLocation();
                    if (!availableLocations.has(currentLocation)) {
                        let fallbackLocation = 'header';
                        if (!availableLocations.has(fallbackLocation)) {
                            const iterator = availableLocations.values().next();
                            if (!iterator.done) {
                                fallbackLocation = iterator.value;
                            }
                        }
                        state.activeMenuLocation = normaliseMenuLocation(
                            fallbackLocation
                        );
                    }
                }
                state.menuItems = items;
                renderMenuItems();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const moveMenuItem = async (id, direction) => {
            if (state.isReorderingMenu || !endpoints.menuItems) {
                return false;
            }

            const step = Number(direction);
            if (!Number.isFinite(step) || step === 0) {
                return false;
            }

            const activeLocation = getActiveMenuLocation();
            const allItems = Array.isArray(state.menuItems)
                ? state.menuItems.slice()
                : [];
            if (!allItems.length) {
                return false;
            }

            const locationItems = allItems
                .filter(
                    (entry) => getMenuItemLocation(entry) === activeLocation
                )
                .sort((a, b) => {
                    const orderDiff = getMenuItemOrder(a) - getMenuItemOrder(b);
                    if (orderDiff !== 0 && Number.isFinite(orderDiff)) {
                        return orderDiff;
                    }
                    const idDiff = getMenuItemId(a) - getMenuItemId(b);
                    if (Number.isFinite(idDiff)) {
                        return idDiff;
                    }
                    return 0;
                });

            if (!locationItems.length) {
                return false;
            }

            const currentIndex = locationItems.findIndex(
                (entry) => String(getMenuItemId(entry)) === String(id)
            );
            if (currentIndex < 0) {
                return false;
            }

            const targetIndex = currentIndex + step;
            if (targetIndex < 0 || targetIndex >= locationItems.length) {
                return false;
            }

            const [movedItem] = locationItems.splice(currentIndex, 1);
            locationItems.splice(targetIndex, 0, movedItem);

            const orders = locationItems
                .map((entry, position) => {
                    const entryId = getMenuItemId(entry);
                    if (!Number.isFinite(entryId) || entryId <= 0) {
                        return null;
                    }
                    return { id: entryId, order: position + 1 };
                })
                .filter(Boolean);

            if (!orders.length) {
                return false;
            }

            state.isReorderingMenu = true;
            renderMenuItems();

            let success = false;
            try {
                await apiRequest(`${endpoints.menuItems}/reorder`, {
                    method: 'PUT',
                    body: JSON.stringify({ orders }),
                });
                const updatedOrders = new Map(
                    orders.map((entry) => [String(entry.id), entry.order])
                );
                state.menuItems = allItems.map((entry) => {
                    if (!entry || typeof entry !== 'object') {
                        return entry;
                    }
                    const entryId = getMenuItemId(entry);
                    const key = String(entryId);
                    if (!updatedOrders.has(key)) {
                        return entry;
                    }
                    const orderValue = updatedOrders.get(key);
                    const updated = { ...entry };
                    if ('order' in updated || !('Order' in updated)) {
                        updated.order = orderValue;
                    }
                    if ('Order' in updated) {
                        updated.Order = orderValue;
                    }
                    return updated;
                });
                success = true;
            } catch (error) {
                handleRequestError(error);
            } finally {
                state.isReorderingMenu = false;
                renderMenuItems();
            }

            return success;
        };

        const handleMenuFormSubmit = async (event) => {
            event.preventDefault();
            if (!menuForm || !endpoints.menuItems) {
                return;
            }

            const titleField = menuForm.querySelector('input[name="title"]');
            const urlField = menuForm.querySelector('input[name="url"]');

            const title = titleField ? titleField.value.trim() : '';
            const url = urlField ? urlField.value.trim() : '';

            if (!title) {
                showAlert('Please provide the menu label.', 'error');
                focusFirstField(menuForm);
                return;
            }

            if (!url) {
                showAlert('Please provide the destination URL.', 'error');
                focusFirstField(menuForm);
                return;
            }

            const location = menuLocationField
                ? normaliseMenuLocation(menuLocationField.value)
                : getActiveMenuLocation();
            const payload = { title, url, location };
            const isEditing = Boolean(state.editingMenuItemId);
            const endpoint = isEditing
                ? `${endpoints.menuItems}/${state.editingMenuItemId}`
                : endpoints.menuItems;
            const method = isEditing ? 'PUT' : 'POST';

            disableForm(menuForm, true);
            clearAlert();

            try {
                await apiRequest(endpoint, {
                    method,
                    body: JSON.stringify(payload),
                });
                state.activeMenuLocation = location;
                await loadMenuItems();
                showAlert(
                    isEditing
                        ? 'Menu item updated successfully.'
                        : 'Menu item created successfully.',
                    'success'
                );
                resetMenuForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(menuForm, false);
            }
        };

        const handleMenuCancelEdit = () => {
            resetMenuForm();
        };

        const handleMenuLocationChange = (event) => {
            const selected = normaliseMenuLocation(event?.target?.value);
            state.activeMenuLocation = selected;
            if (state.editingMenuItemId) {
                resetMenuForm();
            }
            renderMenuItems();
        };

        const handleMenuListClick = async (event) => {
            const button = event.target?.closest('[data-action]');
            if (!button || !menuList || !endpoints.menuItems) {
                return;
            }

            const listItem = button.closest('[data-role="menu-item"]');
            if (!listItem) {
                return;
            }

            const id = listItem.dataset.id;
            if (!id) {
                return;
            }

            if (
                button.dataset.action === 'move-up' ||
                button.dataset.action === 'move-down'
            ) {
                const direction = button.dataset.action === 'move-up' ? -1 : 1;
                const updated = await moveMenuItem(id, direction);
                if (updated) {
                    showAlert('Menu order updated.', 'success');
                }
                return;
            }

            if (button.dataset.action === 'edit') {
                const item = state.menuItems.find(
                    (entry) => String(entry?.id || entry?.ID || entry?.Id) === id
                );
                if (item) {
                    startEditMenuItem(item);
                }
                return;
            }

            if (button.dataset.action === 'delete') {
                if (!window.confirm('Delete this menu item?')) {
                    return;
                }
                disableForm(menuForm, true);
                clearAlert();
                try {
                    await apiRequest(`${endpoints.menuItems}/${id}`, {
                        method: 'DELETE',
                    });
                    showAlert('Menu item deleted.', 'success');
                    if (state.editingMenuItemId === id) {
                        resetMenuForm();
                    }
                    await loadMenuItems();
                } catch (error) {
                    handleRequestError(error);
                } finally {
                    disableForm(menuForm, false);
                }
            }
        };

        const approveComment = async (id, button) => {
            if (!endpoints.comments) {
                return;
            }
            try {
                button.disabled = true;
                await apiRequest(`${endpoints.comments}/${id}/approve`, {
                    method: 'PUT',
                });
                showAlert('Comment approved', 'success');
                await loadComments();
            } catch (error) {
                handleRequestError(error);
            } finally {
                button.disabled = false;
            }
        };

        const rejectComment = async (id, button) => {
            if (!endpoints.comments) {
                return;
            }
            try {
                button.disabled = true;
                await apiRequest(`${endpoints.comments}/${id}/reject`, {
                    method: 'PUT',
                });
                showAlert('Comment rejected', 'info');
                await loadComments();
            } catch (error) {
                handleRequestError(error);
            } finally {
                button.disabled = false;
            }
        };

        const deleteComment = async (id, button) => {
            if (!endpoints.comments) {
                return;
            }
            if (!window.confirm('Delete this comment permanently?')) {
                return;
            }
            try {
                button.disabled = true;
                await apiRequest(`${endpoints.comments}/${id}`, {
                    method: 'DELETE',
                });
                showAlert('Comment deleted', 'success');
                await loadComments();
                await loadStats();
            } catch (error) {
                handleRequestError(error);
            } finally {
                button.disabled = false;
            }
        };

        const handleSiteSettingsSubmit = async (event) => {
            event.preventDefault();
            if (!settingsForm || !endpoints.siteSettings) {
                return;
            }

            const getValue = (name) => {
                const field = settingsForm.querySelector(`[name="${name}"]`);
                return field ? field.value.trim() : '';
            };

            const payload = {
                name: getValue('name'),
                description: getValue('description'),
                url: getValue('url'),
                favicon: getValue('favicon'),
                logo: getValue('logo'),
            };

            const retentionField = settingsForm.querySelector('[name="unused_tag_retention_hours"]');
            const retentionRaw = retentionField ? retentionField.value.trim() : '';
            const retentionHours = Number.parseInt(retentionRaw, 10);

            if (Number.isNaN(retentionHours) || retentionHours < 1) {
                showAlert('Please provide how many hours unused tags should be retained (minimum 1 hour).', 'error');
                return;
            }

            payload.unused_tag_retention_hours = retentionHours;

            if (!payload.name) {
                showAlert('Please provide a site name.', 'error');
                return;
            }

            if (!payload.url) {
                showAlert('Please provide the primary site URL.', 'error');
                return;
            }

            disableForm(settingsForm, true);
            clearAlert();

            try {
                const response = await apiRequest(endpoints.siteSettings, {
                    method: 'PUT',
                    body: JSON.stringify(payload),
                });
                state.site = response?.site || payload;
                populateSiteSettingsForm(state.site);
                showAlert('Site settings updated successfully.', 'success');
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(settingsForm, false);
            }
        };

        const handlePostSubmit = async (event) => {
            event.preventDefault();
            if (!postForm) {
                return;
            }
            const id = postForm.dataset.id;
            const title = postForm.title.value.trim();
            if (!title) {
                showAlert('Please provide a title for the post.', 'error');
                return;
            }
            const description = postForm.description.value.trim();
            const featuredImg = postFeaturedImageInput
                ? postFeaturedImageInput.value.trim()
                : '';
            const content = postContentField
                ? postContentField.value.trim()
                : '';
            const publishedField = postForm.querySelector(
                'input[name="published"]'
            );
            const payload = {
                title,
                description,
                featured_img: featuredImg,
                content,
                published: Boolean(publishedField?.checked),
            };
            if (sectionBuilder) {
                const sections = sectionBuilder.getSections();
                const sectionError = validateSections(sections);
                if (sectionError) {
                    showAlert(sectionError, 'error');
                    return;
                }
                payload.sections = sections;
            }
            const categoryValue = postCategorySelect?.value;
            if (categoryValue) {
                payload.category_id = Number(categoryValue);
            }
            if (postTagsInput) {
                payload.tags = parseTags(postTagsInput.value);
            }
            if (postSectionBuilder) {
                const sectionError = postSectionBuilder.validate?.();
                if (sectionError) {
                    showAlert(sectionError, 'error');
                    return;
                }
                payload.sections = postSectionBuilder.serialize?.() || [];
            }
            disableForm(postForm, true);
            clearAlert();
            try {
                if (id) {
                    await apiRequest(`${endpoints.posts}/${id}`, {
                        method: 'PUT',
                        body: JSON.stringify(payload),
                    });
                    showAlert('Post updated successfully.', 'success');
                } else {
                    await apiRequest(endpoints.posts, {
                        method: 'POST',
                        body: JSON.stringify(payload),
                    });
                    showAlert('Post created successfully.', 'success');
                }
                await loadPosts();
                await loadTags();
                await loadStats();
                resetPostForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(postForm, false);
            }
        };

        const handlePostDelete = async () => {
            if (!postForm || !postForm.dataset.id) {
                return;
            }
            if (!window.confirm('Delete this post permanently?')) {
                return;
            }
            const id = postForm.dataset.id;
            disableForm(postForm, true);
            clearAlert();
            try {
                await apiRequest(`${endpoints.posts}/${id}`, {
                    method: 'DELETE',
                });
                showAlert('Post deleted successfully.', 'success');
                await loadPosts();
                await loadTags();
                await loadStats();
                resetPostForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(postForm, false);
            }
        };

        const handlePageSubmit = async (event) => {
            event.preventDefault();
            if (!pageForm) {
                return;
            }
            const id = pageForm.dataset.id;
            const title = pageForm.title.value.trim();
            if (!title) {
                showAlert('Please provide a title for the page.', 'error');
                return;
            }
            const description = pageForm.description.value.trim();
            const content = pageContentField
                ? pageContentField.value.trim()
                : '';
            const orderInput = pageForm.querySelector('input[name="order"]');
            const orderValue = orderInput ? Number(orderInput.value) : 0;
            const publishedField = pageForm.querySelector(
                'input[name="published"]'
            );
            const payload = {
                title,
                description,
                content,
                order: Number.isNaN(orderValue) ? 0 : orderValue,
                published: Boolean(publishedField?.checked),
            };
            if (!id && pageSlugInput) {
                const slugValue = pageSlugInput.value.trim();
                if (slugValue) {
                    payload.slug = slugValue;
                }
            }
            if (pageSectionBuilder) {
                const sectionError = pageSectionBuilder.validate?.();
                if (sectionError) {
                    showAlert(sectionError, 'error');
                    return;
                }
                payload.sections = pageSectionBuilder.serialize?.() || [];
            }
            disableForm(pageForm, true);
            clearAlert();
            try {
                if (id) {
                    await apiRequest(`${endpoints.pages}/${id}`, {
                        method: 'PUT',
                        body: JSON.stringify(payload),
                    });
                    showAlert('Page updated successfully.', 'success');
                } else {
                    await apiRequest(endpoints.pages, {
                        method: 'POST',
                        body: JSON.stringify(payload),
                    });
                    showAlert('Page created successfully.', 'success');
                }
                await loadPages();
                await loadStats();
                resetPageForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(pageForm, false);
            }
        };

        const handlePageDelete = async () => {
            if (!pageForm || !pageForm.dataset.id) {
                return;
            }
            if (!window.confirm('Delete this page permanently?')) {
                return;
            }
            const id = pageForm.dataset.id;
            disableForm(pageForm, true);
            clearAlert();
            try {
                await apiRequest(`${endpoints.pages}/${id}`, {
                    method: 'DELETE',
                });
                showAlert('Page deleted successfully.', 'success');
                await loadPages();
                await loadStats();
                resetPageForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(pageForm, false);
            }
        };

        const handleCategorySubmit = async (event) => {
            event.preventDefault();
            if (!categoryForm) {
                return;
            }
            const id = categoryForm.dataset.id;
            const name = categoryForm.name.value.trim();
            if (!name) {
                showAlert('Please provide a category name.', 'error');
                return;
            }
            const description = categoryForm.description.value.trim();
            const payload = { name, description };
            disableForm(categoryForm, true);
            clearAlert();
            try {
                if (id) {
                    await apiRequest(`${endpoints.categories}/${id}`, {
                        method: 'PUT',
                        body: JSON.stringify(payload),
                    });
                    showAlert('Category updated successfully.', 'success');
                } else {
                    await apiRequest(endpoints.categories, {
                        method: 'POST',
                        body: JSON.stringify(payload),
                    });
                    showAlert('Category created successfully.', 'success');
                }
                await loadCategories();
                await loadPosts();
                await loadStats();
                resetCategoryForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(categoryForm, false);
            }
        };

        const handleCategoryDelete = async () => {
            if (!categoryForm || !categoryForm.dataset.id) {
                return;
            }
            if (!window.confirm('Delete this category permanently?')) {
                return;
            }
            const id = categoryForm.dataset.id;
            disableForm(categoryForm, true);
            clearAlert();
            try {
                await apiRequest(`${endpoints.categories}/${id}`, {
                    method: 'DELETE',
                });
                showAlert('Category deleted successfully.', 'success');
                await loadCategories();
                await loadPosts();
                await loadStats();
                resetCategoryForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(categoryForm, false);
            }
        };

        const activateTab = (targetId) => {
            root.querySelectorAll('.admin__tab').forEach((tab) => {
                const isActive = tab.dataset.tab === targetId;
                tab.classList.toggle('is-active', isActive);
                tab.setAttribute('aria-selected', String(isActive));
            });
            root.querySelectorAll('.admin-panel').forEach((panel) => {
                const isActive = panel.dataset.panel === targetId;
                panel.toggleAttribute('hidden', !isActive);
                panel.classList.toggle('is-active', isActive);
            });
        };

        root.querySelectorAll('.admin__tab').forEach((tab) => {
            tab.addEventListener('click', () => activateTab(tab.dataset.tab));
        });

        root.querySelector('[data-action="post-reset"]')?.addEventListener(
            'click',
            resetPostForm
        );
        root.querySelector('[data-action="page-reset"]')?.addEventListener(
            'click',
            resetPageForm
        );
        root.querySelector('[data-action="category-reset"]')?.addEventListener(
            'click',
            resetCategoryForm
        );

        const attachSearchHandler = (input, callback) => {
            if (!input || typeof callback !== 'function') {
                return;
            }
            const update = () => callback(input.value);
            input.addEventListener('input', update);
            input.addEventListener('search', update);
        };

        attachSearchHandler(postSearchInput, setPostSearchQuery);
        attachSearchHandler(pageSearchInput, setPageSearchQuery);
        attachSearchHandler(categorySearchInput, setCategorySearchQuery);

        if (postSearchInput?.value) {
            setPostSearchQuery(postSearchInput.value);
        }
        if (pageSearchInput?.value) {
            setPageSearchQuery(pageSearchInput.value);
        }
        if (categorySearchInput?.value) {
            setCategorySearchQuery(categorySearchInput.value);
        }

        postForm?.addEventListener('submit', handlePostSubmit);
        postDeleteButton?.addEventListener('click', handlePostDelete);
        pageForm?.addEventListener('submit', handlePageSubmit);
        pageDeleteButton?.addEventListener('click', handlePageDelete);
        categoryForm?.addEventListener('submit', handleCategorySubmit);
        categoryDeleteButton?.addEventListener('click', handleCategoryDelete);
        settingsForm?.addEventListener('submit', handleSiteSettingsSubmit);
        faviconUploadButton?.addEventListener('click', handleFaviconUploadClick);
        faviconUploadInput?.addEventListener('change', handleFaviconFileChange);
        logoUploadButton?.addEventListener('click', handleLogoUploadClick);
        logoUploadInput?.addEventListener('change', handleLogoFileChange);
        socialForm?.addEventListener('submit', handleSocialFormSubmit);
        socialCancelButton?.addEventListener('click', handleSocialCancelEdit);
        socialList?.addEventListener('click', handleSocialListClick);
        menuForm?.addEventListener('submit', handleMenuFormSubmit);
        menuCancelButton?.addEventListener('click', handleMenuCancelEdit);
        menuLocationField?.addEventListener('change', handleMenuLocationChange);
        menuList?.addEventListener('click', handleMenuListClick);
        postTagsInput?.addEventListener('input', renderTagSuggestions);

        clearAlert();
        renderMetricsChart(state.activityTrend);
        loadStats();
        loadTags();
        loadCategories().then(() => {
            renderCategoryOptions();
            loadPosts();
        });
        loadPages();
        loadComments();
        loadSiteSettings();
        loadSocialLinks();
        loadMenuItems();
    };

    if (document.readyState === 'loading') {
        document.addEventListener(
            'DOMContentLoaded',
            initialiseAdminDashboard,
            {
                once: true,
            }
        );
    } else {
        initialiseAdminDashboard();
    }
})();

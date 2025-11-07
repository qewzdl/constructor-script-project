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
        parseDateInput,
        formatDateTimeInput,
        booleanLabel,
        createElement,
        buildAbsoluteUrl,
        createSvgElement,
        formatNumber,
        formatPeriodLabel,
        normaliseString,
        parseContentDispositionFilename,
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

    const coerceDateValue = (value) => {
        if (value instanceof Date) {
            const time = value.getTime();
            return Number.isNaN(time) ? null : new Date(time);
        }
        if (typeof value === 'number') {
            const date = new Date(value);
            return Number.isNaN(date.getTime()) ? null : date;
        }
        if (typeof value === 'string') {
            const trimmed = value.trim();
            if (!trimmed) {
                return null;
            }
            const date = new Date(trimmed);
            return Number.isNaN(date.getTime()) ? null : date;
        }
        if (value && typeof value === 'object') {
            if (value.Time) {
                return coerceDateValue(value.Time);
            }
            if (value.time) {
                return coerceDateValue(value.time);
            }
        }
        return null;
    };

    const extractDateValue = (entry, ...keys) => {
        if (!entry) {
            return null;
        }
        for (const key of keys) {
            if (Object.prototype.hasOwnProperty.call(entry, key)) {
                const date = coerceDateValue(entry[key]);
                if (date) {
                    return date;
                }
            }
        }
        return null;
    };

    const formatPublicationStatus = (entry) => {
        const published = Boolean(entry?.published ?? entry?.Published);
        const publishAtDate = extractDateValue(entry, 'publish_at', 'publishAt', 'PublishAt');
        const publishedAtDate = extractDateValue(entry, 'published_at', 'publishedAt', 'PublishedAt');
        const now = Date.now();

        if (!published) {
            if (publishAtDate) {
                return publishAtDate.getTime() > now
                    ? `Draft (scheduled ${formatDate(publishAtDate)})`
                    : `Draft (planned ${formatDate(publishAtDate)})`;
            }
            return 'Draft';
        }

        if (publishAtDate && publishAtDate.getTime() > now) {
            return `Scheduled for ${formatDate(publishAtDate)}`;
        }

        if (publishedAtDate) {
            return `Published ${formatDate(publishedAtDate)}`;
        }

        if (publishAtDate) {
            return `Published ${formatDate(publishAtDate)}`;
        }

        return 'Published';
    };

    const formatPercentage = (value, fractionDigits = 1) => {
        const numeric = Number(value);
        if (!Number.isFinite(numeric)) {
            return '0%';
        }
        const digits = Math.max(0, Math.min(4, Number(fractionDigits) || 0));
        try {
            return `${numeric.toLocaleString(undefined, {
                minimumFractionDigits: digits,
                maximumFractionDigits: digits,
            })}%`;
        } catch (error) {
            return `${numeric.toFixed(digits)}%`;
        }
    };

    const describePublication = (entry) => {
        const publishAtDate = extractDateValue(entry, 'publish_at', 'publishAt', 'PublishAt');
        const publishedAtDate = extractDateValue(entry, 'published_at', 'publishedAt', 'PublishedAt');
        const now = Date.now();

        if (publishedAtDate) {
            return `Published on ${formatDate(publishedAtDate)}.`;
        }

        if (publishAtDate) {
            return publishAtDate.getTime() > now
                ? `Scheduled for ${formatDate(publishAtDate)}.`
                : `Planned publish date ${formatDate(publishAtDate)}.`;
        }

        return '';
    };

    const parseOrder = (value, fallback = 0) => {
        const parsed = Number(value);
        return Number.isFinite(parsed) ? parsed : fallback;
    };

    const initialiseAdminDashboard = () => {
        const root = document.querySelector('[data-page="admin"]');
        if (!root) {
            return;
        }

        const ACTIVE_TAB_STORAGE_KEY = 'constructor.admin.activeTab';
        const getStoredActiveTab = () => {
            try {
                const storage = window.localStorage;
                return storage ? storage.getItem(ACTIVE_TAB_STORAGE_KEY) || '' : '';
            } catch (error) {
                return '';
            }
        };
        const setStoredActiveTab = (tabId) => {
            try {
                const storage = window.localStorage;
                if (!storage) {
                    return;
                }
                if (tabId) {
                    storage.setItem(ACTIVE_TAB_STORAGE_KEY, tabId);
                } else {
                    storage.removeItem(ACTIVE_TAB_STORAGE_KEY);
                }
            } catch (error) {
                /* Ignore storage errors (private browsing, storage disabled, etc.) */
            }
        };

        const app = window.App || {};
        const auth = app.auth;
        const stateChangingMethods = new Set([
            'POST',
            'PUT',
            'PATCH',
            'DELETE',
        ]);
        const readCookie = (name) => {
            if (!name || typeof document?.cookie !== 'string') {
                return '';
            }
            const cookies = document.cookie.split('; ');
            for (let index = 0; index < cookies.length; index += 1) {
                const cookie = cookies[index];
                if (!cookie) {
                    continue;
                }
                const [key, ...rest] = cookie.split('=');
                if (key === name) {
                    return decodeURIComponent(rest.join('='));
                }
            }
            return '';
        };
        const getCSRFCookie = () => readCookie('csrf_token');
        const buildAuthenticatedRequestInit = (options = {}) => {
            const init = { ...options };
            const headers = Object.assign({}, options.headers || {});
            const method = (options.method || 'GET').toUpperCase();

            init.method = method;

            if (options.body && !(options.body instanceof FormData)) {
                headers['Content-Type'] =
                    headers['Content-Type'] || 'application/json';
            }

            const token =
                auth && typeof auth.getToken === 'function'
                    ? auth.getToken()
                    : undefined;
            if (token && !headers.Authorization) {
                headers.Authorization = `Bearer ${token}`;
            }

            if (stateChangingMethods.has(method)) {
                const csrfToken = getCSRFCookie();
                if (csrfToken && !headers['X-CSRF-Token']) {
                    headers['X-CSRF-Token'] = csrfToken;
                }
            }

            init.headers = headers;
            init.credentials = 'include';
            return init;
        };

        const authenticatedFetch = (url, options = {}) =>
            fetch(url, buildAuthenticatedRequestInit(options));

        const fallbackApiRequest = async (url, options = {}) => {
            const response = await authenticatedFetch(url, options);

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
            homepage: root.dataset.endpointHomepage,
            faviconUpload: root.dataset.endpointFaviconUpload,
            logoUpload: root.dataset.endpointLogoUpload,
            upload: root.dataset.endpointUpload,
            uploadRename: root.dataset.endpointUploadRename,
            uploads: root.dataset.endpointUploads,
            themes: root.dataset.endpointThemes,
            plugins: root.dataset.endpointPlugins,
            socialLinks: root.dataset.endpointSocialLinks,
            fonts: root.dataset.endpointFonts,
            menuItems: root.dataset.endpointMenuItems,
            users: root.dataset.endpointUsers,
            advertising: root.dataset.endpointAdvertisingSettings,
            backupExport: root.dataset.endpointBackupExport,
            backupImport: root.dataset.endpointBackupImport,
            backupSettings: root.dataset.endpointBackupSettings,
            coursesVideos: root.dataset.endpointCoursesVideos,
            coursesTopics: root.dataset.endpointCoursesTopics,
            coursesPackages: root.dataset.endpointCoursesPackages,
        };

        const currentUserIdValue = Number.parseInt(
            root.dataset.currentUserId || '',
            10
        );
        const currentUserId = Number.isFinite(currentUserIdValue)
            ? String(currentUserIdValue)
            : '';

        const alertElement = document.getElementById('admin-alert');
        const ALERT_AUTO_HIDE_MS = 5000;
        const ALERT_TRANSITION_FALLBACK_MS = 360;
        let alertAutoHideTimeoutId = null;
        let alertDismissFallbackTimeoutId = null;
        let pendingHideHandler = null;

        const cancelPendingHide = () => {
            if (!alertElement) {
                return;
            }
            if (pendingHideHandler) {
                alertElement.removeEventListener('transitionend', pendingHideHandler);
                pendingHideHandler = null;
            }
            if (alertDismissFallbackTimeoutId) {
                window.clearTimeout(alertDismissFallbackTimeoutId);
                alertDismissFallbackTimeoutId = null;
            }
        };

        const updateAlertContent = (message, type = 'info') => {
            if (!alertElement) {
                return;
            }
            if (typeof setAlert === 'function') {
                setAlert(alertElement, message, type);
                return;
            }

            alertElement.classList.remove('is-error', 'is-success', 'is-info');

            if (!message) {
                alertElement.hidden = true;
                alertElement.textContent = '';
                return;
            }

            const statusClass =
                type === 'error' ? 'is-error' : type === 'success' ? 'is-success' : 'is-info';
            alertElement.classList.add(statusClass);
            alertElement.hidden = false;
            alertElement.textContent = message;
        };

        const clearAlertContent = () => updateAlertContent('');

        const hideAlert = () => {
            if (!alertElement) {
                return;
            }

            window.clearTimeout(alertAutoHideTimeoutId);
            alertAutoHideTimeoutId = null;
            cancelPendingHide();
            alertElement.classList.remove('is-visible');

            if (alertElement.hidden) {
                clearAlertContent();
                return;
            }

            pendingHideHandler = (event) => {
                if (event.target !== alertElement) {
                    return;
                }
                cancelPendingHide();
                clearAlertContent();
            };

            alertElement.addEventListener('transitionend', pendingHideHandler);
            alertDismissFallbackTimeoutId = window.setTimeout(() => {
                cancelPendingHide();
                clearAlertContent();
            }, ALERT_TRANSITION_FALLBACK_MS);
        };

        const showAlert = (message, type = 'info') => {
            if (!alertElement) {
                return;
            }

            window.clearTimeout(alertAutoHideTimeoutId);
            alertAutoHideTimeoutId = null;
            cancelPendingHide();

            if (!message) {
                hideAlert();
                return;
            }

            updateAlertContent(message, type);
            // Force reflow so transitions apply consistently.
            void alertElement.offsetWidth;
            alertElement.classList.add('is-visible');
            alertAutoHideTimeoutId = window.setTimeout(() => {
                hideAlert();
            }, ALERT_AUTO_HIDE_MS);
        };

        const clearAlert = () => hideAlert();

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

        const updateBackupSummary = (message) => {
            if (!backupSummary) {
                return;
            }
            if (!message) {
                backupSummary.textContent = '';
                backupSummary.hidden = true;
                return;
            }
            backupSummary.hidden = false;
            backupSummary.textContent = message;
        };

        const metricElements = new Map();
        root.querySelectorAll('.admin__metric').forEach((card) => {
            const key = card.dataset.metric;
            const valueElement = card.querySelector('.admin__metric-value');
            if (key && valueElement) {
                metricElements.set(key, valueElement);
            }
        });

        const chartContainer = root.querySelector('[data-role="metrics-chart"]');
        const chartSeries = [
            { key: 'posts', label: 'Posts', color: 'var(--admin-chart-posts)' },
            {
                key: 'comments',
                label: 'Comments',
                color: 'var(--admin-chart-comments)',
            },
            {
                key: 'views',
                label: 'Views',
                color: 'var(--admin-chart-views)',
            },
            {
                key: 'users',
                label: 'Users',
                color: 'var(--admin-chart-users)',
            },
        ];

        const postAnalyticsContainer = root.querySelector(
            '[data-role="post-analytics"]'
        );
        const postAnalyticsSummary = root.querySelector(
            '[data-role="post-analytics-summary"]'
        );
        const postAnalyticsLoading = root.querySelector(
            '[data-role="post-analytics-loading"]'
        );
        const postAnalyticsEmpty = root.querySelector(
            '[data-role="post-analytics-empty"]'
        );
        const postAnalyticsComparisons = root.querySelector(
            '[data-role="post-analytics-comparisons"]'
        );
        const postAnalyticsComparisonsEmpty = root.querySelector(
            '[data-role="post-analytics-comparisons-empty"]'
        );
        const postAnalyticsChartContainer = root.querySelector(
            '[data-role="post-analytics-chart"]'
        );
        const postAnalyticsSummaryItems = new Map();
        if (postAnalyticsSummary) {
            postAnalyticsSummary.querySelectorAll('[data-metric]').forEach((item) => {
                const key = item.dataset.metric;
                if (!key || postAnalyticsSummaryItems.has(key)) {
                    return;
                }
                postAnalyticsSummaryItems.set(key, {
                    element: item,
                    value: item.querySelector('[data-role="summary-value"]'),
                    subvalue: item.querySelector('[data-role="summary-subvalue"]'),
                    delta: item.querySelector('[data-role="summary-delta"]'),
                });
            });
        }
        const postAnalyticsSeries = [
            { key: 'views', label: 'Views', color: 'var(--admin-chart-views)' },
            {
                key: 'comments',
                label: 'Comments',
                color: 'var(--admin-chart-comments)',
            },
        ];

        const navigationContainer = root.querySelector('[data-role="admin-nav"]');
        const contentScrollContainer = root.querySelector('.admin__content');
        const tables = {
            posts: root.querySelector('#admin-posts-table'),
            pages: root.querySelector('#admin-pages-table'),
            categories: root.querySelector('#admin-categories-table'),
            users: root.querySelector('#admin-users-table'),
            courseVideos: root.querySelector('#admin-course-videos-table'),
            courseTopics: root.querySelector('#admin-course-topics-table'),
            coursePackages: root.querySelector('#admin-course-packages-table'),
        };
        const postSearchInput = root.querySelector('[data-role="post-search"]');
        const pageSearchInput = root.querySelector('[data-role="page-search"]');
        const categorySearchInput = root.querySelector('[data-role="category-search"]');
        const userSearchInput = root.querySelector('[data-role="user-search"]');
        const commentsList = root.querySelector('#admin-comments-list');
        const postForm = root.querySelector('#admin-post-form');
        const pageForm = root.querySelector('#admin-page-form');
        const categoryForm = root.querySelector('#admin-category-form');
        const userForm = root.querySelector('#admin-user-form');
        const settingsForm = root.querySelector('#admin-settings-form');
        const languageForm = root.querySelector('#admin-language-form');
        const homepageForm = root.querySelector('#admin-homepage-form');
        const homepageSelect = homepageForm?.querySelector('[data-role="homepage-select"]');
        const homepageStatus = homepageForm?.querySelector('[data-role="homepage-status"]');
        const homepageSubmitButton = homepageForm?.querySelector('[data-role="homepage-submit"]');
        const homepageOptionsContainer = root.querySelector('[data-role="homepage-options"]');
        const homepageEmptyState = root.querySelector('[data-role="homepage-empty"]');
        const socialList = root.querySelector('[data-role="social-list"]');
        const socialEmpty = root.querySelector('[data-role="social-empty"]');
        const socialForm = document.getElementById('admin-social-form');
        const fontList = root.querySelector('[data-role="font-list"]');
        const fontEmpty = root.querySelector('[data-role="font-empty"]');
        const fontForm = document.getElementById('admin-font-form');
        const fontSubmitButton = fontForm?.querySelector('[data-role="font-submit"]');
        const fontCancelButton = fontForm?.querySelector('[data-role="font-cancel"]');
        const menuList = root.querySelector('[data-role="menu-list"]');
        const menuEmpty = root.querySelector('[data-role="menu-empty"]');
        const menuForm = document.getElementById('admin-menu-form');
        const backupPanel = root.querySelector('#admin-panel-backups');
        const backupSummary = backupPanel?.querySelector('[data-role="backup-summary"]');
        const backupDownloadButton = backupPanel?.querySelector('[data-role="backup-download"]');
        const backupImportForm = document.getElementById('admin-backup-import-form');
        const backupUploadInput = backupImportForm?.querySelector('input[name="backup_file"]');
        const backupSettingsForm = document.getElementById('admin-backup-settings-form');
        const backupSettingsToggle = backupSettingsForm?.querySelector('input[name="auto_enabled"]');
        const backupSettingsIntervalInput = backupSettingsForm?.querySelector('input[name="interval_hours"]');
        const backupSettingsStatus = backupSettingsForm?.querySelector('[data-role="backup-settings-status"]');
        const backupSettingsSubmit = backupSettingsForm?.querySelector('[data-role="backup-settings-submit"]');
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
        const defaultLanguageInput = languageForm?.querySelector('input[name="default_language"]');
        const supportedLanguagesInput = languageForm?.querySelector('[data-role="language-hidden"]');
        const supportedLanguagesList = languageForm?.querySelector('[data-role="language-list"]');
        const supportedLanguagesEmpty = languageForm?.querySelector('[data-role="language-empty"]');
        const supportedLanguagesAddInput = languageForm?.querySelector('[data-role="language-input"]');
        const supportedLanguagesAddButton = languageForm?.querySelector('[data-role="language-add"]');
        const languageSuggestionsList = languageForm?.querySelector('[data-role="language-suggestions"]');
        const languageManagerContainer = languageForm?.querySelector('[data-role="language-manager"]');
        const advertisingForm = root.querySelector('#admin-ads-form');
        const advertisingProviderSelect = advertisingForm?.querySelector('[data-role="ads-provider"]');
        const advertisingEnabledToggle = advertisingForm?.querySelector('[data-role="ads-enabled"]');
        const advertisingSlotsContainer = advertisingForm?.querySelector('[data-role="ads-slots"]');
        const advertisingSlotAddButton = advertisingForm?.querySelector('[data-role="ads-slot-add"]');
        const advertisingPublisherInput = advertisingForm?.querySelector('[data-role="ads-google-publisher"]');
        const advertisingAutoToggle = advertisingForm?.querySelector('[data-role="ads-google-auto"]');
        const advertisingProviderFieldsets = advertisingForm
            ? Array.from(advertisingForm.querySelectorAll('[data-role="ads-provider-fields"]'))
            : [];
        const themeList = root.querySelector('[data-role="theme-list"]');
        const themeEmptyState = root.querySelector('[data-role="theme-empty"]');
        const pluginList = root.querySelector('[data-role="plugin-list"]');
        const pluginEmptyState = root.querySelector('[data-role="plugin-empty"]');
        const pluginInstallForm = root.querySelector('[data-role="plugin-install-form"]');
        const pluginUploadInput = root.querySelector('[data-role="plugin-upload-input"]');
        const pluginInstallButton = root.querySelector('[data-role="plugin-install-button"]');
        const socialSubmitButton = socialForm?.querySelector('[data-role="social-submit"]');
        const socialCancelButton = socialForm?.querySelector('[data-role="social-cancel"]');
        const menuSubmitButton = menuForm?.querySelector('[data-role="menu-submit"]');
        const menuCancelButton = menuForm?.querySelector('[data-role="menu-cancel"]');
        const menuLocationField = menuForm?.querySelector('[data-role="menu-location"]');
        const menuCustomLocationContainer = menuForm?.querySelector(
            '[data-role="menu-custom-location"]'
        );
        const menuCustomLocationInput = menuForm?.querySelector(
            '[data-role="menu-location-name"]'
        );
        const menuCustomLocationHint = menuForm?.querySelector(
            '[data-role="menu-custom-location-hint"]'
        );

        const courseVideoForm = root.querySelector('#admin-course-video-form');
        const courseVideoTitleInput = courseVideoForm?.querySelector('input[name="title"]');
        const courseVideoDescriptionInput = courseVideoForm?.querySelector('textarea[name="description"]');
        const courseVideoDurationField = courseVideoForm?.querySelector('[data-role="course-video-duration"]');
        const courseVideoUploadGroup = courseVideoForm?.querySelector('[data-role="course-video-upload-group"]');
        const courseVideoUploadHint = courseVideoForm?.querySelector('[data-role="course-video-upload-hint"]');
        const courseVideoFileInput = courseVideoForm?.querySelector('input[name="video"]');
        const courseVideoSubmitButton = courseVideoForm?.querySelector('[data-role="course-video-submit"]');
        const courseVideoDeleteButton = courseVideoForm?.querySelector('[data-role="course-video-delete"]');

        const courseTopicForm = root.querySelector('#admin-course-topic-form');
        const courseTopicTitleInput = courseTopicForm?.querySelector('input[name="title"]');
        const courseTopicDescriptionInput = courseTopicForm?.querySelector('textarea[name="description"]');
        const courseTopicVideoSelect = courseTopicForm?.querySelector('[data-role="course-topic-video-select"]');
        const courseTopicVideoAddButton = courseTopicForm?.querySelector('[data-role="course-topic-video-add"]');
        const courseTopicVideoList = courseTopicForm?.querySelector('[data-role="course-topic-video-list"]');
        const courseTopicVideoEmpty = courseTopicForm?.querySelector('[data-role="course-topic-video-empty"]');
        const courseTopicSubmitButton = courseTopicForm?.querySelector('[data-role="course-topic-submit"]');
        const courseTopicDeleteButton = courseTopicForm?.querySelector('[data-role="course-topic-delete"]');

        const coursePackageForm = root.querySelector('#admin-course-package-form');
        const coursePackageTitleInput = coursePackageForm?.querySelector('input[name="title"]');
        const coursePackageDescriptionInput = coursePackageForm?.querySelector('textarea[name="description"]');
        const coursePackagePriceInput = coursePackageForm?.querySelector('input[name="price"]');
        const coursePackageImageInput = coursePackageForm?.querySelector('input[name="image_url"]');
        const coursePackageTopicSelect = coursePackageForm?.querySelector('[data-role="course-package-topic-select"]');
        const coursePackageTopicAddButton = coursePackageForm?.querySelector('[data-role="course-package-topic-add"]');
        const coursePackageTopicList = coursePackageForm?.querySelector('[data-role="course-package-topic-list"]');
        const coursePackageTopicEmpty = coursePackageForm?.querySelector('[data-role="course-package-topic-empty"]');
        const coursePackageSubmitButton = coursePackageForm?.querySelector('[data-role="course-package-submit"]');
        const coursePackageDeleteButton = coursePackageForm?.querySelector('[data-role="course-package-delete"]');

        const CUSTOM_FOOTER_OPTION = '__custom_footer__';
        const COMMON_LANGUAGE_CODES = [
            'en',
            'en-GB',
            'es',
            'es-419',
            'de',
            'fr',
            'it',
            'pt-BR',
            'pt-PT',
            'ru',
            'uk',
            'pl',
            'tr',
            'ar',
            'zh-CN',
            'zh-TW',
            'ja',
            'ko',
        ];
        const defaultMenuLocationValues = [
            'header',
            'footer:explore',
            'footer:account',
            'footer:legal',
            'footer',
        ];
        const postDeleteButton = postForm?.querySelector(
            '[data-role="post-delete"]'
        );
        const postPublishButton = postForm?.querySelector(
            '[data-role="post-submit-publish"]'
        );
        const postDraftButton = postForm?.querySelector(
            '[data-role="post-submit-draft"]'
        );
        const pageDeleteButton = pageForm?.querySelector(
            '[data-role="page-delete"]'
        );
        const pagePublishButton = pageForm?.querySelector(
            '[data-role="page-submit-publish"]'
        );
        const pageDraftButton = pageForm?.querySelector(
            '[data-role="page-submit-draft"]'
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
        const postPublishAtInput = postForm?.querySelector(
            'input[name="publish_at"]'
        );
        const postPublishedAtNote = postForm?.querySelector(
            '[data-role="post-published-at"]'
        );
        const tagList = document.getElementById('admin-tags-list');
        const postTagsList = document.getElementById('admin-post-tags-list');
        const userUsernameField = userForm?.querySelector('input[name="username"]');
        const userEmailField = userForm?.querySelector('input[name="email"]');
        const userRoleField = userForm?.querySelector('[data-role="user-role"]');
        const userStatusField = userForm?.querySelector('[data-role="user-status"]');
        const userSubmitButton = userForm?.querySelector('[data-role="user-submit"]');
        const userDeleteButton = userForm?.querySelector('[data-role="user-delete"]');
        const userHint = userForm?.querySelector('[data-role="user-hint"]');
        const DEFAULT_CATEGORY_SLUG = 'uncategorized';
        const pagePathInput = pageForm?.querySelector('input[name="path"]');
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
        const pagePublishAtInput = pageForm?.querySelector(
            'input[name="publish_at"]'
        );
        const pagePublishedAtNote = pageForm?.querySelector(
            '[data-role="page-published-at"]'
        );

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
            users: [],
            tags: [],
            themes: [],
            plugins: [],
            socialLinks: [],
            fonts: [],
            menuItems: [],
            activeMenuLocation: 'header',
            menuLocations: new Set(defaultMenuLocationValues),
            isReorderingMenu: false,
            isReorderingFonts: false,
            editingSocialLinkId: '',
            editingFontId: '',
            editingMenuItemId: '',
            defaultCategoryId: '',
            site: null,
            advertising: {
                settings: null,
                providers: [],
            },
            postSearchQuery: '',
            pageSearchQuery: '',
            categorySearchQuery: '',
            userSearchQuery: '',
            hasLoadedPosts: false,
            hasLoadedPages: false,
            hasLoadedCategories: false,
            hasLoadedUsers: false,
            postAnalytics: new Map(),
            postAnalyticsLoadingIds: new Set(),
            selectedPostId: '',
            homepage: {
                options: [],
                selected: null,
                selectedId: '',
                hasLoaded: false,
            },
            language: {
                default: '',
                supported: [],
            },
            courses: {
                videos: [],
                topics: [],
                packages: [],
                hasLoadedVideos: false,
                hasLoadedTopics: false,
                hasLoadedPackages: false,
                selectedVideoId: '',
                selectedTopicId: '',
                selectedPackageId: '',
                topicVideoIds: [],
                packageTopicIds: [],
            },
        };

        const ensureHomepageState = () => {
            if (!state.homepage) {
                state.homepage = {
                    options: [],
                    selected: null,
                    selectedId: '',
                    hasLoaded: false,
                };
            }
            return state.homepage;
        };

        const createMediaLibrary = () => {
            if (!window.AdminMediaLibrary || !endpoints.uploads) {
                return null;
            }
            try {
                const fetchUploads = async () => {
                    if (!endpoints.uploads) {
                        throw new Error('Uploads endpoint is not configured.');
                    }
                    try {
                        const response = await apiRequest(endpoints.uploads);
                        const uploads = Array.isArray(response?.uploads)
                            ? response.uploads
                            : [];
                        return uploads;
                    } catch (error) {
                        handleRequestError(error);
                        throw error;
                    }
                };

                const uploadFile = endpoints.upload
                    ? async (file, options = {}) => {
                          if (!file) {
                              throw new Error('Select a file to upload.');
                          }
                          if (!endpoints.upload) {
                              throw new Error('Upload endpoint is not configured.');
                          }
                          const formData = new FormData();
                          formData.append('image', file);
                          const preferredName =
                              options && typeof options.name === 'string'
                                  ? options.name.trim()
                                  : '';
                          if (preferredName) {
                              formData.append('name', preferredName);
                          }
                          try {
                              const result = await apiRequest(endpoints.upload, {
                                  method: 'POST',
                                  body: formData,
                              });
                              return result;
                          } catch (error) {
                              handleRequestError(error);
                              throw error;
                          }
                      }
                    : null;

                const renameUpload = endpoints.uploadRename
                    ? async (upload, newName) => {
                          if (!upload) {
                              throw new Error('Select an image to rename.');
                          }
                          const desiredName =
                              typeof newName === 'string' ? newName.trim() : '';
                          if (!desiredName) {
                              throw new Error('Image name cannot be empty.');
                          }
                          if (!endpoints.uploadRename) {
                              throw new Error('Rename endpoint is not configured.');
                          }

                          const uploadUrl =
                              (upload && typeof upload.url === 'string' && upload.url) ||
                              (upload && typeof upload.URL === 'string' && upload.URL) ||
                              '';
                          const uploadFilename =
                              (upload &&
                                  typeof upload.filename === 'string' &&
                                  upload.filename) ||
                              (upload &&
                                  typeof upload.Filename === 'string' &&
                                  upload.Filename) ||
                              '';

                          let currentValue = uploadUrl || uploadFilename;
                          if (currentValue.includes('/uploads/')) {
                              const index = currentValue.lastIndexOf('/uploads/');
                              currentValue = currentValue.slice(index);
                          }
                          currentValue = (currentValue || '').trim();
                          if (!currentValue) {
                              throw new Error('Unable to determine the current file name.');
                          }

                          try {
                              const result = await apiRequest(endpoints.uploadRename, {
                                  method: 'PUT',
                                  headers: {
                                      'Content-Type': 'application/json',
                                  },
                                  body: JSON.stringify({
                                      current: currentValue,
                                      name: desiredName,
                                  }),
                              });

                              if (result && typeof result === 'object') {
                                  if (result.upload && typeof result.upload === 'object') {
                                      return result.upload;
                                  }
                                  return {
                                      url:
                                          result.url ||
                                          result.URL ||
                                          (result.upload && result.upload.url) ||
                                          uploadUrl ||
                                          currentValue,
                                      filename:
                                          result.filename ||
                                          result.Filename ||
                                          (result.upload && result.upload.filename) ||
                                          uploadFilename,
                                  };
                              }

                              return null;
                          } catch (error) {
                              handleRequestError(error);
                              throw error;
                          }
                      }
                    : null;

                return window.AdminMediaLibrary.create({
                    fetchUploads,
                    uploadFile,
                    renameUpload,
                    onClose: () => {
                        if (document.activeElement) {
                            document.activeElement.blur?.();
                        }
                    },
                });
            } catch (error) {
                console.error('Failed to initialise media library', error);
                return null;
            }
        };

        const mediaLibrary = createMediaLibrary();

        const openMediaLibraryForInput = (input) => {
            if (!(input instanceof HTMLElement)) {
                return;
            }

            if (!mediaLibrary) {
                showAlert('Media library is not available in this environment.', 'error');
                return;
            }

            const currentValue = typeof input.value === 'string' ? input.value.trim() : '';
            mediaLibrary
                .open({
                    currentUrl: currentValue,
                    onSelect: (url) => {
                        if (!url || typeof url !== 'string') {
                            return;
                        }
                        input.value = url;
                        input.dispatchEvent(new Event('input', { bubbles: true }));
                        input.dispatchEvent(new Event('change', { bubbles: true }));
                    },
                })
                .catch((error) => {
                    if (error) {
                        const message =
                            typeof error.message === 'string'
                                ? error.message
                                : 'Failed to open media library.';
                        showAlert(message, 'error');
                    }
                });
        };

        root.addEventListener('click', (event) => {
            const target = event.target instanceof HTMLElement
                ? event.target
                : null;
            if (!target) {
                return;
            }
            const trigger = target.closest('[data-action="open-media-library"]');
            if (!trigger) {
                return;
            }
            event.preventDefault();

            const selector = trigger.dataset.mediaTarget || '';
            let input = null;

            if (selector) {
                try {
                    input = document.querySelector(selector);
                } catch (error) {
                    input = null;
                }
            }

            if (!(input instanceof HTMLElement)) {
                const field = trigger.closest(
                    'label, .admin-form__label, .admin-builder__field, .section-field'
                );
                if (field) {
                    input = field.querySelector('input[type="url"], input, textarea');
                }
            }

            if (!(input instanceof HTMLElement)) {
                showAlert('Unable to locate the image field to update.', 'error');
                return;
            }

            openMediaLibraryForInput(input);
        });

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
            const path = normaliseString(page.path ?? page.Path ?? '').trim();
            if (path) {
                return path.startsWith('/') ? path : `/${path}`;
            }
            const slug = normaliseString(page.slug ?? page.Slug ?? '').trim();
            if (slug) {
                return `/${encodeURIComponent(slug)}`;
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
                const rawTitle =
                    section.title === undefined || section.title === null
                        ? ''
                        : section.title;
                const sectionTitle = String(rawTitle).trim();
                const displayTitle = sectionTitle || `Section ${index + 1}`;
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
                        return `Paragraph ${
                            elementIndex + 1
                        } in section "${displayTitle}" is empty.`;
                    }
                    if (element.type === 'image' && !element.content?.url) {
                        return `Image ${
                            elementIndex + 1
                        } in section "${displayTitle}" is missing a URL.`;
                    }
                    if (element.type === 'image_group') {
                        const images = Array.isArray(element.content?.images)
                            ? element.content.images
                            : [];
                        if (!images.length) {
                            return `The image group in section "${displayTitle}" needs at least one image.`;
                        }
                        const missing = images.findIndex((img) => !img?.url);
                        if (missing !== -1) {
                            return `Image ${
                                missing + 1
                            } in the group for section "${displayTitle}" is missing a URL.`;
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
                            return `List ${
                                elementIndex + 1
                            } in section "${displayTitle}" needs at least one item.`;
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

        const languageCodePattern = /^[a-z]{2,8}(?:-[A-Za-z]{2,3})?$/;

        const normaliseLanguageCode = (value) => {
            if (typeof value !== 'string') {
                return '';
            }
            const trimmed = value.trim();
            if (!trimmed) {
                return '';
            }
            const parts = trimmed.split('-');
            const base = parts[0].toLowerCase();
            if (parts.length === 1) {
                return base;
            }
            const region = parts[1]?.toUpperCase();
            if (!region) {
                return base;
            }
            return `${base}-${region}`;
        };

        const isValidLanguageCode = (value) => languageCodePattern.test(value);

        const parseLanguageCodes = (value) => {
            if (typeof value !== 'string' || !value.trim()) {
                return [];
            }

            const invalid = [];
            const unique = new Map();

            value.split(',').forEach((entry) => {
                const normalized = normaliseLanguageCode(entry);
                if (!normalized) {
                    return;
                }
                if (!isValidLanguageCode(normalized)) {
                    invalid.push(entry.trim());
                    return;
                }
                if (!unique.has(normalized)) {
                    unique.set(normalized, normalized);
                }
            });

            if (invalid.length > 0) {
                const error = new Error(`Invalid language codes: ${invalid.join(', ')}`);
                error.codes = invalid;
                throw error;
            }

            return Array.from(unique.values());
        };

        const getNormalisedLanguageState = () => {
            const seen = new Set();
            const result = [];
            const rawDefault = normaliseLanguageCode(state.language?.default || '');
            let defaultCode = '';

            if (rawDefault && isValidLanguageCode(rawDefault)) {
                defaultCode = rawDefault;
                seen.add(rawDefault);
                result.push(rawDefault);
            }

            const supported = Array.isArray(state.language?.supported)
                ? state.language.supported
                : [];
            supported.forEach((value) => {
                const normalized = normaliseLanguageCode(value);
                if (!normalized || !isValidLanguageCode(normalized) || seen.has(normalized)) {
                    return;
                }
                seen.add(normalized);
                result.push(normalized);
            });

            return {
                defaultCode,
                codes: result,
            };
        };

        const applyLanguageState = (defaultCode, codes) => {
            state.language.default = defaultCode;
            state.language.supported = Array.isArray(codes) ? [...codes] : [];
        };

        const updateLanguageSuggestions = (codes) => {
            if (!languageSuggestionsList) {
                return;
            }

            const unique = new Map();
            languageSuggestionsList.innerHTML = '';

            [...COMMON_LANGUAGE_CODES, ...(codes || [])].forEach((value) => {
                const normalized = normaliseLanguageCode(value);
                if (!normalized || !isValidLanguageCode(normalized) || unique.has(normalized)) {
                    return;
                }
                unique.set(normalized, true);
                const option = document.createElement('option');
                option.value = normalized;
                languageSuggestionsList.appendChild(option);
            });
        };

        const renderLanguageManager = () => {
            if (!languageForm) {
                return;
            }

            let previousTop = null;
            let useWindowScroll = false;
            const scrollContainer = (() => {
                if (!languageManagerContainer) {
                    return null;
                }
                if (contentScrollContainer) {
                    return contentScrollContainer;
                }
                const scrollingElement = document.scrollingElement
                    || document.documentElement
                    || document.body;
                useWindowScroll = true;
                return scrollingElement;
            })();

            if (languageManagerContainer && scrollContainer) {
                previousTop = languageManagerContainer.getBoundingClientRect().top;
            }

            const { defaultCode, codes } = getNormalisedLanguageState();
            applyLanguageState(defaultCode, codes);

            const additional = codes.filter((code) => code !== defaultCode);

            if (supportedLanguagesInput) {
                supportedLanguagesInput.value = additional.join(', ');
            }

            updateLanguageSuggestions(codes);

            if (defaultLanguageInput) {
                const activeElement = document.activeElement;
                const currentValue = defaultLanguageInput.value || '';
                const currentNormalised = normaliseLanguageCode(currentValue);
                const shouldUpdate =
                    activeElement !== defaultLanguageInput ||
                    currentNormalised !== (defaultCode || '');

                if (shouldUpdate) {
                    defaultLanguageInput.value = defaultCode || '';
                }
            }

            if (supportedLanguagesEmpty) {
                supportedLanguagesEmpty.hidden = additional.length > 0;
            }

            if (supportedLanguagesList) {
                supportedLanguagesList.innerHTML = '';

                codes.forEach((code) => {
                    const item = document.createElement('li');
                    item.className = 'admin-languages__item';
                    if (code === defaultCode) {
                        item.dataset.state = 'default';
                    }

                    const codeLabel = document.createElement('span');
                    codeLabel.className = 'admin-languages__code';
                    codeLabel.textContent = code;
                    item.appendChild(codeLabel);

                    if (code === defaultCode) {
                        const badge = document.createElement('span');
                        badge.className = 'admin-languages__badge';
                        badge.textContent = 'Default';
                        item.appendChild(badge);
                    } else {
                        const actions = document.createElement('span');
                        actions.className = 'admin-languages__actions';

                        const defaultButton = document.createElement('button');
                        defaultButton.type = 'button';
                        defaultButton.className = 'admin-languages__action';
                        defaultButton.dataset.action = 'language-default';
                        defaultButton.dataset.code = code;
                        defaultButton.textContent = 'Make default';
                        actions.appendChild(defaultButton);

                        const removeButton = document.createElement('button');
                        removeButton.type = 'button';
                        removeButton.className =
                            'admin-languages__action admin-languages__action--remove';
                        removeButton.dataset.action = 'language-remove';
                        removeButton.dataset.code = code;
                        removeButton.textContent = 'Remove';
                        actions.appendChild(removeButton);

                        item.appendChild(actions);
                    }

                    supportedLanguagesList.appendChild(item);
                });
            }

            if (
                languageManagerContainer &&
                scrollContainer &&
                previousTop !== null
            ) {
                const nextTop = languageManagerContainer.getBoundingClientRect().top;
                const delta = nextTop - previousTop;
                if (Math.abs(delta) > 1) {
                    if (useWindowScroll) {
                        const currentOffset = window.scrollY || window.pageYOffset || 0;
                        window.scrollTo({ top: currentOffset + delta });
                    } else {
                        scrollContainer.scrollTop += delta;
                    }
                }
            }
        };

        function setDefaultLanguage(code, options = {}) {
            const { silent = false } = options;
            const normalized = normaliseLanguageCode(code);
            if (!normalized || !isValidLanguageCode(normalized)) {
                if (!silent) {
                    showAlert(
                        'Please provide a valid language code (e.g. "en" or "en-GB").',
                        'error'
                    );
                }
                return false;
            }

            const { codes } = getNormalisedLanguageState();
            const filtered = codes.filter((value) => value !== normalized);
            applyLanguageState(normalized, [normalized, ...filtered]);
            renderLanguageManager();
            return true;
        }

        function addSupportedLanguage(code) {
            const normalized = normaliseLanguageCode(code);
            if (!normalized) {
                return false;
            }
            if (!isValidLanguageCode(normalized)) {
                showAlert(
                    'Please use a valid language code (e.g. "en" or "en-GB").',
                    'error'
                );
                supportedLanguagesAddInput?.focus();
                supportedLanguagesAddInput?.select?.();
                return false;
            }

            const { defaultCode, codes } = getNormalisedLanguageState();
            if (codes.includes(normalized)) {
                showAlert(`Language "${normalized}" is already configured.`, 'info');
                supportedLanguagesAddInput?.focus();
                supportedLanguagesAddInput?.select?.();
                return false;
            }

            const nextDefault = defaultCode || normalized;
            const nextCodes = nextDefault === normalized
                ? [normalized, ...codes.filter((value) => value !== normalized)]
                : [...codes, normalized];

            applyLanguageState(nextDefault, nextCodes);
            renderLanguageManager();
            return true;
        }

        function removeSupportedLanguage(code) {
            const normalized = normaliseLanguageCode(code);
            if (!normalized) {
                return false;
            }

            const { defaultCode, codes } = getNormalisedLanguageState();
            if (normalized === defaultCode) {
                showAlert('The default language must always be supported.', 'error');
                return false;
            }

            const filtered = codes.filter((value) => value !== normalized);
            applyLanguageState(defaultCode, filtered);
            renderLanguageManager();
            return true;
        }

        function handleLanguageAdd(event) {
            event.preventDefault();
            if (!supportedLanguagesAddInput) {
                return;
            }

            const value = supportedLanguagesAddInput.value.trim();
            if (!value) {
                supportedLanguagesAddInput.focus();
                return;
            }

            const added = addSupportedLanguage(value);
            if (added) {
                supportedLanguagesAddInput.value = '';
                supportedLanguagesAddInput.focus();
            }
        }

        function handleLanguageInputKeydown(event) {
            if (event.key === 'Enter') {
                event.preventDefault();
                handleLanguageAdd(event);
            }
        }

        function handleDefaultLanguageBlur() {
            if (!defaultLanguageInput) {
                return;
            }

            const value = defaultLanguageInput.value.trim();
            if (!value) {
                return;
            }

            const normalized = normaliseLanguageCode(value);
            if (!normalized || !isValidLanguageCode(normalized)) {
                defaultLanguageInput.setCustomValidity(
                    'Please use a valid language code (e.g. "en" or "en-GB").'
                );
                defaultLanguageInput.reportValidity();
                return;
            }

            defaultLanguageInput.setCustomValidity('');
            if (normalized !== value) {
                defaultLanguageInput.value = normalized;
            }
            setDefaultLanguage(normalized, { silent: true });
        }

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
            page?.path,
            page?.Path,
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

        const calculateNiceScale = (maxValue, segments = 4) => {
            const safeSegments = Math.max(1, Number.parseInt(segments, 10) || 1);
            const safeMaxValue = Number.isFinite(maxValue)
                ? Math.max(0, maxValue)
                : 0;
            if (safeMaxValue <= 0) {
                return { max: 0, ticks: [] };
            }

            const safeMax = Math.ceil(safeMaxValue);
            const integerStep = Math.max(1, Math.ceil(safeMax / safeSegments));
            const ticks = Array.from({ length: safeSegments + 1 }, (_, index) =>
                index * integerStep
            );

            let lastTick = ticks[ticks.length - 1];
            while (lastTick < safeMax) {
                lastTick += integerStep;
                ticks.push(lastTick);
            }

            return { max: ticks[ticks.length - 1], ticks };
        };

        const formatAxisTick = (value) => {
            if (!Number.isFinite(value)) {
                return '0';
            }
            const rounded = Math.round(value);
            return formatNumber(rounded);
        };

        const createChartRenderer = (container, series) => {
            const svg = container?.querySelector('svg');
            const legend = container?.querySelector('[data-role="chart-legend"]');
            const summary = container?.querySelector('[data-role="chart-summary"]');
            const empty = container?.querySelector('[data-role="chart-empty"]');

            if (
                !container ||
                !svg ||
                !legend ||
                !summary ||
                !empty ||
                !Array.isArray(series) ||
                !series.length
            ) {
                return () => {};
            }

            return (trend = []) => {
                const normalised = Array.isArray(trend)
                    ? trend
                          .map((entry) => {
                              const period =
                                  entry?.period ||
                                  entry?.Period ||
                                  entry?.date ||
                                  entry?.Date ||
                                  '';
                              const result = { period };
                              series.forEach((definition) => {
                                  const altKey =
                                      typeof definition.key === 'string'
                                          ? definition.key
                                                .charAt(0)
                                                .toUpperCase() + definition.key.slice(1)
                                          : '';
                                  const rawValue = Number(
                                      entry?.[definition.key] ??
                                          (altKey ? entry?.[altKey] : undefined) ??
                                          0
                                  );
                                  result[definition.key] = Number.isFinite(rawValue)
                                      ? Math.max(0, rawValue)
                                      : 0;
                              });
                              return result;
                          })
                          .filter((entry) => entry.period)
                    : [];

                const values = normalised.flatMap((point) =>
                    series.map((definition) => {
                        const numeric = Number(point[definition.key]);
                        return Number.isFinite(numeric) ? Math.max(0, numeric) : 0;
                    })
                );
                const rawMaxValue = values.length ? Math.max(...values, 0) : 0;

                legend.innerHTML = '';
                summary.innerHTML = '';

                if (!normalised.length || rawMaxValue <= 0) {
                    svg.innerHTML = '';
                    empty.hidden = false;
                    legend.hidden = true;
                    summary.hidden = true;
                    container.dataset.state = 'empty';
                    return;
                }

                empty.hidden = true;
                legend.hidden = false;
                summary.hidden = false;
                container.dataset.state = 'ready';

                const segmentCount = Math.max(1, Math.min(4, normalised.length));
                const { max: scaledMax, ticks: yTicks } = calculateNiceScale(
                    rawMaxValue,
                    segmentCount
                );
                const maxValue = scaledMax || rawMaxValue;

                const width = 660;
                const height = 320;
                const leftPadding = 56;
                const rightPadding = 24;
                const topPadding = 24;
                const bottomPadding = 64;
                const chartWidth = width - leftPadding - rightPadding;
                const chartHeight = height - topPadding - bottomPadding;
                const stepX =
                    normalised.length > 1 ? chartWidth / (normalised.length - 1) : 0;

                svg.setAttribute('viewBox', `0 0 ${width} ${height}`);
                svg.innerHTML = '';

                const gridGroup = createSvgElement('g', {
                    class: 'admin-chart__grid',
                });
                const seriesGroup = createSvgElement('g', {
                    class: 'admin-chart__series',
                });
                const pointsGroup = createSvgElement('g', {
                    class: 'admin-chart__points',
                });
                const axisGroup = createSvgElement('g', {
                    class: 'admin-chart__axis',
                });

                svg.appendChild(gridGroup);
                svg.appendChild(seriesGroup);
                svg.appendChild(pointsGroup);
                svg.appendChild(axisGroup);

                const yTickValues = yTicks.length ? yTicks : [0, maxValue];
                yTickValues.forEach((tickValue) => {
                    const ratio = maxValue > 0 ? tickValue / maxValue : 0;
                    const y = topPadding + chartHeight - ratio * chartHeight;
                    const line = createSvgElement('line', {
                        x1: leftPadding.toFixed(2),
                        x2: (width - rightPadding).toFixed(2),
                        y1: y.toFixed(2),
                        y2: y.toFixed(2),
                        class:
                            tickValue === 0
                                ? 'admin-chart__grid-line admin-chart__grid-line--baseline'
                                : 'admin-chart__grid-line',
                    });
                    gridGroup.appendChild(line);

                    const label = createSvgElement('text', {
                        x: (leftPadding - 12).toFixed(2),
                        y: y.toFixed(2),
                        class: 'admin-chart__axis-label admin-chart__axis-label--y',
                    });
                    label.textContent = formatAxisTick(tickValue);
                    axisGroup.appendChild(label);
                });

                const xLabelInterval = Math.max(1, Math.round(normalised.length / 6));
                normalised.forEach((point, index) => {
                    const shouldShowLabel =
                        index === 0 ||
                        index === normalised.length - 1 ||
                        index % xLabelInterval === 0;
                    if (!shouldShowLabel) {
                        return;
                    }
                    const x =
                        normalised.length > 1
                            ? leftPadding + index * stepX
                            : leftPadding + chartWidth / 2;
                    const label = createSvgElement('text', {
                        x: x.toFixed(2),
                        y: (topPadding + chartHeight + 24).toFixed(2),
                        class: 'admin-chart__axis-label admin-chart__axis-label--x',
                    });
                    label.textContent =
                        formatPeriodLabel(point.period) || String(point.period);
                    axisGroup.appendChild(label);
                });

                series.forEach((definition) => {
                    const pathData = normalised
                        .map((point, index) => {
                            const value = Number(point[definition.key]);
                            const safeValue = Number.isFinite(value)
                                ? Math.max(0, value)
                                : 0;
                            const x =
                                normalised.length > 1
                                    ? leftPadding + index * stepX
                                    : leftPadding + chartWidth / 2;
                            const y =
                                topPadding +
                                chartHeight -
                                (maxValue > 0
                                    ? (safeValue / maxValue) * chartHeight
                                    : 0);
                            return `${index === 0 ? 'M' : 'L'}${x.toFixed(2)} ${y.toFixed(
                                2
                            )}`;
                        })
                        .join(' ');

                    const path = createSvgElement('path', {
                        d: pathData,
                        class: 'admin-chart__line',
                        stroke: definition.color,
                    });
                    path.dataset.series = definition.key;
                    seriesGroup.appendChild(path);

                    normalised.forEach((point, index) => {
                        const value = Number(point[definition.key]);
                        const safeValue = Number.isFinite(value)
                            ? Math.max(0, value)
                            : 0;
                        const x =
                            normalised.length > 1
                                ? leftPadding + index * stepX
                                : leftPadding + chartWidth / 2;
                        const y =
                            topPadding +
                            chartHeight -
                            (maxValue > 0
                                ? (safeValue / maxValue) * chartHeight
                                : 0);
                        const circle = createSvgElement('circle', {
                            cx: x.toFixed(2),
                            cy: y.toFixed(2),
                            r: 4,
                            class: 'admin-chart__point',
                            stroke: definition.color,
                        });
                        circle.dataset.series = definition.key;
                        const tooltip = createSvgElement('title');
                        tooltip.textContent = `${definition.label}: ${formatNumber(
                            safeValue
                        )} • ${formatPeriodLabel(point.period) || point.period}`;
                        circle.appendChild(tooltip);
                        pointsGroup.appendChild(circle);
                    });
                });

                series.forEach((definition) => {
                    const legendItem = document.createElement('li');
                    legendItem.className = 'admin-chart__legend-item';
                    legendItem.dataset.series = definition.key;
                    const swatch = document.createElement('span');
                    swatch.className = 'admin-chart__legend-swatch';
                    const label = document.createElement('span');
                    label.className = 'admin-chart__legend-label';
                    label.textContent = definition.label;
                    legendItem.appendChild(swatch);
                    legendItem.appendChild(label);
                    legend.appendChild(legendItem);
                });

                normalised.forEach((point) => {
                    const item = document.createElement('li');
                    item.className = 'admin-chart__summary-item';

                    const period = document.createElement('span');
                    period.className = 'admin-chart__summary-period';
                    period.textContent = formatPeriodLabel(point.period) || '—';
                    item.appendChild(period);

                    series.forEach((definition) => {
                        const value = Number(point[definition.key]);
                        const safeValue = Number.isFinite(value)
                            ? Math.max(0, value)
                            : 0;
                        const valueElement = document.createElement('span');
                        valueElement.className = 'admin-chart__summary-value';
                        valueElement.dataset.series = definition.key;
                        valueElement.textContent = `${formatNumber(
                            safeValue
                        )} ${definition.label.toLowerCase()}`;
                        item.appendChild(valueElement);
                    });

                    summary.appendChild(item);
                });
            };
        };

        const renderMetricsChart = createChartRenderer(
            chartContainer,
            chartSeries
        );
        const renderPostAnalyticsChart = createChartRenderer(
            postAnalyticsChartContainer,
            postAnalyticsSeries
        );

        const safeNumber = (value) => {
            const numeric = Number(value);
            return Number.isFinite(numeric) ? numeric : 0;
        };

        const optionalNumber = (value) => {
            const numeric = Number(value);
            return Number.isFinite(numeric) ? numeric : Number.NaN;
        };

        const resetPostAnalyticsSummary = () => {
            postAnalyticsSummaryItems.forEach((item) => {
                if (item?.value) {
                    item.value.textContent = '—';
                }
                if (item?.subvalue) {
                    item.subvalue.textContent = '—';
                }
                if (item?.delta) {
                    item.delta.textContent = '';
                    item.delta.hidden = true;
                    item.delta.classList.remove('is-positive', 'is-negative');
                }
            });
        };

        const showPostAnalyticsLoading = () => {
            resetPostAnalyticsSummary();
            if (postAnalyticsContainer) {
                postAnalyticsContainer.hidden = true;
            }
            if (postAnalyticsLoading) {
                postAnalyticsLoading.hidden = false;
            }
            if (postAnalyticsEmpty) {
                postAnalyticsEmpty.hidden = true;
            }
            if (postAnalyticsComparisons) {
                postAnalyticsComparisons.innerHTML = '';
                postAnalyticsComparisons.hidden = true;
            }
            if (postAnalyticsComparisonsEmpty) {
                postAnalyticsComparisonsEmpty.hidden = true;
            }
        };

        const showPostAnalyticsEmpty = (message = '') => {
            resetPostAnalyticsSummary();
            if (postAnalyticsContainer) {
                postAnalyticsContainer.hidden = true;
            }
            if (postAnalyticsLoading) {
                postAnalyticsLoading.hidden = true;
            }
            if (postAnalyticsEmpty) {
                postAnalyticsEmpty.textContent =
                    message || 'Select a published post to view analytics.';
                postAnalyticsEmpty.hidden = false;
            }
            if (postAnalyticsComparisons) {
                postAnalyticsComparisons.innerHTML = '';
                postAnalyticsComparisons.hidden = true;
            }
            if (postAnalyticsComparisonsEmpty) {
                postAnalyticsComparisonsEmpty.hidden = true;
            }
        };

        const updateSummaryDelta = (element, change) => {
            if (!element) {
                return;
            }
            element.classList.remove('is-positive', 'is-negative');
            if (!Number.isFinite(change)) {
                element.hidden = true;
                element.textContent = '';
                return;
            }
            const absolute = Math.abs(change);
            if (absolute < 0.1) {
                element.hidden = false;
                element.textContent = 'No change vs previous 7 days';
                return;
            }
            const formatted = formatPercentage(absolute, 1);
            if (change > 0) {
                element.hidden = false;
                element.textContent = `+${formatted} vs previous 7 days`;
                element.classList.add('is-positive');
                return;
            }
            element.hidden = false;
            element.textContent = `-${formatted} vs previous 7 days`;
            element.classList.add('is-negative');
        };

        const renderPostAnalyticsSummary = (metrics = {}) => {
            const totalViews = safeNumber(
                metrics?.total_views ?? metrics?.TotalViews
            );
            const viewsLast = safeNumber(
                metrics?.views_last_7_days ?? metrics?.ViewsLast7Days
            );
            const viewsChange = optionalNumber(
                metrics?.views_change_percent ?? metrics?.ViewsChangePercent
            );

            const viewsItem = postAnalyticsSummaryItems.get('views');
            if (viewsItem?.value) {
                viewsItem.value.textContent = formatNumber(totalViews);
            }
            if (viewsItem?.subvalue) {
                viewsItem.subvalue.textContent = `${formatNumber(
                    viewsLast
                )} in last 7 days`;
            }
            updateSummaryDelta(viewsItem?.delta, viewsChange);

            const totalComments = safeNumber(
                metrics?.total_comments ?? metrics?.TotalComments
            );
            const commentsLast = safeNumber(
                metrics?.comments_last_7_days ?? metrics?.CommentsLast7Days
            );
            const commentsChange = optionalNumber(
                metrics?.comments_change_percent ?? metrics?.CommentsChangePercent
            );

            const commentsItem = postAnalyticsSummaryItems.get('comments');
            if (commentsItem?.value) {
                commentsItem.value.textContent = formatNumber(totalComments);
            }
            if (commentsItem?.subvalue) {
                commentsItem.subvalue.textContent = `${formatNumber(
                    commentsLast
                )} in last 7 days`;
            }
            updateSummaryDelta(commentsItem?.delta, commentsChange);

            const engagement = optionalNumber(
                metrics?.engagement_rate ?? metrics?.EngagementRate
            );
            const engagementItem = postAnalyticsSummaryItems.get('engagement');
            if (engagementItem?.value) {
                engagementItem.value.textContent = formatPercentage(
                    Number.isFinite(engagement) ? engagement : 0,
                    1
                );
            }
            if (engagementItem?.subvalue) {
                engagementItem.subvalue.textContent = `${formatNumber(
                    totalComments
                )} comments · ${formatNumber(totalViews)} views`;
            }
            if (engagementItem?.delta) {
                engagementItem.delta.hidden = true;
                engagementItem.delta.textContent = '';
                engagementItem.delta.classList.remove('is-positive', 'is-negative');
            }
        };

        const formatAverageComparison = (diff, percent, average, noun) => {
            if (!Number.isFinite(diff) || !Number.isFinite(average)) {
                return '';
            }
            const roundedAverage = Math.round(average);
            if (Math.abs(diff) < 0.5) {
                return `Matches the site average of ${formatNumber(
                    roundedAverage
                )} ${noun}.`;
            }
            const direction = diff > 0 ? 'above' : 'below';
            const difference = `${diff > 0 ? '+' : '−'}${formatNumber(
                Math.round(Math.abs(diff))
            )}`;
            let summary = `${difference} ${direction} the site average of ${formatNumber(
                roundedAverage
            )} ${noun}`;
            if (Number.isFinite(percent) && Math.abs(percent) >= 0.5) {
                summary += ` (${formatPercentage(Math.abs(percent), 1)})`;
            }
            return `${summary}.`;
        };

        const formatRankComparison = (rank, total, noun) => {
            if (!Number.isFinite(rank) || !Number.isFinite(total) || total <= 0) {
                return '';
            }
            const position = Math.min(
                Math.max(1, Math.round(rank)),
                Math.round(total)
            );
            const percentile = 100 - ((position - 1) / total) * 100;
            let tier = '';
            if (percentile >= 90) {
                tier = 'Top 10%';
            } else if (percentile >= 75) {
                tier = 'Top 25%';
            } else if (percentile >= 50) {
                tier = 'Top 50%';
            } else {
                tier = 'Lower half';
            }
            return `${tier} • #${formatNumber(position)} of ${formatNumber(
                Math.round(total)
            )} ${noun}`;
        };

        const renderPostAnalyticsComparisons = (comparisons = {}) => {
            if (postAnalyticsComparisons) {
                postAnalyticsComparisons.innerHTML = '';
            }

            const items = [];

            const viewsDifference = optionalNumber(
                comparisons?.views_vs_average_difference ??
                    comparisons?.ViewsVsAverageDifference
            );
            const viewsPercent = optionalNumber(
                comparisons?.views_vs_average_percent ??
                    comparisons?.ViewsVsAveragePercent
            );
            const averageViews = optionalNumber(
                comparisons?.average_views ?? comparisons?.AverageViews
            );
            const viewsAverageText = formatAverageComparison(
                viewsDifference,
                viewsPercent,
                averageViews,
                'views'
            );
            if (viewsAverageText) {
                items.push({
                    label: 'Views vs site average',
                    value: viewsAverageText,
                });
            }

            const commentsDifference = optionalNumber(
                comparisons?.comments_vs_average_difference ??
                    comparisons?.CommentsVsAverageDifference
            );
            const commentsPercent = optionalNumber(
                comparisons?.comments_vs_average_percent ??
                    comparisons?.CommentsVsAveragePercent
            );
            const averageComments = optionalNumber(
                comparisons?.average_comments ?? comparisons?.AverageComments
            );
            const commentsAverageText = formatAverageComparison(
                commentsDifference,
                commentsPercent,
                averageComments,
                'comments'
            );
            if (commentsAverageText) {
                items.push({
                    label: 'Comments vs site average',
                    value: commentsAverageText,
                });
            }

            const viewsRankText = formatRankComparison(
                comparisons?.views_rank_position ?? comparisons?.ViewsRankPosition,
                comparisons?.views_rank_total ?? comparisons?.ViewsRankTotal,
                'posts'
            );
            if (viewsRankText) {
                items.push({ label: 'Views rank', value: viewsRankText });
            }

            const commentsRankText = formatRankComparison(
                comparisons?.comments_rank_position ??
                    comparisons?.CommentsRankPosition,
                comparisons?.comments_rank_total ?? comparisons?.CommentsRankTotal,
                'posts'
            );
            if (commentsRankText) {
                items.push({ label: 'Comments rank', value: commentsRankText });
            }

            if (!postAnalyticsComparisons) {
                if (postAnalyticsComparisonsEmpty) {
                    postAnalyticsComparisonsEmpty.hidden = items.length > 0;
                }
                return;
            }

            if (!items.length) {
                postAnalyticsComparisons.hidden = true;
                if (postAnalyticsComparisonsEmpty) {
                    postAnalyticsComparisonsEmpty.hidden = false;
                }
                return;
            }

            postAnalyticsComparisons.hidden = false;
            if (postAnalyticsComparisonsEmpty) {
                postAnalyticsComparisonsEmpty.hidden = true;
            }

            items.forEach(({ label, value }) => {
                const term = document.createElement('dt');
                term.textContent = label;
                const definition = document.createElement('dd');
                definition.textContent = value;
                postAnalyticsComparisons.appendChild(term);
                postAnalyticsComparisons.appendChild(definition);
            });
        };

        const renderPostAnalytics = (postId, analytics) => {
            const id = String(postId || '');
            state.postAnalytics.set(id, analytics);

            if (String(state.selectedPostId || '') !== id) {
                return;
            }

            if (postAnalyticsLoading) {
                postAnalyticsLoading.hidden = true;
            }
            if (postAnalyticsEmpty) {
                postAnalyticsEmpty.hidden = true;
            }
            if (postAnalyticsContainer) {
                postAnalyticsContainer.hidden = false;
            }

            renderPostAnalyticsSummary(analytics?.metrics || {});
            const trend = Array.isArray(analytics?.trend)
                ? analytics.trend
                : [];
            renderPostAnalyticsChart(trend);
            renderPostAnalyticsComparisons(analytics?.comparisons || {});
        };

        const buildPostEndpoint = (id, action = '') => {
            if (!endpoints.posts) {
                return '';
            }
            const base = endpoints.posts.endsWith('/')
                ? endpoints.posts.slice(0, -1)
                : endpoints.posts;
            if (!id) {
                return base;
            }
            const encodedId = encodeURIComponent(String(id));
            const suffix = action ? `/${action.replace(/^\/+/, '')}` : '';
            return `${base}/${encodedId}${suffix}`;
        };

        const loadPostAnalytics = async (postId) => {
            const id = String(postId || '');
            if (!id) {
                return;
            }
            const endpoint = buildPostEndpoint(id, 'analytics');
            if (!endpoint) {
                return;
            }
            if (!state.postAnalyticsLoadingIds) {
                state.postAnalyticsLoadingIds = new Set();
            }
            if (state.postAnalyticsLoadingIds.has(id)) {
                return;
            }

            const cached = state.postAnalytics?.get(id);
            if (cached) {
                renderPostAnalytics(id, cached);
                return;
            }

            showPostAnalyticsLoading();
            state.postAnalyticsLoadingIds.add(id);

            try {
                const payload = await apiRequest(endpoint);
                const analytics = payload?.analytics || null;
                if (analytics) {
                    renderPostAnalytics(id, analytics);
                } else {
                    showPostAnalyticsEmpty('Analytics data is not available yet.');
                }
            } catch (error) {
                handleRequestError(error);
                showPostAnalyticsEmpty('Failed to load analytics.');
            } finally {
                state.postAnalyticsLoadingIds.delete(id);
            }
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
                        textContent: formatPublicationStatus(post),
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
                cell.colSpan = 5;
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
                const pathText = normaliseString(page.path ?? page.Path ?? '').trim();
                row.appendChild(
                    createElement('td', {
                        textContent: pathText || '—',
                    })
                );
                row.appendChild(
                    createElement('td', {
                        textContent: page.slug || page.Slug || '—',
                    })
                );
                row.appendChild(
                    createElement('td', {
                        textContent: formatPublicationStatus(page),
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

        const extractUserId = (user) => {
            if (!user) {
                return '';
            }
            if (typeof user.id !== 'undefined' && user.id !== null) {
                return String(user.id);
            }
            if (typeof user.ID !== 'undefined' && user.ID !== null) {
                return String(user.ID);
            }
            return '';
        };

        const getUserSearchFields = (user) => [
            user?.id,
            user?.ID,
            user?.username,
            user?.Username,
            user?.email,
            user?.Email,
            user?.role,
            user?.Role,
            user?.status,
            user?.Status,
        ];

        const formatUserLabel = (value) => {
            const text = normaliseString(value).trim();
            if (!text) {
                return '—';
            }
            return text.charAt(0).toUpperCase() + text.slice(1);
        };

        const ensureSelectOption = (select, value) => {
            if (!select || typeof value !== 'string') {
                return;
            }
            const trimmed = value.trim();
            if (!trimmed) {
                return;
            }
            const exists = Array.from(select.options).some(
                (option) => option.value === trimmed
            );
            if (!exists) {
                const option = createElement('option', {
                    textContent: formatUserLabel(trimmed),
                });
                option.value = trimmed;
                select.appendChild(option);
            }
        };

        const setUserFormEnabled = (enabled) => {
            if (!userForm) {
                return;
            }
            userForm.classList.toggle('is-disabled', !enabled);
            [userRoleField, userStatusField, userSubmitButton].forEach(
                (field) => {
                    if (field) {
                        field.disabled = !enabled;
                    }
                }
            );
        };

        const updateUserHint = (user) => {
            if (!userHint) {
                return;
            }
            if (!user) {
                userHint.textContent =
                    'Select a user from the list to view their account details.';
                return;
            }
            const joined =
                user.created_at ||
                user.createdAt ||
                user.CreatedAt ||
                user.created ||
                user.Created;
            const updated =
                user.updated_at ||
                user.updatedAt ||
                user.UpdatedAt ||
                user.updated ||
                user.Updated;
            const details = [];
            if (joined) {
                details.push(`Joined ${formatDate(joined)}`);
            }
            if (updated && updated !== joined) {
                details.push(`Updated ${formatDate(updated)}`);
            }
            userHint.textContent = details.length
                ? details.join(' · ')
                : 'Account details ready for review.';
        };

        const resetUserForm = () => {
            if (!userForm) {
                return;
            }
            delete userForm.dataset.id;
            if (userUsernameField) {
                userUsernameField.value = '';
            }
            if (userEmailField) {
                userEmailField.value = '';
            }
            if (userRoleField && userRoleField.options.length > 0) {
                userRoleField.value = userRoleField.options[0].value;
            }
            if (userStatusField && userStatusField.options.length > 0) {
                userStatusField.value = userStatusField.options[0].value;
            }
            if (userSubmitButton) {
                userSubmitButton.textContent = 'Update user';
                userSubmitButton.disabled = true;
            }
            if (userDeleteButton) {
                userDeleteButton.hidden = true;
                userDeleteButton.disabled = true;
            }
            updateUserHint(null);
            setUserFormEnabled(false);
            highlightRow(tables.users);
        };

        const selectUser = (id) => {
            if (!userForm) {
                return;
            }
            const targetId = String(id || '').trim();
            if (!targetId) {
                resetUserForm();
                return;
            }
            const user = state.users.find(
                (entry) => extractUserId(entry) === targetId
            );
            if (!user) {
                resetUserForm();
                return;
            }
            userForm.dataset.id = targetId;
            if (userUsernameField) {
                userUsernameField.value =
                    user.username || user.Username || '';
            }
            if (userEmailField) {
                userEmailField.value = user.email || user.Email || '';
            }
            const roleValue = normaliseString(user.role || user.Role || '')
                .trim() || 'user';
            ensureSelectOption(userRoleField, roleValue);
            if (userRoleField) {
                userRoleField.value = roleValue;
            }
            const statusValue = normaliseString(
                user.status || user.Status || ''
            ).trim();
            ensureSelectOption(userStatusField, statusValue);
            if (userStatusField) {
                userStatusField.value = statusValue || userStatusField.value;
            }
            setUserFormEnabled(true);
            if (userSubmitButton) {
                userSubmitButton.disabled = false;
                userSubmitButton.textContent = 'Update user';
            }
            if (userDeleteButton) {
                const isSelf = currentUserId && targetId === currentUserId;
                userDeleteButton.hidden = Boolean(isSelf);
                userDeleteButton.disabled = Boolean(isSelf);
            }
            updateUserHint(user);
            highlightRow(tables.users, targetId);
            bringFormIntoView(userForm);
        };

        const renderUsers = () => {
            const table = tables.users;
            if (!table) {
                return;
            }
            table.innerHTML = '';
            const users = state.users.filter((user) =>
                matchesSearchQuery(
                    getUserSearchFields(user),
                    state.userSearchQuery
                )
            );
            if (!users.length) {
                const row = createElement('tr', {
                    className: 'admin-table__placeholder',
                });
                const cell = createElement('td', {
                    textContent: state.userSearchQuery
                        ? 'No users match your search'
                        : 'No users found',
                });
                cell.colSpan = 5;
                row.appendChild(cell);
                table.appendChild(row);
                return;
            }
            users.forEach((user) => {
                const id = extractUserId(user);
                if (!id) {
                    return;
                }
                const row = createElement('tr');
                row.dataset.id = id;
                row.appendChild(
                    createElement('td', {
                        textContent:
                            user.username || user.Username || '(unknown)',
                    })
                );
                row.appendChild(
                    createElement('td', {
                        textContent: user.email || user.Email || '—',
                    })
                );
                const roleValue = normaliseString(
                    user.role || user.Role || ''
                ).trim();
                row.appendChild(
                    createElement('td', {
                        textContent: formatUserLabel(roleValue),
                    })
                );
                const statusValue = normaliseString(
                    user.status || user.Status || ''
                ).trim();
                row.appendChild(
                    createElement('td', {
                        textContent: formatUserLabel(statusValue),
                    })
                );
                const created =
                    user.created_at ||
                    user.createdAt ||
                    user.CreatedAt;
                row.appendChild(
                    createElement('td', { textContent: formatDate(created) })
                );
                row.addEventListener('click', () => selectUser(id));
                table.appendChild(row);
            });
            highlightRow(table, userForm?.dataset.id);
        };

        const setUserSearchQuery = (value) => {
            const next = normaliseSearchQuery(value);
            if (state.userSearchQuery === next) {
                return;
            }
            state.userSearchQuery = next;
            if (state.hasLoadedUsers) {
                renderUsers();
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

        const normaliseIdentifier = (value) => {
            if (value === undefined || value === null) {
                return '';
            }
            if (typeof value === 'object') {
                if (
                    Object.prototype.hasOwnProperty.call(value, 'id') ||
                    Object.prototype.hasOwnProperty.call(value, 'ID')
                ) {
                    return normaliseIdentifier(value.id ?? value.ID);
                }
                if (
                    Object.prototype.hasOwnProperty.call(value, 'value') ||
                    Object.prototype.hasOwnProperty.call(value, 'Value')
                ) {
                    return normaliseIdentifier(value.value ?? value.Value);
                }
            }
            const result = String(value).trim();
            return result;
        };

        const extractCourseVideoId = (video) =>
            normaliseIdentifier(video?.id ?? video?.ID ?? '');
        const extractCourseTopicId = (topic) =>
            normaliseIdentifier(topic?.id ?? topic?.ID ?? '');
        const extractCoursePackageId = (pkg) =>
            normaliseIdentifier(pkg?.id ?? pkg?.ID ?? '');

        const getCourseVideoTitle = (video) =>
            normaliseString(video?.title ?? video?.Title ?? 'Untitled video');

        const getCourseTopicVideos = (topic) => {
            if (Array.isArray(topic?.videos)) {
                return topic.videos;
            }
            if (Array.isArray(topic?.Videos)) {
                return topic.Videos;
            }
            return [];
        };

        const getCoursePackageTopics = (pkg) => {
            if (Array.isArray(pkg?.topics)) {
                return pkg.topics;
            }
            if (Array.isArray(pkg?.Topics)) {
                return pkg.Topics;
            }
            return [];
        };

        const formatVideoDuration = (value) => {
            const totalSeconds = Number(value);
            if (!Number.isFinite(totalSeconds) || totalSeconds <= 0) {
                return '—';
            }
            const wholeSeconds = Math.max(0, Math.floor(totalSeconds));
            const hours = Math.floor(wholeSeconds / 3600);
            const minutes = Math.floor((wholeSeconds % 3600) / 60);
            const seconds = wholeSeconds % 60;
            if (hours > 0) {
                return `${hours}:${String(minutes).padStart(2, '0')}:${String(
                    seconds
                ).padStart(2, '0')}`;
            }
            return `${minutes}:${String(seconds).padStart(2, '0')}`;
        };

        const formatPriceAmount = (value) => {
            const cents = Number(value);
            if (!Number.isFinite(cents)) {
                return '—';
            }
            const amount = cents / 100;
            try {
                return amount.toLocaleString(undefined, {
                    minimumFractionDigits: 2,
                    maximumFractionDigits: 2,
                });
            } catch (error) {
                return amount.toFixed(2);
            }
        };

        const formatPriceInputValue = (value) => {
            const cents = Number(value);
            if (!Number.isFinite(cents)) {
                return '';
            }
            const amount = cents / 100;
            try {
                return amount.toLocaleString(undefined, {
                    minimumFractionDigits: 2,
                    maximumFractionDigits: 2,
                });
            } catch (error) {
                return amount.toFixed(2);
            }
        };

        const parsePriceInputValue = (value) => {
            if (typeof value !== 'string') {
                return null;
            }
            const trimmed = value.trim();
            if (!trimmed) {
                return null;
            }
            const normalised = trimmed.replace(/,/g, '.');
            const amount = Number(normalised);
            if (!Number.isFinite(amount) || amount < 0) {
                return null;
            }
            return Math.round(amount * 100);
        };

        const slugifyPreferredName = (value) => {
            const normalised = normaliseString(value).toLowerCase();
            if (!normalised) {
                return '';
            }
            return normalised
                .replace(/[^a-z0-9]+/g, '-')
                .replace(/^-+|-+$/g, '')
                .slice(0, 80);
        };

        const findCourseVideo = (id) =>
            state.courses.videos.find(
                (video) => extractCourseVideoId(video) === String(id)
            );
        const findCourseTopic = (id) =>
            state.courses.topics.find(
                (topic) => extractCourseTopicId(topic) === String(id)
            );
        const findCoursePackage = (id) =>
            state.courses.packages.find(
                (pkg) => extractCoursePackageId(pkg) === String(id)
            );

        const renderCourseVideos = () => {
            const table = tables.courseVideos;
            if (!table) {
                return;
            }
            table.innerHTML = '';
            const videos = Array.isArray(state.courses.videos)
                ? state.courses.videos.slice()
                : [];
            if (!videos.length) {
                const row = createElement('tr', {
                    className: 'admin-table__placeholder',
                });
                const cell = createElement('td', {
                    textContent: 'No videos uploaded yet.',
                });
                cell.colSpan = 3;
                row.appendChild(cell);
                table.appendChild(row);
                return;
            }
            videos.sort((a, b) => {
                const aDate = new Date(
                    a?.updated_at || a?.updatedAt || a?.UpdatedAt || 0
                ).getTime();
                const bDate = new Date(
                    b?.updated_at || b?.updatedAt || b?.UpdatedAt || 0
                ).getTime();
                return bDate - aDate;
            });
            videos.forEach((video) => {
                const id = extractCourseVideoId(video);
                if (!id) {
                    return;
                }
                const row = createElement('tr');
                row.dataset.id = id;
                row.appendChild(
                    createElement('td', { textContent: getCourseVideoTitle(video) })
                );
                const duration =
                    video?.duration_seconds ??
                    video?.durationSeconds ??
                    video?.DurationSeconds;
                row.appendChild(
                    createElement('td', { textContent: formatVideoDuration(duration) })
                );
                const updated =
                    video?.updated_at || video?.updatedAt || video?.UpdatedAt;
                row.appendChild(
                    createElement('td', { textContent: formatDate(updated) })
                );
                row.addEventListener('click', () => selectCourseVideo(id));
                table.appendChild(row);
            });
            highlightRow(table, courseVideoForm?.dataset.id);
        };

        const renderCourseTopics = () => {
            const table = tables.courseTopics;
            if (!table) {
                return;
            }
            table.innerHTML = '';
            const topics = Array.isArray(state.courses.topics)
                ? state.courses.topics.slice()
                : [];
            if (!topics.length) {
                const row = createElement('tr', {
                    className: 'admin-table__placeholder',
                });
                const cell = createElement('td', {
                    textContent: 'No topics created yet.',
                });
                cell.colSpan = 3;
                row.appendChild(cell);
                table.appendChild(row);
                return;
            }
            topics.sort((a, b) => {
                const aDate = new Date(
                    a?.updated_at || a?.updatedAt || a?.UpdatedAt || 0
                ).getTime();
                const bDate = new Date(
                    b?.updated_at || b?.updatedAt || b?.UpdatedAt || 0
                ).getTime();
                return bDate - aDate;
            });
            topics.forEach((topic) => {
                const id = extractCourseTopicId(topic);
                if (!id) {
                    return;
                }
                const row = createElement('tr');
                row.dataset.id = id;
                row.appendChild(
                    createElement('td', {
                        textContent:
                            normaliseString(topic?.title ?? topic?.Title ?? '') ||
                            'Untitled topic',
                    })
                );
                const videoCount = getCourseTopicVideos(topic).length;
                row.appendChild(
                    createElement('td', {
                        textContent:
                            videoCount === 1
                                ? '1 video'
                                : `${videoCount} videos`,
                    })
                );
                const updated =
                    topic?.updated_at || topic?.updatedAt || topic?.UpdatedAt;
                row.appendChild(
                    createElement('td', { textContent: formatDate(updated) })
                );
                row.addEventListener('click', () => selectCourseTopic(id));
                table.appendChild(row);
            });
            highlightRow(table, courseTopicForm?.dataset.id);
        };

        const renderCoursePackages = () => {
            const table = tables.coursePackages;
            if (!table) {
                return;
            }
            table.innerHTML = '';
            const packages = Array.isArray(state.courses.packages)
                ? state.courses.packages.slice()
                : [];
            if (!packages.length) {
                const row = createElement('tr', {
                    className: 'admin-table__placeholder',
                });
                const cell = createElement('td', {
                    textContent: 'No packages configured yet.',
                });
                cell.colSpan = 4;
                row.appendChild(cell);
                table.appendChild(row);
                return;
            }
            packages.sort((a, b) => {
                const aDate = new Date(
                    a?.updated_at || a?.updatedAt || a?.UpdatedAt || 0
                ).getTime();
                const bDate = new Date(
                    b?.updated_at || b?.updatedAt || b?.UpdatedAt || 0
                ).getTime();
                return bDate - aDate;
            });
            packages.forEach((pkg) => {
                const id = extractCoursePackageId(pkg);
                if (!id) {
                    return;
                }
                const row = createElement('tr');
                row.dataset.id = id;
                row.appendChild(
                    createElement('td', {
                        textContent:
                            normaliseString(pkg?.title ?? pkg?.Title ?? '') ||
                            'Untitled package',
                    })
                );
                row.appendChild(
                    createElement('td', {
                        textContent: formatPriceAmount(
                            pkg?.price_cents ?? pkg?.priceCents ?? pkg?.PriceCents
                        ),
                    })
                );
                const topics = getCoursePackageTopics(pkg);
                row.appendChild(
                    createElement('td', {
                        textContent:
                            topics.length === 1
                                ? '1 topic'
                                : `${topics.length} topics`,
                    })
                );
                const updated =
                    pkg?.updated_at || pkg?.updatedAt || pkg?.UpdatedAt;
                row.appendChild(
                    createElement('td', { textContent: formatDate(updated) })
                );
                row.addEventListener('click', () => selectCoursePackage(id));
                table.appendChild(row);
            });
            highlightRow(table, coursePackageForm?.dataset.id);
        };

        const renderCourseTopicVideoOptions = () => {
            if (!courseTopicVideoSelect) {
                return;
            }
            const currentValue = courseTopicVideoSelect.value;
            courseTopicVideoSelect.innerHTML = '';
            courseTopicVideoSelect.appendChild(
                createElement('option', {
                    value: '',
                    textContent: 'Select a video…',
                })
            );
            const selectedIds = new Set(
                state.courses.topicVideoIds.map((id) => String(id))
            );
            state.courses.videos.forEach((video) => {
                const id = extractCourseVideoId(video);
                if (!id || selectedIds.has(String(id))) {
                    return;
                }
                const option = createElement('option', {
                    value: id,
                    textContent: getCourseVideoTitle(video),
                });
                courseTopicVideoSelect.appendChild(option);
            });
            let found = false;
            Array.from(courseTopicVideoSelect.options).forEach((option) => {
                if (option.value === currentValue) {
                    found = true;
                }
            });
            courseTopicVideoSelect.value = found ? currentValue : '';
        };

        const renderCourseTopicVideoList = () => {
            if (!courseTopicVideoList || !courseTopicVideoEmpty) {
                return;
            }
            courseTopicVideoList.innerHTML = '';
            const ids = state.courses.topicVideoIds.filter((id) =>
                Boolean(findCourseVideo(id))
            );
            state.courses.topicVideoIds = ids.slice();
            if (!ids.length) {
                courseTopicVideoEmpty.hidden = false;
                courseTopicVideoList.appendChild(courseTopicVideoEmpty);
                return;
            }
            courseTopicVideoEmpty.hidden = true;
            ids.forEach((id, index) => {
                const video = findCourseVideo(id);
                if (!video) {
                    return;
                }
                const item = createElement('li', {
                    className: 'admin-courses__selection-item',
                });
                item.dataset.id = String(id);
                const info = createElement('div', {
                    className: 'admin-courses__selection-info',
                });
                info.appendChild(
                    createElement('span', {
                        className: 'admin-courses__selection-label',
                        textContent: getCourseVideoTitle(video),
                    })
                );
                const duration =
                    video?.duration_seconds ??
                    video?.durationSeconds ??
                    video?.DurationSeconds;
                info.appendChild(
                    createElement('span', {
                        className: 'admin-courses__selection-meta',
                        textContent: `Duration ${formatVideoDuration(duration)}`,
                    })
                );
                const actions = createElement('div', {
                    className: 'admin-courses__selection-actions',
                });
                const upButton = createElement('button', {
                    className: 'admin-navigation__reorder-button',
                    textContent: 'Move up',
                });
                upButton.type = 'button';
                upButton.dataset.action = 'move-up';
                upButton.dataset.id = String(id);
                upButton.disabled = index === 0;
                const downButton = createElement('button', {
                    className: 'admin-navigation__reorder-button',
                    textContent: 'Move down',
                });
                downButton.type = 'button';
                downButton.dataset.action = 'move-down';
                downButton.dataset.id = String(id);
                downButton.disabled = index === ids.length - 1;
                const removeButton = createElement('button', {
                    className:
                        'admin-navigation__button admin-navigation__button--danger',
                    textContent: 'Remove',
                });
                removeButton.type = 'button';
                removeButton.dataset.action = 'remove';
                removeButton.dataset.id = String(id);
                actions.appendChild(upButton);
                actions.appendChild(downButton);
                actions.appendChild(removeButton);
                item.appendChild(info);
                item.appendChild(actions);
                courseTopicVideoList.appendChild(item);
            });
        };

        const renderCoursePackageTopicOptions = () => {
            if (!coursePackageTopicSelect) {
                return;
            }
            const currentValue = coursePackageTopicSelect.value;
            coursePackageTopicSelect.innerHTML = '';
            coursePackageTopicSelect.appendChild(
                createElement('option', {
                    value: '',
                    textContent: 'Select a topic…',
                })
            );
            const selectedIds = new Set(
                state.courses.packageTopicIds.map((id) => String(id))
            );
            state.courses.topics.forEach((topic) => {
                const id = extractCourseTopicId(topic);
                if (!id || selectedIds.has(String(id))) {
                    return;
                }
                const option = createElement('option', {
                    value: id,
                    textContent:
                        normaliseString(topic?.title ?? topic?.Title ?? '') ||
                        'Untitled topic',
                });
                coursePackageTopicSelect.appendChild(option);
            });
            let found = false;
            Array.from(coursePackageTopicSelect.options).forEach((option) => {
                if (option.value === currentValue) {
                    found = true;
                }
            });
            coursePackageTopicSelect.value = found ? currentValue : '';
        };

        const renderCoursePackageTopicList = () => {
            if (!coursePackageTopicList || !coursePackageTopicEmpty) {
                return;
            }
            coursePackageTopicList.innerHTML = '';
            const ids = state.courses.packageTopicIds.filter((id) =>
                Boolean(findCourseTopic(id))
            );
            state.courses.packageTopicIds = ids.slice();
            if (!ids.length) {
                coursePackageTopicEmpty.hidden = false;
                coursePackageTopicList.appendChild(coursePackageTopicEmpty);
                return;
            }
            coursePackageTopicEmpty.hidden = true;
            ids.forEach((id, index) => {
                const topic = findCourseTopic(id);
                if (!topic) {
                    return;
                }
                const item = createElement('li', {
                    className: 'admin-courses__selection-item',
                });
                item.dataset.id = String(id);
                const info = createElement('div', {
                    className: 'admin-courses__selection-info',
                });
                info.appendChild(
                    createElement('span', {
                        className: 'admin-courses__selection-label',
                        textContent:
                            normaliseString(topic?.title ?? topic?.Title ?? '') ||
                            'Untitled topic',
                    })
                );
                const videoCount = getCourseTopicVideos(topic).length;
                info.appendChild(
                    createElement('span', {
                        className: 'admin-courses__selection-meta',
                        textContent:
                            videoCount === 1
                                ? 'Includes 1 video'
                                : `Includes ${videoCount} videos`,
                    })
                );
                const actions = createElement('div', {
                    className: 'admin-courses__selection-actions',
                });
                const upButton = createElement('button', {
                    className: 'admin-navigation__reorder-button',
                    textContent: 'Move up',
                });
                upButton.type = 'button';
                upButton.dataset.action = 'move-up';
                upButton.dataset.id = String(id);
                upButton.disabled = index === 0;
                const downButton = createElement('button', {
                    className: 'admin-navigation__reorder-button',
                    textContent: 'Move down',
                });
                downButton.type = 'button';
                downButton.dataset.action = 'move-down';
                downButton.dataset.id = String(id);
                downButton.disabled = index === ids.length - 1;
                const removeButton = createElement('button', {
                    className:
                        'admin-navigation__button admin-navigation__button--danger',
                    textContent: 'Remove',
                });
                removeButton.type = 'button';
                removeButton.dataset.action = 'remove';
                removeButton.dataset.id = String(id);
                actions.appendChild(upButton);
                actions.appendChild(downButton);
                actions.appendChild(removeButton);
                item.appendChild(info);
                item.appendChild(actions);
                coursePackageTopicList.appendChild(item);
            });
        };

        const resetCourseVideoForm = () => {
            if (!courseVideoForm) {
                return;
            }
            courseVideoForm.reset();
            delete courseVideoForm.dataset.id;
            state.courses.selectedVideoId = '';
            if (courseVideoSubmitButton) {
                courseVideoSubmitButton.textContent = 'Upload video';
            }
            if (courseVideoDeleteButton) {
                courseVideoDeleteButton.hidden = true;
            }
            if (courseVideoUploadGroup) {
                courseVideoUploadGroup.hidden = false;
            }
            if (courseVideoUploadHint) {
                courseVideoUploadHint.hidden = true;
            }
            if (courseVideoFileInput) {
                courseVideoFileInput.required = true;
                courseVideoFileInput.value = '';
            }
            if (courseVideoDurationField) {
                courseVideoDurationField.textContent = '—';
            }
            highlightRow(tables.courseVideos);
            bringFormIntoView(courseVideoForm);
        };

        const populateCourseVideoForm = (video, { scroll = true } = {}) => {
            if (!courseVideoForm || !video) {
                return;
            }
            const id = extractCourseVideoId(video);
            if (id) {
                courseVideoForm.dataset.id = id;
                state.courses.selectedVideoId = id;
            } else {
                delete courseVideoForm.dataset.id;
                state.courses.selectedVideoId = '';
            }
            if (courseVideoTitleInput) {
                courseVideoTitleInput.value = normaliseString(
                    video?.title ?? video?.Title ?? ''
                );
            }
            if (courseVideoDescriptionInput) {
                courseVideoDescriptionInput.value = normaliseString(
                    video?.description ?? video?.Description ?? ''
                );
            }
            if (courseVideoUploadGroup) {
                courseVideoUploadGroup.hidden = true;
            }
            if (courseVideoUploadHint) {
                courseVideoUploadHint.hidden = false;
            }
            if (courseVideoFileInput) {
                courseVideoFileInput.required = false;
                courseVideoFileInput.value = '';
            }
            if (courseVideoDeleteButton) {
                courseVideoDeleteButton.hidden = false;
            }
            if (courseVideoSubmitButton) {
                courseVideoSubmitButton.textContent = 'Update video';
            }
            const duration =
                video?.duration_seconds ??
                video?.durationSeconds ??
                video?.DurationSeconds;
            if (courseVideoDurationField) {
                courseVideoDurationField.textContent = formatVideoDuration(duration);
            }
            highlightRow(tables.courseVideos, id);
            if (scroll) {
                bringFormIntoView(courseVideoForm);
            }
        };

        const selectCourseVideo = (id) => {
            if (!courseVideoForm) {
                return;
            }
            const video = findCourseVideo(id);
            if (!video) {
                return;
            }
            populateCourseVideoForm(video);
        };

        const populateCourseTopicForm = (topic, { scroll = true } = {}) => {
            if (!courseTopicForm || !topic) {
                return;
            }
            const id = extractCourseTopicId(topic);
            if (id) {
                courseTopicForm.dataset.id = id;
                state.courses.selectedTopicId = id;
            } else {
                delete courseTopicForm.dataset.id;
                state.courses.selectedTopicId = '';
            }
            if (courseTopicTitleInput) {
                courseTopicTitleInput.value = normaliseString(
                    topic?.title ?? topic?.Title ?? ''
                );
            }
            if (courseTopicDescriptionInput) {
                courseTopicDescriptionInput.value = normaliseString(
                    topic?.description ?? topic?.Description ?? ''
                );
            }
            state.courses.topicVideoIds = getCourseTopicVideos(topic)
                .map((video) => extractCourseVideoId(video))
                .filter(Boolean);
            if (courseTopicSubmitButton) {
                courseTopicSubmitButton.textContent = 'Update topic';
            }
            if (courseTopicDeleteButton) {
                courseTopicDeleteButton.hidden = false;
            }
            renderCourseTopicVideoOptions();
            renderCourseTopicVideoList();
            highlightRow(tables.courseTopics, id);
            if (scroll) {
                bringFormIntoView(courseTopicForm);
            }
        };

        const selectCourseTopic = (id) => {
            if (!courseTopicForm) {
                return;
            }
            const topic = findCourseTopic(id);
            if (!topic) {
                return;
            }
            populateCourseTopicForm(topic);
        };

        const populateCoursePackageForm = (pkg, { scroll = true } = {}) => {
            if (!coursePackageForm || !pkg) {
                return;
            }
            const id = extractCoursePackageId(pkg);
            if (id) {
                coursePackageForm.dataset.id = id;
                state.courses.selectedPackageId = id;
            } else {
                delete coursePackageForm.dataset.id;
                state.courses.selectedPackageId = '';
            }
            if (coursePackageTitleInput) {
                coursePackageTitleInput.value = normaliseString(
                    pkg?.title ?? pkg?.Title ?? ''
                );
            }
            if (coursePackageDescriptionInput) {
                coursePackageDescriptionInput.value = normaliseString(
                    pkg?.description ?? pkg?.Description ?? ''
                );
            }
            if (coursePackagePriceInput) {
                coursePackagePriceInput.value = formatPriceInputValue(
                    pkg?.price_cents ?? pkg?.priceCents ?? pkg?.PriceCents
                );
            }
            if (coursePackageImageInput) {
                coursePackageImageInput.value = normaliseString(
                    pkg?.image_url ?? pkg?.imageUrl ?? pkg?.ImageURL ?? ''
                );
            }
            state.courses.packageTopicIds = getCoursePackageTopics(pkg)
                .map((topic) => extractCourseTopicId(topic))
                .filter(Boolean);
            if (coursePackageSubmitButton) {
                coursePackageSubmitButton.textContent = 'Update package';
            }
            if (coursePackageDeleteButton) {
                coursePackageDeleteButton.hidden = false;
            }
            renderCoursePackageTopicOptions();
            renderCoursePackageTopicList();
            highlightRow(tables.coursePackages, id);
            if (scroll) {
                bringFormIntoView(coursePackageForm);
            }
        };

        const selectCoursePackage = (id) => {
            if (!coursePackageForm) {
                return;
            }
            const pkg = findCoursePackage(id);
            if (!pkg) {
                return;
            }
            populateCoursePackageForm(pkg);
        };

        const resetCourseTopicForm = () => {
            if (!courseTopicForm) {
                return;
            }
            courseTopicForm.reset();
            delete courseTopicForm.dataset.id;
            state.courses.selectedTopicId = '';
            state.courses.topicVideoIds = [];
            if (courseTopicSubmitButton) {
                courseTopicSubmitButton.textContent = 'Create topic';
            }
            if (courseTopicDeleteButton) {
                courseTopicDeleteButton.hidden = true;
            }
            renderCourseTopicVideoOptions();
            renderCourseTopicVideoList();
            highlightRow(tables.courseTopics);
            bringFormIntoView(courseTopicForm);
        };

        const resetCoursePackageForm = () => {
            if (!coursePackageForm) {
                return;
            }
            coursePackageForm.reset();
            delete coursePackageForm.dataset.id;
            state.courses.selectedPackageId = '';
            state.courses.packageTopicIds = [];
            if (coursePackageSubmitButton) {
                coursePackageSubmitButton.textContent = 'Create package';
            }
            if (coursePackageDeleteButton) {
                coursePackageDeleteButton.hidden = true;
            }
            renderCoursePackageTopicOptions();
            renderCoursePackageTopicList();
            highlightRow(tables.coursePackages);
            bringFormIntoView(coursePackageForm);
        };

        const handleCourseVideoSubmit = async (event) => {
            if (!courseVideoForm) {
                return;
            }
            event.preventDefault();
            const title = normaliseString(courseVideoTitleInput?.value).trim();
            const description = normaliseString(
                courseVideoDescriptionInput?.value
            );
            if (!title) {
                showAlert('Please provide a video title.', 'error');
                return;
            }
            if (!endpoints.coursesVideos) {
                showAlert('Video uploads are not configured.', 'error');
                return;
            }
            const id = courseVideoForm.dataset.id;
            try {
                if (id) {
                    await apiRequest(
                        buildCourseEndpoint(endpoints.coursesVideos, id),
                        {
                            method: 'PUT',
                            headers: {
                                'Content-Type': 'application/json',
                            },
                            body: JSON.stringify({
                                title,
                                description,
                            }),
                        }
                    );
                    showAlert('Video updated successfully.', 'success');
                } else {
                    const file = courseVideoFileInput?.files?.[0];
                    if (!file) {
                        showAlert('Select a video file to upload.', 'error');
                        return;
                    }
                    const formData = new FormData();
                    formData.append('title', title);
                    if (description) {
                        formData.append('description', description);
                    }
                    const preferred = slugifyPreferredName(title);
                    if (preferred) {
                        formData.append('preferred_name', preferred);
                    }
                    formData.append('video', file);
                    const response = await apiRequest(endpoints.coursesVideos, {
                        method: 'POST',
                        body: formData,
                    });
                    showAlert('Video uploaded successfully.', 'success');
                    const created = response?.video;
                    await loadCourseVideos(true);
                    await loadCourseTopics(true);
                    await loadCoursePackages(true);
                    if (created) {
                        const createdId = extractCourseVideoId(created);
                        if (createdId) {
                            const video = findCourseVideo(createdId);
                            if (video) {
                                populateCourseVideoForm(video, { scroll: true });
                            }
                        }
                    } else {
                        resetCourseVideoForm();
                    }
                    return;
                }
                await loadCourseVideos(true);
                await loadCourseTopics(true);
                await loadCoursePackages(true);
                const updatedVideo = findCourseVideo(id);
                if (updatedVideo) {
                    populateCourseVideoForm(updatedVideo, { scroll: false });
                }
            } catch (error) {
                handleRequestError(error);
            }
        };

        const handleCourseVideoDelete = async () => {
            if (!courseVideoForm) {
                return;
            }
            const id = courseVideoForm.dataset.id;
            if (!id) {
                showAlert('Select a video to delete first.', 'info');
                return;
            }
            if (
                !window.confirm(
                    'Delete this video permanently? Any topics using it will no longer list the lesson.'
                )
            ) {
                return;
            }
            if (!endpoints.coursesVideos) {
                showAlert('Video deletion is not configured.', 'error');
                return;
            }
            try {
                await apiRequest(buildCourseEndpoint(endpoints.coursesVideos, id), {
                    method: 'DELETE',
                });
                showAlert('Video deleted successfully.', 'success');
                resetCourseVideoForm();
                await loadCourseVideos(true);
                await loadCourseTopics(true);
                await loadCoursePackages(true);
            } catch (error) {
                handleRequestError(error);
            }
        };

        const handleCourseTopicVideoAdd = () => {
            if (!courseTopicVideoSelect) {
                return;
            }
            const value = courseTopicVideoSelect.value;
            if (!value) {
                showAlert('Select a video to add.', 'info');
                return;
            }
            if (
                state.courses.topicVideoIds.some(
                    (entry) => String(entry) === String(value)
                )
            ) {
                showAlert('This video is already attached to the topic.', 'info');
                return;
            }
            state.courses.topicVideoIds.push(String(value));
            renderCourseTopicVideoOptions();
            renderCourseTopicVideoList();
            courseTopicVideoSelect.value = '';
        };

        const handleCourseTopicVideoListClick = (event) => {
            const target = event.target;
            if (!(target instanceof Element)) {
                return;
            }
            const button = target.closest('button[data-action]');
            if (!button || !courseTopicVideoList?.contains(button)) {
                return;
            }
            event.preventDefault();
            const id = button.dataset.id;
            if (!id) {
                return;
            }
            const index = state.courses.topicVideoIds.findIndex(
                (entry) => String(entry) === String(id)
            );
            if (index === -1) {
                return;
            }
            const action = button.dataset.action;
            if (action === 'remove') {
                state.courses.topicVideoIds.splice(index, 1);
            } else if (action === 'move-up' && index > 0) {
                const [entry] = state.courses.topicVideoIds.splice(index, 1);
                state.courses.topicVideoIds.splice(index - 1, 0, entry);
            } else if (
                action === 'move-down' &&
                index < state.courses.topicVideoIds.length - 1
            ) {
                const [entry] = state.courses.topicVideoIds.splice(index, 1);
                state.courses.topicVideoIds.splice(index + 1, 0, entry);
            }
            renderCourseTopicVideoOptions();
            renderCourseTopicVideoList();
        };

        const handleCourseTopicSubmit = async (event) => {
            if (!courseTopicForm) {
                return;
            }
            event.preventDefault();
            const title = normaliseString(courseTopicTitleInput?.value).trim();
            const description = normaliseString(
                courseTopicDescriptionInput?.value
            );
            if (!title) {
                showAlert('Please provide a topic title.', 'error');
                return;
            }
            if (!endpoints.coursesTopics) {
                showAlert('Topic management is not configured.', 'error');
                return;
            }
            const videoIds = state.courses.topicVideoIds
                .map((entry) => Number.parseInt(String(entry), 10))
                .filter((value) => Number.isFinite(value) && value > 0);
            const id = courseTopicForm.dataset.id;
            try {
                if (id) {
                    await apiRequest(
                        buildCourseEndpoint(endpoints.coursesTopics, id),
                        {
                            method: 'PUT',
                            headers: {
                                'Content-Type': 'application/json',
                            },
                            body: JSON.stringify({
                                title,
                                description,
                            }),
                        }
                    );
                    await apiRequest(
                        buildCourseEndpoint(endpoints.coursesTopics, id, 'videos'),
                        {
                            method: 'PUT',
                            headers: {
                                'Content-Type': 'application/json',
                            },
                            body: JSON.stringify({ video_ids: videoIds }),
                        }
                    );
                    showAlert('Topic updated successfully.', 'success');
                } else {
                    const response = await apiRequest(endpoints.coursesTopics, {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({
                            title,
                            description,
                            video_ids: videoIds,
                        }),
                    });
                    showAlert('Topic created successfully.', 'success');
                    const created = response?.topic;
                    await loadCourseTopics(true);
                    await loadCoursePackages(true);
                    if (created) {
                        const createdId = extractCourseTopicId(created);
                        if (createdId) {
                            const topic = findCourseTopic(createdId);
                            if (topic) {
                                populateCourseTopicForm(topic, { scroll: true });
                            }
                        }
                    } else {
                        resetCourseTopicForm();
                    }
                    return;
                }
                await loadCourseTopics(true);
                await loadCoursePackages(true);
                const updatedTopic = findCourseTopic(id);
                if (updatedTopic) {
                    populateCourseTopicForm(updatedTopic, { scroll: false });
                }
            } catch (error) {
                handleRequestError(error);
            }
        };

        const handleCourseTopicDelete = async () => {
            if (!courseTopicForm) {
                return;
            }
            const id = courseTopicForm.dataset.id;
            if (!id) {
                showAlert('Select a topic to delete first.', 'info');
                return;
            }
            if (
                !window.confirm(
                    'Delete this topic permanently? Packages referencing it will lose the topic.'
                )
            ) {
                return;
            }
            if (!endpoints.coursesTopics) {
                showAlert('Topic management is not configured.', 'error');
                return;
            }
            try {
                await apiRequest(buildCourseEndpoint(endpoints.coursesTopics, id), {
                    method: 'DELETE',
                });
                showAlert('Topic deleted successfully.', 'success');
                resetCourseTopicForm();
                await loadCourseTopics(true);
                await loadCoursePackages(true);
            } catch (error) {
                handleRequestError(error);
            }
        };

        const handleCoursePackageTopicAdd = () => {
            if (!coursePackageTopicSelect) {
                return;
            }
            const value = coursePackageTopicSelect.value;
            if (!value) {
                showAlert('Select a topic to add.', 'info');
                return;
            }
            if (
                state.courses.packageTopicIds.some(
                    (entry) => String(entry) === String(value)
                )
            ) {
                showAlert('This topic is already part of the package.', 'info');
                return;
            }
            state.courses.packageTopicIds.push(String(value));
            renderCoursePackageTopicOptions();
            renderCoursePackageTopicList();
            coursePackageTopicSelect.value = '';
        };

        const handleCoursePackageTopicListClick = (event) => {
            const target = event.target;
            if (!(target instanceof Element)) {
                return;
            }
            const button = target.closest('button[data-action]');
            if (!button || !coursePackageTopicList?.contains(button)) {
                return;
            }
            event.preventDefault();
            const id = button.dataset.id;
            if (!id) {
                return;
            }
            const index = state.courses.packageTopicIds.findIndex(
                (entry) => String(entry) === String(id)
            );
            if (index === -1) {
                return;
            }
            const action = button.dataset.action;
            if (action === 'remove') {
                state.courses.packageTopicIds.splice(index, 1);
            } else if (action === 'move-up' && index > 0) {
                const [entry] = state.courses.packageTopicIds.splice(index, 1);
                state.courses.packageTopicIds.splice(index - 1, 0, entry);
            } else if (
                action === 'move-down' &&
                index < state.courses.packageTopicIds.length - 1
            ) {
                const [entry] = state.courses.packageTopicIds.splice(index, 1);
                state.courses.packageTopicIds.splice(index + 1, 0, entry);
            }
            renderCoursePackageTopicOptions();
            renderCoursePackageTopicList();
        };

        const handleCoursePackageSubmit = async (event) => {
            if (!coursePackageForm) {
                return;
            }
            event.preventDefault();
            const title = normaliseString(coursePackageTitleInput?.value).trim();
            const description = normaliseString(
                coursePackageDescriptionInput?.value
            );
            if (!title) {
                showAlert('Please provide a package title.', 'error');
                return;
            }
            const priceValue = coursePackagePriceInput?.value || '';
            const priceCents = parsePriceInputValue(priceValue);
            if (priceCents === null) {
                showAlert('Enter a valid package price (e.g. 99.90).', 'error');
                return;
            }
            if (!endpoints.coursesPackages) {
                showAlert('Package management is not configured.', 'error');
                return;
            }
            const imageUrl = normaliseString(coursePackageImageInput?.value);
            const topicIds = state.courses.packageTopicIds
                .map((entry) => Number.parseInt(String(entry), 10))
                .filter((value) => Number.isFinite(value) && value > 0);
            const id = coursePackageForm.dataset.id;
            try {
                if (id) {
                    await apiRequest(
                        buildCourseEndpoint(endpoints.coursesPackages, id),
                        {
                            method: 'PUT',
                            headers: {
                                'Content-Type': 'application/json',
                            },
                            body: JSON.stringify({
                                title,
                                description,
                                price_cents: priceCents,
                                image_url: imageUrl,
                            }),
                        }
                    );
                    await apiRequest(
                        buildCourseEndpoint(endpoints.coursesPackages, id, 'topics'),
                        {
                            method: 'PUT',
                            headers: {
                                'Content-Type': 'application/json',
                            },
                            body: JSON.stringify({ topic_ids: topicIds }),
                        }
                    );
                    showAlert('Package updated successfully.', 'success');
                } else {
                    const response = await apiRequest(endpoints.coursesPackages, {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({
                            title,
                            description,
                            price_cents: priceCents,
                            image_url: imageUrl,
                            topic_ids: topicIds,
                        }),
                    });
                    showAlert('Package created successfully.', 'success');
                    const created = response?.package;
                    await loadCoursePackages(true);
                    if (created) {
                        const createdId = extractCoursePackageId(created);
                        if (createdId) {
                            const pkg = findCoursePackage(createdId);
                            if (pkg) {
                                populateCoursePackageForm(pkg, { scroll: true });
                            }
                        }
                    } else {
                        resetCoursePackageForm();
                    }
                    return;
                }
                await loadCoursePackages(true);
                const updatedPackage = findCoursePackage(id);
                if (updatedPackage) {
                    populateCoursePackageForm(updatedPackage, { scroll: false });
                }
            } catch (error) {
                handleRequestError(error);
            }
        };

        const handleCoursePackageDelete = async () => {
            if (!coursePackageForm) {
                return;
            }
            const id = coursePackageForm.dataset.id;
            if (!id) {
                showAlert('Select a package to delete first.', 'info');
                return;
            }
            if (
                !window.confirm(
                    'Delete this package permanently? Customers will no longer see it in the catalog.'
                )
            ) {
                return;
            }
            if (!endpoints.coursesPackages) {
                showAlert('Package management is not configured.', 'error');
                return;
            }
            try {
                await apiRequest(
                    buildCourseEndpoint(endpoints.coursesPackages, id),
                    { method: 'DELETE' }
                );
                showAlert('Package deleted successfully.', 'success');
                resetCoursePackageForm();
                await loadCoursePackages(true);
            } catch (error) {
                handleRequestError(error);
            }
        };

        const buildCourseEndpoint = (base, id, suffix = '') => {
            if (!base) {
                return '';
            }
            const trimmed = base.endsWith('/') ? base.slice(0, -1) : base;
            if (!id) {
                return trimmed;
            }
            const encoded = encodeURIComponent(String(id));
            const normalisedSuffix = suffix
                ? `/${suffix.replace(/^\/+/, '')}`
                : '';
            return `${trimmed}/${encoded}${normalisedSuffix}`;
        };

        const syncTopicSelection = () => {
            state.courses.topicVideoIds = state.courses.topicVideoIds.filter(
                (id) => Boolean(findCourseVideo(id))
            );
            renderCourseTopicVideoOptions();
            renderCourseTopicVideoList();
        };

        const syncPackageSelection = () => {
            state.courses.packageTopicIds = state.courses.packageTopicIds.filter(
                (id) => Boolean(findCourseTopic(id))
            );
            renderCoursePackageTopicOptions();
            renderCoursePackageTopicList();
        };

        const updateVideoFormAfterLoad = () => {
            if (!courseVideoForm) {
                return;
            }
            if (!state.courses.selectedVideoId) {
                resetCourseVideoForm();
                return;
            }
            const video = findCourseVideo(state.courses.selectedVideoId);
            if (video) {
                populateCourseVideoForm(video, { scroll: false });
            } else {
                resetCourseVideoForm();
            }
        };

        const updateTopicFormAfterLoad = () => {
            if (!courseTopicForm) {
                return;
            }
            if (!state.courses.selectedTopicId) {
                state.courses.topicVideoIds = [];
                renderCourseTopicVideoOptions();
                renderCourseTopicVideoList();
                return;
            }
            const topic = findCourseTopic(state.courses.selectedTopicId);
            if (topic) {
                populateCourseTopicForm(topic, { scroll: false });
            } else {
                resetCourseTopicForm();
            }
        };

        const updatePackageFormAfterLoad = () => {
            if (!coursePackageForm) {
                return;
            }
            if (!state.courses.selectedPackageId) {
                state.courses.packageTopicIds = [];
                renderCoursePackageTopicOptions();
                renderCoursePackageTopicList();
                return;
            }
            const pkg = findCoursePackage(state.courses.selectedPackageId);
            if (pkg) {
                populateCoursePackageForm(pkg, { scroll: false });
            } else {
                resetCoursePackageForm();
            }
        };

        const loadCourseVideos = async (force = false) => {
            if (!endpoints.coursesVideos) {
                return;
            }
            if (state.courses.hasLoadedVideos && !force) {
                renderCourseVideos();
                return;
            }
            try {
                const response = await apiRequest(endpoints.coursesVideos);
                const videos = Array.isArray(response?.videos)
                    ? response.videos
                    : [];
                state.courses.videos = videos;
                state.courses.hasLoadedVideos = true;
                renderCourseVideos();
                syncTopicSelection();
                updateVideoFormAfterLoad();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const loadCourseTopics = async (force = false) => {
            if (!endpoints.coursesTopics) {
                return;
            }
            if (state.courses.hasLoadedTopics && !force) {
                renderCourseTopics();
                return;
            }
            try {
                const response = await apiRequest(endpoints.coursesTopics);
                const topics = Array.isArray(response?.topics)
                    ? response.topics
                    : [];
                state.courses.topics = topics;
                state.courses.hasLoadedTopics = true;
                renderCourseTopics();
                syncTopicSelection();
                syncPackageSelection();
                updateTopicFormAfterLoad();
                updatePackageFormAfterLoad();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const loadCoursePackages = async (force = false) => {
            if (!endpoints.coursesPackages) {
                return;
            }
            if (state.courses.hasLoadedPackages && !force) {
                renderCoursePackages();
                return;
            }
            try {
                const response = await apiRequest(endpoints.coursesPackages);
                const packages = Array.isArray(response?.packages)
                    ? response.packages
                    : [];
                state.courses.packages = packages;
                state.courses.hasLoadedPackages = true;
                renderCoursePackages();
                syncPackageSelection();
                updatePackageFormAfterLoad();
            } catch (error) {
                handleRequestError(error);
            }
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
            postForm.dataset.published = String(Boolean(post.published));
            if (postPublishAtInput) {
                const publishAt = extractDateValue(
                    post,
                    'publish_at',
                    'publishAt',
                    'PublishAt'
                );
                postPublishAtInput.value = formatDateTimeInput(publishAt);
            }
            if (postPublishedAtNote) {
                const note = describePublication(post);
                postPublishedAtNote.textContent = note;
                postPublishedAtNote.hidden = !note;
            }
            if (postPublishButton) {
                postPublishButton.textContent = 'Update & publish';
            }
            if (postDraftButton) {
                postDraftButton.textContent = 'Save as draft';
            }
            if (postDeleteButton) {
                postDeleteButton.hidden = false;
            }
            postSectionBuilder?.setSections(extractSectionsFromEntry(post));
            renderTagSuggestions();
            highlightRow(tables.posts, post.id);

            state.selectedPostId = String(post.id ?? id ?? '');
            const publishAtDate = extractDateValue(
                post,
                'publish_at',
                'publishAt',
                'PublishAt'
            );
            const now = Date.now();
            const isPublished = Boolean(post.published ?? post.Published);
            const isLive =
                isPublished &&
                (!publishAtDate || publishAtDate.getTime() <= now);
            if (isLive) {
                const cachedAnalytics = state.postAnalytics.get(
                    state.selectedPostId
                );
                if (cachedAnalytics) {
                    renderPostAnalytics(state.selectedPostId, cachedAnalytics);
                } else {
                    showPostAnalyticsLoading();
                    loadPostAnalytics(state.selectedPostId);
                }
            } else {
                showPostAnalyticsEmpty(
                    'Analytics will appear once this post is published.'
                );
            }
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
            if (postPublishAtInput) {
                postPublishAtInput.value = '';
            }
            if (postPublishedAtNote) {
                postPublishedAtNote.textContent = '';
                postPublishedAtNote.hidden = true;
            }
            delete postForm.dataset.published;
            if (postPublishButton) {
                postPublishButton.textContent = 'Save & publish';
            }
            if (postDraftButton) {
                postDraftButton.textContent = 'Save as draft';
            }
            if (postDeleteButton) {
                postDeleteButton.hidden = true;
            }
            postSectionBuilder?.reset();
            renderTagSuggestions();
            highlightRow(tables.posts);
            bringFormIntoView(postForm);
            state.selectedPostId = '';
            showPostAnalyticsEmpty();
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
            if (pagePathInput) {
                pagePathInput.value = page.path || page.Path || '';
            }
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
            pageForm.dataset.published = String(Boolean(page.published));
            if (pagePublishAtInput) {
                const publishAt = extractDateValue(
                    page,
                    'publish_at',
                    'publishAt',
                    'PublishAt'
                );
                pagePublishAtInput.value = formatDateTimeInput(publishAt);
            }
            if (pagePublishedAtNote) {
                const note = describePublication(page);
                pagePublishedAtNote.textContent = note;
                pagePublishedAtNote.hidden = !note;
            }
            const hideHeaderField = pageForm.querySelector(
                'input[name="hide_header"]'
            );
            if (hideHeaderField) {
                const hideHeaderValue =
                    page.hide_header ?? page.HideHeader ?? false;
                hideHeaderField.checked = Boolean(hideHeaderValue);
            }
            if (pagePublishButton) {
                pagePublishButton.textContent = 'Update & publish';
            }
            if (pageDraftButton) {
                pageDraftButton.textContent = 'Save as draft';
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
            delete pageForm.dataset.published;
            if (pagePublishButton) {
                pagePublishButton.textContent = 'Save & publish';
            }
            if (pageDraftButton) {
                pageDraftButton.textContent = 'Save as draft';
            }
            if (pageDeleteButton) {
                pageDeleteButton.hidden = true;
            }
            if (pagePathInput) {
                pagePathInput.value = '';
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
            if (pagePublishAtInput) {
                pagePublishAtInput.value = '';
            }
            if (pagePublishedAtNote) {
                pagePublishedAtNote.textContent = '';
                pagePublishedAtNote.hidden = true;
            }
            const hideHeaderField = pageForm.querySelector(
                'input[name="hide_header"]'
            );
            if (hideHeaderField) {
                hideHeaderField.checked = false;
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

        const buildUserEndpoint = (id, action = '') => {
            if (!endpoints.users) {
                return '';
            }
            const base = endpoints.users.endsWith('/')
                ? endpoints.users.slice(0, -1)
                : endpoints.users;
            if (!id) {
                return base;
            }
            const encodedId = encodeURIComponent(String(id));
            const suffix = action ? `/${action.replace(/^\/+/, '')}` : '';
            return `${base}/${encodedId}${suffix}`;
        };

        const loadUsers = async () => {
            if (!endpoints.users) {
                return;
            }
            const selectedId = userForm?.dataset.id || '';
            try {
                const payload = await apiRequest(endpoints.users);
                const users = Array.isArray(payload?.users)
                    ? payload.users.slice()
                    : [];
                users.sort((a, b) => {
                    const aDate = new Date(
                        a?.created_at || a?.createdAt || a?.CreatedAt || 0
                    ).getTime();
                    const bDate = new Date(
                        b?.created_at || b?.createdAt || b?.CreatedAt || 0
                    ).getTime();
                    if (Number.isFinite(aDate) && Number.isFinite(bDate) && aDate !== bDate) {
                        return bDate - aDate;
                    }
                    const nameA = normaliseString(
                        a?.username || a?.Username || ''
                    ).toLowerCase();
                    const nameB = normaliseString(
                        b?.username || b?.Username || ''
                    ).toLowerCase();
                    return nameA.localeCompare(nameB);
                });
                state.users = users;
                state.hasLoadedUsers = true;
                renderUsers();
                if (selectedId) {
                    const exists = users.some(
                        (user) => extractUserId(user) === selectedId
                    );
                    if (exists) {
                        selectUser(selectedId);
                    } else {
                        resetUserForm();
                    }
                } else {
                    resetUserForm();
                }
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
            if (settingsForm) {
                const entries = [
                    ['name', site?.name],
                    ['description', site?.description],
                    ['url', site?.url],
                    ['favicon', site?.favicon],
                    ['logo', site?.logo],
                    ['unused_tag_retention_hours', site?.unused_tag_retention_hours],
                    ['stripe_secret_key', site?.stripe_secret_key],
                    ['stripe_publishable_key', site?.stripe_publishable_key],
                    ['stripe_webhook_secret', site?.stripe_webhook_secret],
                    ['course_checkout_success_url', site?.course_checkout_success_url],
                    ['course_checkout_cancel_url', site?.course_checkout_cancel_url],
                ];

                entries.forEach(([key, value]) => {
                    const field = settingsForm.querySelector(`[name="${key}"]`);
                    if (!field) {
                        return;
                    }
                    field.value = value || '';
                });

                const currencyField = settingsForm.querySelector('[name="course_checkout_currency"]');
                if (currencyField) {
                    const currencyValue =
                        typeof site?.course_checkout_currency === 'string'
                            ? site.course_checkout_currency.toUpperCase()
                            : '';
                    currencyField.value = currencyValue;
                }
            }

            updateFaviconPreview(site?.favicon || site?.Favicon || '');
            updateLogoPreview(site?.logo || site?.Logo || '');

            const defaultLanguage =
                typeof site?.default_language === 'string' ? site.default_language : '';
            const supportedLanguages = Array.isArray(site?.supported_languages)
                ? [...site.supported_languages]
                : [];

            state.language.default = defaultLanguage;
            if (supportedLanguages.length > 0) {
                state.language.supported = supportedLanguages;
            } else if (defaultLanguage) {
                state.language.supported = [defaultLanguage];
            } else {
                state.language.supported = [];
            }

            renderLanguageManager();
        };

        const getHomepageStatusInfo = (page) => {
            const title = typeof page?.title === 'string' ? page.title.trim() : '';
            if (!page) {
                return {
                    state: 'none',
                    label: 'Not selected',
                    description:
                        'No homepage override selected. The site uses the page assigned to the "/" path.',
                };
            }

            const publishAt = extractDateValue(page, 'publish_at', 'PublishAt');
            const published = Boolean(page.published);
            if (!published) {
                return {
                    state: 'draft',
                    label: 'Draft',
                    description:
                        `${title || 'The selected page'} is not published. Visitors will continue to see the default homepage.`,
                };
            }

            if (publishAt && Number.isFinite(publishAt.getTime()) && publishAt.getTime() > Date.now()) {
                return {
                    state: 'scheduled',
                    label: `Scheduled for ${formatDate(publishAt)}`,
                    description:
                        `${title || 'The selected page'} is scheduled and will become the homepage once it is published.`,
                };
            }

            return {
                state: 'published',
                label: 'Published',
                description: `${title || 'The selected page'} is published and will be shown as the homepage.`,
            };
        };

        const populateHomepageSelect = () => {
            if (!homepageSelect) {
                return;
            }

            const homepageState = ensureHomepageState();
            const options = Array.isArray(homepageState.options) ? homepageState.options : [];
            const selectedId = homepageState.selectedId || '';

            homepageSelect.innerHTML = '';

            const defaultOption = document.createElement('option');
            defaultOption.value = '';
            defaultOption.textContent = 'Use page assigned to "/" path';
            homepageSelect.appendChild(defaultOption);

            options.forEach((option) => {
                if (!option || option.id === undefined || option.id === null) {
                    return;
                }
                const optionId = String(option.id);
                const statusInfo = getHomepageStatusInfo(option);
                const optionElement = document.createElement('option');
                optionElement.value = optionId;
                const label = option.title || `Untitled page #${option.id}`;
                optionElement.textContent = `${label} — ${statusInfo.label}`;
                homepageSelect.appendChild(optionElement);
            });

            if (selectedId) {
                const exists = options.some((option) => String(option?.id ?? '') === selectedId);
                homepageSelect.value = exists ? selectedId : '';
            } else {
                homepageSelect.value = '';
            }
        };

        const updateHomepageStatus = () => {
            if (!homepageStatus) {
                return;
            }

            const homepageState = ensureHomepageState();
            const statusInfo = getHomepageStatusInfo(homepageState.selected);
            homepageStatus.textContent = statusInfo.description;
            homepageStatus.dataset.status = statusInfo.state || '';
        };

        const renderHomepageOptions = () => {
            if (!homepageOptionsContainer) {
                return;
            }

            const homepageState = ensureHomepageState();
            const options = Array.isArray(homepageState.options) ? homepageState.options : [];
            const selectedId = homepageState.selectedId || '';

            homepageOptionsContainer.innerHTML = '';
            if (homepageEmptyState) {
                homepageEmptyState.hidden = true;
            }

            if (options.length === 0) {
                if (homepageEmptyState) {
                    homepageEmptyState.hidden = false;
                    homepageOptionsContainer.appendChild(homepageEmptyState);
                }
                return;
            }

            const list = createElement('ul', {
                className: 'admin-homepage__list',
            });

            options.forEach((option) => {
                if (!option || option.id === undefined || option.id === null) {
                    return;
                }

                const optionId = String(option.id);
                const statusInfo = getHomepageStatusInfo(option);

                const item = createElement('li', {
                    className: 'admin-homepage__item',
                });

                if (optionId === selectedId && selectedId) {
                    item.classList.add('admin-homepage__item--selected');
                }

                item.appendChild(
                    createElement('h4', {
                        className: 'admin-homepage__title',
                        textContent: option.title || `Untitled page #${option.id}`,
                    })
                );

                const meta = createElement('p', {
                    className: 'admin-homepage__meta',
                });

                const statusClass = statusInfo.state
                    ? `admin-homepage__status admin-homepage__status--${statusInfo.state}`
                    : 'admin-homepage__status';
                meta.appendChild(
                    createElement('span', {
                        className: statusClass,
                        textContent: statusInfo.label,
                    })
                );

                meta.appendChild(
                    createElement('span', {
                        className: 'admin-homepage__path',
                        textContent: option.path || '/',
                    })
                );

                const updatedAt = extractDateValue(option, 'updated_at', 'UpdatedAt');
                if (updatedAt) {
                    meta.appendChild(
                        createElement('span', {
                            className: 'admin-homepage__updated',
                            textContent: `Updated ${formatDate(updatedAt)}`,
                        })
                    );
                }

                item.appendChild(meta);

                if (optionId === selectedId && selectedId) {
                    item.appendChild(
                        createElement('p', {
                            className: 'admin-homepage__selected-note',
                            textContent: 'Currently selected homepage',
                        })
                    );
                }

                list.appendChild(item);
            });

            homepageOptionsContainer.appendChild(list);
        };

        const loadHomepageSettings = async () => {
            if (!homepageForm || !endpoints.homepage) {
                return;
            }

            try {
                const payload = await apiRequest(endpoints.homepage);
                const homepageState = ensureHomepageState();
                homepageState.options = Array.isArray(payload?.options) ? payload.options : [];
                homepageState.selected = payload?.homepage || null;
                homepageState.selectedId = homepageState.selected?.id
                    ? String(homepageState.selected.id)
                    : '';
                homepageState.hasLoaded = true;
                populateHomepageSelect();
                updateHomepageStatus();
                renderHomepageOptions();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const handleHomepageSubmit = async (event) => {
            event.preventDefault();

            if (!homepageForm || !endpoints.homepage) {
                return;
            }

            const homepageState = ensureHomepageState();
            const rawValue = homepageSelect ? homepageSelect.value.trim() : '';
            let payload;

            if (!rawValue) {
                payload = { page_id: null };
            } else {
                const parsedId = Number.parseInt(rawValue, 10);
                if (!Number.isFinite(parsedId)) {
                    showAlert('Select a valid page to use as the homepage.', 'error');
                    return;
                }
                payload = { page_id: parsedId };
            }

            disableForm(homepageForm, true);
            const originalLabel = homepageSubmitButton?.textContent;
            if (homepageSubmitButton) {
                homepageSubmitButton.disabled = true;
                homepageSubmitButton.textContent = 'Saving…';
            }

            clearAlert();

            try {
                const response = await apiRequest(endpoints.homepage, {
                    method: 'PUT',
                    body: JSON.stringify(payload),
                });

                homepageState.options = Array.isArray(response?.options) ? response.options : [];
                homepageState.selected = response?.homepage || null;
                homepageState.selectedId = homepageState.selected?.id
                    ? String(homepageState.selected.id)
                    : '';
                homepageState.hasLoaded = true;

                populateHomepageSelect();
                updateHomepageStatus();
                renderHomepageOptions();

                const message = typeof response?.message === 'string' && response.message.trim() !== ''
                    ? response.message
                    : 'Homepage updated successfully.';
                showAlert(message, 'success');
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(homepageForm, false);
                if (homepageSubmitButton) {
                    homepageSubmitButton.disabled = false;
                    homepageSubmitButton.textContent = originalLabel || 'Save homepage';
                }
            }
        };

        const trimString = (value) => (typeof value === 'string' ? value.trim() : '');
        const lowerTrimString = (value) => trimString(value).toLowerCase();

        const defaultAdvertisingSettings = () => ({
            enabled: false,
            provider: '',
            google_ads: {
                publisher_id: '',
                auto_ads: true,
                slots: [],
            },
        });

        const normalizeAdvertisingSettings = (raw) => {
            const defaults = defaultAdvertisingSettings();
            if (!raw || typeof raw !== 'object') {
                return { ...defaults };
            }

            const providerValue = raw.provider ?? raw.Provider ?? defaults.provider;
            const enabledValue = raw.enabled ?? raw.Enabled ?? defaults.enabled;
            const googleRaw = raw.google_ads ?? raw.GoogleAds;

            const settings = {
                enabled: Boolean(enabledValue),
                provider: lowerTrimString(providerValue),
                google_ads: { ...defaults.google_ads },
            };

            if (googleRaw && typeof googleRaw === 'object') {
                const slotsRaw = googleRaw.slots ?? googleRaw.Slots;
                const slots = Array.isArray(slotsRaw)
                    ? slotsRaw.map((slot) => ({
                          placement: lowerTrimString(slot?.placement ?? slot?.Placement ?? ''),
                          slot_id: trimString(slot?.slot_id ?? slot?.SlotID ?? ''),
                          format: lowerTrimString(slot?.format ?? slot?.Format ?? ''),
                          full_width_responsive: Boolean(
                              slot?.full_width_responsive ?? slot?.FullWidthResponsive
                          ),
                      }))
                    : [];

                settings.google_ads = {
                    publisher_id: trimString(
                        googleRaw?.publisher_id ?? googleRaw?.PublisherID ?? defaults.google_ads.publisher_id
                    ),
                    auto_ads: Boolean(
                        googleRaw?.auto_ads ?? googleRaw?.AutoAds ?? defaults.google_ads.auto_ads
                    ),
                    slots,
                };
            }

            return settings;
        };

        const ensureAdvertisingSettings = () => {
            if (!state.advertising.settings || typeof state.advertising.settings !== 'object') {
                state.advertising.settings = defaultAdvertisingSettings();
            }

            const settings = state.advertising.settings;
            settings.enabled = Boolean(settings.enabled);
            settings.provider = lowerTrimString(settings.provider);

            if (!settings.google_ads || typeof settings.google_ads !== 'object') {
                settings.google_ads = { ...defaultAdvertisingSettings().google_ads };
            } else {
                settings.google_ads.publisher_id = trimString(settings.google_ads.publisher_id);
                settings.google_ads.auto_ads = Boolean(settings.google_ads.auto_ads);
                if (!Array.isArray(settings.google_ads.slots)) {
                    settings.google_ads.slots = [];
                } else {
                    settings.google_ads.slots = settings.google_ads.slots.map((slot) => ({
                        placement: lowerTrimString(slot?.placement),
                        slot_id: trimString(slot?.slot_id),
                        format: lowerTrimString(slot?.format) || 'auto',
                        full_width_responsive: Boolean(slot?.full_width_responsive),
                    }));
                }
            }

            return settings;
        };

        const getAdvertisingProviderMeta = (key) => {
            const providers = Array.isArray(state.advertising.providers)
                ? state.advertising.providers
                : [];
            const normalisedKey = lowerTrimString(key);
            return (
                providers.find(
                    (provider) =>
                        lowerTrimString(provider?.key ?? provider?.Key ?? '') === normalisedKey
                ) || null
            );
        };

        const updateAdvertisingProviderOptions = () => {
            if (!advertisingProviderSelect) {
                return;
            }

            const providers = Array.isArray(state.advertising.providers)
                ? state.advertising.providers
                : [];

            advertisingProviderSelect.innerHTML = '';

            const placeholder = document.createElement('option');
            placeholder.value = '';
            placeholder.textContent = providers.length
                ? 'Select provider'
                : 'No providers available';
            advertisingProviderSelect.appendChild(placeholder);

            providers.forEach((provider) => {
                const key = lowerTrimString(provider?.key ?? provider?.Key ?? '');
                if (!key) {
                    return;
                }
                const option = document.createElement('option');
                option.value = key;
                option.textContent = provider?.name ?? provider?.Name ?? key;
                advertisingProviderSelect.appendChild(option);
            });
        };

        const updateAdvertisingProviderVisibility = (providerKey) => {
            const normalisedKey = lowerTrimString(providerKey);
            advertisingProviderFieldsets.forEach((fieldset) => {
                if (!fieldset || !fieldset.dataset) {
                    return;
                }
                const fieldKey = lowerTrimString(fieldset.dataset.provider || '');
                fieldset.hidden = fieldKey !== normalisedKey;
            });
        };

        const activateAdvertisingProvider = (providerKey) => {
            const targetKey = lowerTrimString(providerKey);
            if (!targetKey) {
                return false;
            }

            const providers = Array.isArray(state.advertising.providers)
                ? state.advertising.providers
                : [];
            const hasProvider = providers.some((provider) => {
                const key = lowerTrimString(provider?.key ?? provider?.Key ?? '');
                return key === targetKey;
            });
            if (!hasProvider) {
                return false;
            }

            const settings = ensureAdvertisingSettings();
            if (settings.provider !== targetKey) {
                settings.provider = targetKey;
            }

            if (advertisingProviderSelect) {
                advertisingProviderSelect.value = targetKey;
            }

            updateAdvertisingProviderVisibility(targetKey);

            return true;
        };

        const renderAdvertisingSlots = () => {
            if (!advertisingSlotsContainer) {
                return;
            }

            advertisingSlotsContainer.innerHTML = '';

            const settings = ensureAdvertisingSettings();
            const providerMeta = getAdvertisingProviderMeta(settings.provider);
            const googleSettings = settings.google_ads || defaultAdvertisingSettings().google_ads;
            const slots = Array.isArray(googleSettings.slots) ? googleSettings.slots : [];

            if (!providerMeta || settings.provider !== 'google_ads') {
                const message = document.createElement('p');
                message.className = 'admin-ads__slots-empty';
                message.textContent = settings.provider
                    ? 'Manual placements are not available for this provider.'
                    : 'Select a provider to manage manual placements.';
                advertisingSlotsContainer.appendChild(message);
                return;
            }

            if (slots.length === 0) {
                const message = document.createElement('p');
                message.className = 'admin-ads__slots-empty';
                message.textContent = 'No manual placements configured.';
                advertisingSlotsContainer.appendChild(message);
                return;
            }

            const placements = Array.isArray(providerMeta.placements)
                ? providerMeta.placements
                : [];
            const formats = Array.isArray(providerMeta.formats) ? providerMeta.formats : [];

            slots.forEach((slot, index) => {
                const row = document.createElement('div');
                row.className = 'admin-ads__slot';
                row.dataset.index = String(index);

                const fields = document.createElement('div');
                fields.className = 'admin-ads__slot-fields';

                const placementLabel = document.createElement('label');
                placementLabel.className = 'admin-form__label';
                placementLabel.textContent = 'Placement';
                const placementSelect = document.createElement('select');
                placementSelect.className = 'admin-form__input';
                placementSelect.dataset.role = 'ads-slot-placement';
                placementSelect.dataset.index = String(index);

                const seenPlacements = new Set();
                placements.forEach((placement) => {
                    const key = lowerTrimString(placement?.key ?? placement?.Key ?? '');
                    if (!key || seenPlacements.has(key)) {
                        return;
                    }
                    const option = document.createElement('option');
                    option.value = key;
                    option.textContent = placement?.label ?? placement?.Label ?? key;
                    placementSelect.appendChild(option);
                    seenPlacements.add(key);
                });

                const currentPlacement = lowerTrimString(slot?.placement ?? '');
                if (currentPlacement && !seenPlacements.has(currentPlacement)) {
                    const option = document.createElement('option');
                    option.value = currentPlacement;
                    option.textContent = currentPlacement;
                    placementSelect.appendChild(option);
                }

                if (placementSelect.options.length > 0) {
                    placementSelect.value = currentPlacement || placementSelect.options[0].value;
                }

                placementLabel.appendChild(placementSelect);

                const slotLabel = document.createElement('label');
                slotLabel.className = 'admin-form__label';
                slotLabel.textContent = 'Ad unit ID';
                const slotInput = document.createElement('input');
                slotInput.className = 'admin-form__input';
                slotInput.type = 'text';
                slotInput.placeholder = 'e.g. 1234567890';
                slotInput.dataset.role = 'ads-slot-id';
                slotInput.dataset.index = String(index);
                slotInput.value = trimString(slot?.slot_id ?? '');
                slotLabel.appendChild(slotInput);

                const formatLabel = document.createElement('label');
                formatLabel.className = 'admin-form__label';
                formatLabel.textContent = 'Format';
                const formatSelect = document.createElement('select');
                formatSelect.className = 'admin-form__input';
                formatSelect.dataset.role = 'ads-slot-format';
                formatSelect.dataset.index = String(index);

                const seenFormats = new Set();
                formats.forEach((format) => {
                    const key = lowerTrimString(format?.key ?? format?.Key ?? '');
                    if (!key || seenFormats.has(key)) {
                        return;
                    }
                    const option = document.createElement('option');
                    option.value = key;
                    option.textContent = format?.label ?? format?.Label ?? key;
                    formatSelect.appendChild(option);
                    seenFormats.add(key);
                });

                const currentFormat = lowerTrimString(slot?.format ?? '') || 'auto';
                if (!seenFormats.has(currentFormat)) {
                    const option = document.createElement('option');
                    option.value = currentFormat;
                    option.textContent = currentFormat;
                    formatSelect.appendChild(option);
                }
                formatSelect.value = currentFormat;
                formatLabel.appendChild(formatSelect);

                const responsiveLabel = document.createElement('label');
                responsiveLabel.className = 'admin-form__checkbox admin-ads__slot-checkbox';
                const responsiveInput = document.createElement('input');
                responsiveInput.type = 'checkbox';
                responsiveInput.className = 'checkbox__input';
                responsiveInput.dataset.role = 'ads-slot-responsive';
                responsiveInput.dataset.index = String(index);
                responsiveInput.checked = Boolean(slot?.full_width_responsive);
                responsiveLabel.appendChild(responsiveInput);
                const responsiveText = document.createElement('span');
                responsiveText.textContent = 'Full-width responsive';
                responsiveLabel.appendChild(responsiveText);

                fields.appendChild(placementLabel);
                fields.appendChild(slotLabel);
                fields.appendChild(formatLabel);
                fields.appendChild(responsiveLabel);
                row.appendChild(fields);

                const removeButton = document.createElement('button');
                removeButton.type = 'button';
                removeButton.className = 'admin-form__link-button admin-ads__slot-remove';
                removeButton.dataset.role = 'ads-slot-remove';
                removeButton.dataset.index = String(index);
                removeButton.textContent = 'Remove placement';
                row.appendChild(removeButton);

                advertisingSlotsContainer.appendChild(row);
            });
        };

        const populateAdvertisingForm = () => {
            if (!advertisingForm || !endpoints.advertising) {
                return;
            }

            const settings = ensureAdvertisingSettings();
            updateAdvertisingProviderOptions();

            const providers = Array.isArray(state.advertising.providers)
                ? state.advertising.providers
                : [];
            let providerKey = settings.provider;
            const providerMeta = getAdvertisingProviderMeta(providerKey);
            if (!providerMeta && providers.length > 0) {
                providerKey = lowerTrimString(providers[0]?.key ?? providers[0]?.Key ?? '');
                settings.provider = providerKey;
            }

            if (advertisingEnabledToggle) {
                advertisingEnabledToggle.checked = Boolean(settings.enabled);
            }

            if (advertisingProviderSelect) {
                advertisingProviderSelect.value = providerKey;
            }

            updateAdvertisingProviderVisibility(providerKey);

            if (advertisingSlotAddButton) {
                advertisingSlotAddButton.disabled = providerKey !== 'google_ads';
            }

            if (providerKey === 'google_ads') {
                const google = ensureAdvertisingSettings().google_ads;
                if (advertisingPublisherInput) {
                    advertisingPublisherInput.value = trimString(google.publisher_id);
                }
                if (advertisingAutoToggle) {
                    advertisingAutoToggle.checked = Boolean(google.auto_ads);
                }
            } else {
                if (advertisingPublisherInput) {
                    advertisingPublisherInput.value = '';
                }
                if (advertisingAutoToggle) {
                    advertisingAutoToggle.checked = false;
                }
            }

            renderAdvertisingSlots();
        };

        const loadAdvertisingSettings = async () => {
            if (!advertisingForm || !endpoints.advertising) {
                return;
            }

            try {
                disableForm(advertisingForm, true);
                const response = await apiRequest(endpoints.advertising, { method: 'GET' });
                if (response && Array.isArray(response.providers)) {
                    state.advertising.providers = response.providers;
                } else {
                    state.advertising.providers = [];
                }

                state.advertising.settings = normalizeAdvertisingSettings(response?.settings);
                ensureAdvertisingSettings();
                populateAdvertisingForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(advertisingForm, false);
            }
        };

        const handleAdvertisingSubmit = async (event) => {
            event.preventDefault();

            if (!advertisingForm || !endpoints.advertising) {
                return;
            }

            const settings = ensureAdvertisingSettings();
            const payload = {
                enabled: Boolean(settings.enabled),
                provider: settings.provider,
            };

            if (settings.provider === 'google_ads') {
                const google = settings.google_ads || defaultAdvertisingSettings().google_ads;
                payload.google_ads = {
                    publisher_id: trimString(google.publisher_id),
                    auto_ads: Boolean(google.auto_ads),
                    slots: Array.isArray(google.slots)
                        ? google.slots.map((slot) => ({
                              placement: lowerTrimString(slot?.placement ?? ''),
                              slot_id: trimString(slot?.slot_id ?? ''),
                              format: lowerTrimString(slot?.format ?? ''),
                              full_width_responsive: Boolean(slot?.full_width_responsive),
                          }))
                        : [],
                };
            }

            try {
                disableForm(advertisingForm, true);
                const response = await apiRequest(endpoints.advertising, {
                    method: 'PUT',
                    body: JSON.stringify(payload),
                });

                if (response && Array.isArray(response.providers)) {
                    state.advertising.providers = response.providers;
                }

                state.advertising.settings = normalizeAdvertisingSettings(response?.settings ?? payload);
                ensureAdvertisingSettings();
                populateAdvertisingForm();
                showAlert('Advertising settings updated.', 'success');
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(advertisingForm, false);
            }
        };

        const handleAdvertisingProviderChange = (event) => {
            const value = lowerTrimString(event?.target?.value ?? '');
            const settings = ensureAdvertisingSettings();
            settings.provider = value;
            populateAdvertisingForm();
        };

        const handleAdvertisingEnabledChange = (event) => {
            const settings = ensureAdvertisingSettings();
            settings.enabled = Boolean(event?.target?.checked);
        };

        const handleAdvertisingPublisherInput = (event) => {
            const settings = ensureAdvertisingSettings();
            if (!settings.google_ads) {
                settings.google_ads = { ...defaultAdvertisingSettings().google_ads };
            }
            settings.google_ads.publisher_id = trimString(event?.target?.value ?? '');
        };

        const handleAdvertisingAutoChange = (event) => {
            const settings = ensureAdvertisingSettings();
            if (!settings.google_ads) {
                settings.google_ads = { ...defaultAdvertisingSettings().google_ads };
            }
            settings.google_ads.auto_ads = Boolean(event?.target?.checked);
        };

        const handleAdvertisingSlotChange = (event) => {
            const target = event?.target;
            if (!target || !target.dataset) {
                return;
            }
            const indexValue = Number.parseInt(target.dataset.index || '', 10);
            if (!Number.isFinite(indexValue)) {
                return;
            }

            const settings = ensureAdvertisingSettings();
            const slots = settings.google_ads?.slots;
            if (!Array.isArray(slots) || !slots[indexValue]) {
                return;
            }

            const slot = slots[indexValue];
            switch (target.dataset.role) {
                case 'ads-slot-placement':
                    slot.placement = lowerTrimString(target.value);
                    break;
                case 'ads-slot-id':
                    slot.slot_id = trimString(target.value);
                    break;
                case 'ads-slot-format':
                    slot.format = lowerTrimString(target.value);
                    break;
                case 'ads-slot-responsive':
                    slot.full_width_responsive = Boolean(target.checked);
                    break;
                default:
                    break;
            }
        };

        const handleAdvertisingSlotClick = (event) => {
            const button = event?.target?.closest('[data-role="ads-slot-remove"]');
            if (!button) {
                return;
            }

            event.preventDefault();

            const indexValue = Number.parseInt(button.dataset.index || '', 10);
            if (!Number.isFinite(indexValue)) {
                return;
            }

            const settings = ensureAdvertisingSettings();
            const slots = settings.google_ads?.slots;
            if (!Array.isArray(slots) || indexValue < 0 || indexValue >= slots.length) {
                return;
            }

            slots.splice(indexValue, 1);
            renderAdvertisingSlots();
        };

        const handleAdvertisingAddSlot = (event) => {
            if (event) {
                event.preventDefault();
            }

            const settings = ensureAdvertisingSettings();
            if (settings.provider !== 'google_ads') {
                const providerActivated = activateAdvertisingProvider('google_ads');
                if (!providerActivated) {
                    showAlert('Select Google AdSense to add manual placements.', 'info');
                    return;
                }
                settings.provider = 'google_ads';
            }

            if (advertisingSlotAddButton) {
                advertisingSlotAddButton.disabled = false;
            }

            if (settings.provider !== 'google_ads') {
                showAlert('Select Google AdSense to add manual placements.', 'info');
                return;
            }

            const providerMeta = getAdvertisingProviderMeta('google_ads');
            const placements = Array.isArray(providerMeta?.placements)
                ? providerMeta.placements
                : [];
            const formats = Array.isArray(providerMeta?.formats) ? providerMeta.formats : [];
            const defaultPlacement = lowerTrimString(
                placements[0]?.key ?? placements[0]?.Key ?? 'post_content_top'
            );
            const defaultFormat = lowerTrimString(
                formats[0]?.key ?? formats[0]?.Key ?? 'auto'
            ) || 'auto';

            if (!Array.isArray(settings.google_ads?.slots)) {
                settings.google_ads.slots = [];
            }

            settings.google_ads.slots.push({
                placement: defaultPlacement,
                slot_id: '',
                format: defaultFormat,
                full_width_responsive: true,
            });

            renderAdvertisingSlots();
        };

        const createPluginItem = (plugin) => {
            const item = document.createElement('li');
            item.className = 'admin-plugins__item';
            item.dataset.pluginItem = 'true';

            const slug = normaliseString(plugin?.slug ?? '');
            if (slug) {
                item.dataset.pluginSlug = slug;
            }

            const pluginName = normaliseString(plugin?.name ?? '') || slug || 'Plugin';

            const info = document.createElement('div');
            info.className = 'admin-plugins__info';

            const title = document.createElement('div');
            title.className = 'admin-plugins__title';

            const nameEl = document.createElement('span');
            nameEl.className = 'admin-plugins__name';
            nameEl.textContent = pluginName;
            title.appendChild(nameEl);

            const version = normaliseString(plugin?.version ?? '');
            if (version) {
                const versionEl = document.createElement('span');
                versionEl.className = 'admin-plugins__version';
                versionEl.textContent = `v${version}`;
                title.appendChild(versionEl);
            }

            const badge = document.createElement('span');
            badge.className = 'admin-plugins__badge';
            if (plugin?.missing_files) {
                badge.dataset.status = 'error';
                badge.textContent = 'Files missing';
            } else if (plugin?.active) {
                badge.dataset.status = 'active';
                badge.textContent = 'Active';
            } else {
                badge.dataset.status = 'inactive';
                badge.textContent = 'Inactive';
            }
            title.appendChild(badge);

            info.appendChild(title);

            const description = normaliseString(plugin?.description ?? '');
            if (description) {
                const descEl = document.createElement('p');
                descEl.className = 'admin-plugins__description';
                descEl.textContent = description;
                info.appendChild(descEl);
            }

            const author = normaliseString(plugin?.author ?? '');
            const homepage = normaliseString(plugin?.homepage ?? '');
            if (author || homepage) {
                const authorEl = document.createElement('p');
                authorEl.className = 'admin-plugins__author';
                authorEl.textContent = 'By ';
                if (homepage) {
                    const authorLink = document.createElement('a');
                    authorLink.href = homepage;
                    authorLink.target = '_blank';
                    authorLink.rel = 'noopener noreferrer';
                    authorLink.textContent = author || homepage;
                    authorEl.appendChild(authorLink);
                } else {
                    authorEl.append(author || 'Unknown');
                }
                info.appendChild(authorEl);
            }

            const installedAt = coerceDateValue(plugin?.installed_at);
            const lastActiveAt = coerceDateValue(plugin?.last_active_at);
            const metaParts = [];
            if (installedAt) {
                metaParts.push(`Installed ${formatDate(installedAt)}`);
            }
            if (lastActiveAt) {
                metaParts.push(`Last activated ${formatDate(lastActiveAt)}`);
            }
            if (plugin?.missing_files) {
                metaParts.push('Files missing');
            }
            if (metaParts.length) {
                const metaEl = document.createElement('p');
                metaEl.className = 'admin-plugins__meta';
                metaEl.textContent = metaParts.join(' · ');
                info.appendChild(metaEl);
            }

            item.appendChild(info);

            const actions = document.createElement('div');
            actions.className = 'admin-plugins__actions';

            if (plugin?.missing_files) {
                const warning = document.createElement('p');
                warning.className = 'admin-plugins__warning';
                warning.textContent = 'Plugin files are missing. Reinstall the plugin to restore it.';
                actions.appendChild(warning);
            } else if (plugin?.active) {
                const deactivateButton = document.createElement('button');
                deactivateButton.type = 'button';
                deactivateButton.className = 'admin-plugins__deactivate';
                deactivateButton.dataset.role = 'plugin-deactivate';
                deactivateButton.dataset.pluginName = pluginName;
                if (slug) {
                    deactivateButton.dataset.pluginSlug = slug;
                }
                deactivateButton.textContent = 'Deactivate';
                actions.appendChild(deactivateButton);
            } else {
                const activateButton = document.createElement('button');
                activateButton.type = 'button';
                activateButton.className = 'admin-form__submit';
                activateButton.dataset.role = 'plugin-activate';
                activateButton.dataset.pluginName = pluginName;
                if (slug) {
                    activateButton.dataset.pluginSlug = slug;
                }
                activateButton.textContent = 'Activate';
                actions.appendChild(activateButton);
            }

            const deleteButton = document.createElement('button');
            deleteButton.type = 'button';
            deleteButton.className = 'admin-plugins__delete';
            deleteButton.dataset.role = 'plugin-delete';
            deleteButton.dataset.pluginName = pluginName;
            if (slug) {
                deleteButton.dataset.pluginSlug = slug;
            } else {
                deleteButton.disabled = true;
                deleteButton.title = 'Plugin identifier unavailable';
            }
            deleteButton.textContent = 'Delete';
            actions.appendChild(deleteButton);

            item.appendChild(actions);

            return item;
        };

        const renderPluginList = () => {
            if (!pluginList) {
                return;
            }

            pluginList.querySelectorAll('[data-plugin-item]').forEach((node) => node.remove());

            const plugins = Array.isArray(state.plugins) ? state.plugins : [];
            if (!plugins.length) {
                if (pluginEmptyState) {
                    pluginEmptyState.hidden = false;
                }
                return;
            }

            if (pluginEmptyState) {
                pluginEmptyState.hidden = true;
            }

            const fragment = document.createDocumentFragment();
            plugins.forEach((pluginEntry) => {
                fragment.appendChild(createPluginItem(pluginEntry));
            });
            pluginList.appendChild(fragment);
        };

        const loadPlugins = async () => {
            if (!endpoints.plugins) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.plugins);
                const plugins = Array.isArray(payload?.plugins) ? payload.plugins : [];
                state.plugins = plugins;
                renderPluginList();
            } catch (error) {
                handleRequestError(error);
            }
        };

        let siteReloadTimer = null;
        const scheduleSiteReload = () => {
            if (siteReloadTimer !== null) {
                return;
            }
            siteReloadTimer = window.setTimeout(() => {
                window.location.reload();
            }, 500);
        };

        const handlePluginListClick = async (event) => {
            const activateButton = event.target?.closest('[data-role="plugin-activate"]');
            if (activateButton && pluginList?.contains(activateButton)) {
                event.preventDefault();

                const slug = normaliseString(activateButton.dataset.pluginSlug ?? '');
                if (!slug) {
                    return;
                }

                if (!endpoints.plugins) {
                    showAlert('Plugin activation is not available in this environment.', 'error');
                    return;
                }

                const pluginName = activateButton.dataset.pluginName || slug;
                const base = endpoints.plugins.endsWith('/')
                    ? endpoints.plugins.slice(0, -1)
                    : endpoints.plugins;
                const url = `${base}/${encodeURIComponent(slug)}/activate`;

                const originalText = activateButton.textContent;
                activateButton.disabled = true;
                activateButton.dataset.loading = 'true';
                activateButton.textContent = 'Activating…';
                showAlert(`Activating "${pluginName}"…`, 'info');

                try {
                    await apiRequest(url, { method: 'PUT' });
                    await loadPlugins();
                    showAlert(`Plugin "${pluginName}" activated.`, 'success');
                    scheduleSiteReload();
                    return;
                } catch (error) {
                    activateButton.disabled = false;
                    activateButton.textContent = originalText || 'Activate';
                    handleRequestError(error);
                } finally {
                    activateButton.removeAttribute('data-loading');
                }

                return;
            }

            const deactivateButton = event.target?.closest('[data-role="plugin-deactivate"]');
            if (deactivateButton && pluginList?.contains(deactivateButton)) {
                event.preventDefault();

                const slug = normaliseString(deactivateButton.dataset.pluginSlug ?? '');
                if (!slug) {
                    return;
                }

                if (!endpoints.plugins) {
                    showAlert('Plugin deactivation is not available in this environment.', 'error');
                    return;
                }

                const pluginName = deactivateButton.dataset.pluginName || slug;
                const base = endpoints.plugins.endsWith('/')
                    ? endpoints.plugins.slice(0, -1)
                    : endpoints.plugins;
                const url = `${base}/${encodeURIComponent(slug)}/deactivate`;

                const originalText = deactivateButton.textContent;
                deactivateButton.disabled = true;
                deactivateButton.dataset.loading = 'true';
                deactivateButton.textContent = 'Deactivating…';
                showAlert(`Deactivating "${pluginName}"…`, 'info');

                try {
                    await apiRequest(url, { method: 'PUT' });
                    await loadPlugins();
                    showAlert(`Plugin "${pluginName}" deactivated.`, 'success');
                    scheduleSiteReload();
                    return;
                } catch (error) {
                    deactivateButton.disabled = false;
                    deactivateButton.textContent = originalText || 'Deactivate';
                    handleRequestError(error);
                } finally {
                    deactivateButton.removeAttribute('data-loading');
                }
                return;
            }

            const deleteButton = event.target?.closest('[data-role="plugin-delete"]');
            if (deleteButton && pluginList?.contains(deleteButton)) {
                event.preventDefault();

                const slug = normaliseString(deleteButton.dataset.pluginSlug ?? '');
                if (!slug) {
                    showAlert('Unable to delete this plugin because the identifier is missing.', 'error');
                    return;
                }

                if (!endpoints.plugins) {
                    showAlert('Plugin deletion is not available in this environment.', 'error');
                    return;
                }

                const pluginName = deleteButton.dataset.pluginName || slug;
                const confirmed = window.confirm(
                    `Delete "${pluginName}" permanently? This action cannot be undone.`,
                );
                if (!confirmed) {
                    return;
                }

                const base = endpoints.plugins.endsWith('/')
                    ? endpoints.plugins.slice(0, -1)
                    : endpoints.plugins;
                const url = `${base}/${encodeURIComponent(slug)}`;

                const originalText = deleteButton.textContent;
                deleteButton.disabled = true;
                deleteButton.dataset.loading = 'true';
                deleteButton.textContent = 'Deleting…';
                showAlert(`Deleting "${pluginName}"…`, 'info');

                try {
                    await apiRequest(url, { method: 'DELETE' });
                    await loadPlugins();
                    showAlert(`Plugin "${pluginName}" deleted.`, 'success');
                    scheduleSiteReload();
                    return;
                } catch (error) {
                    deleteButton.disabled = false;
                    deleteButton.textContent = originalText || 'Delete';
                    handleRequestError(error);
                } finally {
                    deleteButton.removeAttribute('data-loading');
                }
            }
        };

        const handlePluginInstallSubmit = async (event) => {
            if (!pluginInstallForm) {
                return;
            }

            event.preventDefault();

            if (!endpoints.plugins) {
                showAlert('Plugin installation is not available in this environment.', 'error');
                return;
            }

            const file = pluginUploadInput?.files?.[0];
            if (!file) {
                showAlert('Please select a plugin archive to upload.', 'error');
                return;
            }

            const formData = new FormData();
            formData.append('file', file);

            const originalText = pluginInstallButton?.textContent || 'Install plugin';
            if (typeof toggleFormDisabled === 'function') {
                toggleFormDisabled(pluginInstallForm, true);
            }
            if (pluginInstallButton) {
                pluginInstallButton.disabled = true;
                pluginInstallButton.dataset.loading = 'true';
                pluginInstallButton.textContent = 'Installing…';
            }

            showAlert(`Installing "${file.name}"…`, 'info');

            try {
                const response = await apiRequest(endpoints.plugins, {
                    method: 'POST',
                    body: formData,
                });
                const installedPlugin = response?.plugin;
                const installedName = normaliseString(installedPlugin?.name ?? '') || file.name;
                showAlert(`Plugin "${installedName}" installed successfully.`, 'success');
                pluginInstallForm.reset();
                await loadPlugins();
            } catch (error) {
                handleRequestError(error);
            } finally {
                if (typeof toggleFormDisabled === 'function') {
                    toggleFormDisabled(pluginInstallForm, false);
                }
                if (pluginInstallButton) {
                    pluginInstallButton.disabled = false;
                    pluginInstallButton.textContent = originalText;
                    pluginInstallButton.removeAttribute('data-loading');
                }
            }
        };

        const createThemeItem = (theme) => {
            const item = document.createElement('li');
            item.className = 'admin-theme__item';
            item.dataset.themeItem = 'true';

            const slug = normaliseString(theme?.slug ?? theme?.Slug ?? '');
            if (slug) {
                item.dataset.themeSlug = slug;
            }
            item.dataset.active = theme?.active ? 'true' : 'false';

            const title = document.createElement('div');
            title.className = 'admin-theme__title';

            const name = document.createElement('span');
            const themeName = normaliseString(theme?.name ?? theme?.Name ?? slug) || 'Theme';
            name.textContent = themeName;
            title.appendChild(name);

            if (theme?.active) {
                const badge = document.createElement('span');
                badge.className = 'admin-theme__badge';
                badge.textContent = 'Active';
                title.appendChild(badge);
            }

            item.appendChild(title);

            const description = normaliseString(theme?.description ?? theme?.Description ?? '');
            if (description) {
                const descEl = document.createElement('p');
                descEl.className = 'admin-theme__description';
                descEl.textContent = description;
                item.appendChild(descEl);
            }

            const metaEntries = [];
            const version = normaliseString(theme?.version ?? theme?.Version ?? '');
            if (version) {
                metaEntries.push(`Version ${version}`);
            }
            const author = normaliseString(theme?.author ?? theme?.Author ?? '');
            if (author) {
                metaEntries.push(`By ${author}`);
            }

            if (metaEntries.length) {
                const metaEl = document.createElement('p');
                metaEl.className = 'admin-theme__meta';
                metaEntries.forEach((entry) => {
                    const span = document.createElement('span');
                    span.textContent = entry;
                    metaEl.appendChild(span);
                });
                item.appendChild(metaEl);
            }

            const actions = document.createElement('div');
            actions.className = 'admin-theme__actions';

            const button = document.createElement('button');
            button.type = 'button';
            button.className = 'admin-form__submit';
            button.dataset.role = 'theme-activate';
            if (slug) {
                button.dataset.themeSlug = slug;
            }
            button.dataset.themeName = themeName;

            if (theme?.active) {
                button.disabled = true;
                button.textContent = 'Current theme';
            } else {
                button.textContent = 'Activate theme';
            }

            actions.appendChild(button);
            if (theme?.active) {
                const reloadButton = document.createElement('button');
                reloadButton.type = 'button';
                reloadButton.className = 'admin-theme__reload';
                reloadButton.dataset.role = 'theme-reload';
                if (slug) {
                    reloadButton.dataset.themeSlug = slug;
                }
                reloadButton.dataset.themeName = themeName;
                reloadButton.textContent = 'Reload from defaults';
                actions.appendChild(reloadButton);
            }
            item.appendChild(actions);

            return item;
        };

        const renderThemeList = () => {
            if (!themeList) {
                return;
            }

            themeList.querySelectorAll('[data-theme-item]').forEach((node) => node.remove());

            const themes = Array.isArray(state.themes) ? state.themes : [];
            if (!themes.length) {
                if (themeEmptyState) {
                    themeEmptyState.hidden = false;
                }
                return;
            }

            if (themeEmptyState) {
                themeEmptyState.hidden = true;
            }

            const fragment = document.createDocumentFragment();
            themes.forEach((theme) => {
                fragment.appendChild(createThemeItem(theme));
            });
            themeList.appendChild(fragment);
        };

        const loadThemes = async () => {
            if (!endpoints.themes) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.themes);
                const themes = Array.isArray(payload?.themes) ? payload.themes : [];
                state.themes = themes;
                renderThemeList();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const handleThemeListClick = async (event) => {
            const reloadButton = event.target?.closest('[data-role="theme-reload"]');
            if (reloadButton && themeList?.contains(reloadButton)) {
                event.preventDefault();

                const slug = normaliseString(reloadButton.dataset.themeSlug ?? '');
                if (!slug) {
                    return;
                }

                if (!endpoints.themes) {
                    showAlert('Theme reload is not available in this environment.', 'error');
                    return;
                }

                const name = reloadButton.dataset.themeName || slug;
                const confirmed = window.confirm(
                    `Reloading "${name}" will replace the theme's default pages and menus. Continue?`,
                );
                if (!confirmed) {
                    return;
                }

                const baseReload = endpoints.themes.endsWith('/')
                    ? endpoints.themes.slice(0, -1)
                    : endpoints.themes;
                const reloadUrl = `${baseReload}/${encodeURIComponent(slug)}/reload`;

                const originalReloadText = reloadButton.textContent;
                reloadButton.disabled = true;
                reloadButton.dataset.loading = 'true';
                reloadButton.textContent = 'Reloading…';
                showAlert(`Reloading "${name}" from defaults…`, 'info');

                try {
                    await apiRequest(reloadUrl, { method: 'PUT' });
                    showAlert(`Theme "${name}" reset to defaults. Reloading…`, 'success');
                    setTimeout(() => {
                        window.location.reload();
                    }, 800);
                } catch (error) {
                    reloadButton.disabled = false;
                    reloadButton.textContent = originalReloadText || 'Reload from defaults';
                    handleRequestError(error);
                } finally {
                    reloadButton.removeAttribute('data-loading');
                }

                return;
            }

            const button = event.target?.closest('[data-role="theme-activate"]');
            if (!button || !themeList?.contains(button)) {
                return;
            }

            event.preventDefault();

            const slug = normaliseString(button.dataset.themeSlug ?? '');
            if (!slug) {
                return;
            }

            if (!endpoints.themes) {
                showAlert('Theme activation is not available in this environment.', 'error');
                return;
            }

            const name = button.dataset.themeName || slug;
            const base = endpoints.themes.endsWith('/')
                ? endpoints.themes.slice(0, -1)
                : endpoints.themes;
            const url = `${base}/${encodeURIComponent(slug)}/activate`;

            const originalText = button.textContent;
            button.disabled = true;
            button.textContent = 'Activating…';
            showAlert(`Activating "${name}"…`, 'info');

            try {
                await apiRequest(url, { method: 'PUT' });
                showAlert(`Theme "${name}" activated. Reloading to apply changes…`, 'success');
                setTimeout(() => {
                    window.location.reload();
                }, 800);
            } catch (error) {
                button.disabled = false;
                button.textContent = originalText || 'Activate theme';
                handleRequestError(error);
            }
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

        const normaliseFontEntry = (font) => {
            const idValue = font?.id ?? font?.ID ?? font?.Id ?? '';
            const preconnectsValue = font?.preconnects ?? font?.Preconnects ?? [];
            const orderValue = Number.parseInt(font?.order ?? font?.Order ?? 0, 10);
            return {
                id: idValue ? String(idValue) : '',
                name: String(font?.name ?? font?.Name ?? 'Font').trim() || 'Font',
                snippet: String(font?.snippet ?? font?.Snippet ?? '').trim(),
                preconnects: Array.isArray(preconnectsValue)
                    ? preconnectsValue
                          .map((entry) => (typeof entry === 'string' ? entry.trim() : ''))
                          .filter(Boolean)
                    : [],
                enabled:
                    font?.enabled !== undefined
                        ? Boolean(font.enabled)
                        : font?.Enabled !== undefined
                          ? Boolean(font.Enabled)
                          : true,
                notes: String(font?.notes ?? font?.Notes ?? '').trim(),
                order: Number.isFinite(orderValue) ? orderValue : 0,
            };
        };

        const ensureFontOrder = (fonts) => {
            if (!Array.isArray(fonts)) {
                return [];
            }
            const sorted = [...fonts].sort((a, b) => {
                const aOrder = Number.isFinite(a.order) ? a.order : 0;
                const bOrder = Number.isFinite(b.order) ? b.order : 0;
                if (aOrder === bOrder) {
                    return (a.name || '').localeCompare(b.name || '');
                }
                return aOrder - bOrder;
            });
            sorted.forEach((font, index) => {
                font.order = index + 1;
            });
            return sorted;
        };

        const formatPreconnectSummary = (values) => {
            if (!Array.isArray(values) || !values.length) {
                return 'No preconnect hints';
            }
            return values.join(', ');
        };

        const renderFonts = () => {
            if (!fontList) {
                return;
            }

            fontList
                .querySelectorAll('[data-role="font-item"]')
                .forEach((item) => item.remove());

            const fonts = ensureFontOrder(state.fonts);
            state.fonts = fonts;

            if (!fonts.length) {
                if (fontEmpty) {
                    fontEmpty.hidden = false;
                }
                return;
            }

            if (fontEmpty) {
                fontEmpty.hidden = true;
            }

            fonts.forEach((font, index) => {
                const item = document.createElement('li');
                item.className = 'admin-fonts__item';
                item.dataset.role = 'font-item';
                item.dataset.id = font.id;
                item.dataset.order = String(index + 1);

                const orderColumn = document.createElement('div');
                orderColumn.className = 'admin-fonts__order';

                const orderNumber = document.createElement('span');
                orderNumber.className = 'admin-fonts__order-number';
                orderNumber.textContent = String(index + 1);
                orderColumn.appendChild(orderNumber);

                const orderButtons = document.createElement('div');
                orderButtons.className = 'admin-fonts__order-buttons';

                const moveUpButton = document.createElement('button');
                moveUpButton.type = 'button';
                moveUpButton.className = 'admin-fonts__order-button';
                moveUpButton.dataset.action = 'font-move-up';
                moveUpButton.textContent = '▲';
                moveUpButton.title = 'Move up';
                if (index === 0 || state.isReorderingFonts) {
                    moveUpButton.disabled = true;
                }
                orderButtons.appendChild(moveUpButton);

                const moveDownButton = document.createElement('button');
                moveDownButton.type = 'button';
                moveDownButton.className = 'admin-fonts__order-button';
                moveDownButton.dataset.action = 'font-move-down';
                moveDownButton.textContent = '▼';
                moveDownButton.title = 'Move down';
                if (index === fonts.length - 1 || state.isReorderingFonts) {
                    moveDownButton.disabled = true;
                }
                orderButtons.appendChild(moveDownButton);

                orderColumn.appendChild(orderButtons);
                item.appendChild(orderColumn);

                const details = document.createElement('div');
                details.className = 'admin-fonts__details';

                const name = document.createElement('span');
                name.className = 'admin-fonts__name';
                name.textContent = font.name || 'Font';
                details.appendChild(name);

                const snippet = document.createElement('pre');
                snippet.className = 'admin-fonts__snippet';
                snippet.textContent = font.snippet || '—';
                details.appendChild(snippet);

                const preconnectInfo = document.createElement('p');
                preconnectInfo.className = 'admin-fonts__meta';
                preconnectInfo.textContent = `Preconnect: ${formatPreconnectSummary(font.preconnects)}`;
                details.appendChild(preconnectInfo);

                if (font.notes) {
                    const notes = document.createElement('p');
                    notes.className = 'admin-fonts__meta admin-fonts__meta--notes';
                    notes.textContent = font.notes;
                    details.appendChild(notes);
                }

                item.appendChild(details);

                const controls = document.createElement('div');
                controls.className = 'admin-fonts__controls';

                const toggleLabel = document.createElement('label');
                toggleLabel.className = 'admin-fonts__toggle';

                const toggle = document.createElement('input');
                toggle.type = 'checkbox';
                toggle.dataset.action = 'font-toggle';
                toggle.checked = Boolean(font.enabled);
                toggleLabel.appendChild(toggle);

                const toggleText = document.createElement('span');
                toggleText.textContent = 'Enabled';
                toggleLabel.appendChild(toggleText);

                controls.appendChild(toggleLabel);

                const actions = document.createElement('div');
                actions.className = 'admin-fonts__actions';

                const editButton = document.createElement('button');
                editButton.type = 'button';
                editButton.className = 'admin-fonts__button';
                editButton.dataset.action = 'font-edit';
                editButton.textContent = 'Edit';
                actions.appendChild(editButton);

                const deleteButton = document.createElement('button');
                deleteButton.type = 'button';
                deleteButton.className = 'admin-fonts__button admin-fonts__button--danger';
                deleteButton.dataset.action = 'font-delete';
                deleteButton.textContent = 'Delete';
                actions.appendChild(deleteButton);

                controls.appendChild(actions);
                item.appendChild(controls);

                fontList.appendChild(item);
            });
        };

        const resetFontForm = () => {
            if (!fontForm) {
                return;
            }
            fontForm.reset();
            const idField = fontForm.querySelector('input[name="id"]');
            if (idField) {
                idField.value = '';
            }
            state.editingFontId = '';
            if (fontSubmitButton) {
                fontSubmitButton.textContent = 'Save font';
            }
            if (fontCancelButton) {
                fontCancelButton.hidden = true;
                fontCancelButton.disabled = false;
            }
        };

        const parsePreconnectInput = (value) => {
            if (typeof value !== 'string') {
                return [];
            }
            return value
                .split(/[,\n]+/)
                .map((entry) => entry.trim())
                .filter(Boolean);
        };

        const populatePreconnectInput = (values) => {
            if (!Array.isArray(values) || !fontForm) {
                return '';
            }
            return values.join(', ');
        };

        const startEditFont = (font) => {
            if (!fontForm || !font) {
                return;
            }
            const entry = normaliseFontEntry(font);
            const idField = fontForm.querySelector('input[name="id"]');
            const nameField = fontForm.querySelector('input[name="name"]');
            const snippetField = fontForm.querySelector('textarea[name="snippet"]');
            const preconnectField = fontForm.querySelector('input[name="preconnects"]');
            const enabledField = fontForm.querySelector('input[name="enabled"]');
            const notesField = fontForm.querySelector('textarea[name="notes"]');

            if (idField) {
                idField.value = entry.id;
            }
            if (nameField) {
                nameField.value = entry.name;
            }
            if (snippetField) {
                snippetField.value = entry.snippet;
            }
            if (preconnectField) {
                preconnectField.value = populatePreconnectInput(entry.preconnects);
            }
            if (enabledField) {
                enabledField.checked = Boolean(entry.enabled);
            }
            if (notesField) {
                notesField.value = entry.notes;
            }

            state.editingFontId = entry.id;

            if (fontSubmitButton) {
                fontSubmitButton.textContent = 'Update font';
            }
            if (fontCancelButton) {
                fontCancelButton.hidden = false;
                fontCancelButton.disabled = false;
            }

            bringFormIntoView(fontForm);
        };

        const loadFonts = async () => {
            if (!endpoints.fonts) {
                return;
            }
            try {
                const response = await apiRequest(endpoints.fonts);
                const fonts = Array.isArray(response?.fonts) ? response.fonts : [];
                state.fonts = ensureFontOrder(fonts.map(normaliseFontEntry));
                renderFonts();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const handleFontFormSubmit = async (event) => {
            event.preventDefault();
            if (!fontForm || !endpoints.fonts) {
                return;
            }

            const nameField = fontForm.querySelector('input[name="name"]');
            const snippetField = fontForm.querySelector('textarea[name="snippet"]');
            const preconnectField = fontForm.querySelector('input[name="preconnects"]');
            const enabledField = fontForm.querySelector('input[name="enabled"]');
            const notesField = fontForm.querySelector('textarea[name="notes"]');

            const name = nameField ? nameField.value.trim() : '';
            const snippet = snippetField ? snippetField.value.trim() : '';
            const preconnects = parsePreconnectInput(preconnectField?.value || '');
            const enabled = enabledField ? Boolean(enabledField.checked) : true;
            const notes = notesField ? notesField.value.trim() : '';

            if (!snippet) {
                showAlert('Please provide the font embed code.', 'error');
                focusFirstField(fontForm);
                return;
            }

            const payload = {
                name: name || 'Font',
                snippet,
                preconnects,
                enabled,
                notes,
            };

            const isEditing = Boolean(state.editingFontId);
            const endpoint = isEditing
                ? `${endpoints.fonts}/${state.editingFontId}`
                : endpoints.fonts;
            const method = isEditing ? 'PUT' : 'POST';

            disableForm(fontForm, true);
            clearAlert();

            try {
                await apiRequest(endpoint, {
                    method,
                    body: JSON.stringify(payload),
                });
                await loadFonts();
                showAlert(
                    isEditing ? 'Font updated successfully.' : 'Font added successfully.',
                    'success'
                );
                resetFontForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(fontForm, false);
            }
        };

        const handleFontCancelEdit = () => {
            resetFontForm();
        };

        const persistFontOrder = async () => {
            if (!endpoints.fonts) {
                return;
            }
            const items = ensureFontOrder(state.fonts).map((font, index) => ({
                id: font.id,
                order: index + 1,
            }));
            await apiRequest(`${endpoints.fonts}/reorder`, {
                method: 'PUT',
                body: JSON.stringify({ items }),
            });
        };

        const moveFont = async (id, direction) => {
            if (!Array.isArray(state.fonts) || state.isReorderingFonts) {
                return;
            }

            const index = state.fonts.findIndex((font) => String(font.id) === String(id));
            if (index === -1) {
                return;
            }

            const targetIndex = direction === 'up' ? index - 1 : index + 1;
            if (targetIndex < 0 || targetIndex >= state.fonts.length) {
                return;
            }

            state.isReorderingFonts = true;

            const fonts = [...state.fonts];
            const [moved] = fonts.splice(index, 1);
            fonts.splice(targetIndex, 0, moved);
            state.fonts = ensureFontOrder(fonts);
            renderFonts();

            try {
                await persistFontOrder();
                showAlert('Font order updated.', 'success');
            } catch (error) {
                handleRequestError(error);
                await loadFonts();
            } finally {
                state.isReorderingFonts = false;
            }
        };

        const handleFontListClick = async (event) => {
            const actionButton = event.target?.closest('[data-action]');
            if (!actionButton || !fontList || !endpoints.fonts) {
                return;
            }

            const listItem = actionButton.closest('[data-role="font-item"]');
            const id = listItem?.dataset?.id;
            if (!id) {
                return;
            }

            const action = actionButton.dataset.action;
            if (action === 'font-edit') {
                const font = state.fonts.find((entry) => String(entry.id) === String(id));
                if (font) {
                    startEditFont(font);
                }
                return;
            }

            if (action === 'font-delete') {
                if (!window.confirm('Delete this font?')) {
                    return;
                }
                disableForm(fontForm, true);
                clearAlert();
                try {
                    await apiRequest(`${endpoints.fonts}/${id}`, {
                        method: 'DELETE',
                    });
                    showAlert('Font deleted.', 'success');
                    if (state.editingFontId === id) {
                        resetFontForm();
                    }
                    await loadFonts();
                } catch (error) {
                    handleRequestError(error);
                } finally {
                    disableForm(fontForm, false);
                }
                return;
            }

            if (action === 'font-move-up') {
                await moveFont(id, 'up');
                return;
            }

            if (action === 'font-move-down') {
                await moveFont(id, 'down');
            }
        };

        const handleFontListChange = async (event) => {
            const checkbox = event.target?.closest('input[data-action="font-toggle"]');
            if (!checkbox || !endpoints.fonts) {
                return;
            }

            const listItem = checkbox.closest('[data-role="font-item"]');
            const id = listItem?.dataset?.id;
            if (!id) {
                return;
            }

            checkbox.disabled = true;
            clearAlert();
            const previous = !checkbox.checked;

            try {
                await apiRequest(`${endpoints.fonts}/${id}`, {
                    method: 'PUT',
                    body: JSON.stringify({ enabled: Boolean(checkbox.checked) }),
                });
                const font = state.fonts.find((entry) => String(entry.id) === String(id));
                if (font) {
                    font.enabled = Boolean(checkbox.checked);
                }
                showAlert(
                    checkbox.checked ? 'Font enabled.' : 'Font disabled.',
                    'success'
                );
            } catch (error) {
                checkbox.checked = previous;
                handleRequestError(error);
            } finally {
                checkbox.disabled = false;
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
            const raw = typeof value === 'string' ? value.trim().toLowerCase() : '';
            if (!raw) {
                return 'header';
            }
            if (raw === 'footer') {
                return raw;
            }
            if (raw.startsWith('footer')) {
                const suffix = raw
                    .slice('footer'.length)
                    .replace(/^[\s:._-]+/, '')
                    .replace(/[\s._]+/g, '-');
                return suffix ? `footer:${suffix}` : 'footer';
            }
            return raw;
        };

        const getMenuItemLocation = (item) => {
            if (!item || typeof item !== 'object') {
                return 'header';
            }
            const locationValue = item.location ?? item.Location ?? '';
            return normaliseMenuLocation(locationValue);
        };

        const getActiveMenuLocation = () => {
            const location = normaliseMenuLocation(state.activeMenuLocation);
            if (location === CUSTOM_FOOTER_OPTION) {
                return 'header';
            }
            return location;
        };

        const menuLocationLabels = {
            header: 'Header',
            footer: 'Footer',
            'footer:explore': 'Footer – Explore',
            'footer:account': 'Footer – Account',
            'footer:legal': 'Footer – Legal',
        };

        const formatMenuLocationLabel = (value) => {
            const location = normaliseMenuLocation(value);
            if (Object.prototype.hasOwnProperty.call(menuLocationLabels, location)) {
                return menuLocationLabels[location];
            }
            if (location.startsWith('footer:')) {
                const suffix = location.slice('footer:'.length);
                const words = suffix
                    .split(/[-_\s/]+/)
                    .filter(Boolean)
                    .map((word) => {
                        if (!word.length) {
                            return word;
                        }
                        const first = word.charAt(0).toUpperCase();
                        const rest = word.slice(1).toLowerCase();
                        return `${first}${rest}`;
                    });
                if (words.length) {
                    return `Footer – ${words.join(' ')}`;
                }
                return 'Footer';
            }
            return location === 'footer' ? 'Footer' : 'Header';
        };

        const slugifyFooterSectionName = (value) => {
            if (typeof value !== 'string') {
                return '';
            }
            const trimmed = value.trim();
            if (!trimmed) {
                return '';
            }
            let normalised = trimmed;
            if (typeof trimmed.normalize === 'function') {
                normalised = trimmed
                    .normalize('NFKD')
                    .replace(/\p{M}+/gu, '')
                    .toLowerCase();
            } else {
                normalised = trimmed.toLowerCase();
            }
            let sanitized;
            try {
                sanitized = normalised.replace(/[^\p{L}\p{N}]+/gu, ' ');
            } catch (error) {
                sanitized = normalised.replace(/[^a-z0-9]+/gi, ' ');
            }
            return sanitized
                .trim()
                .split(/\s+/)
                .filter(Boolean)
                .join('-');
        };

        const ensureMenuLocationsInitialised = () => {
            if (!(state.menuLocations instanceof Set)) {
                state.menuLocations = new Set(defaultMenuLocationValues);
            }
        };

        const updateMenuLocationOptions = () => {
            if (!menuLocationField) {
                return;
            }
            ensureMenuLocationsInitialised();

            const previousValue = menuLocationField.value;
            const previousNormalised =
                previousValue && previousValue !== CUSTOM_FOOTER_OPTION
                    ? normaliseMenuLocation(previousValue)
                    : previousValue;

            const fragment = document.createDocumentFragment();
            const seen = new Set();

            const appendOption = (value) => {
                if (!value || seen.has(value)) {
                    return;
                }
                seen.add(value);
                const option = document.createElement('option');
                option.value = value;
                option.textContent = formatMenuLocationLabel(value);
                fragment.appendChild(option);
            };

            defaultMenuLocationValues.forEach((value) => {
                if (state.menuLocations.has(value)) {
                    appendOption(value);
                }
            });

            const extras = Array.from(state.menuLocations).filter(
                (value) => !seen.has(value)
            );
            extras.sort((a, b) => {
                const labelA = formatMenuLocationLabel(a).toLowerCase();
                const labelB = formatMenuLocationLabel(b).toLowerCase();
                if (labelA === labelB) {
                    return a.localeCompare(b);
                }
                return labelA.localeCompare(labelB);
            });
            extras.forEach(appendOption);

            const customOption = document.createElement('option');
            customOption.value = CUSTOM_FOOTER_OPTION;
            customOption.textContent = 'Create new footer section…';
            fragment.appendChild(customOption);

            menuLocationField.innerHTML = '';
            menuLocationField.appendChild(fragment);

            if (
                previousValue === CUSTOM_FOOTER_OPTION ||
                previousNormalised === CUSTOM_FOOTER_OPTION
            ) {
                menuLocationField.value = CUSTOM_FOOTER_OPTION;
                return;
            }

            if (
                previousNormalised &&
                previousNormalised !== CUSTOM_FOOTER_OPTION &&
                state.menuLocations.has(previousNormalised)
            ) {
                menuLocationField.value = previousNormalised;
                return;
            }

            const activeLocation = getActiveMenuLocation();
            if (state.menuLocations.has(activeLocation)) {
                menuLocationField.value = activeLocation;
                return;
            }

            const fallback = defaultMenuLocationValues.find((value) =>
                state.menuLocations.has(value)
            );
            if (fallback) {
                menuLocationField.value = fallback;
            }
        };

        const ensureMenuLocation = (location) => {
            const normalised = normaliseMenuLocation(location);
            if (!normalised || normalised === CUSTOM_FOOTER_OPTION) {
                return;
            }
            ensureMenuLocationsInitialised();
            if (!state.menuLocations.has(normalised)) {
                state.menuLocations.add(normalised);
                updateMenuLocationOptions();
            }
        };

        const toggleCustomFooterLocation = (visible) => {
            const shouldShow = Boolean(visible);
            if (menuCustomLocationContainer) {
                menuCustomLocationContainer.hidden = !shouldShow;
            }
            if (menuCustomLocationHint) {
                menuCustomLocationHint.hidden = shouldShow;
            }
            if (!shouldShow && menuCustomLocationInput) {
                menuCustomLocationInput.value = '';
            }
            if (shouldShow) {
                menuCustomLocationInput?.focus();
            }
        };

        updateMenuLocationOptions();

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
                if (
                    menuLocationField &&
                    menuLocationField.value !== CUSTOM_FOOTER_OPTION
                ) {
                    menuLocationField.value = activeLocation;
                }
                return;
            }

            if (menuEmpty) {
                menuEmpty.hidden = true;
            }
            if (
                menuLocationField &&
                menuLocationField.value !== CUSTOM_FOOTER_OPTION
            ) {
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
            toggleCustomFooterLocation(false);
            if (menuLocationField) {
                updateMenuLocationOptions();
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
            ensureMenuLocation(location);
            if (menuLocationField) {
                toggleCustomFooterLocation(false);
                updateMenuLocationOptions();
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
                const availableLocations = new Set(defaultMenuLocationValues);
                items.forEach((entry) => {
                    const location = getMenuItemLocation(entry);
                    if (location) {
                        availableLocations.add(location);
                    }
                });

                state.menuLocations = availableLocations;

                const currentLocation = getActiveMenuLocation();
                if (!availableLocations.has(currentLocation)) {
                    let fallbackLocation = 'header';
                    if (!availableLocations.has(fallbackLocation)) {
                        const iterator = availableLocations.values();
                        let next = iterator.next();
                        while (!next.done) {
                            const candidate = next.value;
                            if (candidate) {
                                fallbackLocation = candidate;
                                break;
                            }
                            next = iterator.next();
                        }
                    }
                    state.activeMenuLocation = normaliseMenuLocation(
                        fallbackLocation
                    );
                }

                updateMenuLocationOptions();
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

            let location = getActiveMenuLocation();
            if (menuLocationField) {
                const selectedLocation = menuLocationField.value;
                if (selectedLocation === CUSTOM_FOOTER_OPTION) {
                    const customName = menuCustomLocationInput
                        ? menuCustomLocationInput.value.trim()
                        : '';
                    if (!customName) {
                        showAlert(
                            'Please provide a name for the new footer section.',
                            'error'
                        );
                        menuCustomLocationInput?.focus();
                        return;
                    }
                    const slug = slugifyFooterSectionName(customName);
                    if (!slug) {
                        showAlert(
                            'Footer section names must include letters or numbers.',
                            'error'
                        );
                        menuCustomLocationInput?.focus();
                        return;
                    }
                    location = normaliseMenuLocation(`footer:${slug}`);
                } else {
                    location = normaliseMenuLocation(selectedLocation);
                }
            }

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
                ensureMenuLocation(location);
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
            const rawValue = event?.target?.value || '';
            if (rawValue === CUSTOM_FOOTER_OPTION) {
                toggleCustomFooterLocation(true);
                return;
            }
            toggleCustomFooterLocation(false);
            const selected = normaliseMenuLocation(rawValue);
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

        const handleUserSubmit = async (event) => {
            event.preventDefault();
            if (!userForm || !endpoints.users) {
                return;
            }
            const id = userForm.dataset.id;
            if (!id) {
                showAlert('Select a user to update first.', 'info');
                return;
            }
            const user = state.users.find(
                (entry) => extractUserId(entry) === id
            );
            const roleValue = userRoleField?.value?.trim() || '';
            const statusValue = userStatusField?.value?.trim() || '';
            const updates = [];
            if (roleValue) {
                const currentRole = normaliseString(user?.role || user?.Role || '');
                if (currentRole !== normaliseString(roleValue)) {
                    updates.push(
                        apiRequest(buildUserEndpoint(id, 'role'), {
                            method: 'PUT',
                            body: JSON.stringify({ role: roleValue }),
                        })
                    );
                }
            }
            if (statusValue) {
                const currentStatus = normaliseString(
                    user?.status || user?.Status || ''
                );
                if (currentStatus !== normaliseString(statusValue)) {
                    updates.push(
                        apiRequest(buildUserEndpoint(id, 'status'), {
                            method: 'PUT',
                            body: JSON.stringify({ status: statusValue }),
                        })
                    );
                }
            }
            if (!updates.length) {
                showAlert('No changes to save for this user.', 'info');
                return;
            }
            setUserFormEnabled(false);
            if (userSubmitButton) {
                userSubmitButton.disabled = true;
                userSubmitButton.textContent = 'Saving…';
            }
            clearAlert();
            try {
                await Promise.all(updates);
                showAlert('User updated successfully.', 'success');
                await loadUsers();
                selectUser(id);
                await loadStats();
            } catch (error) {
                handleRequestError(error);
            } finally {
                const hasSelection = Boolean(userForm?.dataset.id);
                setUserFormEnabled(hasSelection);
                if (userSubmitButton) {
                    userSubmitButton.textContent = 'Update user';
                    userSubmitButton.disabled = !hasSelection;
                }
                if (userDeleteButton) {
                    const isSelf =
                        currentUserId && userForm?.dataset.id === currentUserId;
                    userDeleteButton.hidden = !hasSelection || Boolean(isSelf);
                    userDeleteButton.disabled = !hasSelection || Boolean(isSelf);
                }
            }
        };

        const handleUserDelete = async () => {
            if (!userForm || !endpoints.users) {
                return;
            }
            const id = userForm.dataset.id;
            if (!id) {
                return;
            }
            if (currentUserId && id === currentUserId) {
                showAlert('You cannot delete your own account from the admin dashboard.', 'error');
                return;
            }
            if (!window.confirm('Delete this user permanently? This action cannot be undone.')) {
                return;
            }
            setUserFormEnabled(false);
            if (userDeleteButton) {
                userDeleteButton.disabled = true;
            }
            clearAlert();
            try {
                await apiRequest(buildUserEndpoint(id), {
                    method: 'DELETE',
                });
                showAlert('User deleted successfully.', 'success');
                await loadUsers();
                await loadStats();
            } catch (error) {
                handleRequestError(error);
                if (userForm.dataset.id === id) {
                    setUserFormEnabled(true);
                    if (userDeleteButton) {
                        userDeleteButton.disabled = false;
                    }
                }
            }
        };

        const parseBackupCounts = (header) => {
            if (!header || typeof header !== 'string') {
                return {};
            }
            return header.split(';').reduce((accumulator, part) => {
                const [rawKey, rawValue] = part.split('=');
                if (!rawKey || rawValue === undefined) {
                    return accumulator;
                }
                const key = rawKey.trim();
                const value = Number.parseInt(rawValue.trim(), 10);
                if (!Number.isNaN(value)) {
                    accumulator[key] = value;
                }
                return accumulator;
            }, {});
        };

        const handleBackupDownload = async () => {
            if (!backupDownloadButton || !endpoints.backupExport) {
                showAlert('Backup download is not available.', 'error');
                return;
            }
            backupDownloadButton.disabled = true;
            try {
                const response = await authenticatedFetch(endpoints.backupExport, {
                    method: 'GET',
                });
                if (!response.ok) {
                    let message = 'Failed to generate backup.';
                    const contentType = response.headers.get('content-type') || '';
                    if (contentType.includes('application/json')) {
                        const payload = await response.json().catch(() => null);
                        if (payload && typeof payload === 'object' && payload.error) {
                            message = payload.error;
                        }
                    } else {
                        const text = await response.text();
                        if (text) {
                            message = text;
                        }
                    }
                    const error = new Error(message);
                    error.status = response.status;
                    throw error;
                }

                const blob = await response.blob();
                let filename = parseContentDispositionFilename(
                    response.headers.get('content-disposition')
                );
                if (!filename) {
                    const generatedAtHeader = response.headers.get(
                        'x-backup-generated-at'
                    );
                    if (generatedAtHeader) {
                        filename = `backup-${generatedAtHeader.replace(/[:]/g, '-')}.zip`;
                    } else {
                        filename = 'backup.zip';
                    }
                }

                const downloadUrl = window.URL.createObjectURL(blob);
                const link = document.createElement('a');
                link.href = downloadUrl;
                link.download = filename;
                document.body.appendChild(link);
                link.click();
                link.remove();
                window.URL.revokeObjectURL(downloadUrl);

                showAlert('Backup downloaded successfully.', 'success');

                const generatedAt = response.headers.get('x-backup-generated-at');
                const schema = response.headers.get('x-backup-schema');
                const countsHeader = response.headers.get('x-backup-counts');
                const counts = parseBackupCounts(countsHeader);
                const summaryParts = [];
                if (generatedAt) {
                    summaryParts.push(`Generated ${formatDate(generatedAt)}`);
                }
                if (schema) {
                    summaryParts.push(`Schema ${schema}`);
                }
                const highlight = [];
                if (typeof counts.posts === 'number') {
                    highlight.push(`${counts.posts} posts`);
                }
                if (typeof counts.pages === 'number') {
                    highlight.push(`${counts.pages} pages`);
                }
                if (typeof counts.uploads === 'number') {
                    highlight.push(`${counts.uploads} uploads`);
                }
                if (highlight.length > 0) {
                    summaryParts.push(highlight.join(', '));
                }
                updateBackupSummary(summaryParts.join(' · '));
            } catch (error) {
                handleRequestError(error);
            } finally {
                backupDownloadButton.disabled = false;
            }
        };

        const handleBackupImport = async (event) => {
            event.preventDefault();
            if (!backupImportForm || !endpoints.backupImport) {
                showAlert('Backup restore is not available.', 'error');
                return;
            }

            if (!backupUploadInput || backupUploadInput.files.length === 0) {
                showAlert('Select a backup archive to upload.', 'error');
                return;
            }

            const file = backupUploadInput.files[0];
            const formData = new FormData();
            formData.append('file', file);

            disableForm(backupImportForm, true);
            clearAlert();
            try {
                const payload = await apiRequest(endpoints.backupImport, {
                    method: 'POST',
                    body: formData,
                });
                showAlert('Backup restored successfully.', 'success');
                if (payload && typeof payload === 'object' && payload.summary) {
                    const summary = payload.summary;
                    const parts = [];
                    if (summary.generated_at) {
                        parts.push(`Snapshot ${formatDate(summary.generated_at)}`);
                    }
                    if (summary.restored_at) {
                        parts.push(`Restored ${formatDate(summary.restored_at)}`);
                    }
                    const details = [];
                    if (typeof summary.posts === 'number') {
                        details.push(`${summary.posts} posts`);
                    }
                    if (typeof summary.pages === 'number') {
                        details.push(`${summary.pages} pages`);
                    }
                    if (typeof summary.uploads === 'number') {
                        details.push(`${summary.uploads} uploads`);
                    }
                    if (details.length > 0) {
                        parts.push(details.join(', '));
                    }
                    updateBackupSummary(parts.join(' · '));
                } else {
                    updateBackupSummary('Backup restored successfully.');
                }
                backupImportForm.reset();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(backupImportForm, false);
            }
        };

        const handleBackupAutoToggleChange = () => {
            if (!backupSettingsIntervalInput) {
                return;
            }
            const enabled = Boolean(backupSettingsToggle?.checked);
            backupSettingsIntervalInput.disabled = !enabled;
            if (enabled) {
                backupSettingsIntervalInput.removeAttribute('aria-disabled');
            } else {
                backupSettingsIntervalInput.setAttribute('aria-disabled', 'true');
            }
        };

        const renderBackupSettings = (settings) => {
            if (!backupSettingsForm) {
                return;
            }
            const enabled = Boolean(settings?.enabled);
            const interval = Number.parseInt(settings?.interval_hours, 10);

            if (backupSettingsToggle) {
                backupSettingsToggle.checked = enabled;
            }

            if (backupSettingsIntervalInput) {
                const value = Number.isFinite(interval) && interval > 0 ? String(interval) : '24';
                backupSettingsIntervalInput.value = value;
            }

            if (backupSettingsStatus) {
                const parts = [];
                if (enabled) {
                    parts.push('Automatic backups enabled');
                    if (settings?.next_run) {
                        parts.push(`Next backup ${formatDate(settings.next_run)}`);
                    }
                    if (settings?.last_run) {
                        parts.push(`Last backup ${formatDate(settings.last_run)}`);
                    }
                } else {
                    parts.push('Automatic backups disabled');
                }
                const message = parts.join('. ');
                backupSettingsStatus.textContent = message;
                backupSettingsStatus.hidden = message.length === 0;
            }

            handleBackupAutoToggleChange();
        };

        const loadBackupSettings = async () => {
            if (!backupSettingsForm || !endpoints.backupSettings) {
                return;
            }

            try {
                disableForm(backupSettingsForm, true);
                const payload = await apiRequest(endpoints.backupSettings, {
                    method: 'GET',
                });
                if (payload && typeof payload === 'object' && payload.settings) {
                    renderBackupSettings(payload.settings);
                }
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(backupSettingsForm, false);
                handleBackupAutoToggleChange();
            }
        };

        const handleBackupSettingsSubmit = async (event) => {
            event.preventDefault();
            if (!backupSettingsForm || !endpoints.backupSettings) {
                showAlert('Automatic backup configuration is not available.', 'error');
                return;
            }

            const intervalValue = Number.parseInt(
                backupSettingsIntervalInput?.value || '',
                10
            );
            if (!Number.isFinite(intervalValue) || intervalValue < 1) {
                showAlert('Enter a valid backup interval of at least one hour.', 'error');
                return;
            }
            if (intervalValue > 168) {
                showAlert('Automatic backup interval cannot exceed 168 hours (7 days).', 'error');
                return;
            }

            const payload = {
                enabled: Boolean(backupSettingsToggle?.checked),
                interval_hours: intervalValue,
            };

            const originalText = backupSettingsSubmit?.textContent || '';

            disableForm(backupSettingsForm, true);
            if (backupSettingsSubmit) {
                backupSettingsSubmit.disabled = true;
                backupSettingsSubmit.textContent = 'Saving…';
            }

            clearAlert();
            try {
                const response = await apiRequest(endpoints.backupSettings, {
                    method: 'PUT',
                    body: JSON.stringify(payload),
                });
                showAlert('Backup settings updated successfully.', 'success');
                if (response && typeof response === 'object' && response.settings) {
                    renderBackupSettings(response.settings);
                }
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(backupSettingsForm, false);
                if (backupSettingsSubmit) {
                    backupSettingsSubmit.disabled = false;
                    backupSettingsSubmit.textContent =
                        originalText || 'Save backup settings';
                }
                handleBackupAutoToggleChange();
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

            const isValidAbsoluteUrl = (value) => {
                if (!value) {
                    return true;
                }
                try {
                    const parsed = new URL(value);
                    return Boolean(parsed.protocol && parsed.host);
                } catch (error) {
                    return false;
                }
            };

            const payload = {
                name: getValue('name'),
                description: getValue('description'),
                url: getValue('url'),
                favicon: getValue('favicon'),
                logo: getValue('logo'),
            };

            const stripeSecretKey = getValue('stripe_secret_key');
            const stripePublishableKey = getValue('stripe_publishable_key');
            const stripeWebhookSecret = getValue('stripe_webhook_secret');
            const successUrl = getValue('course_checkout_success_url');
            const cancelUrl = getValue('course_checkout_cancel_url');
            const currencyRaw = getValue('course_checkout_currency');

            if (!isValidAbsoluteUrl(successUrl)) {
                showAlert('Please provide a valid checkout success URL, including the protocol (e.g. https://example.com/success).', 'error');
                return;
            }

            if (!isValidAbsoluteUrl(cancelUrl)) {
                showAlert('Please provide a valid checkout cancel URL, including the protocol (e.g. https://example.com/cancel).', 'error');
                return;
            }

            const defaultLanguageValue = defaultLanguageInput
                ? defaultLanguageInput.value.trim()
                : state.language.default;
            const normalisedDefaultLanguage = normaliseLanguageCode(defaultLanguageValue);
            if (!normalisedDefaultLanguage || !isValidLanguageCode(normalisedDefaultLanguage)) {
                showAlert('Please provide a valid default language code (e.g. "en" or "en-GB").', 'error');
                return;
            }

            setDefaultLanguage(normalisedDefaultLanguage, { silent: true });

            let supportedLanguages = [];
            try {
                supportedLanguages = parseLanguageCodes(supportedLanguagesInput?.value || '');
            } catch (languageError) {
                showAlert(languageError.message || 'Please review the supported languages field.', 'error');
                return;
            }

            payload.default_language = normalisedDefaultLanguage;
            payload.supported_languages = supportedLanguages;

            const retentionField = settingsForm.querySelector('[name="unused_tag_retention_hours"]');
            const retentionRaw = retentionField ? retentionField.value.trim() : '';
            const retentionHours = Number.parseInt(retentionRaw, 10);

            if (Number.isNaN(retentionHours) || retentionHours < 1) {
                showAlert('Please provide how many hours unused tags should be retained (minimum 1 hour).', 'error');
                return;
            }

            payload.unused_tag_retention_hours = retentionHours;

            const normalisedCurrency = currencyRaw ? currencyRaw.toLowerCase() : '';
            if (normalisedCurrency && !/^[a-z]{3}$/.test(normalisedCurrency)) {
                showAlert('Please provide a valid three-letter ISO currency code (for example, usd or eur).', 'error');
                return;
            }

            payload.stripe_secret_key = stripeSecretKey;
            payload.stripe_publishable_key = stripePublishableKey;
            payload.stripe_webhook_secret = stripeWebhookSecret;
            payload.course_checkout_success_url = successUrl;
            payload.course_checkout_cancel_url = cancelUrl;
            payload.course_checkout_currency = normalisedCurrency;

            if (!payload.name) {
                showAlert('Please provide a site name.', 'error');
                return;
            }

            if (!payload.url) {
                showAlert('Please provide the primary site URL.', 'error');
                return;
            }

            disableForm(settingsForm, true);
            disableForm(languageForm, true);
            clearAlert();

            try {
                const response = await apiRequest(endpoints.siteSettings, {
                    method: 'PUT',
                    body: JSON.stringify(payload),
                });
                state.site = response?.site || payload;
                if (!state.site.default_language) {
                    state.site.default_language = normalisedDefaultLanguage;
                }
                if (!Array.isArray(state.site.supported_languages)) {
                    state.site.supported_languages = [normalisedDefaultLanguage, ...supportedLanguages];
                }
                state.language.default = normalisedDefaultLanguage;
                state.language.supported = Array.isArray(state.site.supported_languages)
                    ? [...state.site.supported_languages]
                    : [normalisedDefaultLanguage, ...supportedLanguages];
                populateSiteSettingsForm(state.site);
                showAlert('Site settings updated successfully.', 'success');
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(settingsForm, false);
                disableForm(languageForm, false);
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
            const submitter = event.submitter;
            const intent = submitter?.dataset?.intent;
            let published;
            if (intent === 'draft') {
                published = false;
            } else if (intent === 'publish') {
                published = true;
            } else if (postForm.dataset.published) {
                published = postForm.dataset.published === 'true';
            } else {
                published = true;
            }
            postForm.dataset.published = String(published);
            const payload = {
                title,
                description,
                featured_img: featuredImg,
                content,
                published,
            };
            if (postPublishAtInput) {
                const rawPublishAt = postPublishAtInput.value.trim();
                if (rawPublishAt) {
                    const parsedPublishAt = parseDateInput(rawPublishAt);
                    if (!parsedPublishAt) {
                        showAlert(
                            'Please enter a valid publish date and time.',
                            'error'
                        );
                        return;
                    }
                    payload.publish_at = parsedPublishAt.toISOString();
                } else if (id) {
                    payload.publish_at = null;
                }
            }
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
            const pathValue = pagePathInput
                ? pagePathInput.value.trim()
                : '';
            const orderInput = pageForm.querySelector('input[name="order"]');
            const orderValue = orderInput ? Number(orderInput.value) : 0;
            const hideHeaderField = pageForm.querySelector(
                'input[name="hide_header"]'
            );
            const submitter = event.submitter;
            const intent = submitter?.dataset?.intent;
            let published;
            if (intent === 'draft') {
                published = false;
            } else if (intent === 'publish') {
                published = true;
            } else if (pageForm.dataset.published) {
                published = pageForm.dataset.published === 'true';
            } else {
                published = true;
            }
            pageForm.dataset.published = String(published);
            const payload = {
                title,
                description,
                content,
                order: Number.isNaN(orderValue) ? 0 : orderValue,
                published,
                hide_header: Boolean(hideHeaderField?.checked),
            };
            if (pagePublishAtInput) {
                const rawPublishAt = pagePublishAtInput.value.trim();
                if (rawPublishAt) {
                    const parsedPublishAt = parseDateInput(rawPublishAt);
                    if (!parsedPublishAt) {
                        showAlert(
                            'Please enter a valid publish date and time.',
                            'error'
                        );
                        return;
                    }
                    payload.publish_at = parsedPublishAt.toISOString();
                } else if (id) {
                    payload.publish_at = null;
                }
            }
            if (pagePathInput) {
                payload.path = pathValue;
            }
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

        const buildNavigation = () => {
            if (!navigationContainer) {
                return [];
            }

            const panels = Array.from(root.querySelectorAll('.admin-panel'));
            if (panels.length === 0) {
                navigationContainer.innerHTML = '';
                return [];
            }

            const groups = new Map();

            panels.forEach((panel, index) => {
                const panelKey = panel.dataset.panel;
                if (!panelKey) {
                    return;
                }

                const navLabel = panel.dataset.navLabel || panelKey;
                const navOrder = parseOrder(panel.dataset.navOrder, index);
                const groupKey = panel.dataset.navGroup || 'general';
                const groupLabel = panel.dataset.navGroupLabel || 'General';
                const groupOrder = parseOrder(panel.dataset.navGroupOrder, 0);
                const panelElementId = panel.getAttribute('id') || `admin-panel-${panelKey}`;
                panel.id = panelElementId;
                const isActive =
                    panel.classList.contains('is-active') &&
                    !panel.hasAttribute('hidden');

                if (!groups.has(groupKey)) {
                    groups.set(groupKey, {
                        key: groupKey,
                        label: groupLabel,
                        order: groupOrder,
                        panels: [],
                    });
                }

                const group = groups.get(groupKey);
                group.label = groupLabel;
                group.order = groupOrder;
                group.panels.push({
                    id: panelKey,
                    label: navLabel,
                    order: navOrder,
                    isActive,
                    elementId: panelElementId,
                    panel,
                });
            });

            const sortedGroups = Array.from(groups.values()).sort((a, b) => {
                if (a.order !== b.order) {
                    return a.order - b.order;
                }
                return (a.label || '').localeCompare(b.label || '');
            });

            navigationContainer.innerHTML = '';
            const tabs = [];

            sortedGroups.forEach((group, index) => {
                group.panels.sort((a, b) => {
                    if (a.order !== b.order) {
                        return a.order - b.order;
                    }
                    return a.label.localeCompare(b.label);
                });

                const groupElement = createElement('div', {
                    className: 'admin__nav-group',
                });

                const identifier =
                    typeof group.key === 'string' && group.key
                        ? group.key.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '')
                        : `group-${index + 1}`;
                const headingId = `admin-nav-group-${identifier || index + 1}`;
                const heading = createElement('h3', {
                    className: 'admin__nav-title',
                    textContent: group.label || 'Sections',
                });
                heading.id = headingId;
                groupElement.appendChild(heading);

                const tabList = createElement('div', {
                    className: 'admin__tabs',
                });
                tabList.setAttribute('role', 'tablist');
                tabList.setAttribute('aria-labelledby', headingId);
                tabList.setAttribute('aria-orientation', 'vertical');

                group.panels.forEach((panelMeta) => {
                    const tab = createElement('button', {
                        className: 'admin__tab',
                        textContent: panelMeta.label,
                    });
                    const tabId = `admin-tab-${panelMeta.id}`;
                    tab.type = 'button';
                    tab.dataset.tab = panelMeta.id;
                    tab.id = tabId;
                    tab.setAttribute('role', 'tab');
                    tab.setAttribute('aria-controls', panelMeta.elementId);
                    tab.setAttribute('aria-selected', String(Boolean(panelMeta.isActive)));
                    tab.classList.toggle('is-active', panelMeta.isActive);
                    panelMeta.panel.setAttribute('aria-labelledby', tabId);
                    tabList.appendChild(tab);
                    tabs.push(tab);
                });

                groupElement.appendChild(tabList);
                navigationContainer.appendChild(groupElement);
            });

            return tabs;
        };

        const activateTab = (targetId) => {
            if (!targetId) {
                setStoredActiveTab('');
                return;
            }
            const targetPanel = root.querySelector(
                `.admin-panel[data-panel="${targetId}"]`
            );
            if (!targetPanel) {
                setStoredActiveTab('');
                return;
            }
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
            setStoredActiveTab(targetId);
        };

        let navigationTabs = [];

        const refreshNavigation = () => {
            navigationTabs = buildNavigation() || [];

            if (navigationTabs.length === 0) {
                setStoredActiveTab('');
                return;
            }

            let initialTabActivated = false;
            const storedActiveTab = getStoredActiveTab();
            if (storedActiveTab) {
                const storedTab = navigationTabs.find(
                    (tab) => tab.dataset.tab === storedActiveTab
                );
                if (storedTab) {
                    activateTab(storedActiveTab);
                    initialTabActivated = true;
                } else {
                    setStoredActiveTab('');
                }
            }

            if (
                navigationTabs.length > 0 &&
                !navigationTabs.some((tab) => tab.classList.contains('is-active')) &&
                !initialTabActivated
            ) {
                const defaultTab = navigationTabs[0].dataset.tab;
                if (defaultTab) {
                    activateTab(defaultTab);
                }
            }
        };

        refreshNavigation();

        navigationContainer?.addEventListener('click', (event) => {
            const target = event.target;
            if (!(target instanceof Element)) {
                return;
            }
            const tab = target.closest('.admin__tab[data-tab]');
            if (!tab || !navigationContainer.contains(tab)) {
                return;
            }
            event.preventDefault();
            const tabId = tab.dataset.tab;
            if (tabId) {
                activateTab(tabId);
            }
        });

        const quickActionsContainer = root.querySelector('[data-role="admin-quick-actions"]');
        quickActionsContainer?.addEventListener('click', (event) => {
            const target = event.target;
            if (!(target instanceof Element)) {
                return;
            }
            const button = target.closest('[data-nav-target]');
            if (!button || !quickActionsContainer.contains(button)) {
                return;
            }
            event.preventDefault();
            const targetId = button.dataset.navTarget;
            if (targetId) {
                activateTab(targetId);
                const targetPanel = root.querySelector(
                    `.admin-panel[data-panel="${targetId}"]`
                );
                if (targetPanel && typeof targetPanel.scrollIntoView === 'function') {
                    targetPanel.scrollIntoView({ behavior: 'smooth', block: 'start' });
                }
            }
            const actionId = button.dataset.panelAction;
            if (actionId) {
                const actionButton = root.querySelector(
                    `[data-action="${actionId}"]`
                );
                actionButton?.click();
            }
            button.blur();
        });

        root.addEventListener('admin:panels-changed', () => {
            refreshNavigation();
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
        root.querySelector('[data-action="user-reset"]')?.addEventListener(
            'click',
            resetUserForm
        );
        root.querySelector('[data-action="course-video-reset"]')?.addEventListener(
            'click',
            resetCourseVideoForm
        );
        root.querySelector('[data-action="course-topic-reset"]')?.addEventListener(
            'click',
            resetCourseTopicForm
        );
        root.querySelector('[data-action="course-package-reset"]')?.addEventListener(
            'click',
            resetCoursePackageForm
        );

        resetUserForm();
        resetCourseVideoForm();
        resetCourseTopicForm();
        resetCoursePackageForm();

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
        attachSearchHandler(userSearchInput, setUserSearchQuery);

        if (postSearchInput?.value) {
            setPostSearchQuery(postSearchInput.value);
        }
        if (pageSearchInput?.value) {
            setPageSearchQuery(pageSearchInput.value);
        }
        if (categorySearchInput?.value) {
            setCategorySearchQuery(categorySearchInput.value);
        }
        if (userSearchInput?.value) {
            setUserSearchQuery(userSearchInput.value);
        }

        postForm?.addEventListener('submit', handlePostSubmit);
        postDeleteButton?.addEventListener('click', handlePostDelete);
        pageForm?.addEventListener('submit', handlePageSubmit);
        pageDeleteButton?.addEventListener('click', handlePageDelete);
        categoryForm?.addEventListener('submit', handleCategorySubmit);
        categoryDeleteButton?.addEventListener('click', handleCategoryDelete);
        userForm?.addEventListener('submit', handleUserSubmit);
        userDeleteButton?.addEventListener('click', handleUserDelete);
        courseVideoForm?.addEventListener('submit', handleCourseVideoSubmit);
        courseVideoDeleteButton?.addEventListener('click', handleCourseVideoDelete);
        courseTopicForm?.addEventListener('submit', handleCourseTopicSubmit);
        courseTopicDeleteButton?.addEventListener('click', handleCourseTopicDelete);
        courseTopicVideoAddButton?.addEventListener(
            'click',
            handleCourseTopicVideoAdd
        );
        courseTopicVideoList?.addEventListener(
            'click',
            handleCourseTopicVideoListClick
        );
        coursePackageForm?.addEventListener('submit', handleCoursePackageSubmit);
        coursePackageDeleteButton?.addEventListener(
            'click',
            handleCoursePackageDelete
        );
        coursePackageTopicAddButton?.addEventListener(
            'click',
            handleCoursePackageTopicAdd
        );
        coursePackageTopicList?.addEventListener(
            'click',
            handleCoursePackageTopicListClick
        );
        backupDownloadButton?.addEventListener('click', handleBackupDownload);
        backupImportForm?.addEventListener('submit', handleBackupImport);
        backupSettingsForm?.addEventListener('submit', handleBackupSettingsSubmit);
        backupSettingsToggle?.addEventListener('change', handleBackupAutoToggleChange);
        supportedLanguagesAddButton?.addEventListener('click', handleLanguageAdd);
        supportedLanguagesAddInput?.addEventListener('keydown', handleLanguageInputKeydown);
        if (supportedLanguagesList) {
            supportedLanguagesList.addEventListener('click', (event) => {
                const target = event.target;
                if (!(target instanceof Element)) {
                    return;
                }
                const actionButton = target.closest('button[data-action]');
                if (!actionButton) {
                    return;
                }
                const code = actionButton.dataset.code;
                if (!code) {
                    return;
                }
                const action = actionButton.dataset.action;
                if (action === 'language-default') {
                    setDefaultLanguage(code);
                } else if (action === 'language-remove') {
                    removeSupportedLanguage(code);
                }
            });
        }
        defaultLanguageInput?.addEventListener('blur', handleDefaultLanguageBlur);
        defaultLanguageInput?.addEventListener('change', handleDefaultLanguageBlur);
        defaultLanguageInput?.addEventListener('input', () => {
            defaultLanguageInput.setCustomValidity('');
        });
        settingsForm?.addEventListener('submit', handleSiteSettingsSubmit);
        languageForm?.addEventListener('submit', handleSiteSettingsSubmit);
        advertisingForm?.addEventListener('submit', handleAdvertisingSubmit);
        advertisingProviderSelect?.addEventListener('change', handleAdvertisingProviderChange);
        advertisingEnabledToggle?.addEventListener('change', handleAdvertisingEnabledChange);
        advertisingPublisherInput?.addEventListener('input', handleAdvertisingPublisherInput);
        advertisingAutoToggle?.addEventListener('change', handleAdvertisingAutoChange);
        advertisingSlotAddButton?.addEventListener('click', handleAdvertisingAddSlot);
        advertisingSlotsContainer?.addEventListener('input', handleAdvertisingSlotChange);
        advertisingSlotsContainer?.addEventListener('change', handleAdvertisingSlotChange);
        advertisingSlotsContainer?.addEventListener('click', handleAdvertisingSlotClick);
        faviconUploadButton?.addEventListener('click', handleFaviconUploadClick);
        faviconUploadInput?.addEventListener('change', handleFaviconFileChange);
        logoUploadButton?.addEventListener('click', handleLogoUploadClick);
        logoUploadInput?.addEventListener('change', handleLogoFileChange);
        pluginList?.addEventListener('click', handlePluginListClick);
        pluginInstallForm?.addEventListener('submit', handlePluginInstallSubmit);
        themeList?.addEventListener('click', handleThemeListClick);
        socialForm?.addEventListener('submit', handleSocialFormSubmit);
        socialCancelButton?.addEventListener('click', handleSocialCancelEdit);
        socialList?.addEventListener('click', handleSocialListClick);
        fontForm?.addEventListener('submit', handleFontFormSubmit);
        fontCancelButton?.addEventListener('click', handleFontCancelEdit);
        fontList?.addEventListener('click', handleFontListClick);
        fontList?.addEventListener('change', handleFontListChange);
        menuForm?.addEventListener('submit', handleMenuFormSubmit);
        menuCancelButton?.addEventListener('click', handleMenuCancelEdit);
        menuLocationField?.addEventListener('change', handleMenuLocationChange);
        menuList?.addEventListener('click', handleMenuListClick);
        postTagsInput?.addEventListener('input', renderTagSuggestions);
        homepageForm?.addEventListener('submit', handleHomepageSubmit);

        if (languageForm) {
            renderLanguageManager();
        }

        updateHomepageStatus();
        handleBackupAutoToggleChange();
        showPostAnalyticsEmpty();
        clearAlert();
        renderMetricsChart(state.activityTrend);
        loadStats();
        loadTags();
        const loadCourseData = async () => {
            await loadCourseVideos();
            await loadCourseTopics();
            await loadCoursePackages();
        };
        if (
            endpoints.coursesVideos ||
            endpoints.coursesTopics ||
            endpoints.coursesPackages
        ) {
            loadCourseData();
        }
        loadCategories().then(() => {
            renderCategoryOptions();
            loadPosts();
        });
        loadPages();
        loadComments();
        loadUsers();
        loadBackupSettings();
        loadSiteSettings();
        loadHomepageSettings();
        loadAdvertisingSettings();
        loadPlugins();
        loadThemes();
        loadSocialLinks();
        loadFonts();
        loadMenuItems();
    };

    const layoutManager = window.AdminLayout;
    if (layoutManager && typeof layoutManager.whenReady === 'function') {
        layoutManager.whenReady(() => {
            initialiseAdminDashboard();
        });
    } else if (document.readyState === 'loading') {
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

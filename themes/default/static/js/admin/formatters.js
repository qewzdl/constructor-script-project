/**
 * Admin Data Formatters Module
 * Utilities for formatting dates, publication statuses, percentages, and content previews
 */
(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;

    if (!utils || !registry) {
        console.error('AdminFormatters: dependencies are missing.');
        return;
    }

    const { formatDate } = utils;
    const elementDefinitions = registry.getDefinitions();

    /**
     * Coerce various input types into a valid Date object or null
     */
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

    /**
     * Extract date value from entry using provided keys
     */
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

    /**
     * Format publication status with appropriate label
     */
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

    /**
     * Describe publication details
     */
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

    /**
     * Format percentage value
     */
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

    /**
     * Generate content preview from sections
     */
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

    /**
     * Parse order value
     */
    const parseOrder = (value, fallback = 0) => {
        const parsed = Number(value);
        return Number.isFinite(parsed) ? parsed : fallback;
    };

    // Export to global namespace
    window.AdminFormatters = {
        coerceDateValue,
        extractDateValue,
        formatPublicationStatus,
        describePublication,
        formatPercentage,
        generateContentPreview,
        parseOrder,
    };
})();

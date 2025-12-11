(() => {
    if (typeof window.AdminSectionRegistry === 'undefined') {
        console.error('AdminSectionRegistry not found. Make sure registry.js is loaded first.');
        return;
    }

    window.AdminSectionRegistry.register('hero', {
        label: 'Hero section',
        order: 5,
        supportsElements: false,
        description: 'Prominent banner with title, subtitle, image, and call-to-action button.',
        settings: {
            title: {
                label: 'Hero title',
                type: 'text',
                required: true,
                placeholder: 'Welcome to Our Platform',
            },
            subtitle: {
                label: 'Subtitle',
                type: 'textarea',
                placeholder: 'Discover amazing features and possibilities',
            },
            image_url: {
                label: 'Hero image URL',
                type: 'url',
                required: true,
                placeholder: 'https://example.com/hero-image.jpg',
                allowMediaBrowse: true,
            },
            image_alt: {
                label: 'Image alt text',
                type: 'text',
                placeholder: 'Hero image',
            },
            button_text: {
                label: 'Button text',
                type: 'text',
                placeholder: 'Get started',
            },
            button_url: {
                label: 'Button URL',
                type: 'url',
                required: true,
                placeholder: '/',
                allowAnchorPicker: true,
            },
        },
    });
})();

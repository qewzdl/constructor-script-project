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
        variations: [
            {
                id: 'hero-split',
                value: 'split',
                label: 'Split',
                description: 'Content and image side by side',
                isDefault: true,
            },
            {
                id: 'hero-centered',
                value: 'centered',
                label: 'Centered',
                description: 'Centered hero with optional image below',
            },
            {
                id: 'hero-minimal',
                value: 'minimal',
                label: 'Minimal',
                description: 'Compact hero focused on text and action',
            },
            {
                id: 'hero-immersive',
                value: 'immersive',
                label: 'Immersive',
                description: 'Atmospheric hero with layered overlay and glowing accent',
            },
        ],
        settings: {
            title: {
                label: 'Hero title',
                type: 'text',
                required: true,
                placeholder: 'Welcome to Our Platform',
            },
            subtitle: {
                label: 'Subtitle / badge text',
                type: 'text',
                placeholder: 'Discover amazing features and possibilities',
            },
            text: {
                label: 'Description text',
                type: 'textarea',
                placeholder: 'Additional descriptive text for your hero section',
            },
            image_url: {
                label: 'Hero image URL',
                type: 'url',
                required: true,
                placeholder: 'https://example.com/hero-image.jpg',
                allowMediaBrowse: true,
                hiddenForVariations: ['immersive'],
            },
            image_alt: {
                label: 'Image alt text',
                type: 'text',
                placeholder: 'Hero image',
                hiddenForVariations: ['immersive'],
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
            button_icon: {
                label: 'Primary button icon',
                type: 'text',
                placeholder: '✨',
                hiddenForVariations: ['split', 'centered', 'minimal'],
            },
            secondary_button_text: {
                label: 'Secondary button text',
                type: 'text',
                placeholder: 'Learn more',
            },
            secondary_button_url: {
                label: 'Secondary button URL',
                type: 'url',
                placeholder: '/',
                allowAnchorPicker: true,
            },
            secondary_button_icon: {
                label: 'Secondary button icon',
                type: 'text',
                placeholder: '→',
                hiddenForVariations: ['split', 'centered', 'minimal'],
            },
        },
    });
})();

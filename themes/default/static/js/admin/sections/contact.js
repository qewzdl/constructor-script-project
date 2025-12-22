(() => {
    if (typeof window.AdminSectionRegistry === 'undefined') {
        console.error('AdminSectionRegistry not found. Make sure registry.js is loaded first.');
        return;
    }

    window.AdminSectionRegistry.register('contact', {
        label: 'Contact',
        order: 14,
        supportsElements: false,
        description: 'Contact methods paired with a short inquiry form.',
        settings: {
            email: {
                label: 'Contact email',
                type: 'text',
                placeholder: 'team@example.com',
            },
            phone: {
                label: 'Phone number',
                type: 'text',
                placeholder: '+1 (555) 123-4567',
            },
            location: {
                label: 'Location',
                type: 'text',
                placeholder: 'City, country or timezone',
            },
            hours: {
                label: 'Availability',
                type: 'text',
                placeholder: 'Mon-Fri, 9am-6pm',
            },
            response_time: {
                label: 'Response time note',
                type: 'text',
                placeholder: 'We respond within one business day',
            },
            form_title: {
                label: 'Form title',
                type: 'text',
                placeholder: 'Send us a note',
            },
            form_submit_label: {
                label: 'Submit button label',
                type: 'text',
                placeholder: 'Send message',
            },
            form_subject: {
                label: 'Email subject',
                type: 'text',
                placeholder: 'New inquiry from your site',
            },
            privacy_note: {
                label: 'Privacy note',
                type: 'text',
                placeholder: 'We only use your details to reply.',
            },
        },
    });
})();

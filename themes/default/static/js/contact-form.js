(function () {
    const encode = (value) => encodeURIComponent(value || '');

    const buildBody = ({ name, email, topic, message }) => {
        const parts = [
            `Name: ${name || '-'}`,
            `Email: ${email || '-'}`,
            `Topic: ${topic || '-'}`,
            '',
            message || '',
        ];
        return encode(parts.join('\n'));
    };

    const initForm = (form) => {
        if (!form) return;
        const mailto = form.getAttribute('data-contact-mailto');
        if (!mailto) return;

        form.addEventListener('submit', (event) => {
            event.preventDefault();

            const subject = form.getAttribute('data-contact-subject') || 'Contact request';
            const name = form.querySelector('input[name="name"]')?.value || '';
            const email = form.querySelector('input[name="email"]')?.value || '';
            const topic = form.querySelector('select[name="topic"]')?.value || '';
            const message = form.querySelector('textarea[name="message"]')?.value || '';

            const body = buildBody({ name, email, topic, message });
            const href = `${mailto}?subject=${encode(subject)}&body=${body}`;
            window.location.href = href;
        });
    };

    const forms = document.querySelectorAll('form[data-contact-form]');
    forms.forEach(initForm);
})();

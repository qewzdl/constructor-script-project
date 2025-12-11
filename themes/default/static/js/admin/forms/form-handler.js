/**
 * Admin Form Handler Module
 * Provides base functionality for form handling, validation, and submission
 */
(() => {
    /**
     * Create form handler instance
     * @param {Object} config - Configuration object
     * @param {HTMLFormElement} config.form - Form element
     * @param {Object} config.apiClient - API client instance
     * @param {Object} config.uiManager - UI manager instance
     * @param {Function} config.onSubmit - Submit handler
     * @param {Function} config.onSuccess - Success handler
     * @param {Function} config.validate - Validation function
     */
    const createFormHandler = (config) => {
        const {
            form,
            apiClient,
            uiManager,
            onSubmit,
            onSuccess,
            onError,
            validate,
        } = config;

        if (!form) {
            console.error('FormHandler: form element is required');
            return null;
        }

        /**
         * Serialize form data to object
         */
        const serializeForm = () => {
            const formData = new FormData(form);
            const data = {};

            for (const [key, value] of formData.entries()) {
                if (key.endsWith('[]')) {
                    const arrayKey = key.slice(0, -2);
                    if (!data[arrayKey]) {
                        data[arrayKey] = [];
                    }
                    data[arrayKey].push(value);
                } else {
                    data[key] = value;
                }
            }

            return data;
        };

        /**
         * Populate form with data
         */
        const populateForm = (data) => {
            if (!data) {
                return;
            }

            Object.entries(data).forEach(([key, value]) => {
                const field = form.elements[key];
                if (!field) {
                    return;
                }

                if (field.type === 'checkbox') {
                    field.checked = Boolean(value);
                } else if (field.type === 'radio') {
                    const radio = form.querySelector(`input[name="${key}"][value="${value}"]`);
                    if (radio) {
                        radio.checked = true;
                    }
                } else if (field.tagName === 'SELECT' && field.multiple) {
                    const values = Array.isArray(value) ? value : [value];
                    Array.from(field.options).forEach((option) => {
                        option.selected = values.includes(option.value);
                    });
                } else {
                    field.value = value;
                }
            });
        };

        /**
         * Reset form to initial state
         */
        const resetForm = () => {
            form.reset();
            const errorMessages = form.querySelectorAll('.form__error');
            errorMessages.forEach((msg) => msg.remove());
        };

        /**
         * Show field error
         */
        const showFieldError = (fieldName, message) => {
            const field = form.elements[fieldName];
            if (!field) {
                return;
            }

            // Remove existing error
            const existingError = field.parentElement.querySelector('.form__error');
            if (existingError) {
                existingError.remove();
            }

            // Add new error
            const errorElement = document.createElement('div');
            errorElement.className = 'form__error';
            errorElement.textContent = message;
            field.parentElement.appendChild(errorElement);
        };

        /**
         * Clear field errors
         */
        const clearFieldErrors = () => {
            const errorMessages = form.querySelectorAll('.form__error');
            errorMessages.forEach((msg) => msg.remove());
        };

        /**
         * Handle form submission
         */
        const handleSubmit = async (event) => {
            event.preventDefault();

            clearFieldErrors();

            const data = serializeForm();

            // Validate if validation function provided
            if (typeof validate === 'function') {
                const validationErrors = validate(data);
                if (validationErrors && Object.keys(validationErrors).length > 0) {
                    Object.entries(validationErrors).forEach(([field, message]) => {
                        showFieldError(field, message);
                    });
                    uiManager.showAlert('Please fix the errors in the form', 'error');
                    return;
                }
            }

            uiManager.disableForm(form, true);

            try {
                let result;
                if (typeof onSubmit === 'function') {
                    result = await onSubmit(data);
                } else {
                    throw new Error('No submit handler provided');
                }

                if (typeof onSuccess === 'function') {
                    onSuccess(result, data);
                }

                uiManager.showAlert('Changes saved successfully', 'success');
            } catch (error) {
                if (typeof onError === 'function') {
                    onError(error, data);
                } else {
                    uiManager.handleRequestError(error);
                }
            } finally {
                uiManager.disableForm(form, false);
            }
        };

        // Attach submit handler
        form.addEventListener('submit', handleSubmit);

        return {
            serializeForm,
            populateForm,
            resetForm,
            showFieldError,
            clearFieldErrors,
            submit: handleSubmit,
        };
    };

    // Export to global namespace
    window.AdminFormHandler = {
        create: createFormHandler,
    };
})();

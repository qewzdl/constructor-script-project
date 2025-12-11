/**
 * Admin UI Manager Module
 * Handles alerts, form states, and general UI interactions
 */
(() => {
    /**
     * Create UI manager instance
     */
    const createUiManager = ({ alertElement, setAlert, toggleFormDisabled }) => {
        const ALERT_AUTO_HIDE_MS = 5000;
        const ALERT_TRANSITION_FALLBACK_MS = 360;
        let alertAutoHideTimeoutId = null;
        let alertDismissFallbackTimeoutId = null;
        let pendingHideHandler = null;

        /**
         * Cancel pending hide operations
         */
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

        /**
         * Update alert content
         */
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

        /**
         * Clear alert content
         */
        const clearAlertContent = () => updateAlertContent('');

        /**
         * Hide alert with animation
         */
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

        /**
         * Show alert message
         */
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

        /**
         * Clear alert (alias for hideAlert)
         */
        const clearAlert = () => hideAlert();

        /**
         * Handle request errors with appropriate messages
         */
        const handleRequestError = (error, auth = null) => {
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

        /**
         * Disable or enable form elements
         */
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

        /**
         * Focus first available field in form
         */
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

        /**
         * Scroll form into view and focus first field
         */
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

        /**
         * Update status message element
         */
        const updateStatusMessage = (element, message) => {
            if (!element) {
                return;
            }
            if (!message) {
                element.textContent = '';
                element.hidden = true;
                return;
            }
            element.hidden = false;
            element.textContent = message;
        };

        return {
            showAlert,
            hideAlert,
            clearAlert,
            handleRequestError,
            disableForm,
            focusFirstField,
            bringFormIntoView,
            updateStatusMessage,
        };
    };

    // Export to global namespace
    window.AdminUiManager = {
        create: createUiManager,
    };
})();

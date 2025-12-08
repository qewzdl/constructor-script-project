const showAlert = (element, message, type = "error") => {
    element.textContent = message;
    element.className = `setup__alert setup__alert--${type}`;
    element.hidden = false;
};

const clearAlert = (element) => {
    element.hidden = true;
    element.textContent = "";
};

const populateSiteFields = (form, site) => {
    if (!site) {
        return;
    }

    const entries = [
        ["site_name", site.name],
        ["site_description", site.description],
        ["site_url", site.url],
        ["site_favicon", site.favicon],
        ["site_logo", site.logo],
        ["site_default_language", site.default_language],
    ];

    entries.forEach(([name, value]) => {
        if (!value) {
            return;
        }

        const field = form.querySelector(`[name="${name}"]`);
        if (field) {
            field.value = value;
        }
    });

    if (Array.isArray(site.supported_languages)) {
        const supportedField = form.querySelector('[name="site_supported_languages"]');
        if (supportedField) {
            const defaultLanguage = typeof site.default_language === 'string' ? site.default_language : '';
            const additional = site.supported_languages.filter((code) => typeof code === 'string' && code !== defaultLanguage);
            supportedField.value = additional.join(', ');
        }
    }
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

const parseLanguageList = (value) => {
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
        if (!languageCodePattern.test(normalized)) {
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

const buildPayload = (formData) => {
    const payload = {};
    formData.forEach((value, key) => {
        const trimmed = typeof value === 'string' ? value.trim() : value;
        if (key === 'site_supported_languages') {
            payload[key] = parseLanguageList(trimmed || '');
            return;
        }
        payload[key] = trimmed;
    });
    return payload;
};

const disableForm = (form, disabled) => {
    Array.from(form.elements).forEach((el) => {
        el.disabled = disabled;
    });
};

const validatePassword = (password) => {
    const errors = [];
    
    if (password.length < 8) {
        errors.push("at least 8 characters");
    }
    
    if (!/[A-Z]/.test(password)) {
        errors.push("one uppercase letter");
    }
    
    if (!/[a-z]/.test(password)) {
        errors.push("one lowercase letter");
    }
    
    if (!/[0-9]/.test(password)) {
        errors.push("one digit");
    }
    
    if (errors.length > 0) {
        return `Password must contain ${errors.join(", ")}`;
    }
    
    return null;
};

const validateEmail = (email) => {
    const emailRegex = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
    if (!emailRegex.test(email)) {
        return "Please enter a valid email address";
    }
    return null;
};

const validateUsername = (username) => {
    if (username.length < 3) {
        return "Username must be at least 3 characters";
    }
    if (username.length > 50) {
        return "Username must not exceed 50 characters";
    }
    return null;
};

document.addEventListener("DOMContentLoaded", () => {
    const root = document.querySelector('[data-page="setup"]');
    if (!root) {
        return;
    }

    const form = root.querySelector("#setup-form");
    const alertElement = root.querySelector("#setup-alert");
    const action = form?.dataset.action;
    const statusUrl = form?.dataset.status;

    if (!form || !action || !statusUrl) {
        return;
    }

    fetch(statusUrl)
        .then((response) => {
            if (!response.ok) {
                throw new Error("Failed to load setup status");
            }
            return response.json();
        })
        .then((data) => {
            if (!data.setup_required) {
                window.location.href = "/";
                return;
            }

            populateSiteFields(form, data.site);
        })
        .catch((error) => {
            showAlert(alertElement, error.message || "Failed to load setup data");
        });

    form.addEventListener("submit", async (event) => {
        event.preventDefault();
        clearAlert(alertElement);

        const formData = new FormData(form);
        let payload;
        try {
            payload = buildPayload(formData);
        } catch (languageError) {
            showAlert(alertElement, languageError.message || 'Please review the language settings.', 'error');
            return;
        }

        // Validate username
        const usernameError = validateUsername(payload.admin_username || '');
        if (usernameError) {
            showAlert(alertElement, usernameError, 'error');
            return;
        }

        // Validate email
        const emailError = validateEmail(payload.admin_email || '');
        if (emailError) {
            showAlert(alertElement, emailError, 'error');
            return;
        }

        // Validate password
        const passwordError = validatePassword(payload.admin_password || '');
        if (passwordError) {
            showAlert(alertElement, passwordError, 'error');
            return;
        }

        const defaultLanguage = normaliseLanguageCode(payload.site_default_language || '');
        if (!defaultLanguage || !languageCodePattern.test(defaultLanguage)) {
            showAlert(alertElement, 'Please provide a valid default language code (e.g. "en" or "en-GB").', 'error');
            return;
        }
        payload.site_default_language = defaultLanguage;
        if (!Array.isArray(payload.site_supported_languages)) {
            payload.site_supported_languages = [];
        }
        payload.site_supported_languages = payload.site_supported_languages.filter(
            (code) => code && code !== defaultLanguage
        );

        disableForm(form, true);

        try {
            const response = await fetch(action, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(payload),
            });

            if (!response.ok) {
                const error = await response.json().catch(() => ({ error: "Failed to complete setup" }));
                throw new Error(error.error || "Failed to complete setup");
            }

            showAlert(alertElement, "Setup completed successfully. Redirecting to sign in…", "success");
            setTimeout(() => {
                window.location.href = "/login";
            }, 1200);
        } catch (error) {
            showAlert(alertElement, error.message || "Failed to complete setup");
        } finally {
            disableForm(form, false);
        }
    });
});

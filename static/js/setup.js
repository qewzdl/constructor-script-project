const showAlert = (element, message, type = "error") => {
    element.textContent = message;
    element.className = `auth__alert auth__alert--${type}`;
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
};

const buildPayload = (formData) => {
    const payload = {};
    formData.forEach((value, key) => {
        payload[key] = value.trim();
    });
    return payload;
};

const disableForm = (form, disabled) => {
    Array.from(form.elements).forEach((el) => {
        el.disabled = disabled;
    });
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
        const payload = buildPayload(formData);

        if (!payload.admin_password || payload.admin_password.length < 8) {
            showAlert(alertElement, "Password must be at least 8 characters long");
            return;
        }

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
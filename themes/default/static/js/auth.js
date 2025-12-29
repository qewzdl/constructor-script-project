(() => {
    const STORAGE_KEY = "authToken";
    const COOKIE_NAME = "auth_token";
    const CSRF_COOKIE_NAME = "csrf_token";
    const TOKEN_TTL_SECONDS = 72 * 60 * 60;
    const AVATAR_UPLOAD_ENDPOINT = "/api/v1/profile/avatar";

    const bodyElement = document.body;
    const parseBooleanAttribute = (value) => {
        if (value === "true") {
            return true;
        }
        if (value === "false") {
            return false;
        }
        return null;
    };

    const initialServerAuthState = bodyElement
        ? parseBooleanAttribute(bodyElement.dataset?.authenticated)
        : null;

    let serverAuthenticated = initialServerAuthState;
    const avatarState = {
        pendingFile: null,
        previewUrl: null,
    };

    let avatarInput;
    let avatarImage;
    let avatarUrlField;
    let avatarRemoveButton;
    let profileUsernameInput;
    let avatarPreview;

    const isInitialAvatar = (url) => {
        const trimmed = (url || "").trim();
        return trimmed.startsWith("/uploads/avatar-initial-");
    };

    const syncServerAuthState = (value) => {
        if (!bodyElement || typeof value !== "boolean") {
            return;
        }
        bodyElement.dataset.authenticated = value ? "true" : "false";
        serverAuthenticated = value;
    };

    const readCookie = (name) => {
        if (!document.cookie) {
            return null;
        }
        const cookies = document.cookie.split("; ");
        for (const cookie of cookies) {
            const [key, ...rest] = cookie.split("=");
            if (key === name) {
                return decodeURIComponent(rest.join("="));
            }
        }
        return null;
    };

    const secureAttribute =
        window.location.protocol === "https:" ? "; Secure" : "";
    const writeCookie = (name, value, maxAgeSeconds) => {
        const maxAge =
            typeof maxAgeSeconds === "number" ? maxAgeSeconds : TOKEN_TTL_SECONDS;
        document.cookie = `${name}=${encodeURIComponent(
            value || ""
        )}; path=/; max-age=${maxAge}; SameSite=Strict${secureAttribute}`;
    };
    const clearCookie = (name) => writeCookie(name, "", 0);
    const getCSRFCookie = () => readCookie(CSRF_COOKIE_NAME);

    const Auth = {
        getToken() {
            return (
                localStorage.getItem(STORAGE_KEY) ||
                sessionStorage.getItem(STORAGE_KEY) ||
                readCookie(COOKIE_NAME)
            );
        },
        setToken(token, persist) {
            if (persist) {
                localStorage.setItem(STORAGE_KEY, token);
                sessionStorage.removeItem(STORAGE_KEY);
            } else {
                sessionStorage.setItem(STORAGE_KEY, token);
                localStorage.removeItem(STORAGE_KEY);
            }
            writeCookie(COOKIE_NAME, token, TOKEN_TTL_SECONDS);
        },
        clearToken() {
            localStorage.removeItem(STORAGE_KEY);
            sessionStorage.removeItem(STORAGE_KEY);
            clearCookie(COOKIE_NAME);
            clearCookie(CSRF_COOKIE_NAME);
        },
        syncFromCookie() {
            const token = this.getToken();
            if (token) {
                if (
                    !localStorage.getItem(STORAGE_KEY) &&
                    !sessionStorage.getItem(STORAGE_KEY)
                ) {
                    sessionStorage.setItem(STORAGE_KEY, token);
                }
                writeCookie(COOKIE_NAME, token, TOKEN_TTL_SECONDS);
                const csrfToken = getCSRFCookie();
                if (csrfToken) {
                    writeCookie(CSRF_COOKIE_NAME, csrfToken, TOKEN_TTL_SECONDS);
                }
            }
            return token;
        },
    };

    const updateNavVisibility = (explicitState) => {
        const token = Auth.getToken();
        const tokenPresent = Boolean(token);
        let isAuthenticated;

        if (typeof explicitState === "boolean") {
            isAuthenticated = explicitState;
            syncServerAuthState(explicitState);
        } else if (typeof serverAuthenticated === "boolean") {
            isAuthenticated = serverAuthenticated;
            if (!serverAuthenticated && tokenPresent) {
                Auth.clearToken();
            }
        } else if (tokenPresent) {
            isAuthenticated = true;
            syncServerAuthState(true);
        } else {
            isAuthenticated = false;
        }
        document.querySelectorAll('[data-auth="auth"]').forEach((element) => {
            element.hidden = !isAuthenticated;
        });
        document.querySelectorAll('[data-auth="guest"]').forEach((element) => {
            element.hidden = isAuthenticated;
        });
    };

    const setAlert = (target, message, type = "info") => {
        const element =
            typeof target === "string" ? document.getElementById(target) : target;
        if (!element) {
            return;
        }

        element.classList.remove("is-error", "is-success", "is-info");

        if (!message) {
            element.hidden = true;
            element.textContent = "";
            return;
        }

        const statusClass =
            type === "error" ? "is-error" : type === "success" ? "is-success" : "is-info";
        element.classList.add(statusClass);
        element.hidden = false;
        element.textContent = message;
    };

    const toggleFormDisabled = (form, disabled) => {
        const elements = form.querySelectorAll("input, button, select, textarea");
        elements.forEach((element) => {
            element.disabled = disabled;
        });
        form.classList.toggle("is-disabled", disabled);
    };

    const setAvatarValue = (value) => {
        if (avatarUrlField) {
            avatarUrlField.value = value || "";
        }
    };

    const ensureAvatarImage = () => {
        if (avatarImage && avatarImage.isConnected) {
            return avatarImage;
        }
        if (!avatarPreview) {
            return null;
        }
        const img = document.createElement("img");
        img.className = "profile-avatar__image";
        img.loading = "lazy";
        img.setAttribute("data-avatar-image", "");
        img.alt = "Profile avatar";
        avatarPreview.appendChild(img);
        avatarImage = img;
        return img;
    };

    const removeAvatarImage = () => {
        if (avatarImage && avatarImage.parentElement) {
            avatarImage.parentElement.removeChild(avatarImage);
        }
        avatarImage = null;
    };

    const renderAvatar = (url, { commitValue = true } = {}) => {
        const trimmed = (url || "").trim();
        const hasAvatar = Boolean(trimmed);
        const placeholderAvatar = isInitialAvatar(trimmed);
        if (commitValue) {
            setAvatarValue(trimmed);
        }
        if (hasAvatar) {
            const img = ensureAvatarImage();
            if (img) {
                img.src = trimmed;
                img.hidden = false;
            }
        } else {
            removeAvatarImage();
        }
        if (avatarRemoveButton) {
            const hideRemove = !hasAvatar || placeholderAvatar;
            avatarRemoveButton.hidden = hideRemove;
            avatarRemoveButton.disabled = hideRemove;
        }
    };

    const revokeAvatarPreviewURL = () => {
        if (avatarState.previewUrl) {
            if (avatarState.previewUrl.startsWith("blob:")) {
                URL.revokeObjectURL(avatarState.previewUrl);
            }
            avatarState.previewUrl = null;
        }
    };

    const previewAvatarFile = (file) => {
        avatarState.pendingFile = file || null;
        revokeAvatarPreviewURL();

        if (!file) {
            renderAvatar(avatarUrlField ? avatarUrlField.value : "", { commitValue: false });
            return;
        }

        const reader = new FileReader();
        reader.addEventListener("load", () => {
            avatarState.previewUrl = typeof reader.result === "string" ? reader.result : "";
            renderAvatar(avatarState.previewUrl, { commitValue: false });
            if (avatarRemoveButton) {
                avatarRemoveButton.hidden = false;
                avatarRemoveButton.disabled = false;
            }
        });
        reader.addEventListener("error", () => {
            avatarState.previewUrl = null;
            setAlert("profile-details-alert", "Failed to preview avatar. Try another file.", "error");
        });
        reader.readAsDataURL(file);
    };

    const populateAvatarFromUser = (user) => {
        const avatar = (user && user.avatar) || "";
        avatarState.pendingFile = null;
        revokeAvatarPreviewURL();
        renderAvatar(avatar, { commitValue: true });
    };

    const uploadPendingAvatar = async () => {
        if (!avatarState.pendingFile) {
            return avatarUrlField ? avatarUrlField.value.trim() : "";
        }

        const formData = new FormData();
        formData.append("avatar", avatarState.pendingFile);

        const payload = await apiRequest(AVATAR_UPLOAD_ENDPOINT, {
            method: "POST",
            body: formData,
        });

        const newAvatar =
            (payload && (payload.avatar || (payload.user && payload.user.avatar))) || "";

        avatarState.pendingFile = null;
        revokeAvatarPreviewURL();
        renderAvatar(newAvatar, { commitValue: true });
        if (avatarInput) {
            avatarInput.value = "";
        }

        return newAvatar;
    };

    const handleAvatarRemove = async (event) => {
        if (event) {
            event.preventDefault();
        }

        const profileForm = document.getElementById("profile-form");
        const emailInput = document.getElementById("profile-email");
        const alertId = "profile-details-alert";
        const username = profileUsernameInput ? profileUsernameInput.value.trim() : "";
        const email = emailInput ? emailInput.value.trim() : "";

        if (!username || !email) {
            setAlert(alertId, "Username and email cannot be empty.", "error");
            return;
        }

        if (avatarRemoveButton) {
            avatarRemoveButton.disabled = true;
        }
        if (profileForm) {
            toggleFormDisabled(profileForm, true);
        }

        try {
            const payload = await apiRequest("/api/v1/profile", {
                method: "PUT",
                body: JSON.stringify({ username, email, avatar: "" }),
            });

            if (payload && payload.user) {
                populateAvatarFromUser(payload.user);
                if (profileUsernameInput) {
                    profileUsernameInput.value = payload.user.username || profileUsernameInput.value;
                }
                if (emailInput) {
                    emailInput.value = payload.user.email || email;
                }
            } else {
                renderAvatar("", { commitValue: true });
            }

            if (avatarInput) {
                avatarInput.value = "";
            }

            setAlert(alertId, "Avatar removed. Placeholder applied.", "info");
        } catch (error) {
            if (error.status === 401) {
                Auth.clearToken();
                updateNavVisibility(false);
                window.location.href = "/login?redirect=/profile";
                return;
            }
            setAlert(alertId, error.message || "Failed to remove avatar.", "error");
        } finally {
            if (profileForm) {
                toggleFormDisabled(profileForm, false);
            }
            if (avatarRemoveButton) {
                avatarRemoveButton.disabled = false;
            }
        }
    };

    const initPasswordToggles = () => {
        const toggles = document.querySelectorAll("[data-password-toggle]");
        if (!toggles.length) {
            return;
        }

        toggles.forEach((button) => {
            const targetId = button.dataset.passwordToggle;
            if (!targetId) {
                return;
            }

            const input = document.getElementById(targetId);
            if (!input) {
                return;
            }

            const openIcon = button.querySelector('[data-eye="open"]');
            const closedIcon = button.querySelector('[data-eye="closed"]');

            const setState = (visible) => {
                input.type = visible ? "text" : "password";
                button.setAttribute("aria-pressed", visible ? "true" : "false");
                button.setAttribute(
                    "aria-label",
                    visible ? "Hide password" : "Show password"
                );

                if (openIcon) {
                    openIcon.hidden = !visible;
                }
                if (closedIcon) {
                    closedIcon.hidden = visible;
                }
            };

            setState(input.type === "text");

            button.addEventListener("click", (event) => {
                event.preventDefault();
                const nextState = button.getAttribute("aria-pressed") !== "true";
                setState(nextState);
                input.focus();
            });
        });
    };

    const buildPasswordStrengthError = (password) => {
        if (typeof password !== "string" || password.length < 6) {
            return "Password must be at least 6 characters long.";
        }

        return "";
    };

    const apiRequest = async (url, options = {}) => {
        const headers = Object.assign({}, options.headers || {});
        const token = Auth.getToken();
        const method = (options.method || "GET").toUpperCase();

        if (options.body && !(options.body instanceof FormData)) {
            headers["Content-Type"] = headers["Content-Type"] || "application/json";
        }

        if (token) {
            headers.Authorization = `Bearer ${token}`;
        }

        if (["POST", "PUT", "PATCH", "DELETE"].includes(method)) {
            const csrfToken = getCSRFCookie();
            if (csrfToken) {
                headers["X-CSRF-Token"] = csrfToken;
            }
        }

        const response = await fetch(url, {
            credentials: "include",
            ...options,
            headers,
        });

        const contentType = response.headers.get("content-type") || "";
        const isJson = contentType.includes("application/json");
        const payload = isJson ? await response.json().catch(() => null) : await response.text();

        if (!response.ok) {
            let message = "Request failed";
            if (payload) {
                if (typeof payload === "string") {
                    message = payload;
                } else if (payload.error) {
                    message = payload.error;
                }
            }
            const error = new Error(message);
            error.status = response.status;
            error.payload = payload;
            throw error;
        }

        return payload;
    };

    const handleLogout = async () => {
        try {
            await apiRequest("/api/v1/logout", {
                method: "POST",
            });
        } catch (error) {
            console.warn("Failed to notify server about logout:", error);
        } finally {
            Auth.clearToken();
            updateNavVisibility(false);
            window.location.href = "/login";
        }
    };

    const handleLogin = async (event) => {
        event.preventDefault();
        const form = event.currentTarget;
        const alertId = "login-alert";
        setAlert(alertId, "");

        const email = form.email.value.trim();
        const password = form.password.value;
        const remember = form.remember.checked;

        if (!email || !password) {
            setAlert(alertId, "Please provide both email and password.", "error");
            return;
        }

        toggleFormDisabled(form, true);

        try {
            const payload = await apiRequest(
                form.dataset.action || form.action,
                {
                    method: "POST",
                    body: JSON.stringify({ email, password }),
                }
            );

            if (!payload || !payload.token) {
                throw new Error("Unable to sign in. Please try again.");
            }

            Auth.setToken(payload.token, remember);
            if (payload.csrf_token) {
                writeCookie(CSRF_COOKIE_NAME, payload.csrf_token, TOKEN_TTL_SECONDS);
            }
            updateNavVisibility(true);
            setAlert(alertId, "Signed in successfully. Redirecting…", "success");

            const redirectTarget = form.dataset.redirect || "/profile";
            window.setTimeout(() => {
                window.location.href = redirectTarget;
            }, 800);
        } catch (error) {
            setAlert(alertId, error.message, "error");
        } finally {
            toggleFormDisabled(form, false);
        }
    };

    const handleRegister = async (event) => {
        event.preventDefault();
        const form = event.currentTarget;
        const alertId = "register-alert";
        setAlert(alertId, "");

        const username = form.username.value.trim();
        const email = form.email.value.trim();
        const password = form.password.value;
        const confirmPassword = form.password_confirm.value;
        const acceptedTerms = form.agree.checked;

        if (!username || !email || !password) {
            setAlert(alertId, "All fields are required.", "error");
            return;
        }

        if (password !== confirmPassword) {
            setAlert(alertId, "Passwords do not match.", "error");
            return;
        }

        const passwordError = buildPasswordStrengthError(password);
        if (passwordError) {
            setAlert(alertId, passwordError, "error");
            return;
        }

        if (!acceptedTerms) {
            setAlert(alertId, "Please accept the terms to continue.", "error");
            return;
        }

        toggleFormDisabled(form, true);

        try {
            await apiRequest(form.dataset.action || form.action, {
                method: "POST",
                body: JSON.stringify({ username, email, password }),
            });

            setAlert(
                alertId,
                "Account created successfully. You can sign in now.",
                "success"
            );

            window.setTimeout(() => {
                window.location.href = "/login";
            }, 1200);
        } catch (error) {
            setAlert(alertId, error.message, "error");
        } finally {
            toggleFormDisabled(form, false);
        }
    };

    const handleForgotPassword = async (event) => {
        event.preventDefault();
        const form = event.currentTarget;
        const alertId = "forgot-password-alert";
        setAlert(alertId, "");

        const email = form.email.value.trim();
        if (!email) {
            setAlert(alertId, "Please enter your email address.", "error");
            return;
        }

        toggleFormDisabled(form, true);

        try {
            await apiRequest(form.dataset.action || form.action, {
                method: "POST",
                body: JSON.stringify({ email }),
            });

            setAlert(
                alertId,
                "If this email is registered, we sent password reset instructions.",
                "success"
            );
        } catch (error) {
            setAlert(alertId, error.message, "error");
        } finally {
            toggleFormDisabled(form, false);
        }
    };

    const resolveResetToken = (form) => {
        const hiddenToken = (form.token && form.token.value ? form.token.value : "").trim();
        const dataToken = (form.dataset.token || "").trim();
        return hiddenToken || dataToken;
    };

    const handlePasswordReset = async (event) => {
        event.preventDefault();
        const form = event.currentTarget;
        const alertId = "reset-password-alert";
        setAlert(alertId, "");

        const token = resolveResetToken(form);
        const password = form.password.value;
        const confirmPassword = form.password_confirm.value;

        if (!token) {
            setAlert(alertId, "The reset link is missing or has expired.", "error");
            return;
        }

        if (!password || !confirmPassword) {
            setAlert(alertId, "Please fill in the new password fields.", "error");
            return;
        }

        if (password !== confirmPassword) {
            setAlert(alertId, "Passwords do not match.", "error");
            return;
        }

        const passwordError = buildPasswordStrengthError(password);
        if (passwordError) {
            setAlert(alertId, passwordError, "error");
            return;
        }

        toggleFormDisabled(form, true);

        try {
            await apiRequest(form.dataset.action || form.action, {
                method: "POST",
                body: JSON.stringify({
                    token,
                    password,
                    password_confirm: confirmPassword,
                }),
            });

            setAlert(alertId, "Password updated. You can sign in now.", "success");
            window.setTimeout(() => {
                window.location.href = "/login";
            }, 900);
        } catch (error) {
            setAlert(alertId, error.message, "error");
        } finally {
            toggleFormDisabled(form, false);
        }
    };

    const handleProfileUpdate = async (event) => {
        event.preventDefault();
        const form = event.currentTarget;
        const alertId = "profile-details-alert";
        setAlert(alertId, "");

        const username = form.username.value.trim();
        const email = form.email.value.trim();
        let avatarValue = avatarUrlField ? avatarUrlField.value : undefined;

        if (!username || !email) {
            setAlert(alertId, "Username and email cannot be empty.", "error");
            return;
        }

        toggleFormDisabled(form, true);

        try {
            if (avatarState.pendingFile) {
                avatarValue = await uploadPendingAvatar();
            }

            const body = { username, email };
            if (typeof avatarValue === "string") {
                body.avatar = avatarValue;
            }

            const payload = await apiRequest(form.dataset.action, {
                method: "PUT",
                body: JSON.stringify(body),
            });

            if (payload && payload.user) {
                const { username: updatedUsername, email: updatedEmail } = payload.user;
                form.username.value = updatedUsername;
                form.email.value = updatedEmail;
                populateProfileFromResponse(payload.user);
            }

            setAlert(alertId, "Profile updated successfully.", "success");
        } catch (error) {
            if (error.status === 401) {
                Auth.clearToken();
                updateNavVisibility(false);
                window.location.href = "/login?redirect=/profile";
                return;
            }
            setAlert(alertId, error.message, "error");
        } finally {
            toggleFormDisabled(form, false);
        }
    };

    const handlePasswordUpdate = async (event) => {
        event.preventDefault();
        const form = event.currentTarget;
        const alertId = "profile-password-alert";
        setAlert(alertId, "");

        const currentPassword = form.old_password.value;
        const newPassword = form.new_password.value;
        const confirmPassword = form.confirm_password.value;

        if (!currentPassword || !newPassword) {
            setAlert(alertId, "Please fill in all password fields.", "error");
            return;
        }

        if (newPassword !== confirmPassword) {
            setAlert(alertId, "New passwords do not match.", "error");
            return;
        }

        const passwordError = buildPasswordStrengthError(newPassword);
        if (passwordError) {
            setAlert(alertId, passwordError, "error");
            return;
        }

        toggleFormDisabled(form, true);

        try {
            await apiRequest(form.dataset.action, {
                method: "PUT",
                body: JSON.stringify({
                    old_password: currentPassword,
                    new_password: newPassword,
                }),
            });

            form.reset();
            setAlert(alertId, "Password updated successfully.", "success");
        } catch (error) {
            if (error.status === 401) {
                Auth.clearToken();
                updateNavVisibility(false);
                window.location.href = "/login?redirect=/profile";
                return;
            }
            setAlert(alertId, error.message, "error");
        } finally {
            toggleFormDisabled(form, false);
        }
    };

    const renderUserCourses = (courses) => {
        const container =
            document.getElementById("profile-courses") ||
            document.querySelector('[data-role="profile-courses"]');
        if (!container) {
            return;
        }

        const list = container.querySelector(".profile-courses__list");
        const empty = container.querySelector(".profile-courses__empty");
        const entries = Array.isArray(courses) ? courses : [];
        const limitValue = Number.parseInt(
            container.dataset.courseLimit,
            10
        );
        const maxEntries = Number.isFinite(limitValue) && limitValue > 0
            ? limitValue
            : entries.length;
        const limitedEntries = entries.slice(0, maxEntries);

        if (empty) {
            empty.hidden = limitedEntries.length > 0;
        }

        if (!list) {
            return;
        }

        list.innerHTML = "";

        if (limitedEntries.length === 0) {
            list.hidden = true;
            return;
        }

        list.hidden = false;

        limitedEntries.forEach((entry, index) => {
            if (!entry || typeof entry !== "object") {
                return;
            }
            const pkg = entry.package || {};
            const access = entry.access || {};

            const cardIndex = index + 1;
            const headingId = `profile-course-${cardIndex}-title`;
            const descriptionId = pkg.description ? `profile-course-${cardIndex}-description` : "";

            const item = document.createElement("li");
            item.className = "profile-courses__item";

            const cardElement = pkg.id
                ? document.createElement("a")
                : document.createElement("article");
            cardElement.className = pkg.id
                ? "profile-course post-card profile-course--link"
                : "profile-course post-card";

            if (pkg.id) {
                cardElement.href = `/courses/${pkg.id}`;
            }

            cardElement.setAttribute("aria-labelledby", headingId);
            if (descriptionId) {
                cardElement.setAttribute("aria-describedby", descriptionId);
            }

            if (pkg.image_url) {
                const figure = document.createElement("figure");
                figure.className = "profile-course__media post-card__figure";

                const img = document.createElement("img");
                img.src = pkg.image_url;
                img.alt = pkg.title ? `${pkg.title} cover` : "Course cover";
                img.loading = "lazy";
                img.className = "profile-course__image post-card__image";

                figure.appendChild(img);
                cardElement.appendChild(figure);
            }

            const content = document.createElement("div");
            content.className = "profile-course__content post-card__content";

            const title = document.createElement("h3");
            title.className = "profile-course__title post-card__title";
            title.id = headingId;
            title.textContent = pkg.title || "Untitled course";
            content.appendChild(title);

            if (pkg.description) {
                const description = document.createElement("p");
                description.className = "profile-course__description post-card__description";
                description.id = descriptionId;
                description.textContent = pkg.description;
                content.appendChild(description);
            }

            const meta = document.createElement("div");
            meta.className = "profile-course__meta post-card__meta";

            const grantedItem = document.createElement("span");
            grantedItem.className = "profile-course__meta-item";
            grantedItem.append(document.createTextNode("Granted"));

            if (access.created_at) {
                grantedItem.append(" ");
                const grantedDate = new Date(access.created_at);
                const grantedTime = document.createElement("time");
                if (!Number.isNaN(grantedDate.getTime())) {
                    grantedTime.dateTime = grantedDate.toISOString();
                    grantedTime.textContent = grantedDate.toLocaleDateString();
                } else {
                    grantedTime.textContent = access.created_at;
                }
                grantedItem.appendChild(grantedTime);
            }

            meta.appendChild(grantedItem);

            const expiresItem = document.createElement("span");
            expiresItem.className = "profile-course__meta-item";

            if (access.expires_at) {
                expiresItem.append(document.createTextNode("Expires"), " ");
                const expiresDate = new Date(access.expires_at);
                const expiresTime = document.createElement("time");
                if (!Number.isNaN(expiresDate.getTime())) {
                    expiresTime.dateTime = expiresDate.toISOString();
                    expiresTime.textContent = expiresDate.toLocaleDateString();
                } else {
                    expiresTime.textContent = access.expires_at;
                }
                expiresItem.appendChild(expiresTime);
            } else {
                expiresItem.textContent = "No expiration";
            }

            meta.appendChild(expiresItem);
            content.appendChild(meta);

            cardElement.appendChild(content);

            const boxIcon = document.createElement("span");
            boxIcon.className = "course-card__box-icon";
            boxIcon.setAttribute("aria-hidden", "true");
            boxIcon.innerHTML = `<svg fill="currentColor" viewBox="0 0 32 32" role="img" aria-hidden="true"><title>box-open</title><path d="M29.742 5.39c-.002-.012-.01-.022-.012-.034-.014-.057-.032-.106-.055-.152l.002.004c-.017-.046-.036-.086-.059-.124l.002.003c-.033-.044-.069-.082-.108-.117l-.001-.001c-.023-.028-.046-.053-.071-.076l-.023-.011c-.044-.027-.095-.05-.149-.067l-.005-.002c-.034-.016-.073-.031-.115-.043l-.005-.001-.028-.01-12.999-2c-.034-.006-.074-.009-.114-.009s-.08.003-.119.009l.004-.001-13.026 2.01c-.054.014-.101.032-.146.054l.004-.002c-.052.018-.096.039-.138.064l.003-.002-.024.011c-.025.023-.047.048-.068.074l-.001.001c-.041.036-.078.075-.11.118l-.001.002c-.02.034-.039.074-.055.115l-.002.005c-.021.042-.039.09-.052.141l-.001.005c-.003.013-.011.023-.013.036l-1 6.75c-.005.033-.008.071-.008.11 0 .361.255.663.595.734l.005.001 1.445.296c-.025.065-.041.14-.044.218l-.002 12.502c0 .36.254.66.592.733l.005.001 12 2.5c.046.01.099.016.153.016s.107-.006.158-.017l-.005.001 11.999-2.5c.344-.073.597-.374.598-.734v-12.5c-.004-.08-.02-.155-.046-.225l.002.005 1.445-.296c.345-.072.6-.373.6-.734 0-.039-.003-.077-.009-.115l.001.004zm-13.742-1.131 8.351 1.285-8.351 1.446-8.351-1.446zm-12.371 2.111 11.295 1.955-2.364 5.319-9.714-1.987zm1.121 7.208 8.1 1.657c.046.01.099.016.153.016.303 0 .564-.181.681-.441l.002-.005 1.564-3.52v16.294l-10.5-2.188zm22.5 11.813-10.5 2.188v-16.294l1.564 3.52c.12.264.382.445.685.445h0c0 0 0 0 0 0 .053 0 .105-.006.155-.017l-.005.001 8.1-1.657zm-7.809-11.746-2.365-5.319 11.295-1.955.783 5.287z"></path></svg>`;
            cardElement.appendChild(boxIcon);
            item.appendChild(cardElement);
            list.appendChild(item);
        });
    };

    const initProfileTabs = () => {
        const profileRoot =
            document.querySelector('[data-page="profile"]') ||
            document.querySelector(".page-view--profile");
        if (!profileRoot) {
            return;
        }

        const nav = profileRoot.querySelector("[data-profile-nav]");
        const panelsContainer = profileRoot.querySelector("[data-profile-panels]");
        if (!nav || !panelsContainer) {
            return;
        }

        const buttons = Array.from(
            nav.querySelectorAll("[data-profile-tab-target]")
        );
        const panels = Array.from(
            panelsContainer.querySelectorAll("[data-profile-tab-panel]")
        );

        if (!buttons.length || !panels.length) {
            return;
        }

        const panelByTab = new Map();
        panels.forEach((panel) => {
            const tabKey =
                (panel.dataset.profileTabPanel || panel.dataset.profileTab || "")
                    .trim();
            if (tabKey) {
                panelByTab.set(tabKey, panel);
            }
        });

        let defaultTab = buttons[0].dataset.profileTabTarget || "";
        const hashTab = window.location.hash.replace("#", "").trim();
        if (hashTab && panelByTab.has(hashTab)) {
            defaultTab = hashTab;
        }

        const activate = (tab) => {
            let resolvedTab = tab;
            if (!panelByTab.has(resolvedTab)) {
                resolvedTab = defaultTab;
            }

            const activePanel = panelByTab.get(resolvedTab);
            if (!activePanel) {
                return;
            }

            panelsContainer.classList.add("is-ready");

            panels.forEach((panel) => {
                const isActive = panel === activePanel;
                panel.classList.toggle("is-active", isActive);
                panel.hidden = !isActive;
            });

            buttons.forEach((button) => {
                const isActive = button.dataset.profileTabTarget === resolvedTab;
                button.classList.toggle("is-active", isActive);
                if (isActive) {
                    button.setAttribute("aria-current", "true");
                } else {
                    button.removeAttribute("aria-current");
                }
            });

            if (window.history && window.history.replaceState) {
                window.history.replaceState(null, "", `#${resolvedTab}`);
            }
        };

        buttons.forEach((button) => {
            button.addEventListener("click", () => {
                const target = button.dataset.profileTabTarget;
                if (!target) {
                    return;
                }
                activate(target);
            });
        });

        activate(defaultTab);
    };

    const populateProfileFromResponse = (user) => {
        const usernameField = document.getElementById("profile-username");
        const emailField = document.getElementById("profile-email");
        const roleField = document.getElementById("profile-role");

        if (user && usernameField) {
            usernameField.value = user.username || "";
        }
        if (user && emailField) {
            emailField.value = user.email || "";
        }
        if (user && roleField) {
            roleField.value = user.role || "user";
        }
        if (user) {
            populateAvatarFromUser(user);
        } else {
            populateAvatarFromUser(null);
        }
    };

    const loadProfileData = async () => {
        try {
            const payload = await apiRequest("/api/v1/profile", {
                method: "GET",
            });

            if (payload && payload.user) {
                populateProfileFromResponse(payload.user);
            }

            renderUserCourses(payload?.courses || []);
        } catch (error) {
            if (error.status === 401) {
                Auth.clearToken();
                updateNavVisibility(false);
                window.location.href = "/login?redirect=/profile";
                return;
            }
            setAlert("profile-details-alert", error.message, "error");
        }
    };

    window.App = Object.assign(window.App || {}, {
        auth: Auth,
        apiRequest,
        setAlert,
        toggleFormDisabled,
        renderUserCourses,
    });

    document.addEventListener("DOMContentLoaded", () => {
        Auth.syncFromCookie();
        updateNavVisibility();
        initPasswordToggles();

        const logoutButtons = [];
        const legacyLogoutButton = document.getElementById("logout-button");
        if (legacyLogoutButton) {
            logoutButtons.push(legacyLogoutButton);
        }
        document.querySelectorAll("[data-profile-logout]").forEach((button) => {
            logoutButtons.push(button);
        });
        logoutButtons.forEach((button) => {
            button.addEventListener("click", handleLogout);
        });

        const loginForm = document.getElementById("login-form");
        if (loginForm) {
            loginForm.addEventListener("submit", handleLogin);
        }

        const registerForm = document.getElementById("register-form");
        if (registerForm) {
            registerForm.addEventListener("submit", handleRegister);
        }

        const forgotPasswordForm = document.getElementById("forgot-password-form");
        if (forgotPasswordForm) {
            forgotPasswordForm.addEventListener("submit", handleForgotPassword);
        }

        const resetPasswordForm = document.getElementById("reset-password-form");
        if (resetPasswordForm) {
            const tokenValue = resolveResetToken(resetPasswordForm);
            if (!tokenValue) {
                setAlert(
                    "reset-password-alert",
                    "The reset link is missing or has expired. Request a new one.",
                    "error"
                );
                toggleFormDisabled(resetPasswordForm, true);
            }
            resetPasswordForm.addEventListener("submit", handlePasswordReset);
        }

        const profileForm = document.getElementById("profile-form");
        const passwordForm = document.getElementById("password-form");
        const profilePage = document.querySelector('[data-page="profile"], .page-view--profile');
        avatarInput = document.querySelector("[data-avatar-input]");
        avatarPreview = document.querySelector("[data-avatar-preview]");
        avatarImage = document.querySelector("[data-avatar-image]");
        avatarUrlField = document.querySelector("[data-avatar-url]");
        avatarRemoveButton = document.querySelector("[data-avatar-remove]");
        profileUsernameInput = document.getElementById("profile-username");

        if (profilePage) {
            if (!Auth.getToken() && serverAuthenticated !== true) {
                window.location.href = "/login?redirect=/profile";
                return;
            }

            if (avatarUrlField) {
                renderAvatar(avatarUrlField.value, { commitValue: true });
            }

        if (avatarInput) {
            avatarInput.addEventListener("change", (event) => {
                const [file] = event.target.files || [];
                if (file) {
                    previewAvatarFile(file);
                        setAlert(
                            "profile-details-alert",
                            "Avatar selected. Save changes to apply.",
                            "info"
                        );
                } else {
                    previewAvatarFile(null);
                }
            });
        }

        document.querySelectorAll("[data-avatar-trigger]").forEach((button) => {
            button.addEventListener("click", (event) => {
                event.preventDefault();
                if (avatarInput) {
                    avatarInput.click();
                }
            });
        });

        if (avatarRemoveButton) {
            avatarRemoveButton.addEventListener("click", handleAvatarRemove);
        }

            initProfileTabs();
            loadProfileData();
        }

        if (profileForm) {
            profileForm.addEventListener("submit", handleProfileUpdate);
        }

        if (passwordForm) {
            passwordForm.addEventListener("submit", handlePasswordUpdate);
        }
    });
})();

(() => {
    const STORAGE_KEY = "authToken";
    const COOKIE_NAME = "auth_token";
    const TOKEN_TTL_SECONDS = 72 * 60 * 60;

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

    const writeCookie = (value, maxAgeSeconds) => {
        const maxAge = typeof maxAgeSeconds === "number" ? maxAgeSeconds : TOKEN_TTL_SECONDS;
        document.cookie = `${COOKIE_NAME}=${encodeURIComponent(value || "")}; path=/; max-age=${maxAge}; SameSite=Strict`;
    };

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
            writeCookie(token, TOKEN_TTL_SECONDS);
        },
        clearToken() {
            localStorage.removeItem(STORAGE_KEY);
            sessionStorage.removeItem(STORAGE_KEY);
            writeCookie("", 0);
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
                writeCookie(token, TOKEN_TTL_SECONDS);
            }
            return token;
        },
    };

    const updateNavVisibility = (explicitState) => {
        const tokenPresent = Boolean(Auth.getToken());
        let isAuthenticated;

        if (typeof explicitState === "boolean") {
            isAuthenticated = explicitState;
            syncServerAuthState(explicitState);
        } else if (tokenPresent) {
            isAuthenticated = true;
            syncServerAuthState(true);
        } else if (typeof serverAuthenticated === "boolean") {
            isAuthenticated = serverAuthenticated;
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

    const apiRequest = async (url, options = {}) => {
        const headers = Object.assign({}, options.headers || {});
        const token = Auth.getToken();

        if (options.body && !(options.body instanceof FormData)) {
            headers["Content-Type"] = headers["Content-Type"] || "application/json";
        }

        if (token) {
            headers.Authorization = `Bearer ${token}`;
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

    const handleProfileUpdate = async (event) => {
        event.preventDefault();
        const form = event.currentTarget;
        const alertId = "profile-details-alert";
        setAlert(alertId, "");

        const username = form.username.value.trim();
        const email = form.email.value.trim();

        if (!username || !email) {
            setAlert(alertId, "Username and email cannot be empty.", "error");
            return;
        }

        toggleFormDisabled(form, true);

        try {
            const payload = await apiRequest(form.dataset.action, {
                method: "PUT",
                body: JSON.stringify({ username, email }),
            });

            if (payload && payload.user) {
                const { username: updatedUsername, email: updatedEmail } = payload.user;
                form.username.value = updatedUsername;
                form.email.value = updatedEmail;
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
    };

    const loadProfileData = async () => {
        try {
            const payload = await apiRequest("/api/v1/profile", {
                method: "GET",
            });

            if (payload && payload.user) {
                populateProfileFromResponse(payload.user);
            }
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
    });

    document.addEventListener("DOMContentLoaded", () => {
        Auth.syncFromCookie();
        updateNavVisibility();

        const logoutButton = document.getElementById("logout-button");
        if (logoutButton) {
            logoutButton.addEventListener("click", handleLogout);
        }

        const loginForm = document.getElementById("login-form");
        if (loginForm) {
            loginForm.addEventListener("submit", handleLogin);
        }

        const registerForm = document.getElementById("register-form");
        if (registerForm) {
            registerForm.addEventListener("submit", handleRegister);
        }

        const profileForm = document.getElementById("profile-form");
        const passwordForm = document.getElementById("password-form");
        const profilePage = document.querySelector('[data-page="profile"]');

        if (profilePage) {
            if (!Auth.getToken() && serverAuthenticated !== true) {
                window.location.href = "/login?redirect=/profile";
                return;
            }

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
(() => {
    const storageKey = "theme";
    const root = document.documentElement;
    const toggle = document.querySelector("[data-theme-toggle]");
    const label = toggle ? toggle.querySelector("[data-theme-toggle-label]") : null;
    const icon = toggle ? toggle.querySelector(".header__theme-icon") : null;
    const mediaQuery =
        window.matchMedia && typeof window.matchMedia === "function"
            ? window.matchMedia("(prefers-color-scheme: dark)")
            : null;

    const getStoredTheme = () => {
        try {
            const storedTheme = localStorage.getItem(storageKey);
            if (storedTheme === "light" || storedTheme === "dark") {
                return storedTheme;
            }
        } catch (error) {
            /* no-op */
        }
        return null;
    };

    const updateToggle = (theme) => {
        if (!toggle) {
            return;
        }
        if (label) {
            label.textContent = theme === "dark" ? "Light mode" : "Dark mode";
        }
        toggle.setAttribute(
            "aria-label",
            theme === "dark" ? "Switch to light mode" : "Switch to dark mode",
        );
        if (icon) {
            icon.textContent = theme === "dark" ? "☀️" : "🌙";
        }
    };

    let transitionTimeoutId;

    const startThemeTransition = () => {
        root.classList.add("theme-transition");
        if (transitionTimeoutId) {
            window.clearTimeout(transitionTimeoutId);
        }
        transitionTimeoutId = window.setTimeout(() => {
            root.classList.remove("theme-transition");
        }, 800);
    };

    const applyTheme = (theme, options = {}) => {
        const { persist = true, withTransition = true } = options;
        if (withTransition) {
            startThemeTransition();
        }
        root.setAttribute("data-theme", theme);
        if (persist) {
            try {
                localStorage.setItem(storageKey, theme);
            } catch (error) {
                /* no-op */
            }
        }
        updateToggle(theme);
    };

    const storedTheme = getStoredTheme();
    const prefersDark = mediaQuery ? mediaQuery.matches : false;
    const preferredTheme = storedTheme ?? (prefersDark ? "dark" : "light");
    const initialTheme = root.getAttribute("data-theme");

    if (initialTheme !== preferredTheme) {
        applyTheme(preferredTheme, { persist: false, withTransition: false });
    } else {
        updateToggle(preferredTheme);
    }

    if (toggle) {
        toggle.addEventListener("click", () => {
            const theme = root.getAttribute("data-theme") === "dark" ? "light" : "dark";
            applyTheme(theme);
        });
    }

    if (mediaQuery) {
        const handleMediaChange = (event) => {
            if (getStoredTheme()) {
                return;
            }
            applyTheme(event.matches ? "dark" : "light", { persist: false });
        };

        if (typeof mediaQuery.addEventListener === "function") {
            mediaQuery.addEventListener("change", handleMediaChange);
        } else if (typeof mediaQuery.addListener === "function") {
            mediaQuery.addListener(handleMediaChange);
        }
    }
})();

(() => {
    const storageKey = "theme";
    const root = document.documentElement;
    const toggle = document.querySelector("[data-theme-toggle]");
    if (!toggle) {
        return;
    }

    const label = toggle.querySelector("[data-theme-toggle-label]");
    const icon = toggle.querySelector(".header__theme-icon");

    const updateToggle = (theme) => {
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

    const applyTheme = (theme) => {
        startThemeTransition();
        root.setAttribute("data-theme", theme);
        try {
            localStorage.setItem(storageKey, theme);
        } catch (error) {
            /* no-op */
        }
        updateToggle(theme);
    };

    const currentTheme = root.getAttribute("data-theme") === "dark" ? "dark" : "light";
    updateToggle(currentTheme);

    toggle.addEventListener("click", () => {
        const theme = root.getAttribute("data-theme") === "dark" ? "light" : "dark";
        applyTheme(theme);
    });
})();
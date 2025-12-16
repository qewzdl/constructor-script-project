(() => {
    const header = document.querySelector(".header");
    if (!header) {
        return;
    }

    const toggle = header.querySelector("[data-menu-toggle]");
    const menu = header.querySelector("[data-menu]");
    if (!toggle || !menu) {
        return;
    }

    const breakpoint = window.matchMedia("(min-width: 768px)");

    const setMenuVisibility = (open) => {
        header.classList.toggle("is-menu-open", open);
        toggle.setAttribute("aria-expanded", open ? "true" : "false");
        if (open) {
            menu.removeAttribute("hidden");
        } else {
            menu.setAttribute("hidden", "");
        }
    };

    const syncWithViewport = () => {
        if (breakpoint.matches) {
            header.classList.remove("is-menu-open");
            toggle.setAttribute("aria-expanded", "false");
            menu.removeAttribute("hidden");
        } else if (!header.classList.contains("is-menu-open")) {
            menu.setAttribute("hidden", "");
        }
    };

    toggle.addEventListener("click", () => {
        const open = header.classList.contains("is-menu-open");
        setMenuVisibility(!open);
    });

    document.addEventListener("click", (event) => {
        if (!header.contains(event.target) && header.classList.contains("is-menu-open")) {
            setMenuVisibility(false);
        }
    });

    breakpoint.addEventListener("change", syncWithViewport);
    syncWithViewport();
    // Добавляем обработчик прокрутки для смены стиля header
    const onScroll = () => {
        if (window.scrollY > 0) {
            header.classList.add("is-scrolled");
        } else {
            header.classList.remove("is-scrolled");
        }
    };
    window.addEventListener("scroll", onScroll);
    // Инициализация состояния при загрузке
    onScroll();
})();
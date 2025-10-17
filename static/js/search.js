(function () {
    "use strict";

    function initCustomSelect(container) {
        const trigger = container.querySelector("[data-custom-select-trigger]");
        const list = container.querySelector("[data-custom-select-list]");
        const input = container.querySelector("[data-custom-select-input]");
        const label = container.querySelector("[data-custom-select-label]");
        const options = Array.from(container.querySelectorAll("[data-custom-select-option]"));

        if (!trigger || !list || !input || !label || options.length === 0) {
            return;
        }

        let isOpen = false;
        let activeIndex = options.findIndex((option) =>
            option.classList.contains("search__custom-option--selected")
        );

        if (activeIndex < 0) {
            activeIndex = 0;
        }

        function updateActiveDescendant(index) {
            options.forEach((option, optionIndex) => {
                if (optionIndex === index) {
                    option.classList.add("search__custom-option--active");
                    list.setAttribute("aria-activedescendant", option.id);
                } else {
                    option.classList.remove("search__custom-option--active");
                }
            });
        }

        function ensureVisible(index) {
            const option = options[index];
            if (!option) {
                return;
            }
            option.scrollIntoView({ block: "nearest" });
        }

        function close(focusTrigger = true) {
            if (!isOpen) {
                return;
            }
            isOpen = false;
            container.classList.remove("is-open");
            trigger.setAttribute("aria-expanded", "false");
            list.hidden = true;
            if (focusTrigger) {
                trigger.focus();
            }
        }

        function open() {
            if (isOpen) {
                return;
            }
            isOpen = true;
            container.classList.add("is-open");
            trigger.setAttribute("aria-expanded", "true");
            list.hidden = false;
            list.focus({ preventScroll: true });
            updateActiveDescendant(activeIndex);
            ensureVisible(activeIndex);
        }

        function selectOption(index) {
            const option = options[index];
            if (!option) {
                return;
            }

            options.forEach((item) => {
                const isSelected = item === option;
                item.classList.toggle("search__custom-option--selected", isSelected);
                item.setAttribute("aria-selected", isSelected ? "true" : "false");
            });

            input.value = option.getAttribute("data-value") || "";
            label.textContent = option.textContent.trim();
            activeIndex = index;
            updateActiveDescendant(index);
            close();
        }

        trigger.addEventListener("click", (event) => {
            event.preventDefault();
            if (isOpen) {
                close(false);
            } else {
                open();
            }
        });

        trigger.addEventListener("keydown", (event) => {
            const { key } = event;

            if (key === "ArrowDown" || key === "ArrowUp") {
                event.preventDefault();
                if (!isOpen) {
                    open();
                    return;
                }
                activeIndex = key === "ArrowDown"
                    ? Math.min(activeIndex + 1, options.length - 1)
                    : Math.max(activeIndex - 1, 0);
                updateActiveDescendant(activeIndex);
                ensureVisible(activeIndex);
            } else if (key === "Enter" || key === " ") {
                event.preventDefault();
                if (isOpen) {
                    selectOption(activeIndex);
                } else {
                    open();
                }
            } else if (key === "Escape") {
                if (isOpen) {
                    event.preventDefault();
                    close();
                }
            }
        });

        list.addEventListener("keydown", (event) => {
            const { key } = event;

            if (key === "ArrowDown" || key === "ArrowUp") {
                event.preventDefault();
                activeIndex = key === "ArrowDown"
                    ? Math.min(activeIndex + 1, options.length - 1)
                    : Math.max(activeIndex - 1, 0);
                updateActiveDescendant(activeIndex);
                ensureVisible(activeIndex);
            } else if (key === "Home") {
                event.preventDefault();
                activeIndex = 0;
                updateActiveDescendant(activeIndex);
                ensureVisible(activeIndex);
            } else if (key === "End") {
                event.preventDefault();
                activeIndex = options.length - 1;
                updateActiveDescendant(activeIndex);
                ensureVisible(activeIndex);
            } else if (key === "Enter" || key === " ") {
                event.preventDefault();
                selectOption(activeIndex);
            } else if (key === "Escape") {
                event.preventDefault();
                close();
            } else if (key === "Tab") {
                close(false);
            }
        });

        options.forEach((option, index) => {
            option.addEventListener("click", (event) => {
                event.preventDefault();
                selectOption(index);
            });

            option.addEventListener("mousemove", () => {
                if (activeIndex !== index) {
                    activeIndex = index;
                    updateActiveDescendant(activeIndex);
                }
            });
        });

        document.addEventListener("click", (event) => {
            if (!container.contains(event.target)) {
                close(false);
            }
        });

        const selectedOption = options[activeIndex];
        if (selectedOption) {
            label.textContent = selectedOption.textContent.trim();
            updateActiveDescendant(activeIndex);
        }
    }

    document.addEventListener("DOMContentLoaded", () => {
        document.querySelectorAll("[data-custom-select]").forEach((container) => {
            initCustomSelect(container);
        });
    });
})();
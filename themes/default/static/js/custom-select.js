(function () {
    "use strict";

    const INITIALIZED_ATTR = "data-custom-select-initialized";
    const states = new WeakMap();
    let idCounter = 0;

    function uniqueId(prefix = "custom-select") {
        idCounter += 1;
        return `${prefix}-${idCounter}`;
    }

    function getPlaceholder(select) {
        const placeholder = select.getAttribute("data-placeholder");
        if (placeholder) {
            return placeholder;
        }
        const firstOption = select.querySelector("option[value='']");
        if (firstOption) {
            return firstOption.textContent.trim();
        }
        return "Select an option";
    }

    function isDisabledOption(option) {
        return option.disabled || option.hasAttribute("aria-disabled");
    }

    function setActiveIndex(state, nextIndex) {
        if (!state.optionElements.length) {
            state.activeIndex = -1;
            return;
        }

        const clampedIndex = Math.min(
            Math.max(nextIndex, 0),
            state.optionElements.length - 1,
        );

        const target = state.optionElements[clampedIndex];
        if (!target || target.classList.contains("custom-select__option--disabled")) {
            state.activeIndex = -1;
            state.list.removeAttribute("aria-activedescendant");
            state.optionElements.forEach((el) => {
                el.classList.remove("custom-select__option--active");
            });
            return;
        }

        state.activeIndex = clampedIndex;
        state.optionElements.forEach((el, index) => {
            el.classList.toggle("custom-select__option--active", index === state.activeIndex);
        });
        state.list.setAttribute(
            "aria-activedescendant",
            target.id,
        );
        target.scrollIntoView({ block: "nearest" });
    }

    function updateDisabledState(state) {
        const disabled = state.select.disabled;
        state.trigger.disabled = disabled;
        state.trigger.setAttribute("aria-disabled", disabled ? "true" : "false");
        state.container.classList.toggle("is-disabled", disabled);
        if (disabled && state.isOpen) {
            closeSelect(state, false);
        }
    }

    function updateSelection(state, { shouldSetActive = true } = {}) {
        const { select, label, optionElements, options } = state;
        let selectedIndex = select.selectedIndex;

        if (selectedIndex < 0) {
            selectedIndex = options.findIndex((option) => !isDisabledOption(option));
        }

        const selectedOption = options[selectedIndex];

        optionElements.forEach((element, index) => {
            const isSelected = index === selectedIndex && !element.classList.contains("custom-select__option--disabled");
            element.classList.toggle("custom-select__option--selected", isSelected);
            element.setAttribute("aria-selected", isSelected ? "true" : "false");
        });

        if (selectedOption && !isDisabledOption(selectedOption)) {
            label.textContent = selectedOption.textContent.trim();
            if (shouldSetActive) {
                setActiveIndex(state, selectedIndex);
            }
        } else {
            label.textContent = getPlaceholder(select);
            if (shouldSetActive) {
                const fallbackIndex = options.findIndex((option) => !isDisabledOption(option));
                if (fallbackIndex >= 0) {
                    setActiveIndex(state, fallbackIndex);
                }
            }
        }
    }

    function buildOption(state, option, index) {
        const element = document.createElement("li");
        element.className = "custom-select__option";
        element.setAttribute("role", "option");
        element.id = `${state.list.id}-option-${index}`;
        element.dataset.value = option.value;
        element.textContent = option.textContent;

        if (isDisabledOption(option)) {
            element.classList.add("custom-select__option--disabled");
            element.setAttribute("aria-disabled", "true");
        }

        if (option.selected && !isDisabledOption(option)) {
            element.classList.add("custom-select__option--selected");
            element.setAttribute("aria-selected", "true");
            state.activeIndex = index;
        }

        element.addEventListener("click", (event) => {
            event.preventDefault();
            if (isDisabledOption(option)) {
                return;
            }
            selectOption(state, index, { triggerChange: true });
        });

        element.addEventListener("mousemove", () => {
            if (state.activeIndex !== index && !isDisabledOption(option)) {
                setActiveIndex(state, index);
            }
        });

        return element;
    }

    function rebuildOptions(state) {
        const { select, list } = state;
        const options = Array.from(select.options);

        list.innerHTML = "";
        state.optionElements = [];
        state.options = options;

        options.forEach((option, index) => {
            const optionElement = buildOption(state, option, index);
            state.optionElements.push(optionElement);
            list.appendChild(optionElement);
        });

        updateSelection(state, { shouldSetActive: true });
    }

    function openSelect(state) {
        if (state.isOpen || state.select.disabled) {
            return;
        }
        state.isOpen = true;
        state.container.classList.add("is-open");
        state.trigger.setAttribute("aria-expanded", "true");
        state.list.hidden = false;
        state.list.focus({ preventScroll: true });
        if (state.activeIndex >= 0) {
            setActiveIndex(state, state.activeIndex);
        }
    }

    function closeSelect(state, focusTrigger = true) {
        if (!state.isOpen) {
            return;
        }
        state.isOpen = false;
        state.container.classList.remove("is-open");
        state.trigger.setAttribute("aria-expanded", "false");
        state.list.hidden = true;
        if (focusTrigger) {
            state.trigger.focus();
        }
    }

    function selectOption(state, index, { triggerChange } = { triggerChange: true }) {
        const option = state.options[index];
        if (!option || isDisabledOption(option)) {
            return;
        }

        if (state.select.selectedIndex !== index) {
            state.select.selectedIndex = index;
            updateSelection(state, { shouldSetActive: false });
            if (triggerChange) {
                const event = new Event("change", { bubbles: true });
                state.select.dispatchEvent(event);
            }
        } else {
            updateSelection(state, { shouldSetActive: false });
        }

        closeSelect(state);
    }

    function handleTriggerKeydown(state, event) {
        const { key } = event;
        if (key === "ArrowDown" || key === "ArrowUp") {
            event.preventDefault();
            if (!state.isOpen) {
                openSelect(state);
                return;
            }
            const delta = key === "ArrowDown" ? 1 : -1;
            let nextIndex = state.activeIndex + delta;
            while (
                nextIndex >= 0 &&
                nextIndex < state.optionElements.length &&
                state.optionElements[nextIndex].classList.contains("custom-select__option--disabled")
            ) {
                nextIndex += delta;
            }
            setActiveIndex(state, nextIndex);
        } else if (key === "Enter" || key === " ") {
            event.preventDefault();
            if (state.isOpen && state.activeIndex >= 0) {
                selectOption(state, state.activeIndex, { triggerChange: true });
            } else {
                openSelect(state);
            }
        } else if (key === "Escape") {
            if (state.isOpen) {
                event.preventDefault();
                closeSelect(state);
            }
        }
    }

    function handleListKeydown(state, event) {
        const { key } = event;
        if (key === "ArrowDown" || key === "ArrowUp") {
            event.preventDefault();
            const delta = key === "ArrowDown" ? 1 : -1;
            let nextIndex = state.activeIndex + delta;
            while (
                nextIndex >= 0 &&
                nextIndex < state.optionElements.length &&
                state.optionElements[nextIndex].classList.contains("custom-select__option--disabled")
            ) {
                nextIndex += delta;
            }
            setActiveIndex(state, nextIndex);
        } else if (key === "Home") {
            event.preventDefault();
            const firstEnabled = state.optionElements.findIndex(
                (element) => !element.classList.contains("custom-select__option--disabled"),
            );
            if (firstEnabled >= 0) {
                setActiveIndex(state, firstEnabled);
            }
        } else if (key === "End") {
            event.preventDefault();
            for (let index = state.optionElements.length - 1; index >= 0; index -= 1) {
                if (!state.optionElements[index].classList.contains("custom-select__option--disabled")) {
                    setActiveIndex(state, index);
                    break;
                }
            }
        } else if (key === "Enter" || key === " ") {
            event.preventDefault();
            if (state.activeIndex >= 0) {
                selectOption(state, state.activeIndex, { triggerChange: true });
            }
        } else if (key === "Escape") {
            event.preventDefault();
            closeSelect(state);
        } else if (key === "Tab") {
            closeSelect(state, false);
        }
    }

    function enhanceSelect(select) {
        if (
            select.dataset.customSelect === "native" ||
            select.getAttribute(INITIALIZED_ATTR) === "true" ||
            select.multiple ||
            select.size > 1
        ) {
            return;
        }

        const container = document.createElement("div");
        container.className = "custom-select";
        container.dataset.customSelect = "container";

        const trigger = document.createElement("button");
        trigger.type = "button";
        trigger.className = "custom-select__trigger";
        trigger.setAttribute("aria-haspopup", "listbox");
        trigger.setAttribute("aria-expanded", "false");

        const label = document.createElement("span");
        label.className = "custom-select__label";
        trigger.appendChild(label);

        const icon = document.createElement("span");
        icon.className = "custom-select__icon";
        trigger.appendChild(icon);

        const list = document.createElement("ul");
        list.className = "custom-select__options";
        list.setAttribute("role", "listbox");
        list.setAttribute("aria-multiselectable", "false");
        list.tabIndex = -1;
        list.hidden = true;

        const baseId = select.id || uniqueId();
        const triggerId = `${baseId}-trigger`;
        const listId = `${baseId}-listbox`;
        trigger.id = triggerId;
        trigger.setAttribute("aria-controls", listId);
        list.id = listId;

        const labels = select.labels ? Array.from(select.labels) : [];
        if (labels.length) {
            const labelIds = [];
            labels.forEach((associatedLabel, index) => {
                if (!associatedLabel.id) {
                    associatedLabel.id = `${baseId}-label-${index + 1}`;
                }
                if (associatedLabel.htmlFor === select.id) {
                    associatedLabel.setAttribute("data-custom-select-original-for", associatedLabel.htmlFor);
                    associatedLabel.htmlFor = triggerId;
                }
                associatedLabel.addEventListener("click", () => {
                    if (!select.disabled) {
                        trigger.focus();
                    }
                });
                labelIds.push(associatedLabel.id);
            });
            trigger.setAttribute("aria-labelledby", `${labelIds.join(" ")} ${triggerId}`.trim());
            list.setAttribute("aria-labelledby", labelIds.join(" "));
        }

        select.setAttribute(INITIALIZED_ATTR, "true");
        select.classList.add("custom-select__native");
        select.setAttribute("aria-hidden", "true");
        select.tabIndex = -1;

        if (select.parentNode) {
            select.parentNode.insertBefore(container, select);
        }
        container.appendChild(select);
        container.appendChild(trigger);
        container.appendChild(list);

        select.classList.forEach((className) => {
            if (className && className !== "custom-select__native") {
                trigger.classList.add(className);
            }
        });

        const state = {
            select,
            container,
            trigger,
            label,
            list,
            options: [],
            optionElements: [],
            activeIndex: -1,
            isOpen: false,
        };

        states.set(select, state);

        rebuildOptions(state);
        updateSelection(state, { shouldSetActive: true });
        updateDisabledState(state);

        trigger.addEventListener("click", (event) => {
            event.preventDefault();
            if (state.isOpen) {
                closeSelect(state, false);
            } else {
                openSelect(state);
            }
        });

        trigger.addEventListener("keydown", (event) => {
            handleTriggerKeydown(state, event);
        });

        list.addEventListener("keydown", (event) => {
            handleListKeydown(state, event);
        });

        select.addEventListener("change", () => {
            updateSelection(state, { shouldSetActive: true });
        });

        select.addEventListener("focus", () => {
            trigger.focus();
        });

        if (select.form) {
            select.form.addEventListener("reset", () => {
                window.requestAnimationFrame(() => {
                    rebuildOptions(state);
                });
            });
        }

        const documentClickHandler = (event) => {
            if (!container.contains(event.target)) {
                closeSelect(state, false);
            }
        };

        document.addEventListener("click", documentClickHandler);

        const observer = new MutationObserver((mutations) => {
            let needsRebuild = false;
            let needsDisabledUpdate = false;

            mutations.forEach((mutation) => {
                if (mutation.type === "childList") {
                    needsRebuild = true;
                }
                if (
                    mutation.type === "attributes" &&
                    mutation.attributeName === "disabled"
                ) {
                    needsDisabledUpdate = true;
                }
            });

            if (needsRebuild) {
                rebuildOptions(state);
            }
            if (needsDisabledUpdate) {
                updateDisabledState(state);
            }
        });

        observer.observe(select, {
            childList: true,
            subtree: true,
            attributes: true,
            attributeFilter: ["disabled"],
        });

        state.cleanup = () => {
            document.removeEventListener("click", documentClickHandler);
            observer.disconnect();
        };
    }

    function refreshSelect(select) {
        const state = states.get(select);
        if (!state) {
            enhanceSelect(select);
            return;
        }
        rebuildOptions(state);
        updateDisabledState(state);
        updateSelection(state, { shouldSetActive: true });
    }

    function initAll(root = document) {
        const selects = root.querySelectorAll("select");
        selects.forEach((select) => {
            enhanceSelect(select);
        });
    }

    function handleMutations(mutations) {
        mutations.forEach((mutation) => {
            mutation.addedNodes.forEach((node) => {
                if (node.nodeType !== Node.ELEMENT_NODE) {
                    return;
                }
                if (node.tagName === "SELECT") {
                    enhanceSelect(node);
                } else {
                    const nestedSelects = node.querySelectorAll ? node.querySelectorAll("select") : [];
                    nestedSelects.forEach((select) => {
                        enhanceSelect(select);
                    });
                }
            });
        });
    }

    document.addEventListener("DOMContentLoaded", () => {
        initAll();

        const observer = new MutationObserver(handleMutations);
        observer.observe(document.body, {
            childList: true,
            subtree: true,
        });

        window.Constructor = window.Constructor || {};
        window.Constructor.customSelect = {
            enhance: enhanceSelect,
            refresh: refreshSelect,
        };
    });
})();

(function () {
    "use strict";

    const focusableSelectors = [
        'a[href]:not([tabindex="-1"])',
        'button:not([disabled]):not([tabindex="-1"])',
        'input:not([disabled]):not([type="hidden"]):not([tabindex="-1"])',
        'textarea:not([disabled]):not([tabindex="-1"])',
        'select:not([disabled]):not([tabindex="-1"])',
        '[tabindex]:not([tabindex="-1"])'
    ].join(",");

    function ready(fn) {
        if (document.readyState === "loading") {
            document.addEventListener("DOMContentLoaded", fn, { once: true });
        } else {
            fn();
        }
    }

    function collectCourseTopics(card) {
        const topicNodes = card.querySelectorAll(".courses-list__topics .post-card__tag");
        return Array.from(topicNodes)
            .map((node) => node.textContent.trim())
            .filter(Boolean);
    }

    function prepareTopicItem(text) {
        const item = document.createElement("li");
        item.className = "course-modal__topics-item";
        item.textContent = text;
        return item;
    }

    ready(() => {
        const modal = document.querySelector("[data-course-modal]");
        if (!modal) {
            return;
        }

        const dialog = modal.querySelector(".course-modal__dialog");
        const closeButton = modal.querySelector("[data-course-modal-close]");
        const purchaseButton = modal.querySelector("[data-course-modal-purchase]");
        const titleElement = modal.querySelector("[data-course-modal-title]");
        const priceElement = modal.querySelector("[data-course-modal-price]");
        const descriptionElement = modal.querySelector("[data-course-modal-description]");
        const topicsWrapper = modal.querySelector("[data-course-modal-topics-wrapper]");
        const topicsList = modal.querySelector("[data-course-modal-topics]");
        const mediaWrapper = modal.querySelector("[data-course-modal-media]");
        const imageElement = modal.querySelector("[data-course-modal-image]");

        if (!dialog || !closeButton || !purchaseButton || !titleElement || !priceElement || !descriptionElement || !topicsWrapper || !topicsList || !mediaWrapper || !imageElement) {
            return;
        }

        let activeCard = null;
        let lastFocusedElement = null;
        let escapeListenerAttached = false;

        function setHidden(element, hidden) {
            if (!element) {
                return;
            }
            element.hidden = hidden;
        }

        function dispatchLifecycleEvent(name, detail) {
            const event = new CustomEvent(name, {
                bubbles: true,
                detail: detail || {}
            });
            modal.dispatchEvent(event);
        }

        function trapFocus(event) {
            if (!modal.classList.contains("course-modal--active")) {
                return;
            }
            if (event.key !== "Tab") {
                return;
            }

            const focusable = dialog.querySelectorAll(focusableSelectors);
            if (focusable.length === 0) {
                event.preventDefault();
                dialog.focus();
                return;
            }

            const first = focusable[0];
            const last = focusable[focusable.length - 1];

            if (event.shiftKey) {
                if (document.activeElement === first) {
                    event.preventDefault();
                    last.focus();
                }
            } else if (document.activeElement === last) {
                event.preventDefault();
                first.focus();
            }
        }

        function handleEscape(event) {
            if (event.key === "Escape" && modal.classList.contains("course-modal--active")) {
                event.preventDefault();
                closeModal();
            }
        }

        function bindEscape() {
            if (!escapeListenerAttached) {
                document.addEventListener("keydown", handleEscape);
                escapeListenerAttached = true;
            }
        }

        function unbindEscape() {
            if (escapeListenerAttached) {
                document.removeEventListener("keydown", handleEscape);
                escapeListenerAttached = false;
            }
        }

        function openModal(card) {
            activeCard = card;
            lastFocusedElement = document.activeElement instanceof HTMLElement ? document.activeElement : null;

            const title = card.querySelector(".post-card__link")?.textContent.trim() || "";
            const priceText = card.querySelector(".courses-list__price")?.textContent.trim() || "";
            const descriptionSource = card.querySelector(".post-card__description");
            const image = card.querySelector(".post-card__image");
            const topics = collectCourseTopics(card);
            const courseId = card.getAttribute("data-course-id") || "";

            titleElement.textContent = title;

            if (priceText) {
                priceElement.textContent = priceText;
                setHidden(priceElement, false);
            } else {
                priceElement.textContent = "";
                setHidden(priceElement, true);
            }

            if (descriptionSource) {
                descriptionElement.innerHTML = descriptionSource.innerHTML;
                setHidden(descriptionElement, false);
            } else {
                descriptionElement.innerHTML = "";
                setHidden(descriptionElement, true);
            }

            topicsList.innerHTML = "";
            if (topics.length > 0) {
                topics.forEach((text) => {
                    topicsList.appendChild(prepareTopicItem(text));
                });
                setHidden(topicsWrapper, false);
            } else {
                setHidden(topicsWrapper, true);
            }

            if (image instanceof HTMLImageElement && image.src) {
                imageElement.src = image.src;
                imageElement.alt = image.alt || title;
                setHidden(mediaWrapper, false);
            } else {
                imageElement.removeAttribute("src");
                imageElement.alt = "";
                setHidden(mediaWrapper, true);
            }

            purchaseButton.dataset.courseId = courseId;
            purchaseButton.dataset.courseTitle = title;
            purchaseButton.dataset.coursePrice = priceText;

            modal.hidden = false;
            requestAnimationFrame(() => {
                modal.classList.add("course-modal--active");
                modal.setAttribute("aria-hidden", "false");
                document.body.classList.add("course-modal-open");
                dialog.focus();
            });

            bindEscape();
            document.addEventListener("keydown", trapFocus);

            dispatchLifecycleEvent("courses:modal-open", {
                id: courseId,
                title: title,
                price: priceText,
                card: card
            });
        }

        function closeModal() {
            if (!modal.classList.contains("course-modal--active")) {
                return;
            }

            modal.classList.remove("course-modal--active");
            modal.setAttribute("aria-hidden", "true");
            document.body.classList.remove("course-modal-open");
            unbindEscape();
            document.removeEventListener("keydown", trapFocus);

            setTimeout(() => {
                if (!modal.classList.contains("course-modal--active")) {
                    modal.hidden = true;
                }
            }, 200);

            const detail = {
                id: purchaseButton.dataset.courseId || "",
                title: titleElement.textContent || "",
                price: priceElement.textContent || "",
                card: activeCard
            };

            dispatchLifecycleEvent("courses:modal-close", detail);

            if (lastFocusedElement && typeof lastFocusedElement.focus === "function") {
                lastFocusedElement.focus();
            }
        }

        function onCardActivate(event) {
            const card = event.currentTarget;
            if (!(card instanceof HTMLElement)) {
                return;
            }
            event.preventDefault();
            openModal(card);
        }

        function onCardKeydown(event) {
            if (event.defaultPrevented) {
                return;
            }
            if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                onCardActivate.call(event.currentTarget, event);
            }
        }

        function attachCardInteractions(card) {
            card.addEventListener("click", onCardActivate);
            card.addEventListener("keydown", onCardKeydown);
        }

        function detachCardInteractions(card) {
            card.removeEventListener("click", onCardActivate);
            card.removeEventListener("keydown", onCardKeydown);
        }

        Array.from(document.querySelectorAll("[data-course-card]"))
            .filter((card) => card instanceof HTMLElement)
            .forEach((card) => {
                attachCardInteractions(card);
            });

        closeButton.addEventListener("click", (event) => {
            event.preventDefault();
            closeModal();
        });

        modal.addEventListener("click", (event) => {
            if (event.target === modal) {
                closeModal();
            }
        });

        purchaseButton.addEventListener("click", () => {
            const detail = {
                id: purchaseButton.dataset.courseId || "",
                title: purchaseButton.dataset.courseTitle || "",
                price: purchaseButton.dataset.coursePrice || ""
            };
            dispatchLifecycleEvent("courses:purchase", detail);
        });

        window.addEventListener("courses:refresh", () => {
            Array.from(document.querySelectorAll("[data-course-card]"))
                .filter((card) => card instanceof HTMLElement)
                .forEach((card) => {
                    detachCardInteractions(card);
                    attachCardInteractions(card);
                });
        });
    });
})();

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

    function parseCourseDetails(card) {
        const dataNode = card.querySelector("[data-course-details]");
        if (!dataNode) {
            return null;
        }

        const raw = dataNode.textContent || "";
        const text = raw.trim();
        if (!text) {
            return null;
        }

        try {
            const parsed = JSON.parse(text);
            if (parsed && typeof parsed === "object") {
                return parsed;
            }
        } catch (error) {
            console.error("Failed to parse course details", error);
        }

        return null;
    }

    function prepareTopicItem(topic) {
        const item = document.createElement("li");
        item.className = "course-modal__topics-item";

        if (typeof topic === "string") {
            item.textContent = topic;
            return item;
        }

        if (!topic || typeof topic !== "object") {
            return item;
        }

        const title = document.createElement("h5");
        title.className = "course-modal__topic-title";
        title.textContent = typeof topic.title === "string" ? topic.title : "";
        item.appendChild(title);

        const metaParts = [];
        if (typeof topic.lesson_label === "string" && topic.lesson_label.trim() !== "") {
            metaParts.push(topic.lesson_label.trim());
        }

        if (typeof topic.duration_label === "string" && topic.duration_label.trim() !== "") {
            metaParts.push(topic.duration_label.trim());
        }

        if (metaParts.length > 0) {
            const meta = document.createElement("p");
            meta.className = "course-modal__topic-meta";
            meta.textContent = metaParts.join(" • ");
            item.appendChild(meta);
        }

        if (typeof topic.description_html === "string" && topic.description_html.trim() !== "") {
            const description = document.createElement("div");
            description.className = "course-modal__topic-description";
            description.innerHTML = topic.description_html;
            item.appendChild(description);
        }

        if (Array.isArray(topic.lessons) && topic.lessons.length > 0) {
            const lessonsList = document.createElement("ul");
            lessonsList.className = "course-modal__lessons-list";

            topic.lessons.forEach((lesson) => {
                if (!lesson || typeof lesson !== "object") {
                    return;
                }

                const lessonItem = document.createElement("li");
                lessonItem.className = "course-modal__lesson-item";

                const lessonTitle = document.createElement("span");
                lessonTitle.className = "course-modal__lesson-title";
                lessonTitle.textContent = typeof lesson.title === "string" ? lesson.title : "";
                lessonItem.appendChild(lessonTitle);

                if (typeof lesson.duration_label === "string" && lesson.duration_label.trim() !== "") {
                    const lessonDuration = document.createElement("span");
                    lessonDuration.className = "course-modal__lesson-duration";
                    lessonDuration.textContent = lesson.duration_label.trim();
                    lessonItem.appendChild(lessonDuration);
                }

                lessonsList.appendChild(lessonItem);
            });

            if (lessonsList.children.length > 0) {
                item.appendChild(lessonsList);
            }
        }

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
        const errorElement = modal.querySelector("[data-course-modal-error]");
        const titleElement = modal.querySelector("[data-course-modal-title]");
        const priceElement = modal.querySelector("[data-course-modal-price]");
        const descriptionElement = modal.querySelector("[data-course-modal-description]");
        const metaElement = modal.querySelector("[data-course-modal-meta]");
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

            const details = parseCourseDetails(card) || {};

            const title = typeof details.title === "string" && details.title.trim() !== ""
                ? details.title.trim()
                : card.querySelector(".post-card__link")?.textContent.trim() || "";

            const priceText = typeof details.price_text === "string" && details.price_text.trim() !== ""
                ? details.price_text.trim()
                : card.querySelector(".courses-list__price")?.textContent.trim() || "";

            const descriptionSource = typeof details.description_html === "string" && details.description_html.trim() !== ""
                ? details.description_html
                : null;

            const image = card.querySelector(".post-card__image");
            const topics = Array.isArray(details.topics) && details.topics.length > 0
                ? details.topics
                : collectCourseTopics(card);
            const courseId = card.getAttribute("data-course-id") || "";

            titleElement.textContent = title;

            if (priceText) {
                priceElement.textContent = priceText;
                setHidden(priceElement, false);
            } else {
                priceElement.textContent = "";
                setHidden(priceElement, true);
            }

            if (metaElement) {
                metaElement.innerHTML = "";
                if (Array.isArray(details.meta) && details.meta.length > 0) {
                    details.meta.forEach((metaText) => {
                        if (typeof metaText !== "string" || metaText.trim() === "") {
                            return;
                        }
                        const metaItem = document.createElement("li");
                        metaItem.className = "course-modal__meta-item";
                        metaItem.textContent = metaText.trim();
                        metaElement.appendChild(metaItem);
                    });
                    setHidden(metaElement, metaElement.children.length === 0);
                } else {
                    setHidden(metaElement, true);
                }
            }

            if (descriptionSource) {
                descriptionElement.innerHTML = descriptionSource;
                setHidden(descriptionElement, false);
            } else {
                const fallbackDescription = card.querySelector(".post-card__description");
                if (fallbackDescription) {
                    descriptionElement.innerHTML = fallbackDescription.innerHTML;
                    setHidden(descriptionElement, false);
                } else {
                    descriptionElement.innerHTML = "";
                    setHidden(descriptionElement, true);
                }
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

            const detailImageURL = typeof details.image_url === "string" && details.image_url.trim() !== ""
                ? details.image_url.trim()
                : "";
            const detailImageAlt = typeof details.image_alt === "string" && details.image_alt.trim() !== ""
                ? details.image_alt.trim()
                : "";

            if (detailImageURL) {
                imageElement.src = detailImageURL;
                imageElement.alt = detailImageAlt || title;
                setHidden(mediaWrapper, false);
            } else if (image instanceof HTMLImageElement && image.src) {
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
            purchaseButton.classList.remove("course-modal__purchase--loading");
            purchaseButton.removeAttribute("aria-busy");
            purchaseButton.disabled = false;

            if (errorElement) {
                errorElement.textContent = "";
                errorElement.hidden = true;
            }

            modal.hidden = false;
            requestAnimationFrame(() => {
                modal.classList.add("course-modal--active");
                modal.setAttribute("aria-hidden", "false");
                document.body.classList.add("course-modal-open");
                document.documentElement.classList.add("course-modal-open");
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
            document.documentElement.classList.remove("course-modal-open");
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
                price: purchaseButton.dataset.coursePrice || "",
                button: purchaseButton
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

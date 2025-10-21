(function () {
    "use strict";

    const isActivatingKey = (event) => {
        if (!event) {
            return false;
        }
        const { key } = event;
        return key === "Enter" || key === " ";
    };

    const shouldIgnoreEvent = (event) => {
        if (!event) {
            return false;
        }
        const target = event.target;
        if (!(target instanceof Element)) {
            return false;
        }
        return Boolean(target.closest("a, button, input, textarea, select, label"));
    };

    const navigateToPost = (card) => {
        if (!card) {
            return;
        }
        const url = card.dataset.postUrl;
        if (url) {
            window.location.href = url;
        }
    };

    const enhancePostCards = () => {
        const cards = document.querySelectorAll("[data-post-url]");
        if (!cards.length) {
            return;
        }

        cards.forEach((card) => {
            if (!(card instanceof HTMLElement)) {
                return;
            }

            const handleClick = (event) => {
                if (shouldIgnoreEvent(event)) {
                    return;
                }
                navigateToPost(card);
            };

            const handleKeydown = (event) => {
                if (!isActivatingKey(event)) {
                    return;
                }
                if (shouldIgnoreEvent(event)) {
                    return;
                }
                event.preventDefault();
                navigateToPost(card);
            };

            card.addEventListener("click", handleClick);
            card.addEventListener("keydown", handleKeydown);
        });
    };

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", enhancePostCards);
    } else {
        enhancePostCards();
    }
})();
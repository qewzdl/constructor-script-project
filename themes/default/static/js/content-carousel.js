(function () {
    "use strict";

    const selectCarousels = () => Array.from(document.querySelectorAll("[data-carousel]"));

    const parseColumns = (root) => {
        if (!(root instanceof HTMLElement)) {
            return 1;
        }
        const attr = parseInt(root.dataset.carouselColumns || "", 10);
        if (Number.isFinite(attr) && attr > 0) {
            return attr;
        }
        const cssValue = parseFloat(getComputedStyle(root).getPropertyValue("--carousel-columns"));
        if (Number.isFinite(cssValue) && cssValue > 0) {
            return Math.round(cssValue);
        }
        return 1;
    };

    const measure = (track) => {
        if (!(track instanceof HTMLElement)) {
            return null;
        }
        const slides = Array.from(track.querySelectorAll(".content-carousel__slide"));
        if (!slides.length || !(slides[0] instanceof HTMLElement)) {
            return null;
        }
        const { width } = slides[0].getBoundingClientRect();
        const gapValue = parseFloat(getComputedStyle(track).gap || "0");
        const gap = Number.isFinite(gapValue) ? gapValue : 0;
        return { slides, width, gap };
    };

    const clampIndex = (index, metrics, columns) => {
        if (!metrics) {
            return 0;
        }
        const maxIndex = Math.max(0, metrics.slides.length - columns);
        if (index < 0) return 0;
        if (index > maxIndex) return maxIndex;
        return index;
    };

    const currentIndex = (track, metrics) => {
        if (!metrics) {
            return 0;
        }
        const step = metrics.width + metrics.gap;
        if (step <= 0) {
            return 0;
        }
        return Math.round(track.scrollLeft / step);
    };

    const scrollToIndex = (track, metrics, index) => {
        if (!metrics) {
            return;
        }
        const step = metrics.width + metrics.gap;
        const target = step * index;
        track.scrollTo({ left: target, behavior: "smooth" });
    };

    const updateButtons = (track, metrics, columns, prevBtn, nextBtn) => {
        if (!metrics) {
            if (prevBtn instanceof HTMLButtonElement) prevBtn.disabled = true;
            if (nextBtn instanceof HTMLButtonElement) nextBtn.disabled = true;
            return;
        }
        const step = metrics.width + metrics.gap;
        const maxIndex = Math.max(0, metrics.slides.length - columns);
        const position = step > 0 ? track.scrollLeft / step : 0;
        const atStart = position <= 0.1;
        const atEnd = position >= maxIndex - 0.1;
        if (prevBtn instanceof HTMLButtonElement) prevBtn.disabled = atStart;
        if (nextBtn instanceof HTMLButtonElement) nextBtn.disabled = atEnd;
    };

    const attachCarousel = (root) => {
        if (!(root instanceof HTMLElement)) return;
        if (root.dataset.carouselReady === "true") return;

        const track = root.querySelector("[data-carousel-track]");
        if (!(track instanceof HTMLElement)) return;
        const prevBtn = root.querySelector("[data-carousel-prev]");
        const nextBtn = root.querySelector("[data-carousel-next]");

        let metrics = measure(track);
        let columns = parseColumns(root);

        const refresh = () => {
            metrics = measure(track);
            columns = parseColumns(root);
            updateButtons(track, metrics, columns, prevBtn, nextBtn);
        };

        const go = (direction) => {
            refresh();
            if (!metrics) return;
            const target = clampIndex(currentIndex(track, metrics) + direction, metrics, columns);
            scrollToIndex(track, metrics, target);
        };

        if (prevBtn instanceof HTMLButtonElement) {
            prevBtn.addEventListener("click", () => go(-1));
        }
        if (nextBtn instanceof HTMLButtonElement) {
            nextBtn.addEventListener("click", () => go(1));
        }

        track.addEventListener(
            "scroll",
            () => updateButtons(track, metrics, columns, prevBtn, nextBtn),
            { passive: true },
        );
        window.addEventListener("resize", refresh);

        refresh();
        root.dataset.carouselReady = "true";
    };

    const init = () => {
        selectCarousels().forEach(attachCarousel);
    };

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", init);
    } else {
        init();
    }
})();

(() => {
    const selectors = [
        ".page-view__sections > section",
        ".page-view__section",
        ".post__section",
        "[data-scroll-reveal]",
    ];

    const animationOptions = new Set([
        "float-up",
        "fade-in",
        "slide-left",
        "zoom-in",
        "none",
    ]);
    const defaultAnimation = "float-up";

    const prefersReducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)");
    const supportsObserver = "IntersectionObserver" in window;

    const normaliseAnimation = (value) => {
        const normalised =
            typeof value === "string" ? value.trim().toLowerCase() : "";
        if (animationOptions.has(normalised)) {
            return normalised;
        }
        return defaultAnimation;
    };

    const collectSections = () => {
        const seen = new Set();
        const sections = [];

        selectors.forEach((selector) => {
            const nodes = Array.from(document.querySelectorAll(selector));
            nodes.forEach((node) => {
                if (!(node instanceof HTMLElement)) {
                    return;
                }
                if (node.dataset.revealInitialized === "true" || seen.has(node)) {
                    return;
                }
                seen.add(node);
                sections.push(node);
            });
        });

        return sections;
    };

    const showAll = (sections) => {
        sections.forEach((section) => section.classList.add("is-visible"));
    };

    const init = () => {
        const sections = collectSections();
        if (!sections.length) {
            return;
        }

        const revealable = [];
        sections.forEach((section, index) => {
            section.dataset.revealInitialized = "true";
            const animation = normaliseAnimation(section.dataset.sectionAnimation);
            section.dataset.sectionAnimation = animation;
            const blurEnabled =
                section.dataset.sectionAnimationBlur !== "false" &&
                section.dataset.sectionAnimationBlur !== "0";

            if (animation === "none") {
                section.classList.add("section-reveal--none", "is-visible");
                return;
            }

            section.classList.add("section-reveal", `section-reveal--${animation}`);
            if (!blurEnabled) {
                section.classList.add("section-reveal--no-blur");
            }
            if (!section.style.getPropertyValue("--section-reveal-delay")) {
                const delay = Math.min(index * 70, 420);
                section.style.setProperty("--section-reveal-delay", `${delay}ms`);
            }
            revealable.push(section);
        });

        if (!revealable.length) {
            return;
        }

        if (prefersReducedMotion.matches || !supportsObserver) {
            showAll(revealable);
            return;
        }

        const observer = new IntersectionObserver(
            (entries, obs) => {
                entries.forEach((entry) => {
                    if (entry.isIntersecting) {
                        entry.target.classList.add("is-visible");
                        obs.unobserve(entry.target);
                    }
                });
            },
            {
                threshold: 0.15,
                rootMargin: "0px 0px -10% 0px",
            }
        );

        revealable.forEach((section) => observer.observe(section));

        const handleMotionChange = (event) => {
            if (!event.matches) {
                return;
            }
            showAll(revealable);
            observer.disconnect();
        };

        if (typeof prefersReducedMotion.addEventListener === "function") {
            prefersReducedMotion.addEventListener("change", handleMotionChange);
        } else if (typeof prefersReducedMotion.addListener === "function") {
            prefersReducedMotion.addListener(handleMotionChange);
        }
    };

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", init);
    } else {
        init();
    }
})();

(() => {
    const IMMERSIVE_GROUP_SELECTOR =
        ".page-view__section-group--background.page-view__section-group--style-immersive";
    const HERO_STAGE_SELECTOR =
        ".page-view__section--hero.page-view__section--variation-immersive .page-view__hero-stage";
    const GLOW_CLASS = "page-view__hero-cursor-glow";

    const supportsFinePointer =
        typeof window !== "undefined" &&
        typeof window.matchMedia === "function" &&
        window.matchMedia("(hover: hover) and (pointer: fine)").matches;

    if (!supportsFinePointer) {
        return;
    }

    const createCursorGlow = (target) => {
        if (!(target instanceof HTMLElement)) {
            return null;
        }
        if (target.querySelector(`.${GLOW_CLASS}`)) {
            return target.querySelector(`.${GLOW_CLASS}`);
        }

        const glow = document.createElement("span");
        glow.className = GLOW_CLASS;
        glow.setAttribute("aria-hidden", "true");
        target.appendChild(glow);
        return glow;
    };

    const attachTracking = (target) => {
        const glow = createCursorGlow(target);
        if (!glow) {
            return;
        }

        const rect = () => target.getBoundingClientRect();
        const clamp = (value, min, max) => Math.min(Math.max(value, min), max);
        const clampVelocity = (value, maxAbs) => clamp(value, -maxAbs, maxAbs);

        let bounds = rect();
        let rawTargetX = bounds.width / 2;
        let rawTargetY = bounds.height / 2;
        let delayedTargetX = rawTargetX;
        let delayedTargetY = rawTargetY;
        let currentX = delayedTargetX;
        let currentY = delayedTargetY;
        let velocityX = 0;
        let velocityY = 0;
        let targetOpacity = 0;
        let currentOpacity = 0;
        let targetScale = 1.22;
        let currentScale = targetScale;
        const targetLag = 0.11;
        const acceleration = 0.045;
        const velocityDamping = 0.82;
        const maxVelocity = 26;

        const syncBounds = () => {
            bounds = rect();
            rawTargetX = clamp(rawTargetX, 0, bounds.width);
            rawTargetY = clamp(rawTargetY, 0, bounds.height);
            delayedTargetX = clamp(delayedTargetX, 0, bounds.width);
            delayedTargetY = clamp(delayedTargetY, 0, bounds.height);
            currentX = clamp(currentX, 0, bounds.width);
            currentY = clamp(currentY, 0, bounds.height);
        };

        const setTargetFromPointer = (event) => {
            const x = event.clientX - bounds.left;
            const y = event.clientY - bounds.top;
            rawTargetX = clamp(x, 0, bounds.width);
            rawTargetY = clamp(y, 0, bounds.height);
            return { x: rawTargetX, y: rawTargetY };
        };

        const syncToPointer = (event) => {
            const { x, y } = setTargetFromPointer(event);
            delayedTargetX = x;
            delayedTargetY = y;
            currentX = x;
            currentY = y;
            velocityX = 0;
            velocityY = 0;
        };

        const renderGlow = () => {
            glow.style.transform = `translate3d(${currentX}px, ${currentY}px, 0) translate(-50%, -50%) scale(${currentScale})`;
            glow.style.opacity = String(currentOpacity);
        };

        const handleEnter = (event) => {
            syncBounds();
            syncToPointer(event);
            targetOpacity = 1;
            targetScale = 1;
            currentScale = targetScale;
            currentOpacity = Math.max(currentOpacity, 0.1);
            renderGlow();
        };

        const handleMove = (event) => {
            setTargetFromPointer(event);
            targetOpacity = 1;
            targetScale = 1;
        };

        const handleLeave = () => {
            targetOpacity = 0;
            targetScale = 1.22;
        };

        target.addEventListener("pointerenter", handleEnter);
        target.addEventListener("pointermove", handleMove);
        target.addEventListener("pointerleave", handleLeave);
        target.addEventListener("pointercancel", handleLeave);
        window.addEventListener("resize", syncBounds);
        window.addEventListener("scroll", syncBounds, { passive: true });

        const animate = () => {
            delayedTargetX += (rawTargetX - delayedTargetX) * targetLag;
            delayedTargetY += (rawTargetY - delayedTargetY) * targetLag;

            velocityX += (delayedTargetX - currentX) * acceleration;
            velocityY += (delayedTargetY - currentY) * acceleration;
            velocityX = clampVelocity(velocityX * velocityDamping, maxVelocity);
            velocityY = clampVelocity(velocityY * velocityDamping, maxVelocity);

            currentX = clamp(currentX + velocityX, 0, bounds.width);
            currentY = clamp(currentY + velocityY, 0, bounds.height);
            currentOpacity += (targetOpacity - currentOpacity) * 0.08;
            currentScale += (targetScale - currentScale) * 0.1;

            renderGlow();

            if (target.isConnected) {
                window.requestAnimationFrame(animate);
            }
        };

        window.requestAnimationFrame(animate);
    };

    const init = () => {
        const groupTargets = Array.from(
            document.querySelectorAll(IMMERSIVE_GROUP_SELECTOR)
        );
        const fallbackStageTargets = Array.from(
            document.querySelectorAll(HERO_STAGE_SELECTOR)
        ).filter((stage) => !stage.closest(IMMERSIVE_GROUP_SELECTOR));

        [...groupTargets, ...fallbackStageTargets].forEach(attachTracking);
    };

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", init, { once: true });
    } else {
        init();
    }
})();

(function () {
    "use strict";

    function ready(fn) {
        if (document.readyState === "loading") {
            document.addEventListener("DOMContentLoaded", fn, { once: true });
        } else {
            fn();
        }
    }

    ready(() => {
        const modal = document.querySelector("[data-course-modal]");
        if (!modal) {
            return;
        }

        const checkoutEnabled = modal.getAttribute("data-course-checkout-enabled") === "true";
        const endpoint = modal.getAttribute("data-course-checkout-endpoint") || "/api/v1/courses/checkout";
        const publishableKey = modal.getAttribute("data-course-checkout-publishable-key") || "";
        const errorElement = modal.querySelector("[data-course-modal-error]");

        const CSRF_COOKIE_NAME = "csrf_token";

        function readCookie(name) {
            if (!name || typeof document?.cookie !== "string") {
                return "";
            }
            const cookies = document.cookie.split("; ");
            for (let index = 0; index < cookies.length; index += 1) {
                const cookie = cookies[index];
                if (!cookie) {
                    continue;
                }
                const [key, ...rest] = cookie.split("=");
                if (key === name) {
                    return decodeURIComponent(rest.join("="));
                }
            }
            return "";
        }

        function getCSRFToken() {
            return readCookie(CSRF_COOKIE_NAME);
        }

        function resolveButton(detail) {
            if (detail && detail.button instanceof HTMLButtonElement) {
                return detail.button;
            }
            return modal.querySelector("[data-course-modal-purchase]");
        }

        function showError(message) {
            if (!errorElement) {
                return;
            }
            if (message) {
                errorElement.textContent = message;
                errorElement.hidden = false;
            } else {
                errorElement.textContent = "";
                errorElement.hidden = true;
            }
        }

        function clearError() {
            showError("");
        }

        function setButtonLoading(button, loading) {
            const target = button || resolveButton(null);
            if (!(target instanceof HTMLButtonElement)) {
                return;
            }
            if (loading) {
                if (!target.dataset.originalLabel) {
                    target.dataset.originalLabel = target.textContent.trim();
                }
                const loadingLabel = target.getAttribute("data-course-modal-loading-label") || "Processing...";
                target.textContent = loadingLabel;
                target.disabled = true;
                target.classList.add("course-modal__purchase--loading");
                target.setAttribute("aria-busy", "true");
            } else {
                if (target.dataset.originalLabel) {
                    target.textContent = target.dataset.originalLabel;
                }
                target.disabled = false;
                target.classList.remove("course-modal__purchase--loading");
                target.removeAttribute("aria-busy");
            }
        }

        let stripePromise = null;
        function loadStripe() {
            if (!publishableKey) {
                return Promise.resolve(null);
            }
            if (typeof window.Stripe === "function") {
                try {
                    return Promise.resolve(window.Stripe(publishableKey));
                } catch (err) {
                    return Promise.reject(err);
                }
            }
            if (stripePromise) {
                return stripePromise;
            }
            stripePromise = new Promise((resolve, reject) => {
                const script = document.createElement("script");
                script.src = "https://js.stripe.com/v3";
                script.async = true;
                script.onload = () => {
                    try {
                        if (typeof window.Stripe === "function") {
                            resolve(window.Stripe(publishableKey));
                        } else {
                            resolve(null);
                        }
                    } catch (error) {
                        reject(error);
                    }
                };
                script.onerror = () => {
                    reject(new Error("Failed to load Stripe.js"));
                };
                document.head.appendChild(script);
            });
            return stripePromise;
        }

        async function redirectToCheckout(sessionId, checkoutURL) {
            if (publishableKey && sessionId) {
                try {
                    const stripe = await loadStripe();
                    if (stripe) {
                        const result = await stripe.redirectToCheckout({ sessionId: sessionId });
                        if (!result || !result.error) {
                            return true;
                        }
                        console.error("Stripe redirect error", result.error.message);
                    }
                } catch (stripeError) {
                    console.error("Failed to use Stripe.js", stripeError);
                }
            }

            if (checkoutURL) {
                window.location.assign(checkoutURL);
                return true;
            }

            return false;
        }

        let isProcessing = false;

        modal.addEventListener("courses:modal-open", () => {
            isProcessing = false;
            clearError();
            setButtonLoading(resolveButton(null), false);
        });

        modal.addEventListener("courses:modal-close", () => {
            isProcessing = false;
            setButtonLoading(resolveButton(null), false);
        });

        if (!checkoutEnabled) {
            modal.addEventListener("courses:purchase", (event) => {
                clearError();
                setButtonLoading(resolveButton(event.detail), false);
                showError("Course checkout is currently unavailable. Please try again later.");
            });
            return;
        }

        modal.addEventListener("courses:purchase", async (event) => {
            const detail = event.detail || {};
            const button = resolveButton(detail);

            if (isProcessing) {
                return;
            }

            isProcessing = true;
            clearError();
            setButtonLoading(button, true);

            const courseId = detail.id;
            if (!courseId) {
                showError("Unable to start checkout: course id is missing.");
                setButtonLoading(button, false);
                isProcessing = false;
                return;
            }

            const payload = {
                package_id: Number(courseId)
            };
            if (Number.isNaN(payload.package_id) || payload.package_id <= 0) {
                showError("Invalid course id.");
                setButtonLoading(button, false);
                isProcessing = false;
                return;
            }

            try {
                const headers = {
                    "Content-Type": "application/json"
                };
                const csrfToken = getCSRFToken();
                if (csrfToken) {
                    headers["X-CSRF-Token"] = csrfToken;
                }

                const response = await fetch(endpoint, {
                    method: "POST",
                    headers: headers,
                    credentials: "same-origin",
                    body: JSON.stringify(payload)
                });

                if (!response.ok) {
                    let message = "Unable to start checkout. Please try again.";
                    if (response.status === 401) {
                        message = "Please sign in to purchase this course.";
                    } else if (response.status === 503) {
                        message = "Checkout is temporarily unavailable. Please try again later.";
                    } else if (response.status === 409) {
                        message = "You already own this course.";
                    }
                    try {
                        const errorPayload = await response.json();
                        if (errorPayload && typeof errorPayload.error === "string" && errorPayload.error.trim() !== "") {
                            message = errorPayload.error;
                        }
                    } catch (parseError) {
                        console.error("Failed to parse checkout error response", parseError);
                    }
                    throw new Error(message);
                }

                const data = await response.json();
                const sessionId = data && (data.session_id || data.sessionId);
                const checkoutURL = data && (data.checkout_url || data.checkoutUrl);

                const redirected = await redirectToCheckout(sessionId, checkoutURL);
                if (!redirected) {
                    throw new Error("Unable to redirect to payment. Please try again.");
                }
            } catch (error) {
                console.error("Failed to start course checkout", error);
                showError(error && error.message ? error.message : "Unable to start checkout. Please try again.");
            } finally {
                setButtonLoading(button, false);
                isProcessing = false;
            }
        });
    });
})();

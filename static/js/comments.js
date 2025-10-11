(() => {
    const formatTimestamp = (value) => {
        if (!value) {
            return {
                iso: "",
                label: "",
            };
        }

        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            return {
                iso: "",
                label: value,
            };
        }

        return {
            iso: date.toISOString(),
            label: date.toLocaleString(undefined, {
                dateStyle: "medium",
                timeStyle: "short",
            }),
        };
    };

    const escapeHTML = (value) =>
        String(value)
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/"/g, "&quot;")
            .replace(/'/g, "&#39;");

    const renderContentHTML = (text) => {
        if (!text) {
            return "";
        }
        return escapeHTML(text).replace(/\r?\n/g, "<br />");
    };

    const createReplyList = () => {
        const repliesList = document.createElement("ol");
        repliesList.className = "comments__replies";
        repliesList.dataset.commentsReplies = "";
        return repliesList;
    };

    const createCommentElement = (comment, { isReply = false, canReply = false } = {}) => {
        const item = document.createElement("li");
        item.className = "comments__item";
        if (isReply) {
            item.classList.add("comments__item--reply");
        }
        item.dataset.commentId = String(comment.id);

        const card = document.createElement("article");
        card.className = "comments__card";

        const header = document.createElement("header");
        header.className = "comments__header";

        const author = document.createElement("span");
        author.className = "comments__author";
        author.textContent = comment.author?.username || comment.author_name || "Anonymous";
        header.appendChild(author);

        const timeInfo = formatTimestamp(comment.created_at || comment.createdAt);
        if (timeInfo.label) {
            const time = document.createElement("time");
            time.className = "comments__time";
            if (timeInfo.iso) {
                time.dateTime = timeInfo.iso;
            }
            time.textContent = timeInfo.label;
            header.appendChild(time);
        }

        card.appendChild(header);

        const content = document.createElement("div");
        content.className = "comments__content";
        content.innerHTML = renderContentHTML(comment.content);
        card.appendChild(content);

        if (canReply) {
            const actions = document.createElement("div");
            actions.className = "comments__actions";

            const replyButton = document.createElement("button");
            replyButton.type = "button";
            replyButton.className = "comments__reply-button";
            replyButton.dataset.action = "reply";
            replyButton.dataset.commentId = String(comment.id);
            replyButton.dataset.author = author.textContent || "Anonymous";
            replyButton.textContent = "Reply";

            actions.appendChild(replyButton);
            card.appendChild(actions);
        }

        item.appendChild(card);

        const repliesList = createReplyList();
        if (Array.isArray(comment.replies)) {
            comment.replies.forEach((reply) => {
                repliesList.appendChild(
                    createCommentElement(reply, {
                        isReply: true,
                        canReply,
                    })
                );
            });
        }
        item.appendChild(repliesList);

        return item;
    };

    const ensureRepliesList = (commentElement) => {
        const existing = commentElement.querySelector("[data-comments-replies]");
        if (existing) {
            return existing;
        }
        const repliesList = createReplyList();
        commentElement.appendChild(repliesList);
        return repliesList;
    };

    const defaultToggleDisabled = (form, disabled) => {
        if (!form) {
            return;
        }
        const elements = form.querySelectorAll("input, button, select, textarea");
        elements.forEach((element) => {
            element.disabled = disabled;
        });
        if (disabled) {
            form.classList.add("is-disabled");
        } else {
            form.classList.remove("is-disabled");
        }
    };

    const fallbackAlert = (target, message, type = "info") => {
        const element = typeof target === "string" ? document.getElementById(target) : target;
        if (!element) {
            return;
        }

        element.classList.remove("is-error", "is-success", "is-info");

        if (!message) {
            element.hidden = true;
            element.textContent = "";
            return;
        }

        const statusClass =
            type === "error" ? "is-error" : type === "success" ? "is-success" : "is-info";
        element.classList.add(statusClass);
        element.hidden = false;
        element.textContent = message;
    };

    document.addEventListener("DOMContentLoaded", () => {
        const commentsSection = document.querySelector("[data-comments]");
        if (!commentsSection) {
            return;
        }

        const app = window.App || {};
        const auth = app.auth;
        const apiRequest =
            app.apiRequest ||
            (async (url, options = {}) => {
                const headers = Object.assign({}, options.headers || {});
                const token =
                    auth && typeof auth.getToken === "function" ? auth.getToken() : undefined;

                if (options.body && !(options.body instanceof FormData)) {
                    headers["Content-Type"] = headers["Content-Type"] || "application/json";
                }

                if (token) {
                    headers.Authorization = headers.Authorization || `Bearer ${token}`;
                }

                const response = await fetch(url, {
                    credentials: "include",
                    ...options,
                    headers,
                });

                const contentType = response.headers.get("content-type") || "";
                const isJson = contentType.includes("application/json");
                const payload = isJson
                    ? await response.json().catch(() => null)
                    : await response.text();

                if (!response.ok) {
                    const error = new Error(
                        payload && typeof payload === "object" && payload.error
                            ? payload.error
                            : typeof payload === "string"
                            ? payload
                            : "Request failed"
                    );
                    error.status = response.status;
                    error.payload = payload;
                    throw error;
                }

                return payload;
            });
        const setAlert = typeof app.setAlert === "function" ? app.setAlert : fallbackAlert;
        const toggleFormDisabled =
            typeof app.toggleFormDisabled === "function"
                ? app.toggleFormDisabled
                : defaultToggleDisabled;

        const form = document.getElementById("comment-form");
        const alertElement = document.getElementById("comment-alert");
        const countElement = commentsSection.querySelector("[data-comment-count]");
        const emptyState = commentsSection.querySelector("[data-comments-empty]");
        const commentsList = commentsSection.querySelector("[data-comments-list]");
        const parentInput = form ? form.querySelector('input[name="parent_id"]') : null;
        const replyContext = form
            ? form.querySelector("[data-reply-context]")
            : null;
        const replyTarget = form ? form.querySelector("[data-reply-to]") : null;
        const textarea = form ? form.querySelector('textarea[name="content"]') : null;

        const updateCount = (delta) => {
            if (!countElement) {
                return;
            }
            const current = parseInt(countElement.textContent || "0", 10);
            const next = Number.isNaN(current) ? delta : current + delta;
            countElement.textContent = Math.max(next, 0);
        };

        const clearReplyState = () => {
            if (parentInput) {
                parentInput.value = "";
            }
            if (replyContext) {
                replyContext.hidden = true;
            }
            if (replyTarget) {
                replyTarget.textContent = "";
            }
        };

        const startReply = (button) => {
            if (!form || !parentInput) {
                return;
            }

            const token = auth && typeof auth.getToken === "function" ? auth.getToken() : null;
            if (!token) {
                setAlert(alertElement, "Please sign in to reply to a comment.", "error");
                return;
            }

            parentInput.value = button.dataset.commentId || "";
            if (replyContext) {
                replyContext.hidden = false;
            }
            if (replyTarget) {
                replyTarget.textContent = button.dataset.author || "this comment";
            }
            if (textarea) {
                textarea.focus();
            }
        };

        const insertComment = (comment) => {
            if (!commentsList) {
                return;
            }

            const canReply = Boolean(form);
            const parentId = comment.parent_id || comment.parentId;
            const commentElement = createCommentElement(comment, {
                isReply: Boolean(parentId),
                canReply,
            });

            if (parentId) {
                const parentNode = commentsSection.querySelector(
                    `[data-comment-id="${parentId}"]`
                );
                if (parentNode) {
                    const repliesList = ensureRepliesList(parentNode);
                    repliesList.appendChild(commentElement);
                } else {
                    commentsList.appendChild(commentElement);
                }
            } else {
                commentsList.appendChild(commentElement);
            }

            if (emptyState) {
                emptyState.hidden = true;
            }
        };

        const handleSubmit = async (event) => {
            event.preventDefault();
            if (!form) {
                return;
            }

            const token = auth && typeof auth.getToken === "function" ? auth.getToken() : null;
            if (!token) {
                setAlert(alertElement, "Please sign in to post a comment.", "error");
                return;
            }

            const formData = new FormData(form);
            const content = (formData.get("content") || "").toString().trim();
            if (!content) {
                setAlert(alertElement, "Please write a comment before posting.", "error");
                return;
            }

            const body = { content };
            const parentValue = parentInput ? parentInput.value.trim() : "";
            if (parentValue) {
                body.parent_id = Number(parentValue);
            }

            setAlert(alertElement, "");
            toggleFormDisabled(form, true);

            try {
                const payload = await apiRequest(form.dataset.action, {
                    method: "POST",
                    body: JSON.stringify(body),
                });

                if (!payload || !payload.comment) {
                    throw new Error("Unexpected server response. Please try again.");
                }

                insertComment(payload.comment);
                updateCount(1);
                form.reset();
                clearReplyState();
                setAlert(alertElement, "Your comment has been posted.", "success");
            } catch (error) {
                if (error && error.status === 401) {
                    setAlert(alertElement, "Your session expired. Please sign in again.", "error");
                } else {
                    setAlert(
                        alertElement,
                        error && error.message ? error.message : "Failed to post comment.",
                        "error"
                    );
                }
            } finally {
                toggleFormDisabled(form, false);
            }
        };

        if (form) {
            form.addEventListener("submit", handleSubmit);
        }

        commentsSection.addEventListener("click", (event) => {
            const target = event.target;
            if (!(target instanceof HTMLElement)) {
                return;
            }

            if (target.dataset.action === "reply") {
                event.preventDefault();
                startReply(target);
                return;
            }

            if (target.dataset.action === "cancel-reply") {
                event.preventDefault();
                clearReplyState();
                setAlert(alertElement, "", "info");
            }
        });
    });
})();
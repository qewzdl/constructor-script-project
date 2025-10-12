(() => {
    let currentUserId = null;
    let isAdmin = false;
    let commentEndpoint = "/api/v1/comments";

    const getAuthorIdFromComment = (comment) => {
        if (!comment) {
            return null;
        }

        const value = comment.author_id ?? comment.authorId ?? comment.authorID ?? null;
        if (value === null || value === undefined) {
            return null;
        }

        const parsed = Number(value);
        if (!Number.isFinite(parsed) || parsed <= 0) {
            return null;
        }

        return parsed;
    };

    const canModifyComment = (comment) => {
        if (isAdmin) {
            return true;
        }

        if (currentUserId === null) {
            return false;
        }

        const authorId = getAuthorIdFromComment(comment);
        if (authorId === null) {
            return false;
        }

        return authorId === currentUserId;
    };

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

    const createCommentElement = (
        comment,
        { isReply = false, canReply = false, canEdit = false, canDelete = false } = {}
    ) => {
        const item = document.createElement("li");
        item.className = "comments__item";
        if (isReply) {
            item.classList.add("comments__item--reply");
        }
        item.dataset.commentId = String(comment.id);

        const authorId = getAuthorIdFromComment(comment);
        if (authorId !== null) {
            item.dataset.authorId = String(authorId);
        }

        item.dataset.commentRaw = typeof comment.content === "string" ? comment.content : "";

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

        const actions = document.createElement("div");
        actions.className = "comments__actions";
        let hasActions = false;

        if (canReply) {
            const replyButton = document.createElement("button");
            replyButton.type = "button";
            replyButton.className = "comments__reply-button";
            replyButton.dataset.action = "reply";
            replyButton.dataset.commentId = String(comment.id);
            replyButton.dataset.author = author.textContent || "Anonymous";
            replyButton.textContent = "Reply";

            actions.appendChild(replyButton);
            hasActions = true;
        }

        if (canEdit) {
            const editButton = document.createElement("button");
            editButton.type = "button";
            editButton.className = "comments__edit-button";
            editButton.dataset.action = "edit";
            editButton.dataset.commentId = String(comment.id);
            editButton.textContent = "Edit";

            actions.appendChild(editButton);
            hasActions = true;
        }

        if (canDelete) {
            const deleteButton = document.createElement("button");
            deleteButton.type = "button";
            deleteButton.className = "comments__delete-button";
            deleteButton.dataset.action = "delete";
            deleteButton.dataset.commentId = String(comment.id);
            deleteButton.textContent = "Delete";

            actions.appendChild(deleteButton);
            hasActions = true;
        }

        if (hasActions) {
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
                        canEdit: canModifyComment(reply),
                        canDelete: canModifyComment(reply),
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

        if (commentsSection.dataset.commentEndpoint) {
            commentEndpoint = commentsSection.dataset.commentEndpoint;
        }

        const userIdValue = commentsSection.dataset.currentUserId || "";
        if (userIdValue) {
            const parsedUserId = Number(userIdValue);
            currentUserId = Number.isFinite(parsedUserId) && parsedUserId > 0 ? parsedUserId : null;
        } else {
            currentUserId = null;
        }

        isAdmin = commentsSection.dataset.isAdmin === "true";

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

        const finishEditComment = (commentElement) => {
            if (!commentElement) {
                return;
            }

            const editForm = commentElement.querySelector(".comments__edit-form");
            if (editForm) {
                editForm.remove();
            }

            const contentElement = commentElement.querySelector(".comments__content");
            if (contentElement) {
                contentElement.hidden = false;
            }

            delete commentElement.dataset.editing;
        };

        const updateCommentElement = (commentElement, comment) => {
            if (!commentElement || !comment) {
                return;
            }

            const rawContent = typeof comment.content === "string" ? comment.content : "";
            commentElement.dataset.commentRaw = rawContent;

            const contentElement = commentElement.querySelector(".comments__content");
            if (contentElement) {
                contentElement.innerHTML = renderContentHTML(rawContent);
            }

            const authorElement = commentElement.querySelector(".comments__author");
            if (authorElement && comment.author && comment.author.username) {
                authorElement.textContent = comment.author.username;
            }

            const authorId = getAuthorIdFromComment(comment);
            if (authorId !== null) {
                commentElement.dataset.authorId = String(authorId);
            }

            const timeElement = commentElement.querySelector(".comments__time");
            if (timeElement) {
                const timestamp =
                    comment.updated_at ||
                    comment.updatedAt ||
                    comment.created_at ||
                    comment.createdAt ||
                    null;
                const timeInfo = formatTimestamp(timestamp);
                if (timeInfo.iso) {
                    timeElement.dateTime = timeInfo.iso;
                }
                if (timeInfo.label) {
                    timeElement.textContent = timeInfo.label;
                }
            }
        };

        const handleEditSubmit = async (event, commentElement) => {
            event.preventDefault();

            const formElement = event.currentTarget;
            if (!(formElement instanceof HTMLFormElement) || !commentElement) {
                return;
            }

            const textareaElement = formElement.querySelector("textarea[name=\"content\"]");
            const contentValue = textareaElement ? textareaElement.value.trim() : "";

            if (!contentValue) {
                setAlert(alertElement, "Please write a comment before saving.", "error");
                return;
            }

            const commentId = Number(commentElement.dataset.commentId || 0);
            if (!commentId) {
                setAlert(alertElement, "Unable to update this comment.", "error");
                return;
            }

            toggleFormDisabled(formElement, true);

            try {
                const payload = await apiRequest(`${commentEndpoint}/${commentId}`, {
                    method: "PUT",
                    body: JSON.stringify({ content: contentValue }),
                });

                if (!payload || !payload.comment) {
                    throw new Error("Unexpected server response. Please try again.");
                }

                updateCommentElement(commentElement, payload.comment);
                finishEditComment(commentElement);
                setAlert(alertElement, "Your comment has been updated.", "success");
            } catch (error) {
                if (error && error.status === 401) {
                    setAlert(alertElement, "Your session expired. Please sign in again.", "error");
                } else if (error && error.status === 403) {
                    setAlert(alertElement, "You can only edit your own comments.", "error");
                } else if (error && error.message) {
                    setAlert(alertElement, error.message, "error");
                } else {
                    setAlert(alertElement, "Failed to update comment.", "error");
                }
            } finally {
                toggleFormDisabled(formElement, false);
            }
        };

        const startEditComment = (button) => {
            const commentElement = button.closest("[data-comment-id]");
            if (!commentElement || commentElement.dataset.editing === "true") {
                return;
            }

            const contentElement = commentElement.querySelector(".comments__content");
            if (!contentElement) {
                return;
            }

            commentElement.dataset.editing = "true";

            const formElement = document.createElement("form");
            formElement.className = "comments__edit-form";

            const textareaElement = document.createElement("textarea");
            textareaElement.name = "content";
            textareaElement.required = true;
            textareaElement.rows = 4;
            textareaElement.className = "comments__edit-textarea";
            textareaElement.value = commentElement.dataset.commentRaw || "";

            const actionsElement = document.createElement("div");
            actionsElement.className = "comments__edit-actions";

            const saveButton = document.createElement("button");
            saveButton.type = "submit";
            saveButton.className = "comments__edit-save";
            saveButton.textContent = "Save";

            const cancelButton = document.createElement("button");
            cancelButton.type = "button";
            cancelButton.className = "comments__edit-cancel";
            cancelButton.dataset.action = "cancel-edit";
            cancelButton.textContent = "Cancel";

            actionsElement.appendChild(saveButton);
            actionsElement.appendChild(cancelButton);

            formElement.appendChild(textareaElement);
            formElement.appendChild(actionsElement);

            formElement.addEventListener("submit", (event) => handleEditSubmit(event, commentElement));

            contentElement.hidden = true;
            contentElement.insertAdjacentElement("afterend", formElement);

            textareaElement.focus();
        };

        const cancelEditComment = (button) => {
            const commentElement = button.closest("[data-comment-id]");
            if (!commentElement) {
                return;
            }

            finishEditComment(commentElement);
        };

        const countNestedComments = (commentElement) => {
            if (!commentElement) {
                return 0;
            }

            const descendantItems = commentElement.querySelectorAll("[data-comment-id]");
            return 1 + descendantItems.length;
        };

        const handleDeleteComment = async (button) => {
            const commentElement = button.closest("li[data-comment-id]");
            if (!commentElement) {
                console.warn("Comment element not found for delete");
                return;
            }
        
            const commentId = Number(commentElement.dataset.commentId || 0);
            if (!commentId) {
                setAlert(alertElement, "Unable to delete this comment.", "error");
                return;
            }
        
            const confirmed = window.confirm(
                "Delete this comment? All of its replies will be removed as well."
            );
            if (!confirmed) {
                return;
            }
        
            button.disabled = true;
        
            try {
                await apiRequest(`${commentEndpoint}/${commentId}`, { method: "DELETE" });
        
                const removedCount = countNestedComments(commentElement);
                commentElement.remove(); 
                updateCount(-removedCount);
        
                if (parentInput && Number(parentInput.value) === commentId) {
                    clearReplyState();
                }
        
                if (commentsList && !commentsList.querySelector("[data-comment-id]")) {
                    if (emptyState) emptyState.hidden = false;
                }
        
                setAlert(alertElement, "Comment deleted.", "success");
            } catch (error) {
                setAlert(alertElement, error.message || "Failed to delete comment.", "error");
            } finally {
                button.disabled = false;
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
            const canManage = canModifyComment(comment);
            const commentElement = createCommentElement(comment, {
                isReply: Boolean(parentId),
                canReply,
                canEdit: canManage,
                canDelete: canManage,
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

            if (target.dataset.action === "edit") {
                event.preventDefault();
                startEditComment(target);
                return;
            }

            if (target.dataset.action === "delete") {
                event.preventDefault();
                handleDeleteComment(target);
                return;
            }

            if (target.dataset.action === "cancel-edit") {
                event.preventDefault();
                cancelEditComment(target);
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
(() => {
    const app = window.App || {};

    const getCookie = (name) => {
        const pattern = new RegExp(`(?:^|; )${name}=([^;]*)`);
        const match = document.cookie.match(pattern);
        return match ? decodeURIComponent(match[1]) : "";
    };

    const fallbackApiRequest = async (url, options = {}) => {
        const headers = Object.assign({}, options.headers || {});
        const method = (options.method || "GET").toUpperCase();
        const auth = app.auth;
        const token = auth && typeof auth.getToken === "function" ? auth.getToken() : "";

        if (options.body && !(options.body instanceof FormData)) {
            headers["Content-Type"] = headers["Content-Type"] || "application/json";
        }

        if (token) {
            headers.Authorization = `Bearer ${token}`;
        }

        if (["POST", "PUT", "PATCH", "DELETE"].includes(method)) {
            const csrfToken = getCookie("csrf_token");
            if (csrfToken) {
                headers["X-CSRF-Token"] = csrfToken;
            }
        }

        const response = await fetch(url, {
            credentials: "include",
            ...options,
            headers,
        });

        const contentType = response.headers.get("content-type") || "";
        const isJson = contentType.includes("application/json");
        const payload = isJson ? await response.json().catch(() => null) : await response.text();

        if (!response.ok) {
            let message = "Request failed";
            if (payload) {
                if (typeof payload === "string") {
                    message = payload;
                } else if (payload.error) {
                    message = payload.error;
                }
            }
            const error = new Error(message);
            error.status = response.status;
            error.payload = payload;
            throw error;
        }

        return payload;
    };

    const setAlert = typeof app.setAlert === "function"
        ? app.setAlert
        : (target, message, type = "info") => {
              const element = typeof target === "string" ? document.getElementById(target) : target;
              if (!element) {
                  if (message && type === "error") {
                      console.error(message);
                  }
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

    const toggleFormDisabled = typeof app.toggleFormDisabled === "function"
        ? app.toggleFormDisabled
        : (form, disabled) => {
              if (!form) {
                  return;
              }
              form.querySelectorAll("input, textarea, button, select").forEach((element) => {
                  element.disabled = disabled;
              });
              form.classList.toggle("is-disabled", disabled);
          };

    const apiRequest = typeof app.apiRequest === "function" ? app.apiRequest : fallbackApiRequest;

    const isAuthenticated = () => {
        if (document.body && document.body.dataset.authenticated === "true") {
            return true;
        }
        if (app.auth && typeof app.auth.getToken === "function") {
            return Boolean(app.auth.getToken());
        }
        return false;
    };

    const showAlert = (element, message, type = "info") => {
        if (!element) {
            if (message && type === "error") {
                console.error(message);
            }
            return;
        }
        setAlert(element, message, type);
    };

    const normalizeEndpoint = (value) => {
        if (typeof value !== "string") {
            return "";
        }
        return value.replace(/\/+$/, "");
    };

    const getNumber = (entry, ...keys) => {
        for (const key of keys) {
            if (entry && Object.prototype.hasOwnProperty.call(entry, key)) {
                const value = Number(entry[key]);
                if (Number.isFinite(value)) {
                    return value;
                }
            }
        }
        return 0;
    };

    const getString = (entry, ...keys) => {
        for (const key of keys) {
            if (entry && Object.prototype.hasOwnProperty.call(entry, key)) {
                const value = entry[key];
                if (value !== undefined && value !== null) {
                    return String(value);
                }
            }
        }
        return "";
    };

    const getAuthorName = (entry) => {
        if (!entry) {
            return "";
        }
        const author = entry.author || entry.Author || null;
        if (!author) {
            return "";
        }
        return author.username || author.Username || "";
    };

    const formatDateTime = (value) => {
        if (!value) {
            return { iso: "", label: "" };
        }
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            return { iso: "", label: String(value) };
        }
        let label = "";
        try {
            label = date.toLocaleString(undefined, {
                dateStyle: "medium",
                timeStyle: "short",
            });
        } catch (_error) {
            label = date.toISOString();
        }
        return { iso: date.toISOString(), label };
    };

    const buildAnswerElement = (answer) => {
        const answerId = getNumber(answer, "id", "ID");
        const rating = getNumber(answer, "rating", "Rating");
        const content = getString(answer, "content", "Content");
        const { iso, label } = formatDateTime(
            getString(answer, "created_at", "createdAt", "CreatedAt")
        );
        const authorName = getAuthorName(answer);

        const item = document.createElement("li");
        item.className = "forum-answer";
        if (answerId) {
            item.dataset.answerId = String(answerId);
        }

        const votes = document.createElement("div");
        votes.className = "forum-answer__votes";

        const upvote = document.createElement("button");
        upvote.type = "button";
        upvote.className = "forum-vote forum-vote--up";
        upvote.dataset.role = "answer-vote";
        upvote.dataset.value = "1";
        upvote.setAttribute("aria-label", "Upvote this answer");

        const ratingOutput = document.createElement("output");
        ratingOutput.className = "forum-vote__value";
        ratingOutput.dataset.role = "answer-rating";
        ratingOutput.textContent = String(rating);

        const downvote = document.createElement("button");
        downvote.type = "button";
        downvote.className = "forum-vote forum-vote--down";
        downvote.dataset.role = "answer-vote";
        downvote.dataset.value = "-1";
        downvote.setAttribute("aria-label", "Downvote this answer");

        votes.append(upvote, ratingOutput, downvote);

        const body = document.createElement("article");
        body.className = "forum-answer__body";

        const header = document.createElement("header");
        header.className = "forum-answer__meta";

        const author = document.createElement("span");
        author.className = "forum-answer__meta-item";
        author.textContent = authorName
            ? `Answered by ${authorName}`
            : "Community member";

        const time = document.createElement("time");
        time.className = "forum-answer__meta-item";
        if (iso) {
            time.setAttribute("datetime", iso);
        }
        time.textContent = label || "";

        header.append(author);
        if (label) {
            header.append(time);
        }

        const contentElement = document.createElement("div");
        contentElement.className = "forum-answer__content";
        contentElement.textContent = content;

        body.append(header, contentElement);
        item.append(votes, body);

        return item;
    };

    const focusableSelector =
        'a[href]:not([tabindex="-1"]), button:not([disabled]):not([tabindex="-1"]), input:not([disabled]):not([tabindex="-1"]), textarea:not([disabled]):not([tabindex="-1"]), select:not([disabled]):not([tabindex="-1"]), [tabindex]:not([tabindex="-1"])';
    const interactiveElementSelector =
        'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [role="button"]';

    const initForumTableNavigation = (root) => {
        const list = root.querySelector('[data-role="forum-list"]');
        if (!list) {
            return;
        }

        const rows = Array.from(list.querySelectorAll('.forum-table__row[data-question-url]'));
        rows.forEach((row) => {
            if (!(row instanceof HTMLElement)) {
                return;
            }

            if (row.dataset.linkEnhanced === "true") {
                return;
            }

            row.dataset.linkEnhanced = "true";

            row.addEventListener("click", (event) => {
                if (event.defaultPrevented) {
                    return;
                }

                const target = event.target;
                if (target instanceof Element && target.closest(interactiveElementSelector)) {
                    return;
                }

                const url = row.dataset.questionUrl;
                if (url) {
                    window.location.href = url;
                }
            });

            row.addEventListener("keydown", (event) => {
                if (event.defaultPrevented) {
                    return;
                }

                const target = event.target;
                if (target instanceof Element && target !== row && target.closest(interactiveElementSelector)) {
                    return;
                }

                if (event.key === "Enter" || event.key === " ") {
                    const url = row.dataset.questionUrl;
                    if (url) {
                        event.preventDefault();
                        window.location.href = url;
                    }
                }
            });
        });
    };

    const initForumList = (root) => {
        const modal = root.querySelector('[data-role="question-modal"]');
        const container = modal || root;
        const alertElement = container.querySelector('[data-role="forum-alert"]');
        const form = container.querySelector('[data-role="question-form"]');
        if (!form) {
            return;
        }

        const endpoint = normalizeEndpoint(root.dataset.endpointCreate || "");
        const loginURL = root.dataset.loginUrl || "/login";

        let lastFocusedElement = null;

        const getFocusableElements = () => {
            if (!modal) {
                return [];
            }
            return Array.from(modal.querySelectorAll(focusableSelector)).filter((element) => {
                if (!(element instanceof HTMLElement)) {
                    return false;
                }
                return !element.hasAttribute("hidden") && !element.closest("[hidden]");
            });
        };

        const openModal = () => {
            if (!modal) {
                return;
            }

            if (modal.classList.contains("forum-question-modal--active")) {
                return;
            }

            if (!isAuthenticated()) {
                window.location.href = loginURL;
                return;
            }

            lastFocusedElement =
                document.activeElement && document.activeElement instanceof HTMLElement
                    ? document.activeElement
                    : null;

            if (form.id && window.location.hash !== `#${form.id}`) {
                try {
                    history.replaceState(null, document.title, `#${form.id}`);
                } catch (_error) {
                    // Ignore history errors in unsupported environments.
                }
            }

            modal.hidden = false;
            modal.setAttribute("aria-hidden", "false");
            requestAnimationFrame(() => {
                modal.classList.add("forum-question-modal--active");
            });
            document.body.classList.add("forum-question-modal-open");
            document.documentElement.classList.add("forum-question-modal-open");

            const focusable = getFocusableElements();
            const preferredTarget = focusable.find((element) =>
                element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement
            );
            const initialFocus = preferredTarget || focusable[0];
            if (initialFocus) {
                initialFocus.focus({ preventScroll: true });
            }
        };

        const clearHash = () => {
            if (modal && window.location.hash === `#${form.id}`) {
                const url = `${window.location.pathname}${window.location.search}`;
                try {
                    history.replaceState(null, document.title, url);
                } catch (_error) {
                    // Ignore history errors in unsupported environments.
                }
            }
        };

        const closeModal = () => {
            if (!modal) {
                return;
            }

            if (!modal.classList.contains("forum-question-modal--active")) {
                clearHash();
                return;
            }

            modal.classList.remove("forum-question-modal--active");
            modal.setAttribute("aria-hidden", "true");
            document.body.classList.remove("forum-question-modal-open");
            document.documentElement.classList.remove("forum-question-modal-open");

            const handleTransitionEnd = (event) => {
                if (event.target === modal) {
                    modal.hidden = true;
                    modal.removeEventListener("transitionend", handleTransitionEnd);
                }
            };
            modal.addEventListener("transitionend", handleTransitionEnd);
            window.setTimeout(() => {
                modal.hidden = true;
                modal.removeEventListener("transitionend", handleTransitionEnd);
            }, 320);

            clearHash();

            if (lastFocusedElement && typeof lastFocusedElement.focus === "function") {
                lastFocusedElement.focus({ preventScroll: true });
            }
        };

        if (modal) {
            const openButtons = root.querySelectorAll('[data-role="question-modal-open"]');
            openButtons.forEach((button) => {
                button.addEventListener("click", (event) => {
                    event.preventDefault();
                    openModal();
                });
            });

            const closeButton = modal.querySelector('[data-role="question-modal-close"]');
            if (closeButton) {
                closeButton.addEventListener("click", (event) => {
                    event.preventDefault();
                    closeModal();
                });
            }

            modal.addEventListener("click", (event) => {
                if (event.target === modal) {
                    closeModal();
                }
            });

            modal.addEventListener("keydown", (event) => {
                if (event.key === "Escape") {
                    event.preventDefault();
                    closeModal();
                    return;
                }

                if (event.key !== "Tab") {
                    return;
                }

                const focusable = getFocusableElements();
                if (focusable.length === 0) {
                    return;
                }

                const first = focusable[0];
                const last = focusable[focusable.length - 1];
                const active = document.activeElement;

                if (event.shiftKey) {
                    if (active === first || !modal.contains(active)) {
                        event.preventDefault();
                        last.focus({ preventScroll: true });
                    }
                } else if (active === last) {
                    event.preventDefault();
                    first.focus({ preventScroll: true });
                }
            });

            const shouldOpenFromHash = () => `#${form.id}` === window.location.hash;
            if (shouldOpenFromHash()) {
                openModal();
            }

            window.addEventListener("hashchange", () => {
                if (shouldOpenFromHash()) {
                    openModal();
                }
            });
        }

        form.addEventListener("submit", async (event) => {
            event.preventDefault();
            showAlert(alertElement, "");

            if (!endpoint) {
                showAlert(alertElement, "Question submission is unavailable right now.", "error");
                return;
            }

            if (!isAuthenticated()) {
                window.location.href = loginURL;
                return;
            }

            const formData = new FormData(form);
            const title = (formData.get("title") || "").toString().trim();
            const content = (formData.get("content") || "").toString().trim();
            const categoryValue = (formData.get("category_id") || "").toString().trim();
            let categoryId = null;
            if (categoryValue !== "") {
                const parsed = Number(categoryValue);
                if (!Number.isFinite(parsed) || parsed <= 0) {
                    showAlert(alertElement, "Please choose a valid category.", "error");
                    return;
                }
                categoryId = parsed;
            }

            if (!title || !content) {
                showAlert(alertElement, "Please provide both a title and description.", "error");
                return;
            }

            try {
                toggleFormDisabled(form, true);
                const body = { title, content };
                if (categoryId !== null) {
                    body.category_id = categoryId;
                }
                const payload = await apiRequest(endpoint, {
                    method: "POST",
                    body: JSON.stringify(body),
                });
                const question = payload?.question;
                if (question) {
                    const slug = getString(question, "slug", "Slug") || String(getNumber(question, "id", "ID"));
                    showAlert(alertElement, "Your question has been posted. Redirecting…", "success");
                    if (slug) {
                        window.location.href = `/forum/${slug}`;
                        return;
                    }
                }
                showAlert(alertElement, "Question created successfully.", "success");
                form.reset();
            } catch (error) {
                if (error && error.status === 401) {
                    window.location.href = loginURL;
                    return;
                }
                showAlert(alertElement, error?.message || "Failed to submit question.", "error");
            } finally {
                toggleFormDisabled(form, false);
            }
        });
    };

    const initForumQuestion = (root) => {
        const alertElement = root.querySelector('[data-role="forum-alert"]');
        const questionRatingOutput = root.querySelector('[data-role="question-rating"]');
        const answerList = root.querySelector('[data-role="answer-list"]');
        const answerEmpty = root.querySelector('[data-role="answer-empty"]');
        const answerCountElement = root.querySelector('[data-role="answer-count"]');
        const answerForm = root.querySelector('[data-role="answer-form"]');
        const answerTextarea = root.querySelector('[data-role="answer-content"]');
        const questionDeleteButton = root.querySelector('[data-role="question-delete"]');

        const loginURL = root.dataset.loginUrl || "/login";
        const questionEndpoint = normalizeEndpoint(root.dataset.endpointQuestion || "");
        const questionVoteEndpoint = normalizeEndpoint(root.dataset.endpointQuestionVote || "");
        const answerCreateEndpoint = normalizeEndpoint(root.dataset.endpointAnswerCreate || "");
        const answerVoteEndpoint = normalizeEndpoint(root.dataset.endpointAnswerVote || "");
        const forumPath = root.dataset.forumPath || "/forum";

        const answerVotes = new Map();
        let questionVoteState = 0;

        const getCurrentAnswerCount = () => {
            const value = Number(root.dataset.answerCount || "0");
            if (Number.isFinite(value)) {
                return value;
            }
            if (answerList) {
                return answerList.children.length;
            }
            return 0;
        };

        const updateAnswerCount = (count) => {
            const safeCount = Math.max(0, count);
            root.dataset.answerCount = String(safeCount);
            if (answerCountElement) {
                answerCountElement.textContent = `${safeCount} ${safeCount === 1 ? "answer" : "answers"}`;
            }
        };

        const updateVoteIndicators = (container, currentValue) => {
            if (!container) {
                return;
            }
            container.querySelectorAll('[data-role="answer-vote"], [data-role="question-vote"]').forEach((button) => {
                const value = Number(button.dataset.value || "0");
                button.classList.toggle("is-active", currentValue !== 0 && value === currentValue);
            });
        };

        const handleQuestionDelete = async () => {
            showAlert(alertElement, "");
            if (!questionEndpoint) {
                showAlert(alertElement, "Question deletion is unavailable right now.", "error");
                return;
            }
            if (!isAuthenticated()) {
                window.location.href = loginURL;
                return;
            }
            const confirmation = window.confirm(
                "Are you sure you want to delete this question? This action cannot be undone."
            );
            if (!confirmation) {
                return;
            }
            if (questionDeleteButton) {
                questionDeleteButton.disabled = true;
            }
            try {
                await apiRequest(questionEndpoint, { method: "DELETE" });
                window.location.href = forumPath || "/forum";
            } catch (error) {
                if (questionDeleteButton) {
                    questionDeleteButton.disabled = false;
                }
                if (error && error.status === 401) {
                    window.location.href = loginURL;
                    return;
                }
                showAlert(alertElement, error?.message || "Failed to delete question.", "error");
            }
        };

        const handleQuestionVote = async (button) => {
            showAlert(alertElement, "");
            if (!questionVoteEndpoint) {
                showAlert(alertElement, "Voting is unavailable right now.", "error");
                return;
            }
            const value = Number(button.dataset.value || "0");
            if (!Number.isFinite(value) || value === 0) {
                return;
            }
            if (!isAuthenticated()) {
                window.location.href = loginURL;
                return;
            }

            const submitValue = questionVoteState === value ? 0 : value;

            try {
                const payload = await apiRequest(questionVoteEndpoint, {
                    method: "POST",
                    body: JSON.stringify({ value: submitValue }),
                });
                const rating = Number(payload?.rating);
                if (Number.isFinite(rating) && questionRatingOutput) {
                    questionRatingOutput.textContent = String(rating);
                }
                questionVoteState = submitValue === 0 ? 0 : value;
                updateVoteIndicators(root.querySelector('[data-role="question-votes"]'), questionVoteState);
                showAlert(alertElement, "Thanks for your feedback.", "success");
            } catch (error) {
                if (error && error.status === 401) {
                    window.location.href = loginURL;
                    return;
                }
                showAlert(alertElement, error?.message || "Failed to submit your vote.", "error");
            }
        };

        const handleAnswerVote = async (button) => {
            showAlert(alertElement, "");
            if (!answerVoteEndpoint) {
                showAlert(alertElement, "Voting is unavailable right now.", "error");
                return;
            }
            const item = button.closest(".forum-answer");
            if (!item) {
                return;
            }
            const answerId = Number(item.dataset.answerId || "0");
            if (!Number.isFinite(answerId) || answerId <= 0) {
                return;
            }
            const value = Number(button.dataset.value || "0");
            if (!Number.isFinite(value) || value === 0) {
                return;
            }
            if (!isAuthenticated()) {
                window.location.href = loginURL;
                return;
            }

            const currentValue = answerVotes.get(answerId) || 0;
            const submitValue = currentValue === value ? 0 : value;
            const endpoint = `${answerVoteEndpoint}/${answerId}/vote`;

            try {
                const payload = await apiRequest(endpoint, {
                    method: "POST",
                    body: JSON.stringify({ value: submitValue }),
                });
                const rating = Number(payload?.rating);
                const ratingElement = item.querySelector('[data-role="answer-rating"]');
                if (Number.isFinite(rating) && ratingElement) {
                    ratingElement.textContent = String(rating);
                }
                const newValue = submitValue === 0 ? 0 : value;
                answerVotes.set(answerId, newValue);
                updateVoteIndicators(item.querySelector(".forum-answer__votes"), newValue);
                showAlert(alertElement, "Thanks for your feedback.", "success");
            } catch (error) {
                if (error && error.status === 401) {
                    window.location.href = loginURL;
                    return;
                }
                showAlert(alertElement, error?.message || "Failed to submit your vote.", "error");
            }
        };

        if (questionDeleteButton) {
            questionDeleteButton.addEventListener("click", (event) => {
                event.preventDefault();
                handleQuestionDelete();
            });
        }

        root.addEventListener("click", (event) => {
            const target = event.target;
            if (!(target instanceof Element)) {
                return;
            }
            const questionVoteButton = target.closest('[data-role="question-vote"]');
            if (questionVoteButton) {
                event.preventDefault();
                handleQuestionVote(questionVoteButton);
                return;
            }
            const answerVoteButton = target.closest('[data-role="answer-vote"]');
            if (answerVoteButton) {
                event.preventDefault();
                handleAnswerVote(answerVoteButton);
            }
        });

        if (answerForm && answerTextarea) {
            answerForm.addEventListener("submit", async (event) => {
                event.preventDefault();
                showAlert(alertElement, "");

                if (!answerCreateEndpoint) {
                    showAlert(alertElement, "Answer submission is unavailable right now.", "error");
                    return;
                }

                if (!isAuthenticated()) {
                    window.location.href = loginURL;
                    return;
                }

                const content = answerTextarea.value.trim();
                if (!content) {
                    showAlert(alertElement, "Please write an answer before submitting.", "error");
                    return;
                }

                try {
                    toggleFormDisabled(answerForm, true);
                    const payload = await apiRequest(answerCreateEndpoint, {
                        method: "POST",
                        body: JSON.stringify({ content }),
                    });
                    const answer = payload?.answer;
                    if (answer && answerList) {
                        const element = buildAnswerElement(answer);
                        if (answerList.firstChild) {
                            answerList.prepend(element);
                        } else {
                            answerList.appendChild(element);
                        }
                        const newAnswerId = getNumber(answer, "id", "ID");
                        if (newAnswerId > 0) {
                            answerVotes.set(newAnswerId, 0);
                        }
                        if (answerEmpty) {
                            answerEmpty.hidden = true;
                        }
                        updateAnswerCount(getCurrentAnswerCount() + 1);
                        answerTextarea.value = "";
                        showAlert(alertElement, "Your answer has been posted.", "success");
                    } else {
                        showAlert(alertElement, "Answer saved, but the interface could not refresh automatically.", "info");
                    }
                } catch (error) {
                    if (error && error.status === 401) {
                        window.location.href = loginURL;
                        return;
                    }
                    showAlert(alertElement, error?.message || "Failed to post answer.", "error");
                } finally {
                    toggleFormDisabled(answerForm, false);
                }
            });
        }
    };

    const initialize = () => {
        const forumListRoot = document.querySelector('[data-forum="list"]');
        if (forumListRoot) {
            initForumList(forumListRoot);
            initForumTableNavigation(forumListRoot);
        }
        const forumQuestionRoot = document.querySelector('[data-forum="question"]');
        if (forumQuestionRoot) {
            initForumQuestion(forumQuestionRoot);
        }
    };

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", initialize);
    } else {
        initialize();
    }
})();

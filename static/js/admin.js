(() => {
    const formatDate = (value) => {
        if (!value) {
            return "—";
        }
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            return value;
        }
        try {
            return new Intl.DateTimeFormat(undefined, {
                dateStyle: "medium",
                timeStyle: "short",
            }).format(date);
        } catch (error) {
            return date.toLocaleString();
        }
    };

    const booleanLabel = (value) => (value ? "Yes" : "No");

    const createElement = (tag, options = {}) => {
        const element = document.createElement(tag);
        if (options.className) {
            element.className = options.className;
        }
        if (options.textContent !== undefined) {
            element.textContent = options.textContent;
        }
        if (options.html !== undefined) {
            element.innerHTML = options.html;
        }
        return element;
    };

    document.addEventListener("DOMContentLoaded", () => {
        const root = document.querySelector('[data-page="admin"]');
        if (!root) {
            return;
        }

        const app = window.App || {};
        const { apiRequest, auth, setAlert, toggleFormDisabled } = app;
        if (typeof apiRequest !== "function") {
            console.warn("Admin dashboard requires App.apiRequest to be available.");
            return;
        }

        const requireAuth = () => {
            if (!auth || typeof auth.getToken !== "function") {
                return true;
            }
            if (!auth.getToken()) {
                window.location.href = "/login?redirect=/admin";
                return false;
            }
            return true;
        };

        if (!requireAuth()) {
            return;
        }

        const endpoints = {
            stats: root.dataset.endpointStats,
            posts: root.dataset.endpointPosts,
            pages: root.dataset.endpointPages,
            categories: root.dataset.endpointCategories,
            categoriesIndex: root.dataset.endpointCategoriesIndex,
            comments: root.dataset.endpointComments,
            tags: root.dataset.endpointTags,
        };

        const alertElement = document.getElementById("admin-alert");
        const showAlert = (message, type = "info") => {
            if (!alertElement) {
                return;
            }
            if (typeof setAlert === "function") {
                setAlert(alertElement, message, type);
                return;
            }
            alertElement.textContent = message || "";
            alertElement.hidden = !message;
        };

        const clearAlert = () => showAlert("");

        const handleRequestError = (error) => {
            if (!error) {
                return;
            }
            if (error.status === 401) {
                if (auth && typeof auth.clearToken === "function") {
                    auth.clearToken();
                }
                window.location.href = "/login?redirect=/admin";
                return;
            }
            if (error.status === 403) {
                showAlert("You do not have permission to perform this action.", "error");
                return;
            }
            const message = error.message || "Request failed. Please try again.";
            showAlert(message, "error");
            console.error("Admin dashboard request failed", error);
        };

        const disableForm = (form, disabled) => {
            if (!form) {
                return;
            }
            if (typeof toggleFormDisabled === "function") {
                toggleFormDisabled(form, disabled);
                return;
            }
            form.querySelectorAll("input, select, textarea, button").forEach((field) => {
                field.disabled = disabled;
            });
        };

        const metricElements = new Map();
        root.querySelectorAll(".admin__metric").forEach((card) => {
            const key = card.dataset.metric;
            const valueElement = card.querySelector(".admin__metric-value");
            if (key && valueElement) {
                metricElements.set(key, valueElement);
            }
        });

        const tables = {
            posts: root.querySelector("#admin-posts-table"),
            pages: root.querySelector("#admin-pages-table"),
            categories: root.querySelector("#admin-categories-table"),
        };
        const commentsList = root.querySelector("#admin-comments-list");
        const postForm = root.querySelector("#admin-post-form");
        const pageForm = root.querySelector("#admin-page-form");
        const categoryForm = root.querySelector("#admin-category-form");
        const postDeleteButton = postForm?.querySelector('[data-role="post-delete"]');
        const postSubmitButton = postForm?.querySelector('[data-role="post-submit"]');
        const pageDeleteButton = pageForm?.querySelector('[data-role="page-delete"]');
        const pageSubmitButton = pageForm?.querySelector('[data-role="page-submit"]');
        const categoryDeleteButton = categoryForm?.querySelector('[data-role="category-delete"]');
        const categorySubmitButton = categoryForm?.querySelector('[data-role="category-submit"]');
        const postCategorySelect = postForm?.querySelector("#admin-post-category");
        const postTagsInput = postForm?.querySelector("#admin-post-tags");
        const postTagsList = document.getElementById("admin-post-tags-list");
        const DEFAULT_CATEGORY_SLUG = "uncategorized";
        const pageSlugInput = pageForm?.querySelector('input[name="slug"]');

        const state = {
            metrics: {},
            posts: [],
            pages: [],
            categories: [],
            comments: [],
            tags: [],
            defaultCategoryId: "",
        };

        const normaliseSlug = (value) => (typeof value === "string" ? value.toLowerCase() : "");

        const extractCategorySlug = (category) => {
            if (!category) {
                return "";
            }
            const candidates = [category.slug, category.Slug];
            for (const candidate of candidates) {
                const normalised = normaliseSlug(candidate);
                if (normalised) {
                    return normalised;
                }
                if (candidate && typeof candidate.value === "string") {
                    const nested = normaliseSlug(candidate.value);
                    if (nested) {
                        return nested;
                    }
                }
            }
            return normaliseSlug(category.name || category.Name || "");
        };

        const extractCategoryId = (category) => {
            if (!category) {
                return "";
            }
            const candidates = [category.id, category.ID];
            for (const candidate of candidates) {
                if (candidate === undefined || candidate === null) {
                    continue;
                }
                if (typeof candidate === "object" && candidate !== null) {
                    const value = candidate.value ?? candidate.Value;
                    if (value !== undefined && value !== null) {
                        const normalised = String(value).trim();
                        if (normalised) {
                            return normalised;
                        }
                    }
                    continue;
                }
                const normalised = String(candidate).trim();
                if (normalised) {
                    return normalised;
                }
            }
            return "";
        };

        const refreshDefaultCategoryId = () => {
            const defaultSlug = normaliseSlug(DEFAULT_CATEGORY_SLUG);
            const matchBySlug = state.categories.find((category) => extractCategorySlug(category) === defaultSlug);
            if (matchBySlug) {
                state.defaultCategoryId = extractCategoryId(matchBySlug);
                return;
            }
            const fallback = state.categories.find((category) => extractCategoryId(category));
            state.defaultCategoryId = fallback ? extractCategoryId(fallback) : "";
        };

        const ensureDefaultCategorySelection = () => {
            if (!postCategorySelect) {
                return;
            }
            if (!state.defaultCategoryId) {
                refreshDefaultCategoryId();
            }
            if (state.defaultCategoryId) {
                postCategorySelect.value = state.defaultCategoryId;
            }
            if (!postCategorySelect.value && postCategorySelect.options.length) {
                const firstUsable = Array.from(postCategorySelect.options).find((option) => option.value);
                if (firstUsable) {
                    postCategorySelect.value = firstUsable.value;
                }
            }
            if (!postCategorySelect.value && postCategorySelect.options.length) {
                postCategorySelect.selectedIndex = 0;
            }
            if (postCategorySelect.value) {
                state.defaultCategoryId = postCategorySelect.value;
            }
        };

        const normaliseTagName = (value) => (typeof value === "string" ? value.trim() : "");

        const parseTags = (value) => {
            if (typeof value !== "string" || !value.trim()) {
                return [];
            }
            const unique = new Map();
            value
                .split(",")
                .map((entry) => normaliseTagName(entry))
                .filter(Boolean)
                .forEach((name) => {
                    const key = name.toLowerCase();
                    if (!unique.has(key)) {
                        unique.set(key, name);
                    }
                });
            return Array.from(unique.values());
        };

        const extractTagNames = (entry) => {
            const tags = entry?.tags || entry?.Tags;
            if (!Array.isArray(tags)) {
                return [];
            }
            const unique = new Map();
            tags.forEach((tag) => {
                const name = normaliseTagName(tag?.name || tag?.Name);
                if (!name) {
                    return;
                }
                const key = name.toLowerCase();
                if (!unique.has(key)) {
                    unique.set(key, name);
                }
            });
            return Array.from(unique.values());
        };

        const renderTagSuggestions = () => {
            if (!postTagsList) {
                return;
            }
            const suggestions = new Map();
            const addSuggestion = (name) => {
                const cleaned = normaliseTagName(name);
                if (!cleaned) {
                    return;
                }
                const key = cleaned.toLowerCase();
                if (!suggestions.has(key)) {
                    suggestions.set(key, cleaned);
                }
            };

            state.tags.forEach((tag) => addSuggestion(tag?.name || tag?.Name));
            state.posts.forEach((post) => {
                extractTagNames(post).forEach(addSuggestion);
            });
            if (postTagsInput && postTagsInput.value) {
                parseTags(postTagsInput.value).forEach(addSuggestion);
            }

            const ordered = Array.from(suggestions.values()).sort((a, b) =>
                a.localeCompare(b, undefined, { sensitivity: "base" })
            );

            postTagsList.innerHTML = "";
            ordered.forEach((name) => {
                const option = document.createElement("option");
                option.value = name;
                postTagsList.appendChild(option);
            });
        };

        const highlightRow = (table, id) => {
            if (!table) {
                return;
            }
            table.querySelectorAll("tr").forEach((row) => {
                row.classList.toggle("is-selected", id && String(row.dataset.id) === String(id));
            });
        };

        const renderMetrics = (metrics = {}) => {
            Object.entries(metrics).forEach(([key, value]) => {
                const target = metricElements.get(key);
                if (target) {
                    target.textContent = Number.isFinite(Number(value))
                        ? Number(value).toLocaleString()
                        : String(value ?? "—");
                }
            });
        };

        const renderPosts = () => {
            const table = tables.posts;
            if (!table) {
                return;
            }
            table.innerHTML = "";
            if (!state.posts.length) {
                const row = createElement("tr", { className: "admin-table__placeholder" });
                const cell = createElement("td", { textContent: "No posts found" });
                cell.colSpan = 5;
                row.appendChild(cell);
                table.appendChild(row);
                return;
            }
            state.posts.forEach((post) => {
                const row = createElement("tr");
                row.dataset.id = post.id;
                row.appendChild(createElement("td", { textContent: post.title || "Untitled" }));
                const categoryName = post.category?.name || post.category_name || "—";
                row.appendChild(createElement("td", { textContent: categoryName || "—" }));
                const tagNames = extractTagNames(post).join(", ");
                row.appendChild(createElement("td", { textContent: tagNames || "—" }));
                row.appendChild(createElement("td", { textContent: booleanLabel(post.published) }));
                const updated = post.updated_at || post.updatedAt || post.UpdatedAt;
                row.appendChild(createElement("td", { textContent: formatDate(updated) }));
                row.addEventListener("click", () => selectPost(post.id));
                table.appendChild(row);
            });
            highlightRow(table, postForm?.dataset.id);
        };

        const renderPages = () => {
            const table = tables.pages;
            if (!table) {
                return;
            }
            table.innerHTML = "";
            if (!state.pages.length) {
                const row = createElement("tr", { className: "admin-table__placeholder" });
                const cell = createElement("td", { textContent: "No pages found" });
                cell.colSpan = 4;
                row.appendChild(cell);
                table.appendChild(row);
                return;
            }
            state.pages.forEach((page) => {
                const row = createElement("tr");
                row.dataset.id = page.id;
                row.appendChild(createElement("td", { textContent: page.title || "Untitled" }));
                row.appendChild(createElement("td", { textContent: page.slug || "—" }));
                row.appendChild(createElement("td", { textContent: booleanLabel(page.published) }));
                const updated = page.updated_at || page.updatedAt || page.UpdatedAt;
                row.appendChild(createElement("td", { textContent: formatDate(updated) }));
                row.addEventListener("click", () => selectPage(page.id));
                table.appendChild(row);
            });
            highlightRow(table, pageForm?.dataset.id);
        };

        const renderCategories = () => {
            const table = tables.categories;
            if (!table) {
                return;
            }
            table.innerHTML = "";
            if (!state.categories.length) {
                const row = createElement("tr", { className: "admin-table__placeholder" });
                const cell = createElement("td", { textContent: "No categories found" });
                cell.colSpan = 3;
                row.appendChild(cell);
                table.appendChild(row);
                return;
            }
            state.categories.forEach((category) => {
                const id = extractCategoryId(category);
                if (!id) {
                    return;
                }
                const row = createElement("tr");
                row.dataset.id = id;
                row.appendChild(createElement("td", { textContent: category.name || "Untitled" }));
                row.appendChild(createElement("td", { textContent: category.slug || "—" }));
                const updated = category.updated_at || category.updatedAt || category.UpdatedAt;
                row.appendChild(createElement("td", { textContent: formatDate(updated) }));
                row.addEventListener("click", () => selectCategory(id));
                table.appendChild(row);
            });
            highlightRow(table, categoryForm?.dataset.id);
        };

        const renderCategoryOptions = () => {
            if (!postCategorySelect) {
                return;
            }
            const currentValue = postCategorySelect.value;
            postCategorySelect.innerHTML = "";

            const seen = new Set();
            state.categories.forEach((category) => {
                const id = extractCategoryId(category);
                if (!id) {
                    return;
                }
                if (seen.has(id)) {
                    return;
                }
                seen.add(id);
                const option = createElement("option", { textContent: category.name || "Untitled" });
                option.value = id;
                postCategorySelect.appendChild(option);
            });

            if (currentValue && state.categories.some((category) => extractCategoryId(category) === currentValue)) {
                postCategorySelect.value = currentValue;
            } else {
                ensureDefaultCategorySelection();
            }
        };

        const renderComments = () => {
            if (!commentsList) {
                return;
            }
            commentsList.innerHTML = "";
            if (!state.comments.length) {
                const item = createElement("li", {
                    className: "admin-comment-list__item admin-comment-list__item--empty",
                    textContent: "No comments available",
                });
                commentsList.appendChild(item);
                return;
            }
            state.comments.forEach((comment) => {
                const item = createElement("li", { className: "admin-comment-list__item" });
                const meta = createElement("div", { className: "admin-comment-list__meta" });
                const pieces = [];
                if (comment.author?.username) {
                    pieces.push(`by ${comment.author.username}`);
                }
                if (comment.post?.title) {
                    pieces.push(`on "${comment.post.title}"`);
                }
                pieces.push(comment.approved ? "approved" : "pending approval");
                const created = comment.created_at || comment.createdAt || comment.CreatedAt;
                pieces.push(formatDate(created));
                meta.textContent = pieces.join(" · ");
                const content = createElement("p", {
                    className: "admin-comment-list__content",
                    textContent: comment.content || "(no content)",
                });
                const actions = createElement("div", { className: "admin-comment-list__actions" });
                if (!comment.approved) {
                    const approveButton = createElement("button", {
                        className: "admin-comment-button",
                        textContent: "Approve",
                    });
                    approveButton.dataset.action = "approve";
                    approveButton.addEventListener("click", () => approveComment(comment.id, approveButton));
                    actions.appendChild(approveButton);
                } else {
                    const rejectButton = createElement("button", {
                        className: "admin-comment-button",
                        textContent: "Reject",
                    });
                    rejectButton.dataset.action = "reject";
                    rejectButton.addEventListener("click", () => rejectComment(comment.id, rejectButton));
                    actions.appendChild(rejectButton);
                }
                const deleteButton = createElement("button", {
                    className: "admin-comment-button",
                    textContent: "Delete",
                });
                deleteButton.dataset.action = "delete";
                deleteButton.addEventListener("click", () => deleteComment(comment.id, deleteButton));
                actions.appendChild(deleteButton);
                item.appendChild(meta);
                item.appendChild(content);
                item.appendChild(actions);
                commentsList.appendChild(item);
            });
        };

        const selectPost = (id) => {
            if (!postForm) {
                return;
            }
            const post = state.posts.find((entry) => String(entry.id) === String(id));
            if (!post) {
                return;
            }
            postForm.dataset.id = post.id;
            postForm.title.value = post.title || "";
            postForm.description.value = post.description || "";
            postForm.content.value = post.content || "";
            const categoryId =
                post.category?.id ||
                post.category?.ID ||
                post.category_id ||
                post.CategoryID;
            if (postCategorySelect) {
                if (categoryId) {
                    postCategorySelect.value = String(categoryId);
                } else {
                    ensureDefaultCategorySelection();
                }
            }
            if (postTagsInput) {
                postTagsInput.value = extractTagNames(post).join(", ");
            }
            const publishedField = postForm.querySelector('input[name="published"]');
            if (publishedField) {
                publishedField.checked = Boolean(post.published);
            }
            if (postSubmitButton) {
                postSubmitButton.textContent = "Update post";
            }
            if (postDeleteButton) {
                postDeleteButton.hidden = false;
            }
            renderTagSuggestions();
            highlightRow(tables.posts, post.id);
        };

        const resetPostForm = () => {
            if (!postForm) {
                return;
            }
            postForm.reset();
            delete postForm.dataset.id;
            ensureDefaultCategorySelection();
            if (postTagsInput) {
                postTagsInput.value = "";
            }
            if (postSubmitButton) {
                postSubmitButton.textContent = "Create post";
            }
            if (postDeleteButton) {
                postDeleteButton.hidden = true;
            }
            renderTagSuggestions();
            highlightRow(tables.posts);
        };

        const selectPage = (id) => {
            if (!pageForm) {
                return;
            }
            const page = state.pages.find((entry) => String(entry.id) === String(id));
            if (!page) {
                return;
            }
            pageForm.dataset.id = page.id;
            pageForm.title.value = page.title || "";
            if (pageSlugInput) {
                pageSlugInput.value = page.slug || "";
                pageSlugInput.disabled = true;
                pageSlugInput.title = "The slug is generated from the title when updating";
            }
            pageForm.description.value = page.description || "";
            const orderInput = pageForm.querySelector('input[name="order"]');
            if (orderInput) {
                orderInput.value = page.order ?? 0;
            }
            const publishedField = pageForm.querySelector('input[name="published"]');
            if (publishedField) {
                publishedField.checked = Boolean(page.published);
            }
            if (pageSubmitButton) {
                pageSubmitButton.textContent = "Update page";
            }
            if (pageDeleteButton) {
                pageDeleteButton.hidden = false;
            }
            highlightRow(tables.pages, page.id);
        };

        const resetPageForm = () => {
            if (!pageForm) {
                return;
            }
            pageForm.reset();
            delete pageForm.dataset.id;
            if (pageSubmitButton) {
                pageSubmitButton.textContent = "Create page";
            }
            if (pageDeleteButton) {
                pageDeleteButton.hidden = true;
            }
            if (pageSlugInput) {
                pageSlugInput.disabled = false;
                pageSlugInput.title = "Optional custom slug";
            }
            const orderInput = pageForm.querySelector('input[name="order"]');
            if (orderInput) {
                orderInput.value = 0;
            }
            highlightRow(tables.pages);
        };

        const selectCategory = (id) => {
            if (!categoryForm) {
                return;
            }
            const category = state.categories.find((entry) => extractCategoryId(entry) === String(id));
            if (!category) {
                return;
            }
            const categoryId = extractCategoryId(category);
            if (categoryId) {
                categoryForm.dataset.id = categoryId;
            } else {
                delete categoryForm.dataset.id;
            }
            categoryForm.name.value = category.name || "";
            categoryForm.description.value = category.description || "";
            if (categorySubmitButton) {
                categorySubmitButton.textContent = "Update category";
            }
            if (categoryDeleteButton) {
                categoryDeleteButton.hidden = false;
            }
            highlightRow(tables.categories, categoryId);
        };

        const resetCategoryForm = () => {
            if (!categoryForm) {
                return;
            }
            categoryForm.reset();
            delete categoryForm.dataset.id;
            if (categorySubmitButton) {
                categorySubmitButton.textContent = "Create category";
            }
            if (categoryDeleteButton) {
                categoryDeleteButton.hidden = true;
            }
            highlightRow(tables.categories);
        };

        const loadStats = async () => {
            if (!endpoints.stats) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.stats);
                const metrics = payload?.statistics || {};
                state.metrics = metrics;
                renderMetrics(metrics);
            } catch (error) {
                handleRequestError(error);
            }
        };

        const loadPosts = async () => {
            if (!endpoints.posts) {
                return;
            }
            try {
                const payload = await apiRequest(`${endpoints.posts}?limit=50`);
                state.posts = payload?.posts || [];
                renderPosts();
                renderTagSuggestions();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const loadPages = async () => {
            if (!endpoints.pages) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.pages);
                state.pages = payload?.pages || [];
                renderPages();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const loadCategories = async () => {
            if (!endpoints.categoriesIndex) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.categoriesIndex);
                state.categories = payload?.categories || [];
                refreshDefaultCategoryId();
                renderCategories();
                renderCategoryOptions();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const loadComments = async () => {
            if (!endpoints.comments) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.comments);
                const comments = payload?.comments || [];
                comments.sort((a, b) => {
                    const aDate = new Date(a.created_at || a.createdAt || a.CreatedAt || 0).getTime();
                    const bDate = new Date(b.created_at || b.createdAt || b.CreatedAt || 0).getTime();
                    return bDate - aDate;
                });
                state.comments = comments.slice(0, 15);
                renderComments();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const loadTags = async () => {
            if (!endpoints.tags) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.tags);
                state.tags = payload?.tags || [];
                renderTagSuggestions();
            } catch (error) {
                handleRequestError(error);
            }
        };

        const approveComment = async (id, button) => {
            if (!endpoints.comments) {
                return;
            }
            try {
                button.disabled = true;
                await apiRequest(`${endpoints.comments}/${id}/approve`, { method: "PUT" });
                showAlert("Comment approved", "success");
                await loadComments();
            } catch (error) {
                handleRequestError(error);
            } finally {
                button.disabled = false;
            }
        };

        const rejectComment = async (id, button) => {
            if (!endpoints.comments) {
                return;
            }
            try {
                button.disabled = true;
                await apiRequest(`${endpoints.comments}/${id}/reject`, { method: "PUT" });
                showAlert("Comment rejected", "info");
                await loadComments();
            } catch (error) {
                handleRequestError(error);
            } finally {
                button.disabled = false;
            }
        };

        const deleteComment = async (id, button) => {
            if (!endpoints.comments) {
                return;
            }
            if (!window.confirm("Delete this comment permanently?")) {
                return;
            }
            try {
                button.disabled = true;
                await apiRequest(`${endpoints.comments}/${id}`, { method: "DELETE" });
                showAlert("Comment deleted", "success");
                await loadComments();
                await loadStats();
            } catch (error) {
                handleRequestError(error);
            } finally {
                button.disabled = false;
            }
        };

        const handlePostSubmit = async (event) => {
            event.preventDefault();
            if (!postForm) {
                return;
            }
            const id = postForm.dataset.id;
            const title = postForm.title.value.trim();
            if (!title) {
                showAlert("Please provide a title for the post.", "error");
                return;
            }
            const description = postForm.description.value.trim();
            const content = postForm.content.value.trim();
            const publishedField = postForm.querySelector('input[name="published"]');
            const payload = {
                title,
                description,
                content,
                published: Boolean(publishedField?.checked),
            };
            const categoryValue = postCategorySelect?.value;
            if (categoryValue) {
                payload.category_id = Number(categoryValue);
            }
            if (postTagsInput) {
                payload.tags = parseTags(postTagsInput.value);
            }
            disableForm(postForm, true);
            clearAlert();
            try {
                if (id) {
                    await apiRequest(`${endpoints.posts}/${id}`, {
                        method: "PUT",
                        body: JSON.stringify(payload),
                    });
                    showAlert("Post updated successfully.", "success");
                } else {
                    await apiRequest(endpoints.posts, {
                        method: "POST",
                        body: JSON.stringify(payload),
                    });
                    showAlert("Post created successfully.", "success");
                }
                await loadPosts();
                await loadTags();
                await loadStats();
                resetPostForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(postForm, false);
            }
        };

        const handlePostDelete = async () => {
            if (!postForm || !postForm.dataset.id) {
                return;
            }
            if (!window.confirm("Delete this post permanently?")) {
                return;
            }
            const id = postForm.dataset.id;
            disableForm(postForm, true);
            clearAlert();
            try {
                await apiRequest(`${endpoints.posts}/${id}`, { method: "DELETE" });
                showAlert("Post deleted successfully.", "success");
                await loadPosts();
                await loadTags();
                await loadStats();
                resetPostForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(postForm, false);
            }
        };

        const handlePageSubmit = async (event) => {
            event.preventDefault();
            if (!pageForm) {
                return;
            }
            const id = pageForm.dataset.id;
            const title = pageForm.title.value.trim();
            if (!title) {
                showAlert("Please provide a title for the page.", "error");
                return;
            }
            const description = pageForm.description.value.trim();
            const orderInput = pageForm.querySelector('input[name="order"]');
            const orderValue = orderInput ? Number(orderInput.value) : 0;
            const publishedField = pageForm.querySelector('input[name="published"]');
            const payload = {
                title,
                description,
                order: Number.isNaN(orderValue) ? 0 : orderValue,
                published: Boolean(publishedField?.checked),
            };
            if (!id && pageSlugInput) {
                const slugValue = pageSlugInput.value.trim();
                if (slugValue) {
                    payload.slug = slugValue;
                }
            }
            disableForm(pageForm, true);
            clearAlert();
            try {
                if (id) {
                    await apiRequest(`${endpoints.pages}/${id}`, {
                        method: "PUT",
                        body: JSON.stringify(payload),
                    });
                    showAlert("Page updated successfully.", "success");
                } else {
                    await apiRequest(endpoints.pages, {
                        method: "POST",
                        body: JSON.stringify(payload),
                    });
                    showAlert("Page created successfully.", "success");
                }
                await loadPages();
                await loadStats();
                resetPageForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(pageForm, false);
            }
        };

        const handlePageDelete = async () => {
            if (!pageForm || !pageForm.dataset.id) {
                return;
            }
            if (!window.confirm("Delete this page permanently?")) {
                return;
            }
            const id = pageForm.dataset.id;
            disableForm(pageForm, true);
            clearAlert();
            try {
                await apiRequest(`${endpoints.pages}/${id}`, { method: "DELETE" });
                showAlert("Page deleted successfully.", "success");
                await loadPages();
                await loadStats();
                resetPageForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(pageForm, false);
            }
        };

        const handleCategorySubmit = async (event) => {
            event.preventDefault();
            if (!categoryForm) {
                return;
            }
            const id = categoryForm.dataset.id;
            const name = categoryForm.name.value.trim();
            if (!name) {
                showAlert("Please provide a category name.", "error");
                return;
            }
            const description = categoryForm.description.value.trim();
            const payload = { name, description };
            disableForm(categoryForm, true);
            clearAlert();
            try {
                if (id) {
                    await apiRequest(`${endpoints.categories}/${id}`, {
                        method: "PUT",
                        body: JSON.stringify(payload),
                    });
                    showAlert("Category updated successfully.", "success");
                } else {
                    await apiRequest(endpoints.categories, {
                        method: "POST",
                        body: JSON.stringify(payload),
                    });
                    showAlert("Category created successfully.", "success");
                }
                await loadCategories();
                await loadPosts();
                await loadStats();
                resetCategoryForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(categoryForm, false);
            }
        };

        const handleCategoryDelete = async () => {
            if (!categoryForm || !categoryForm.dataset.id) {
                return;
            }
            if (!window.confirm("Delete this category permanently?")) {
                return;
            }
            const id = categoryForm.dataset.id;
            disableForm(categoryForm, true);
            clearAlert();
            try {
                await apiRequest(`${endpoints.categories}/${id}`, { method: "DELETE" });
                showAlert("Category deleted successfully.", "success");
                await loadCategories();
                await loadPosts();
                await loadStats();
                resetCategoryForm();
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(categoryForm, false);
            }
        };

        const activateTab = (targetId) => {
            root.querySelectorAll(".admin__tab").forEach((tab) => {
                const isActive = tab.dataset.tab === targetId;
                tab.classList.toggle("is-active", isActive);
                tab.setAttribute("aria-selected", String(isActive));
            });
            root.querySelectorAll(".admin-panel").forEach((panel) => {
                const isActive = panel.dataset.panel === targetId;
                panel.toggleAttribute("hidden", !isActive);
                panel.classList.toggle("is-active", isActive);
            });
        };

        root.querySelectorAll(".admin__tab").forEach((tab) => {
            tab.addEventListener("click", () => activateTab(tab.dataset.tab));
        });

        root.querySelector('[data-action="post-reset"]')?.addEventListener("click", resetPostForm);
        root.querySelector('[data-action="page-reset"]')?.addEventListener("click", resetPageForm);
        root.querySelector('[data-action="category-reset"]')?.addEventListener("click", resetCategoryForm);

        postForm?.addEventListener("submit", handlePostSubmit);
        postDeleteButton?.addEventListener("click", handlePostDelete);
        pageForm?.addEventListener("submit", handlePageSubmit);
        pageDeleteButton?.addEventListener("click", handlePageDelete);
        categoryForm?.addEventListener("submit", handleCategorySubmit);
        categoryDeleteButton?.addEventListener("click", handleCategoryDelete);
        postTagsInput?.addEventListener("input", renderTagSuggestions);

        clearAlert();
        loadStats();
        loadTags();
        loadCategories().then(() => {
            renderCategoryOptions();
            loadPosts();
        });
        loadPages();
        loadComments();
    });
})();
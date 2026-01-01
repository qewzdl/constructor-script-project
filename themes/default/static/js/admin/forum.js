(() => {
    const layout = window.AdminLayout;
    if (!layout) {
        return;
    }

    const app = window.App || {};
    const globalAlertId = 'admin-alert';

    const fallbackApiRequest = async (url, options = {}) => {
        const headers = Object.assign({}, options.headers || {});
        const auth = app.auth;
        const token = auth && typeof auth.getToken === 'function' ? auth.getToken() : '';
        if (options.body && !(options.body instanceof FormData)) {
            headers['Content-Type'] = headers['Content-Type'] || 'application/json';
        }
        if (token) {
            headers.Authorization = `Bearer ${token}`;
        }
        const response = await fetch(url, {
            credentials: 'include',
            ...options,
            headers,
        });
        const contentType = response.headers.get('content-type') || '';
        const isJson = contentType.includes('application/json');
        const payload = isJson ? await response.json().catch(() => null) : await response.text();
        if (!response.ok) {
            const message =
                payload && typeof payload === 'object' && payload.error
                    ? payload.error
                    : typeof payload === 'string'
                    ? payload
                    : 'Request failed';
            const error = new Error(message);
            error.status = response.status;
            error.payload = payload;
            throw error;
        }
        return payload;
    };

    const apiClient = typeof app.apiRequest === 'function' ? app.apiRequest : fallbackApiRequest;
    const setAlert = typeof app.setAlert === 'function'
        ? app.setAlert
        : (_target, message, type) => {
              if (message && type === 'error') {
                  console.error(message);
              }
          };
    const toggleFormDisabled = typeof app.toggleFormDisabled === 'function'
        ? app.toggleFormDisabled
        : (form, disabled) => {
              if (!form) {
                  return;
              }
              form.querySelectorAll('input, textarea, button, select').forEach((element) => {
                  element.disabled = disabled;
              });
          };

    const escapeHtml = (value) => {
        if (value === null || value === undefined) {
            return '';
        }
        return String(value)
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/"/g, '&quot;')
            .replace(/'/g, '&#039;');
    };

    const formatDateTime = (value) => {
        if (!value) {
            return '—';
        }
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
            return typeof value === 'string' ? value : '—';
        }
        try {
            return date.toLocaleString(undefined, {
                year: 'numeric',
                month: 'short',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
            });
        } catch (error) {
            return date.toISOString();
        }
    };

    const resolveDateValue = (entry, ...keys) => {
        for (const key of keys) {
            if (entry && Object.prototype.hasOwnProperty.call(entry, key)) {
                const value = entry[key];
                if (value) {
                    return value;
                }
            }
        }
        return null;
    };

    layout.whenReady((context) => {
        if (!context || !context.forumEnabled) {
            return;
        }

        const root = context.root || document;
        const panel = root.querySelector('#admin-panel-forum');
        if (!panel) {
            return;
        }

        const topicsEndpointRaw = (context.dataset?.endpointForumTopics || '').trim();
        const adminTopicsEndpointRaw = (context.dataset?.endpointAdminForumTopics || '').trim();
        if (!topicsEndpointRaw) {
            return;
        }
        const answersEndpointRaw = (context.dataset?.endpointForumAnswers || '').trim();
        const categoriesEndpointRaw = (context.dataset?.endpointForumCategories || '').trim();
        const topicsEndpoint = topicsEndpointRaw.replace(/\/+$/, '');
        const adminTopicsEndpoint = adminTopicsEndpointRaw
            ? adminTopicsEndpointRaw.replace(/\/+$/, '')
            : '';
        const answersEndpoint = answersEndpointRaw ? answersEndpointRaw.replace(/\/+$/, '') : '';
        const categoriesEndpoint = categoriesEndpointRaw ? categoriesEndpointRaw.replace(/\/+$/, '') : '';
        const topicManageEndpoint = adminTopicsEndpoint || topicsEndpoint;

        const topicTable = panel.querySelector('#admin-forum-topics-table');
        const topicForm = panel.querySelector('#admin-forum-topic-form');
        const topicStatus = panel.querySelector('[data-role="forum-topic-status"]');
        const topicSubmitButton = panel.querySelector('[data-role="forum-topic-submit"]');
        const topicDeleteButton = panel.querySelector('[data-role="forum-topic-delete"]');
        const topicResetButton = panel.querySelector('[data-action="forum-topic-reset"]');
        const topicSearchInput = panel.querySelector('[data-role="forum-topic-search"]');
        const topicCategorySelect = panel.querySelector('[data-role="forum-topic-category"]');

        const answersContainer = panel.querySelector('[data-role="forum-answer-container"]');
        const answersList = panel.querySelector('[data-role="forum-answer-list"]');
        const answerEmpty = panel.querySelector('[data-role="forum-answer-empty"]');
        const answerForm = panel.querySelector('#admin-forum-answer-form');
        const answerContentInput = panel.querySelector('[data-role="forum-answer-content"]');
        const answerSubmitButton = panel.querySelector('[data-role="forum-answer-submit"]');
        const answerCancelButton = panel.querySelector('[data-role="forum-answer-cancel"]');
        const answerDeleteButton = panel.querySelector('[data-role="forum-answer-delete"]');

        const categoryTable = panel.querySelector('#admin-forum-categories-table');
        const categoryForm = panel.querySelector('#admin-forum-category-form');
        const categoryStatus = panel.querySelector('[data-role="forum-category-status"]');
        const categorySubmitButton = panel.querySelector('[data-role="forum-category-submit"]');
        const categoryDeleteButton = panel.querySelector('[data-role="forum-category-delete"]');
        const categoryResetButton = panel.querySelector('[data-action="forum-category-reset"]');
        const categorySearchInput = panel.querySelector('[data-role="forum-category-search"]');

        const state = {
            topics: [],
            categories: [],
            selectedTopicId: '',
            selectedAnswerId: '',
            selectedCategoryId: '',
        };

        const showAlert = (message, type = 'info') => {
            if (!message) {
                setAlert(globalAlertId, '', type);
                return;
            }
            setAlert(globalAlertId, message, type);
        };

        const buildTopicEndpoint = (id, suffix = '', options = {}) => {
            const useManagementEndpoint = Boolean(options?.useManagementEndpoint);
            const baseEndpoint = useManagementEndpoint ? topicManageEndpoint : topicsEndpoint;
            if (!id) {
                return baseEndpoint;
            }
            if (!baseEndpoint) {
                return '';
            }
            const base = `${baseEndpoint}/${id}`;
            if (!suffix) {
                return base;
            }
            return `${base}${suffix.startsWith('/') ? '' : '/'}${suffix}`;
        };

        const buildAnswerEndpoint = (id) => {
            if (!answersEndpoint) {
                return '';
            }
            return `${answersEndpoint}/${id}`;
        };

        const setAnswerFormEnabled = (enabled) => {
            if (!answerForm) {
                return;
            }
            answerForm.querySelectorAll('textarea, button').forEach((element) => {
                element.disabled = !enabled;
            });
        };

        const resetAnswerForm = () => {
            if (!answerForm) {
                return;
            }
            answerForm.reset();
            delete answerForm.dataset.id;
            answerCancelButton.hidden = true;
            answerDeleteButton.hidden = true;
            if (answerSubmitButton) {
                answerSubmitButton.textContent = 'Save answer';
            }
        };

        const resetAnswersView = () => {
            if (answersList) {
                answersList.innerHTML = '';
            }
            if (answerEmpty) {
                answerEmpty.textContent = 'Select a topic to see submitted answers.';
                answerEmpty.hidden = false;
            }
            if (answerForm) {
                delete answerForm.dataset.topicId;
            }
            resetAnswerForm();
            setAnswerFormEnabled(false);
        };

        const resolveTopicCategoryId = (topic) => {
            if (!topic) {
                return "";
            }
            const direct = topic.category_id ?? topic.CategoryID;
            if (direct !== undefined && direct !== null) {
                const numeric = Number(direct);
                return Number.isFinite(numeric) && numeric > 0 ? String(numeric) : "";
            }
            const category = topic.category || topic.Category;
            if (category && (category.id || category.ID)) {
                const numeric = Number(category.id ?? category.ID);
                return Number.isFinite(numeric) && numeric > 0 ? String(numeric) : "";
            }
            return "";
        };

        const resolveTopicCategoryName = (topic) => {
            if (!topic) {
                return "";
            }
            const category = topic.category || topic.Category;
            if (category) {
                return category.name || category.Name || "";
            }
            const categoryId = resolveTopicCategoryId(topic);
            if (!categoryId) {
                return "";
            }
            const matched = state.categories.find((entry) => {
                const identifier = entry?.id ?? entry?.ID;
                return identifier !== undefined && String(identifier) === categoryId;
            });
            if (matched) {
                return matched.name || matched.Name || "";
            }
            return "";
        };

        const updateTopicCategoryOptions = () => {
            if (!topicCategorySelect) {
                return;
            }
            const previousValue = topicCategorySelect.value;
            topicCategorySelect.innerHTML = '';
            const defaultOption = document.createElement('option');
            defaultOption.value = '';
            defaultOption.textContent = 'No category';
            topicCategorySelect.appendChild(defaultOption);
            state.categories.forEach((category) => {
                const identifier = category?.id ?? category?.ID;
                if (!identifier) {
                    return;
                }
                const option = document.createElement('option');
                option.value = String(identifier);
                option.textContent = category?.name || category?.Name || `Category ${identifier}`;
                topicCategorySelect.appendChild(option);
            });

            const selectedTopic = state.topics.find(
                (topic) => String(topic?.id) === state.selectedTopicId
            );
            const desiredValue = selectedTopic ? resolveTopicCategoryId(selectedTopic) : previousValue;
            if (desiredValue && topicCategorySelect.querySelector(`option[value="${CSS.escape(desiredValue)}"]`)) {
                topicCategorySelect.value = desiredValue;
            } else {
                topicCategorySelect.value = '';
            }
        };

        const highlightCategoryRow = (categoryId) => {
            if (!categoryTable) {
                return;
            }
            Array.from(categoryTable.querySelectorAll('tr')).forEach((row) => {
                if (!row.dataset || !row.dataset.id) {
                    return;
                }
                row.classList.toggle('is-selected', row.dataset.id === categoryId);
            });
        };

        const renderCategories = () => {
            if (!categoryTable) {
                return;
            }
            const filterValue = categorySearchInput?.value ? categorySearchInput.value.trim().toLowerCase() : '';
            const filtered = state.categories.filter((category) => {
                if (!filterValue) {
                    return true;
                }
                const name = (category?.name || category?.Name || '').toLowerCase();
                return name.includes(filterValue);
            });

            categoryTable.innerHTML = '';

            if (!filtered.length) {
                const placeholder = document.createElement('tr');
                placeholder.className = 'admin-table__placeholder';
                const cell = document.createElement('td');
                cell.colSpan = 3;
                cell.textContent = filterValue
                    ? 'No categories match your search.'
                    : 'No categories available yet.';
                placeholder.appendChild(cell);
                categoryTable.appendChild(placeholder);
                return;
            }

            const fragment = document.createDocumentFragment();
            filtered.forEach((category) => {
                const row = document.createElement('tr');
                const identifier = category?.id ?? category?.ID;
                if (identifier) {
                    row.dataset.id = String(identifier);
                }
                if (String(identifier) === state.selectedCategoryId) {
                    row.classList.add('is-selected');
                }
                const topicCount = Number(category?.question_count ?? category?.QuestionCount ?? 0);
                const updatedValue =
                    resolveDateValue(category, 'updated_at', 'updatedAt', 'UpdatedAt') ||
                    resolveDateValue(category, 'created_at', 'createdAt', 'CreatedAt');

                row.innerHTML = `
                    <td>${escapeHtml(category?.name || category?.Name || '')}</td>
                    <td>${Number.isFinite(topicCount) ? topicCount : 0}</td>
                    <td>${escapeHtml(formatDateTime(updatedValue))}</td>
                `;

                fragment.appendChild(row);
            });

            categoryTable.appendChild(fragment);
        };

        const resetCategoryForm = () => {
            if (!categoryForm) {
                return;
            }
            categoryForm.reset();
            delete categoryForm.dataset.id;
            state.selectedCategoryId = '';
            highlightCategoryRow('');
            if (categorySubmitButton) {
                categorySubmitButton.textContent = 'Create category';
            }
            if (categoryDeleteButton) {
                categoryDeleteButton.hidden = true;
            }
            if (categoryStatus) {
                categoryStatus.textContent = '';
                categoryStatus.hidden = true;
            }
        };

        const populateCategoryForm = (category) => {
            if (!categoryForm || !category) {
                return;
            }
            const identifier = category?.id ?? category?.ID;
            if (identifier) {
                categoryForm.dataset.id = String(identifier);
                state.selectedCategoryId = String(identifier);
            }
            const nameField = categoryForm.querySelector('input[name="name"]');
            if (nameField) {
                nameField.value = category?.name || category?.Name || '';
            }
            if (categorySubmitButton) {
                categorySubmitButton.textContent = 'Update category';
            }
            if (categoryDeleteButton) {
                categoryDeleteButton.hidden = false;
            }
            if (categoryStatus) {
                const topicCount = Number(category?.question_count ?? category?.QuestionCount ?? 0);
                categoryStatus.textContent = `Assigned to ${topicCount} ${topicCount === 1 ? 'topic' : 'topics'}.`;
                categoryStatus.hidden = false;
            }
        };

        const selectCategory = (categoryId) => {
            if (!categoryId) {
                resetCategoryForm();
                return;
            }
            const category = state.categories.find((entry) => String(entry?.id ?? entry?.ID) === String(categoryId));
            if (!category) {
                resetCategoryForm();
                return;
            }
            highlightCategoryRow(String(categoryId));
            populateCategoryForm(category);
        };

        const loadCategories = async () => {
            if (!categoriesEndpoint) {
                return;
            }
            try {
                const payload = await apiClient(`${categoriesEndpoint}?include_counts=true`);
                state.categories = Array.isArray(payload?.categories) ? payload.categories : [];
                renderCategories();
                updateTopicCategoryOptions();
                if (state.selectedCategoryId) {
                    highlightCategoryRow(state.selectedCategoryId);
                }
            } catch (error) {
                showAlert(error.message || 'Failed to load forum categories.', 'error');
            }
        };

        const highlightTopicRow = (topicId) => {
            if (!topicTable) {
                return;
            }
            Array.from(topicTable.querySelectorAll('tr')).forEach((row) => {
                if (!row.dataset || !row.dataset.id) {
                    return;
                }
                row.classList.toggle('is-selected', row.dataset.id === topicId);
            });
        };

        const renderTopics = () => {
            if (!topicTable) {
                return;
            }
            const filterValue = topicSearchInput?.value ? topicSearchInput.value.trim().toLowerCase() : '';
            const filtered = state.topics.filter((topic) => {
                if (!filterValue) {
                    return true;
                }
                const titleMatch = (topic.title || '').toLowerCase().includes(filterValue);
                const authorMatch = (topic.author?.username || '')
                    .toLowerCase()
                    .includes(filterValue);
                const categoryMatch = resolveTopicCategoryName(topic)
                    .toLowerCase()
                    .includes(filterValue);
                return titleMatch || authorMatch || categoryMatch;
            });

            topicTable.innerHTML = '';

            if (!filtered.length) {
                const placeholder = document.createElement('tr');
                placeholder.className = 'admin-table__placeholder';
                const cell = document.createElement('td');
                cell.colSpan = 6;
                cell.textContent = filterValue
                    ? 'No topics match your search.'
                    : 'No topics available yet.';
                placeholder.appendChild(cell);
                topicTable.appendChild(placeholder);
                return;
            }

            const fragment = document.createDocumentFragment();
            filtered.forEach((topic) => {
                const row = document.createElement('tr');
                row.dataset.id = String(topic.id);
                if (String(topic.id) === state.selectedTopicId) {
                    row.classList.add('is-selected');
                }
                const author = topic.author?.username || '—';
                const categoryName = resolveTopicCategoryName(topic) || '—';
                const answersCount = Number.isFinite(topic.answers_count)
                    ? topic.answers_count
                    : Array.isArray(topic.answers)
                    ? topic.answers.length
                    : 0;
                const updatedValue =
                    resolveDateValue(topic, 'updated_at', 'updatedAt', 'UpdatedAt') ||
                    resolveDateValue(topic, 'created_at', 'createdAt', 'CreatedAt');

                row.innerHTML = `
                    <td>${escapeHtml(topic.title || '')}</td>
                    <td>${escapeHtml(categoryName)}</td>
                    <td>${escapeHtml(author)}</td>
                    <td>${answersCount}</td>
                    <td>${Number.isFinite(topic.rating) ? topic.rating : 0}</td>
                    <td>${escapeHtml(formatDateTime(updatedValue))}</td>
                `;

                fragment.appendChild(row);
            });
            topicTable.appendChild(fragment);
        };

        const renderAnswers = (answers) => {
            if (!answersList || !answerEmpty) {
                return;
            }

            answersList.innerHTML = '';
            state.selectedAnswerId = '';
            resetAnswerForm();

            if (!Array.isArray(answers) || answers.length === 0) {
                answerEmpty.textContent = 'No answers yet. Add the first response below.';
                answerEmpty.hidden = false;
                return;
            }

            answerEmpty.hidden = true;
            const fragment = document.createDocumentFragment();
            answers.forEach((answer) => {
                const item = document.createElement('li');
                item.className = 'admin-forum-answer';
                item.dataset.answerId = String(answer.id);

                const meta = document.createElement('div');
                meta.className = 'admin-forum-answer__meta';
                const author = document.createElement('span');
                author.textContent = answer.author?.username ? `by ${answer.author.username}` : 'Unknown author';
                const timing = document.createElement('span');
                const createdValue =
                    resolveDateValue(answer, 'updated_at', 'updatedAt', 'UpdatedAt', 'created_at', 'createdAt', 'CreatedAt');
                timing.textContent = formatDateTime(createdValue);
                meta.append(author, timing);

                const body = document.createElement('div');
                body.className = 'admin-forum-answer__content';
                body.textContent = answer.content || '';

                const actions = document.createElement('div');
                actions.className = 'admin-forum-answer__actions';
                const editButton = document.createElement('button');
                editButton.type = 'button';
                editButton.className = 'admin-navigation__button';
                editButton.dataset.action = 'forum-answer-edit';
                editButton.dataset.answerId = String(answer.id);
                editButton.textContent = 'Edit';
                const deleteButton = document.createElement('button');
                deleteButton.type = 'button';
                deleteButton.className = 'admin-navigation__button admin-navigation__button--danger';
                deleteButton.dataset.action = 'forum-answer-delete';
                deleteButton.dataset.answerId = String(answer.id);
                deleteButton.textContent = 'Delete';
                actions.append(editButton, deleteButton);

                const rating = document.createElement('span');
                rating.className = 'admin-card__description';
                rating.textContent = `Rating: ${Number.isFinite(answer.rating) ? answer.rating : 0}`;

                item.append(meta, body, rating, actions);
                fragment.appendChild(item);
            });

            answersList.appendChild(fragment);
        };

        const resetTopicForm = () => {
            if (topicForm) {
                topicForm.reset();
                delete topicForm.dataset.id;
            }
            state.selectedTopicId = '';
            highlightTopicRow('');
            if (topicCategorySelect) {
                topicCategorySelect.value = '';
            }
            if (topicStatus) {
                topicStatus.textContent = '';
                topicStatus.hidden = true;
            }
            if (topicDeleteButton) {
                topicDeleteButton.hidden = true;
            }
            if (topicSubmitButton) {
                topicSubmitButton.textContent = 'Save topic';
            }
            resetAnswersView();
            renderTopics();
        };

        const populateTopicForm = (topic) => {
            if (!topicForm || !topic) {
                return;
            }
            topicForm.dataset.id = String(topic.id);
            const titleField = topicForm.querySelector('input[name="title"]');
            const contentField = topicForm.querySelector('textarea[name="content"]');
            if (titleField) {
                titleField.value = topic.title || '';
            }
            if (contentField) {
                contentField.value = topic.content || '';
            }
            if (topicCategorySelect) {
                const categoryId = resolveTopicCategoryId(topic);
                if (
                    categoryId &&
                    topicCategorySelect.querySelector(`option[value="${CSS.escape(categoryId)}"]`)
                ) {
                    topicCategorySelect.value = categoryId;
                } else {
                    topicCategorySelect.value = '';
                }
            }
            if (topicSubmitButton) {
                topicSubmitButton.textContent = 'Update topic';
            }
            if (topicDeleteButton) {
                topicDeleteButton.hidden = false;
            }
            if (topicStatus) {
                const answersCount = Array.isArray(topic.answers) ? topic.answers.length : topic.answers_count || 0;
                topicStatus.textContent = `Rating ${Number.isFinite(topic.rating) ? topic.rating : 0} · ${answersCount} answers · ${Number.isFinite(topic.views) ? topic.views : 0} views`;
                topicStatus.hidden = false;
            }
        };

        const loadTopicDetail = async (topicId) => {
            if (!topicId) {
                return null;
            }
            try {
                const payload = await apiClient(`${buildTopicEndpoint(topicId)}?increment=false`);
                return payload?.topic || payload?.question || null;
            } catch (error) {
                showAlert(error.message || 'Failed to load topic details.', 'error');
                return null;
            }
        };

        const selectTopic = async (topicId) => {
            if (!topicId) {
                resetTopicForm();
                return;
            }
            const topic = await loadTopicDetail(topicId);
            if (!topic) {
                return;
            }
            state.selectedTopicId = String(topic.id);
            highlightTopicRow(state.selectedTopicId);
            populateTopicForm(topic);
            renderAnswers(topic.answers || []);
            setAnswerFormEnabled(true);
            if (answerForm) {
                answerForm.dataset.topicId = String(topic.id);
            }
            if (answerEmpty) {
                answerEmpty.textContent = 'No answers yet. Add the first response below.';
            }
        };

        const loadTopics = async () => {
            try {
                const payload = await apiClient(`${topicsEndpoint}?limit=100`);
                const list = Array.isArray(payload?.topics) ? payload.topics : payload?.questions;
                state.topics = Array.isArray(list) ? list : [];
                renderTopics();
                if (state.selectedTopicId) {
                    highlightTopicRow(state.selectedTopicId);
                }
            } catch (error) {
                showAlert(error.message || 'Failed to load forum topics.', 'error');
            }
        };

        if (topicTable) {
            topicTable.addEventListener('click', (event) => {
                const target = event.target;
                if (!(target instanceof Element)) {
                    return;
                }
                const row = target.closest('tr');
                if (!row || !row.dataset.id) {
                    return;
                }
                event.preventDefault();
                selectTopic(row.dataset.id);
            });
        }

        if (topicSearchInput) {
            topicSearchInput.addEventListener('input', () => {
                renderTopics();
            });
        }

        if (topicResetButton) {
            topicResetButton.addEventListener('click', () => {
                resetTopicForm();
            });
        }

        if (topicForm) {
            topicForm.addEventListener('submit', async (event) => {
                event.preventDefault();
                const form = event.currentTarget;
                const formData = new FormData(form);
                const title = (formData.get('title') || '').toString().trim();
                const content = (formData.get('content') || '').toString().trim();
                if (!title || !content) {
                    showAlert('Title and content are required to save a topic.', 'error');
                    return;
                }
                const topicId = topicForm.dataset.id;
                const method = topicId ? 'PUT' : 'POST';
                const endpoint = topicId
                    ? buildTopicEndpoint(topicId, '', { useManagementEndpoint: true })
                    : topicManageEndpoint;
                const rawCategory = formData.get('category_id');
                const categoryValue = rawCategory === null ? '' : String(rawCategory).trim();
                let categoryId = null;
                if (categoryValue) {
                    const parsed = Number(categoryValue);
                    if (Number.isFinite(parsed) && parsed > 0) {
                        categoryId = parsed;
                    }
                }
                const payload = { title, content };
                if (topicId) {
                    payload.category_id = categoryValue ? categoryId : null;
                } else if (categoryId !== null) {
                    payload.category_id = categoryId;
                }
                try {
                    toggleFormDisabled(topicForm, true);
                    const response = await apiClient(endpoint, {
                        method,
                        body: JSON.stringify(payload),
                    });
                    showAlert(topicId ? 'Topic updated successfully.' : 'Topic created successfully.', 'success');
                    await loadTopics();
                    const createdId = response?.topic?.id ?? response?.question?.id;
                    if (createdId) {
                        await selectTopic(createdId);
                    } else if (topicId) {
                        await selectTopic(topicId);
                    } else if (state.topics.length) {
                        const latest = state.topics[0];
                        await selectTopic(latest.id);
                    }
                } catch (error) {
                    showAlert(error.message || 'Failed to save topic.', 'error');
                } finally {
                    toggleFormDisabled(topicForm, false);
                }
            });
        }

        if (topicDeleteButton) {
            topicDeleteButton.addEventListener('click', async () => {
                const topicId = topicForm?.dataset.id;
                if (!topicId) {
                    return;
                }
                const confirmed = window.confirm('Delete this topic and all associated answers?');
                if (!confirmed) {
                    return;
                }
                try {
                    toggleFormDisabled(topicForm, true);
                    await apiClient(buildTopicEndpoint(topicId, '', { useManagementEndpoint: true }), {
                        method: 'DELETE',
                    });
                    showAlert('Topic deleted successfully.', 'success');
                    resetTopicForm();
                    await loadTopics();
                } catch (error) {
                    showAlert(error.message || 'Failed to delete topic.', 'error');
                } finally {
                    toggleFormDisabled(topicForm, false);
                }
            });
        }

        if (categoryTable) {
            categoryTable.addEventListener('click', (event) => {
                const target = event.target;
                if (!(target instanceof Element)) {
                    return;
                }
                const row = target.closest('tr');
                if (!row || !row.dataset.id) {
                    return;
                }
                event.preventDefault();
                selectCategory(row.dataset.id);
            });
        }

        if (categorySearchInput) {
            categorySearchInput.addEventListener('input', () => {
                renderCategories();
            });
        }

        if (categoryResetButton) {
            categoryResetButton.addEventListener('click', () => {
                resetCategoryForm();
                renderCategories();
            });
        }

        if (categoryForm) {
            categoryForm.addEventListener('submit', async (event) => {
                event.preventDefault();
                if (!categoriesEndpoint) {
                    showAlert('Category management is unavailable right now.', 'error');
                    return;
                }
                const form = event.currentTarget;
                const formData = new FormData(form);
                const name = (formData.get('name') || '').toString().trim();
                if (!name) {
                    showAlert('Category name is required.', 'error');
                    return;
                }
                const categoryId = categoryForm.dataset.id;
                const method = categoryId ? 'PUT' : 'POST';
                const endpoint = categoryId
                    ? `${categoriesEndpoint}/${categoryId}`
                    : categoriesEndpoint;
                const payload = { name };
                try {
                    toggleFormDisabled(categoryForm, true);
                    const response = await apiClient(endpoint, {
                        method,
                        body: JSON.stringify(payload),
                    });
                    showAlert(
                        categoryId ? 'Category updated successfully.' : 'Category created successfully.',
                        'success'
                    );
                    await loadCategories();
                    const createdId = response?.category?.id ?? response?.category?.ID;
                    if (createdId) {
                        selectCategory(createdId);
                    } else if (categoryId) {
                        selectCategory(categoryId);
                    } else {
                        resetCategoryForm();
                    }
                } catch (error) {
                    showAlert(error.message || 'Failed to save category.', 'error');
                } finally {
                    toggleFormDisabled(categoryForm, false);
                }
            });
        }

        if (categoryDeleteButton) {
            categoryDeleteButton.addEventListener('click', async () => {
                if (!categoriesEndpoint) {
                    showAlert('Category management is unavailable right now.', 'error');
                    return;
                }
                const categoryId = categoryForm?.dataset.id;
                if (!categoryId) {
                    return;
                }
                const confirmed = window.confirm(
                    'Delete this category? Topics assigned to it will become uncategorised.'
                );
                if (!confirmed) {
                    return;
                }
                try {
                    toggleFormDisabled(categoryForm, true);
                    await apiClient(`${categoriesEndpoint}/${categoryId}`, { method: 'DELETE' });
                    showAlert('Category deleted successfully.', 'success');
                    resetCategoryForm();
                    await loadCategories();
                } catch (error) {
                    showAlert(error.message || 'Failed to delete category.', 'error');
                } finally {
                    toggleFormDisabled(categoryForm, false);
                }
            });
        }

        if (answersList) {
            answersList.addEventListener('click', (event) => {
                const target = event.target;
                if (!(target instanceof Element)) {
                    return;
                }
                const actionButton = target.closest('[data-action]');
                if (!actionButton) {
                    return;
                }
                const answerId = actionButton.dataset.answerId;
                if (!answerId) {
                    return;
                }
                if (actionButton.dataset.action === 'forum-answer-edit') {
                    const listItem = actionButton.closest('.admin-forum-answer');
                    if (!listItem) {
                        return;
                    }
                    const content = listItem.querySelector('.admin-forum-answer__content')?.textContent || '';
                    state.selectedAnswerId = answerId;
                    if (answerContentInput) {
                        answerContentInput.value = content;
                        answerContentInput.focus();
                    }
                    if (answerForm) {
                        answerForm.dataset.id = answerId;
                    }
                    if (answerSubmitButton) {
                        answerSubmitButton.textContent = 'Update answer';
                    }
                    answerCancelButton.hidden = false;
                    answerDeleteButton.hidden = false;
                } else if (actionButton.dataset.action === 'forum-answer-delete') {
                    const confirmed = window.confirm('Delete this answer?');
                    if (!confirmed) {
                        return;
                    }
                    const topicId = state.selectedTopicId;
                    if (!topicId) {
                        return;
                    }
                    const endpoint = buildAnswerEndpoint(answerId);
                    if (!endpoint) {
                        showAlert('Answer endpoint not configured.', 'error');
                        return;
                    }
                    toggleFormDisabled(answerForm, true);
                    apiClient(endpoint, { method: 'DELETE' })
                        .then(async () => {
                            showAlert('Answer deleted.', 'success');
                            resetAnswerForm();
                            const updated = await loadTopicDetail(topicId);
                            if (updated) {
                                renderAnswers(updated.answers || []);
                            }
                            await loadTopics();
                        })
                        .catch((error) => {
                            showAlert(error.message || 'Failed to delete answer.', 'error');
                        })
                        .finally(() => {
                            toggleFormDisabled(answerForm, false);
                        });
                }
            });
        }

        if (answerForm) {
            answerForm.addEventListener('submit', async (event) => {
                event.preventDefault();
                const topicId = state.selectedTopicId;
                if (!topicId) {
                    showAlert('Select a topic before adding an answer.', 'error');
                    return;
                }
                const content = (answerContentInput?.value || '').trim();
                if (!content) {
                    showAlert('Answer content cannot be empty.', 'error');
                    return;
                }
                const answerId = answerForm.dataset.id;
                const isUpdate = Boolean(answerId);
                const method = isUpdate ? 'PUT' : 'POST';
                const endpoint = isUpdate
                    ? buildAnswerEndpoint(answerId)
                    : `${buildTopicEndpoint(topicId, '', { useManagementEndpoint: true })}/answers`;
                const payload = isUpdate ? { content } : { content };
                try {
                    toggleFormDisabled(answerForm, true);
                    await apiClient(endpoint, {
                        method,
                        body: JSON.stringify(payload),
                    });
                    showAlert(isUpdate ? 'Answer updated.' : 'Answer created.', 'success');
                    resetAnswerForm();
                    const updated = await loadTopicDetail(topicId);
                    if (updated) {
                        renderAnswers(updated.answers || []);
                    }
                    await loadTopics();
                } catch (error) {
                    showAlert(error.message || 'Failed to save answer.', 'error');
                } finally {
                    toggleFormDisabled(answerForm, false);
                }
            });
        }

        if (answerCancelButton) {
            answerCancelButton.addEventListener('click', () => {
                resetAnswerForm();
            });
        }

        if (answerDeleteButton) {
            answerDeleteButton.addEventListener('click', async () => {
                const topicId = state.selectedTopicId;
                const answerId = answerForm?.dataset.id;
                if (!topicId || !answerId) {
                    return;
                }
                const confirmed = window.confirm('Delete this answer?');
                if (!confirmed) {
                    return;
                }
                const endpoint = buildAnswerEndpoint(answerId);
                if (!endpoint) {
                    showAlert('Answer endpoint not configured.', 'error');
                    return;
                }
                try {
                    toggleFormDisabled(answerForm, true);
                    await apiClient(endpoint, { method: 'DELETE' });
                    showAlert('Answer deleted.', 'success');
                    resetAnswerForm();
                    const updated = await loadTopicDetail(topicId);
                    if (updated) {
                        renderAnswers(updated.answers || []);
                    }
                    await loadTopics();
                } catch (error) {
                    showAlert(error.message || 'Failed to delete answer.', 'error');
                } finally {
                    toggleFormDisabled(answerForm, false);
                }
            });
        }

        resetTopicForm();
        loadCategories();
        loadTopics();
    });
})();

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

        const questionsEndpointRaw = (context.dataset?.endpointForumQuestions || '').trim();
        const adminQuestionsEndpointRaw = (context.dataset?.endpointAdminForumQuestions || '').trim();
        if (!questionsEndpointRaw) {
            return;
        }
        const answersEndpointRaw = (context.dataset?.endpointForumAnswers || '').trim();
        const categoriesEndpointRaw = (context.dataset?.endpointForumCategories || '').trim();
        const questionsEndpoint = questionsEndpointRaw.replace(/\/+$/, '');
        const adminQuestionsEndpoint = adminQuestionsEndpointRaw
            ? adminQuestionsEndpointRaw.replace(/\/+$/, '')
            : '';
        const answersEndpoint = answersEndpointRaw ? answersEndpointRaw.replace(/\/+$/, '') : '';
        const categoriesEndpoint = categoriesEndpointRaw ? categoriesEndpointRaw.replace(/\/+$/, '') : '';
        const questionManageEndpoint = adminQuestionsEndpoint || questionsEndpoint;

        const questionTable = panel.querySelector('#admin-forum-questions-table');
        const questionForm = panel.querySelector('#admin-forum-question-form');
        const questionStatus = panel.querySelector('[data-role="forum-question-status"]');
        const questionSubmitButton = panel.querySelector('[data-role="forum-question-submit"]');
        const questionDeleteButton = panel.querySelector('[data-role="forum-question-delete"]');
        const questionResetButton = panel.querySelector('[data-action="forum-question-reset"]');
        const searchInput = panel.querySelector('[data-role="forum-question-search"]');
        const questionCategorySelect = panel.querySelector('[data-role="forum-question-category"]');

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
            questions: [],
            categories: [],
            selectedQuestionId: '',
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

        const buildQuestionEndpoint = (id, suffix = '', options = {}) => {
            const useManagementEndpoint = Boolean(options?.useManagementEndpoint);
            const baseEndpoint = useManagementEndpoint ? questionManageEndpoint : questionsEndpoint;
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
                answerEmpty.textContent = 'Select a question to see submitted answers.';
                answerEmpty.hidden = false;
            }
            if (answerForm) {
                delete answerForm.dataset.questionId;
            }
            resetAnswerForm();
            setAnswerFormEnabled(false);
        };

        const resolveQuestionCategoryId = (question) => {
            if (!question) {
                return '';
            }
            const direct = question.category_id ?? question.CategoryID;
            if (direct !== undefined && direct !== null) {
                const numeric = Number(direct);
                return Number.isFinite(numeric) && numeric > 0 ? String(numeric) : '';
            }
            const category = question.category || question.Category;
            if (category && (category.id || category.ID)) {
                const numeric = Number(category.id ?? category.ID);
                return Number.isFinite(numeric) && numeric > 0 ? String(numeric) : '';
            }
            return '';
        };

        const resolveQuestionCategoryName = (question) => {
            if (!question) {
                return '';
            }
            const category = question.category || question.Category;
            if (category && (category.name || category.Name)) {
                return category.name || category.Name || '';
            }
            const categoryId = resolveQuestionCategoryId(question);
            if (!categoryId) {
                return '';
            }
            const matched = state.categories.find((entry) => {
                const identifier = entry?.id ?? entry?.ID;
                return identifier !== undefined && String(identifier) === categoryId;
            });
            if (matched) {
                return matched.name || matched.Name || '';
            }
            return '';
        };

        const updateQuestionCategoryOptions = () => {
            if (!questionCategorySelect) {
                return;
            }
            const previousValue = questionCategorySelect.value;
            questionCategorySelect.innerHTML = '';
            const defaultOption = document.createElement('option');
            defaultOption.value = '';
            defaultOption.textContent = 'No category';
            questionCategorySelect.appendChild(defaultOption);
            state.categories.forEach((category) => {
                const identifier = category?.id ?? category?.ID;
                if (!identifier) {
                    return;
                }
                const option = document.createElement('option');
                option.value = String(identifier);
                option.textContent = category?.name || category?.Name || `Category ${identifier}`;
                questionCategorySelect.appendChild(option);
            });

            const selectedQuestion = state.questions.find(
                (question) => String(question?.id) === state.selectedQuestionId
            );
            const desiredValue = selectedQuestion ? resolveQuestionCategoryId(selectedQuestion) : previousValue;
            if (desiredValue && questionCategorySelect.querySelector(`option[value="${CSS.escape(desiredValue)}"]`)) {
                questionCategorySelect.value = desiredValue;
            } else {
                questionCategorySelect.value = '';
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
                const questions = Number(category?.question_count ?? category?.QuestionCount ?? 0);
                const updatedValue =
                    resolveDateValue(category, 'updated_at', 'updatedAt', 'UpdatedAt') ||
                    resolveDateValue(category, 'created_at', 'createdAt', 'CreatedAt');

                row.innerHTML = `
                    <td>${escapeHtml(category?.name || category?.Name || '')}</td>
                    <td>${Number.isFinite(questions) ? questions : 0}</td>
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
                const questions = Number(category?.question_count ?? category?.QuestionCount ?? 0);
                categoryStatus.textContent = `Assigned to ${questions} ${questions === 1 ? 'question' : 'questions'}.`;
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
                updateQuestionCategoryOptions();
                if (state.selectedCategoryId) {
                    highlightCategoryRow(state.selectedCategoryId);
                }
            } catch (error) {
                showAlert(error.message || 'Failed to load forum categories.', 'error');
            }
        };

        const highlightQuestionRow = (questionId) => {
            if (!questionTable) {
                return;
            }
            Array.from(questionTable.querySelectorAll('tr')).forEach((row) => {
                if (!row.dataset || !row.dataset.id) {
                    return;
                }
                row.classList.toggle('is-selected', row.dataset.id === questionId);
            });
        };

        const renderQuestions = () => {
            if (!questionTable) {
                return;
            }
            const filterValue = searchInput?.value ? searchInput.value.trim().toLowerCase() : '';
            const filtered = state.questions.filter((question) => {
                if (!filterValue) {
                    return true;
                }
                const titleMatch = (question.title || '').toLowerCase().includes(filterValue);
                const authorMatch = (question.author?.username || '')
                    .toLowerCase()
                    .includes(filterValue);
                const categoryMatch = resolveQuestionCategoryName(question)
                    .toLowerCase()
                    .includes(filterValue);
                return titleMatch || authorMatch || categoryMatch;
            });

            questionTable.innerHTML = '';

            if (!filtered.length) {
                const placeholder = document.createElement('tr');
                placeholder.className = 'admin-table__placeholder';
                const cell = document.createElement('td');
                cell.colSpan = 6;
                cell.textContent = filterValue
                    ? 'No questions match your search.'
                    : 'No questions available yet.';
                placeholder.appendChild(cell);
                questionTable.appendChild(placeholder);
                return;
            }

            const fragment = document.createDocumentFragment();
            filtered.forEach((question) => {
                const row = document.createElement('tr');
                row.dataset.id = String(question.id);
                if (String(question.id) === state.selectedQuestionId) {
                    row.classList.add('is-selected');
                }
                const author = question.author?.username || '—';
                const categoryName = resolveQuestionCategoryName(question) || '—';
                const answersCount = Number.isFinite(question.answers_count)
                    ? question.answers_count
                    : Array.isArray(question.answers)
                    ? question.answers.length
                    : 0;
                const updatedValue =
                    resolveDateValue(question, 'updated_at', 'updatedAt', 'UpdatedAt') ||
                    resolveDateValue(question, 'created_at', 'createdAt', 'CreatedAt');

                row.innerHTML = `
                    <td>${escapeHtml(question.title || '')}</td>
                    <td>${escapeHtml(categoryName)}</td>
                    <td>${escapeHtml(author)}</td>
                    <td>${answersCount}</td>
                    <td>${Number.isFinite(question.rating) ? question.rating : 0}</td>
                    <td>${escapeHtml(formatDateTime(updatedValue))}</td>
                `;

                fragment.appendChild(row);
            });
            questionTable.appendChild(fragment);
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

        const resetQuestionForm = () => {
            if (questionForm) {
                questionForm.reset();
                delete questionForm.dataset.id;
            }
            state.selectedQuestionId = '';
            highlightQuestionRow('');
            if (questionCategorySelect) {
                questionCategorySelect.value = '';
            }
            if (questionStatus) {
                questionStatus.textContent = '';
                questionStatus.hidden = true;
            }
            if (questionDeleteButton) {
                questionDeleteButton.hidden = true;
            }
            if (questionSubmitButton) {
                questionSubmitButton.textContent = 'Save question';
            }
            resetAnswersView();
            renderQuestions();
        };

        const populateQuestionForm = (question) => {
            if (!questionForm || !question) {
                return;
            }
            questionForm.dataset.id = String(question.id);
            const titleField = questionForm.querySelector('input[name="title"]');
            const contentField = questionForm.querySelector('textarea[name="content"]');
            if (titleField) {
                titleField.value = question.title || '';
            }
            if (contentField) {
                contentField.value = question.content || '';
            }
            if (questionCategorySelect) {
                const categoryId = resolveQuestionCategoryId(question);
                if (
                    categoryId &&
                    questionCategorySelect.querySelector(`option[value="${CSS.escape(categoryId)}"]`)
                ) {
                    questionCategorySelect.value = categoryId;
                } else {
                    questionCategorySelect.value = '';
                }
            }
            if (questionSubmitButton) {
                questionSubmitButton.textContent = 'Update question';
            }
            if (questionDeleteButton) {
                questionDeleteButton.hidden = false;
            }
            if (questionStatus) {
                const answersCount = Array.isArray(question.answers) ? question.answers.length : question.answers_count || 0;
                questionStatus.textContent = `Rating ${Number.isFinite(question.rating) ? question.rating : 0} · ${answersCount} answers · ${Number.isFinite(question.views) ? question.views : 0} views`;
                questionStatus.hidden = false;
            }
        };

        const loadQuestionDetail = async (questionId) => {
            if (!questionId) {
                return null;
            }
            try {
                const payload = await apiClient(`${buildQuestionEndpoint(questionId)}?increment=false`);
                return payload?.question || null;
            } catch (error) {
                showAlert(error.message || 'Failed to load question details.', 'error');
                return null;
            }
        };

        const selectQuestion = async (questionId) => {
            if (!questionId) {
                resetQuestionForm();
                return;
            }
            const question = await loadQuestionDetail(questionId);
            if (!question) {
                return;
            }
            state.selectedQuestionId = String(question.id);
            highlightQuestionRow(state.selectedQuestionId);
            populateQuestionForm(question);
            renderAnswers(question.answers || []);
            setAnswerFormEnabled(true);
            if (answerForm) {
                answerForm.dataset.questionId = String(question.id);
            }
            if (answerEmpty) {
                answerEmpty.textContent = 'No answers yet. Add the first response below.';
            }
        };

        const loadQuestions = async () => {
            try {
                const payload = await apiClient(`${questionsEndpoint}?limit=100`);
                state.questions = Array.isArray(payload?.questions) ? payload.questions : [];
                renderQuestions();
                if (state.selectedQuestionId) {
                    highlightQuestionRow(state.selectedQuestionId);
                }
            } catch (error) {
                showAlert(error.message || 'Failed to load forum questions.', 'error');
            }
        };

        if (questionTable) {
            questionTable.addEventListener('click', (event) => {
                const target = event.target;
                if (!(target instanceof Element)) {
                    return;
                }
                const row = target.closest('tr');
                if (!row || !row.dataset.id) {
                    return;
                }
                event.preventDefault();
                selectQuestion(row.dataset.id);
            });
        }

        if (searchInput) {
            searchInput.addEventListener('input', () => {
                renderQuestions();
            });
        }

        if (questionResetButton) {
            questionResetButton.addEventListener('click', () => {
                resetQuestionForm();
            });
        }

        if (questionForm) {
            questionForm.addEventListener('submit', async (event) => {
                event.preventDefault();
                const form = event.currentTarget;
                const formData = new FormData(form);
                const title = (formData.get('title') || '').toString().trim();
                const content = (formData.get('content') || '').toString().trim();
                if (!title || !content) {
                    showAlert('Title and content are required to save a question.', 'error');
                    return;
                }
                const questionId = questionForm.dataset.id;
                const method = questionId ? 'PUT' : 'POST';
                const endpoint = questionId
                    ? buildQuestionEndpoint(questionId, '', { useManagementEndpoint: true })
                    : questionManageEndpoint;
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
                if (questionId) {
                    payload.category_id = categoryValue ? categoryId : null;
                } else if (categoryId !== null) {
                    payload.category_id = categoryId;
                }
                try {
                    toggleFormDisabled(questionForm, true);
                    const response = await apiClient(endpoint, {
                        method,
                        body: JSON.stringify(payload),
                    });
                    showAlert(questionId ? 'Question updated successfully.' : 'Question created successfully.', 'success');
                    await loadQuestions();
                    const createdId = response?.question?.id;
                    if (createdId) {
                        await selectQuestion(createdId);
                    } else if (questionId) {
                        await selectQuestion(questionId);
                    } else if (state.questions.length) {
                        const latest = state.questions[0];
                        await selectQuestion(latest.id);
                    }
                } catch (error) {
                    showAlert(error.message || 'Failed to save question.', 'error');
                } finally {
                    toggleFormDisabled(questionForm, false);
                }
            });
        }

        if (questionDeleteButton) {
            questionDeleteButton.addEventListener('click', async () => {
                const questionId = questionForm?.dataset.id;
                if (!questionId) {
                    return;
                }
                const confirmed = window.confirm('Delete this question and all associated answers?');
                if (!confirmed) {
                    return;
                }
                try {
                    toggleFormDisabled(questionForm, true);
                    await apiClient(buildQuestionEndpoint(questionId, '', { useManagementEndpoint: true }), {
                        method: 'DELETE',
                    });
                    showAlert('Question deleted successfully.', 'success');
                    resetQuestionForm();
                    await loadQuestions();
                } catch (error) {
                    showAlert(error.message || 'Failed to delete question.', 'error');
                } finally {
                    toggleFormDisabled(questionForm, false);
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
                    'Delete this category? Questions assigned to it will become uncategorised.'
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
                    const questionId = state.selectedQuestionId;
                    if (!questionId) {
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
                            const updated = await loadQuestionDetail(questionId);
                            if (updated) {
                                renderAnswers(updated.answers || []);
                            }
                            await loadQuestions();
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
                const questionId = state.selectedQuestionId;
                if (!questionId) {
                    showAlert('Select a question before adding an answer.', 'error');
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
                    : `${buildQuestionEndpoint(questionId, '', { useManagementEndpoint: true })}/answers`;
                const payload = isUpdate ? { content } : { content };
                try {
                    toggleFormDisabled(answerForm, true);
                    await apiClient(endpoint, {
                        method,
                        body: JSON.stringify(payload),
                    });
                    showAlert(isUpdate ? 'Answer updated.' : 'Answer created.', 'success');
                    resetAnswerForm();
                    const updated = await loadQuestionDetail(questionId);
                    if (updated) {
                        renderAnswers(updated.answers || []);
                    }
                    await loadQuestions();
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
                const questionId = state.selectedQuestionId;
                const answerId = answerForm?.dataset.id;
                if (!questionId || !answerId) {
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
                    const updated = await loadQuestionDetail(questionId);
                    if (updated) {
                        renderAnswers(updated.answers || []);
                    }
                    await loadQuestions();
                } catch (error) {
                    showAlert(error.message || 'Failed to delete answer.', 'error');
                } finally {
                    toggleFormDisabled(answerForm, false);
                }
            });
        }

        resetQuestionForm();
        loadCategories();
        loadQuestions();
    });
})();

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
        if (!questionsEndpointRaw) {
            return;
        }
        const answersEndpointRaw = (context.dataset?.endpointForumAnswers || '').trim();
        const questionsEndpoint = questionsEndpointRaw.replace(/\/+$/, '');
        const answersEndpoint = answersEndpointRaw ? answersEndpointRaw.replace(/\/+$/, '') : '';

        const questionTable = panel.querySelector('#admin-forum-questions-table');
        const questionForm = panel.querySelector('#admin-forum-question-form');
        const questionStatus = panel.querySelector('[data-role="forum-question-status"]');
        const questionSubmitButton = panel.querySelector('[data-role="forum-question-submit"]');
        const questionDeleteButton = panel.querySelector('[data-role="forum-question-delete"]');
        const questionResetButton = panel.querySelector('[data-action="forum-question-reset"]');
        const searchInput = panel.querySelector('[data-role="forum-question-search"]');

        const answersContainer = panel.querySelector('[data-role="forum-answer-container"]');
        const answersList = panel.querySelector('[data-role="forum-answer-list"]');
        const answerEmpty = panel.querySelector('[data-role="forum-answer-empty"]');
        const answerForm = panel.querySelector('#admin-forum-answer-form');
        const answerContentInput = panel.querySelector('[data-role="forum-answer-content"]');
        const answerSubmitButton = panel.querySelector('[data-role="forum-answer-submit"]');
        const answerCancelButton = panel.querySelector('[data-role="forum-answer-cancel"]');
        const answerDeleteButton = panel.querySelector('[data-role="forum-answer-delete"]');

        const state = {
            questions: [],
            selectedQuestionId: '',
            selectedAnswerId: '',
        };

        const showAlert = (message, type = 'info') => {
            if (!message) {
                setAlert(globalAlertId, '', type);
                return;
            }
            setAlert(globalAlertId, message, type);
        };

        const buildQuestionEndpoint = (id, suffix = '') => {
            if (!id) {
                return questionsEndpoint;
            }
            const base = `${questionsEndpoint}/${id}`;
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
                return titleMatch || authorMatch;
            });

            questionTable.innerHTML = '';

            if (!filtered.length) {
                const placeholder = document.createElement('tr');
                placeholder.className = 'admin-table__placeholder';
                const cell = document.createElement('td');
                cell.colSpan = 5;
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
                const endpoint = questionId ? buildQuestionEndpoint(questionId) : questionsEndpoint;
                const payload = questionId ? { title, content } : { title, content };
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
                    await apiClient(buildQuestionEndpoint(questionId), {
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
                    : `${buildQuestionEndpoint(questionId)}/answers`;
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
        loadQuestions();
    });
})();

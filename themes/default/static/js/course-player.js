(() => {
    const formatDuration = (totalSeconds) => {
        if (typeof totalSeconds !== "number" || Number.isNaN(totalSeconds) || totalSeconds <= 0) {
            return "";
        }
        const seconds = Math.floor(totalSeconds % 60);
        const minutes = Math.floor(totalSeconds / 60) % 60;
        const hours = Math.floor(totalSeconds / 3600);
        if (hours > 0) {
            const minPart = minutes > 0 ? `${minutes}m` : "";
            return `${hours}h${minPart ? ` ${minPart}` : ""}`;
        }
        if (minutes > 0) {
            return `${minutes}m${seconds > 0 ? ` ${seconds}s` : ""}`;
        }
        return `${seconds}s`;
    };

    const pluralize = (count, singular, plural) => {
        return `${count} ${count === 1 ? singular : plural}`;
    };

    const parseInitialData = (element) => {
        if (!element) {
            return null;
        }
        const raw = element.textContent || element.innerText || "";
        if (!raw.trim()) {
            return null;
        }
        try {
            return JSON.parse(raw);
        } catch (error) {
            console.warn("Failed to parse course payload", error);
            return null;
        }
    };

    const normalizeNumber = (value) => {
        const parsed = Number(value);
        return Number.isFinite(parsed) ? parsed : undefined;
    };

    const normaliseAttachment = (attachment) => {
        if (!attachment || typeof attachment !== "object") {
            return null;
        }
        const rawUrl =
            (typeof attachment.url === "string" && attachment.url) ||
            (typeof attachment.URL === "string" && attachment.URL) ||
            "";
        const url = rawUrl.trim();
        if (!url) {
            return null;
        }
        const title =
            (typeof attachment.title === "string" && attachment.title) ||
            (typeof attachment.Title === "string" && attachment.Title) ||
            "";
        return {
            url,
            title,
        };
    };

    const deriveAttachmentLabel = (attachment) => {
        if (!attachment) {
            return "Download";
        }
        const rawTitle = typeof attachment.title === "string" ? attachment.title.trim() : "";
        if (rawTitle) {
            return rawTitle;
        }
        const url = typeof attachment.url === "string" ? attachment.url.trim() : "";
        if (!url) {
            return "Download";
        }
        let candidate = url;
        try {
            const parsed = new URL(url, window.location?.origin || "http://localhost");
            if (parsed.pathname) {
                candidate = parsed.pathname;
            }
        } catch (error) {
            // fall back to the raw URL
        }
        const segments = candidate.split("/").filter(Boolean);
        const lastSegment = segments.length ? segments[segments.length - 1] : candidate;
        try {
            const decoded = decodeURIComponent(lastSegment);
            if (decoded.trim()) {
                return decoded.trim();
            }
        } catch (error) {
            // ignore decoding failure
        }
        return lastSegment || "Download";
    };

    const deriveAttachmentFileName = (attachment, fallbackLabel = "") => {
        const label = fallbackLabel.trim();
        if (label) {
            return label;
        }
        const url = typeof attachment?.url === "string" ? attachment.url : "";
        if (!url) {
            return "";
        }
        const segments = url.split("/").filter(Boolean);
        const lastSegment = segments.length ? segments[segments.length - 1] : url;
        try {
            return decodeURIComponent(lastSegment);
        } catch (error) {
            return lastSegment;
        }
    };

    const getAttachmentExtension = (url) => {
        if (typeof url !== "string") {
            return "";
        }
        const match = url.match(/\.([a-z0-9]{2,8})(?:$|[?#])/i);
        return match ? match[1].toUpperCase() : "";
    };

    const normaliseVideoAttachments = (video) => {
        const source = Array.isArray(video?.attachments)
            ? video.attachments
            : Array.isArray(video?.Attachments)
            ? video.Attachments
            : [];
        const seen = new Set();
        const attachments = [];
        source.forEach((entry) => {
            const normalised = normaliseAttachment(entry);
            if (!normalised || seen.has(normalised.url)) {
                return;
            }
            seen.add(normalised.url);
            attachments.push(normalised);
        });
        return attachments;
    };

    const createOptionInput = (question, option, type) => {
        const item = document.createElement("li");
        item.className = "course-player__option";

        const label = document.createElement("label");
        label.className = "course-player__option-label";

        const input = document.createElement("input");
        input.type = type === "multiple_choice" ? "checkbox" : "radio";
        input.name = `question-${question.id}`;
        input.value = option?.id != null ? option.id : "";
        input.dataset.questionId = question.id;
        label.appendChild(input);

        const text = document.createElement("span");
        text.className = "course-player__option-text";
        text.textContent = option?.text || "Option";
        label.appendChild(text);

        item.appendChild(label);
        return item;
    };

    document.addEventListener("DOMContentLoaded", () => {
        const root = document.querySelector("[data-course-player]");
        if (!root) {
            return;
        }

        const dataset = root.dataset || {};
        const endpoint = dataset.courseEndpoint || "";
        const testEndpointBase = (dataset.courseTestEndpoint || "/api/v1/courses/tests").replace(/\/$/, "");

        const elements = {
            topicList: root.querySelector("[data-course-player-topic-list]"),
            placeholder: root.querySelector("[data-course-player-placeholder]"),
            content: root.querySelector("[data-course-player-content]"),
            error: root.querySelector("[data-course-player-error]"),
            stats: root.querySelector("[data-course-player-stats]"),
        };

        const app = window.App || {};
        const apiRequest = app.apiRequest || (async (url, options = {}) => {
            const response = await fetch(url, {
                credentials: "include",
                ...options,
            });
            if (!response.ok) {
                const message = await response.text();
                const error = new Error(message || "Request failed");
                error.status = response.status;
                throw error;
            }
            const contentType = response.headers.get("content-type") || "";
            if (contentType.includes("application/json")) {
                return response.json();
            }
            return null;
        });

        const redirectToLogin = () => {
            if (app.auth && typeof app.auth.clearToken === "function") {
                app.auth.clearToken();
            }
            const redirectTarget = encodeURIComponent(window.location.pathname + window.location.search);
            window.location.href = `/login?redirect=${redirectTarget}`;
        };

        const state = {
            course: null,
            topicIndex: null,
            stepIndex: null,
        };

        const setPlaceholder = (message) => {
            if (!elements.placeholder) {
                return;
            }
            elements.placeholder.innerHTML = "";
            if (!message) {
                elements.placeholder.hidden = true;
                return;
            }
            const paragraph = document.createElement("p");
            paragraph.textContent = message;
            elements.placeholder.appendChild(paragraph);
            elements.placeholder.hidden = false;
        };

        const showError = (message) => {
            if (!elements.error) {
                return;
            }
            if (!message) {
                elements.error.textContent = "";
                elements.error.hidden = true;
                return;
            }
            elements.error.textContent = message;
            elements.error.hidden = false;
        };

        const updateStats = () => {
            if (!elements.stats || !state.course || !state.course.package) {
                return;
            }
            const topics = Array.isArray(state.course.package.topics)
                ? state.course.package.topics
                : [];
            const topicCount = topics.length;
            let lessonCount = 0;
            topics.forEach((topic) => {
                if (Array.isArray(topic.steps)) {
                    lessonCount += topic.steps.length;
                }
            });
            if (topicCount === 0) {
                elements.stats.hidden = true;
                elements.stats.textContent = "";
                return;
            }
            const lessonText = lessonCount > 0 ? ` • ${pluralize(lessonCount, "lesson", "lessons")}` : "";
            elements.stats.textContent = `${pluralize(topicCount, "topic", "topics")}${lessonText}`;
            elements.stats.hidden = false;
        };

        const updateActiveButtons = () => {
            if (!elements.topicList) {
                return;
            }
            const buttons = elements.topicList.querySelectorAll("[data-course-player-step]");
            buttons.forEach((button) => {
                const topicIndex = normalizeNumber(button.dataset.topicIndex);
                const stepIndex = normalizeNumber(button.dataset.stepIndex);
                const isActive =
                    topicIndex === state.topicIndex && stepIndex === state.stepIndex;
                button.classList.toggle("is-active", Boolean(isActive));
            });
        };

        const renderTopics = () => {
            if (!elements.topicList || !state.course || !state.course.package) {
                return;
            }
            elements.topicList.innerHTML = "";

            const topics = Array.isArray(state.course.package.topics)
                ? state.course.package.topics
                : [];

            if (topics.length === 0) {
                const emptyItem = document.createElement("li");
                emptyItem.className = "course-player__empty";
                emptyItem.textContent = "This course doesn't have any lessons yet.";
                elements.topicList.appendChild(emptyItem);
                setPlaceholder("Lessons will appear here once they're published.");
                return;
            }

            topics.forEach((topic, topicIndex) => {
                const topicItem = document.createElement("li");
                topicItem.className = "course-player__topic";

                const title = document.createElement("h3");
                title.className = "course-player__topic-title";
                title.textContent = topic?.title || `Topic ${topicIndex + 1}`;
                topicItem.appendChild(title);

                if (topic?.description) {
                    const description = document.createElement("p");
                    description.className = "course-player__topic-description";
                    description.textContent = topic.description;
                    topicItem.appendChild(description);
                }

                const stepsList = document.createElement("ul");
                stepsList.className = "course-player__steps";
                const steps = Array.isArray(topic?.steps) ? topic.steps : [];

                if (steps.length === 0) {
                    const placeholderStep = document.createElement("li");
                    placeholderStep.className = "course-player__step course-player__step--empty";
                    placeholderStep.textContent = "No lessons in this topic yet.";
                    stepsList.appendChild(placeholderStep);
                } else {
                    steps.forEach((step, stepIndex) => {
                        const stepItem = document.createElement("li");
                        stepItem.className = "course-player__step";

                        const button = document.createElement("button");
                        button.type = "button";
                        button.className = "course-player__step-button";
                        button.dataset.coursePlayerStep = "";
                        button.dataset.topicIndex = topicIndex;
                        button.dataset.stepIndex = stepIndex;
                        button.dataset.stepType = step?.type || "";

                        const order = document.createElement("span");
                        order.className = "course-player__step-order";
                        order.textContent = `${topicIndex + 1}.${stepIndex + 1}`;
                        button.appendChild(order);

                        const contentWrap = document.createElement("span");
                        contentWrap.className = "course-player__step-content";

                        const label = document.createElement("span");
                        label.className = "course-player__step-label";
                        if (step?.type === "test" && step?.test?.title) {
                            label.textContent = step.test.title;
                        } else if (step?.type === "video" && step?.video?.title) {
                            label.textContent = step.video.title;
                        } else {
                            label.textContent = `Lesson ${stepIndex + 1}`;
                        }
                        contentWrap.appendChild(label);

                        const metaText = (() => {
                            if (step?.type === "video") {
                                const parts = ["Video"];
                                const duration = formatDuration(step?.video?.duration_seconds);
                                if (duration) {
                                    parts.push(duration);
                                }
                                return parts.join(" • ");
                            }
                            if (step?.type === "test") {
                                const questionCount = Array.isArray(step?.test?.questions)
                                    ? step.test.questions.length
                                    : 0;
                                if (questionCount > 0) {
                                    return `Test • ${pluralize(questionCount, "question", "questions")}`;
                                }
                                return "Test";
                            }
                            return "";
                        })();

                        if (metaText) {
                            const meta = document.createElement("span");
                            meta.className = "course-player__step-meta";
                            meta.textContent = metaText;
                            contentWrap.appendChild(meta);
                        }

                        button.appendChild(contentWrap);
                        stepItem.appendChild(button);
                        stepsList.appendChild(stepItem);
                    });
                }

                topicItem.appendChild(stepsList);
                elements.topicList.appendChild(topicItem);
            });

            updateActiveButtons();
        };

        const renderVideo = (topic, step) => {
            const video = step?.video || {};
            const container = document.createElement("div");
            container.className = "course-player__lesson-content";

            if (video?.description) {
                const description = document.createElement("p");
                description.className = "course-player__lesson-description";
                description.textContent = video.description;
                container.appendChild(description);
            }

            const sectionsHTML =
                video?.sections_html ?? video?.sectionsHtml ?? video?.SectionsHTML;
            if (sectionsHTML) {
                const sections = document.createElement("div");
                sections.className = "course-player__lesson-sections";
                sections.innerHTML = sectionsHTML;
                container.appendChild(sections);
            }

            if (video?.file_url) {
                const wrapper = document.createElement("div");
                wrapper.className = "course-player__media";

                const videoEl = document.createElement("video");
                videoEl.className = "course-player__video";
                videoEl.controls = true;
                videoEl.src = video.file_url;
                if (video?.filename) {
                    videoEl.setAttribute("title", video.filename);
                }
                wrapper.appendChild(videoEl);
                container.appendChild(wrapper);
            }

            const attachments = normaliseVideoAttachments(video);
            if (attachments.length > 0) {
                const resources = document.createElement("section");
                resources.className = "course-player__resources";

                const heading = document.createElement("h4");
                heading.className = "course-player__resources-title";
                heading.textContent = "Downloadable files";
                resources.appendChild(heading);

                const list = document.createElement("ul");
                list.className = "course-player__resources-list";

                attachments.forEach((attachment) => {
                    const label = deriveAttachmentLabel(attachment);
                    const item = document.createElement("li");
                    item.className = "course-player__resource-item";

                    const link = document.createElement("a");
                    link.className = "course-player__resource-link";
                    link.href = attachment.url;
                    link.target = "_blank";
                    link.rel = "noopener noreferrer";

                    const fileName = deriveAttachmentFileName(attachment, label);
                    if (fileName) {
                        link.download = fileName;
                    }

                    const extension = getAttachmentExtension(attachment.url);
                    if (extension) {
                        const badge = document.createElement("span");
                        badge.className = "course-player__resource-extension";
                        badge.textContent = extension;
                        link.appendChild(badge);
                    }

                    const text = document.createElement("span");
                    text.textContent = label;
                    link.appendChild(text);

                    item.appendChild(link);
                    list.appendChild(item);
                });

                resources.appendChild(list);
                container.appendChild(resources);
            }

            const fragment = document.createDocumentFragment();
            fragment.appendChild(container);
            return fragment;
        };

        const renderTest = (topicIndex, stepIndex) => {
            const topic = state.course?.package?.topics?.[topicIndex];
            const step = topic?.steps?.[stepIndex];
            const test = step?.test;
            const container = document.createElement("div");
            container.className = "course-player__test";

            if (test?.description) {
                const intro = document.createElement("p");
                intro.className = "course-player__test-description";
                intro.textContent = test.description;
                container.appendChild(intro);
            }

            const form = document.createElement("form");
            form.className = "course-player__test-form";
            form.dataset.testId = test?.id != null ? test.id : "";
            form.dataset.topicIndex = topicIndex;
            form.dataset.stepIndex = stepIndex;

            const questionsList = document.createElement("ol");
            questionsList.className = "course-player__questions";
            const questions = Array.isArray(test?.questions) ? test.questions : [];

            questions.forEach((question, index) => {
                const item = document.createElement("li");
                item.className = "course-player__question";

                const title = document.createElement("h3");
                title.className = "course-player__question-title";
                title.textContent = `${index + 1}. ${question?.prompt || "Question"}`;
                item.appendChild(title);

                const type = question?.type || "text";
                if (type === "text") {
                    const textarea = document.createElement("textarea");
                    textarea.className = "course-player__answer course-player__answer--text";
                    textarea.rows = 4;
                    textarea.dataset.questionId = question.id;
                    textarea.dataset.questionType = "text";
                    textarea.placeholder = "Type your answer";
                    item.appendChild(textarea);
                } else {
                    const options = Array.isArray(question?.options) ? question.options : [];
                    const optionsList = document.createElement("ul");
                    optionsList.className = "course-player__options";
                    options.forEach((option) => {
                        optionsList.appendChild(createOptionInput(question, option, type));
                    });
                    item.appendChild(optionsList);
                }

                questionsList.appendChild(item);
            });

            form.appendChild(questionsList);

            const actions = document.createElement("div");
            actions.className = "course-player__actions";

            const submit = document.createElement("button");
            submit.type = "submit";
            submit.className = "button button--primary course-player__submit";
            submit.textContent = "Submit answers";
            submit.dataset.loadingLabel = "Submitting...";
            actions.appendChild(submit);
            form.appendChild(actions);

            const error = document.createElement("p");
            error.className = "course-player__test-error";
            error.hidden = true;
            form.appendChild(error);

            const result = document.createElement("div");
            result.className = "course-player__test-result";
            result.hidden = true;
            result.setAttribute("aria-live", "polite");
            form.appendChild(result);

            form.__resultContainer = result;
            form.__errorElement = error;
            form.__submitButton = submit;
            form.addEventListener("submit", async (event) => {
                event.preventDefault();
                if (!test?.id) {
                    return;
                }

                const answers = [];
                questions.forEach((question) => {
                    const submission = { question_id: question.id };
                    if (question?.type === "text") {
                        const field = form.querySelector(
                            `[data-question-id="${question.id}"][data-question-type="text"]`
                        );
                        submission.text = field ? field.value.trim() : "";
                    } else if (question?.type === "single_choice") {
                        const selected = form.querySelector(
                            `input[name="question-${question.id}"]:checked`
                        );
                        const selectedId = normalizeNumber(selected?.value);
                        submission.option_ids = selectedId != null ? [selectedId] : [];
                    } else if (question?.type === "multiple_choice") {
                        const selected = Array.from(
                            form.querySelectorAll(`input[name="question-${question.id}"]:checked`)
                        )
                            .map((input) => normalizeNumber(input.value))
                            .filter((value) => value != null);
                        submission.option_ids = selected;
                    } else {
                        submission.text = "";
                    }
                    answers.push(submission);
                });

                if (form.__errorElement) {
                    form.__errorElement.hidden = true;
                    form.__errorElement.textContent = "";
                }
                if (form.__resultContainer) {
                    form.__resultContainer.hidden = true;
                    form.__resultContainer.innerHTML = "";
                }

                const submitButton = form.__submitButton;
                const originalLabel = submitButton ? submitButton.textContent : "";
                if (submitButton) {
                    submitButton.disabled = true;
                    submitButton.textContent = submitButton.dataset.loadingLabel || "Submitting...";
                }

                try {
                    const payload = await apiRequest(`${testEndpointBase}/${test.id}/submit`, {
                        method: "POST",
                        headers: { "Content-Type": "application/json" },
                        body: JSON.stringify({ answers }),
                    });
                    const resultPayload = payload?.result;
                    if (!resultPayload) {
                        throw new Error("Unexpected server response");
                    }

                    if (form.__resultContainer) {
                        form.__resultContainer.innerHTML = "";
                        const summary = document.createElement("p");
                        summary.className = "course-player__test-summary";
                        summary.textContent = `Score: ${resultPayload.score} / ${resultPayload.max_score}`;
                        form.__resultContainer.appendChild(summary);

                        const answersList = document.createElement("ul");
                        answersList.className = "course-player__test-answers";
                        resultPayload.answers?.forEach((answer) => {
                            const question = questions.find((q) => q.id === answer.question_id) || {};
                            const item = document.createElement("li");
                            item.className = "course-player__test-answer";
                            item.classList.add(
                                answer.correct
                                    ? "course-player__test-answer--correct"
                                    : "course-player__test-answer--incorrect"
                            );

                            const titleEl = document.createElement("h4");
                            titleEl.className = "course-player__test-answer-title";
                            titleEl.textContent = question.prompt || "Question";
                            item.appendChild(titleEl);

                            const status = document.createElement("p");
                            status.className = "course-player__test-answer-status";
                            status.textContent = answer.correct ? "Correct" : "Incorrect";
                            item.appendChild(status);

                            if (answer.explanation) {
                                const explanation = document.createElement("p");
                                explanation.className = "course-player__test-answer-explanation";
                                explanation.textContent = answer.explanation;
                                item.appendChild(explanation);
                            }

                            answersList.appendChild(item);
                        });

                        if (answersList.childElementCount > 0) {
                            form.__resultContainer.appendChild(answersList);
                        }

                        form.__resultContainer.hidden = false;
                    }
                } catch (error) {
                    if (error && error.status === 401) {
                        redirectToLogin();
                        return;
                    }
                    if (form.__errorElement) {
                        form.__errorElement.textContent = error?.message || "Unable to submit answers.";
                        form.__errorElement.hidden = false;
                    }
                } finally {
                    if (submitButton) {
                        submitButton.disabled = false;
                        submitButton.textContent = originalLabel || "Submit answers";
                    }
                }
            });

            container.appendChild(form);
            return container;
        };

        const renderStep = (topicIndex, stepIndex) => {
            if (!elements.content || !state.course || !state.course.package) {
                return;
            }
            const topics = Array.isArray(state.course.package.topics)
                ? state.course.package.topics
                : [];
            const topic = topics[topicIndex];
            const step = topic?.steps?.[stepIndex];

            elements.content.innerHTML = "";

            if (!topic || !step) {
                setPlaceholder("This lesson could not be found.");
                return;
            }

            const header = document.createElement("header");
            header.className = "course-player__content-header";

            const breadcrumb = document.createElement("p");
            breadcrumb.className = "course-player__breadcrumb";
            breadcrumb.textContent = `${topic.title || `Topic ${topicIndex + 1}`} • Lesson ${stepIndex + 1}`;
            header.appendChild(breadcrumb);

            const title = document.createElement("h2");
            title.className = "course-player__lesson-title";
            if (step?.type === "test" && step?.test?.title) {
                title.textContent = step.test.title;
            } else if (step?.type === "video" && step?.video?.title) {
                title.textContent = step.video.title;
            } else {
                title.textContent = `Lesson ${stepIndex + 1}`;
            }
            header.appendChild(title);

            if (step?.type === "video" && step?.video?.duration_seconds) {
                const meta = document.createElement("p");
                meta.className = "course-player__lesson-meta";
                meta.textContent = `Duration: ${formatDuration(step.video.duration_seconds)}`;
                header.appendChild(meta);
            }

            elements.content.appendChild(header);

            if (step.type === "video") {
                elements.content.appendChild(renderVideo(topic, step));
            } else if (step.type === "test") {
                elements.content.appendChild(renderTest(topicIndex, stepIndex));
            } else {
                const message = document.createElement("p");
                message.className = "course-player__lesson-description";
                message.textContent = "This lesson type isn't supported yet.";
                elements.content.appendChild(message);
            }
        };

        const selectStep = (topicIndex, stepIndex) => {
            if (!state.course || !state.course.package) {
                return;
            }
            const topics = Array.isArray(state.course.package.topics)
                ? state.course.package.topics
                : [];
            if (!topics[topicIndex] || !topics[topicIndex].steps || !topics[topicIndex].steps[stepIndex]) {
                showError("Selected lesson is not available.");
                return;
            }

            showError("");
            setPlaceholder(null);
            state.topicIndex = topicIndex;
            state.stepIndex = stepIndex;
            updateActiveButtons();
            renderStep(topicIndex, stepIndex);
        };

        const setCourse = (course) => {
            if (!course || typeof course !== "object") {
                showError("Course details are not available right now.");
                return;
            }
            state.course = course;
            state.topicIndex = null;
            state.stepIndex = null;
            updateStats();
            renderTopics();

            const topics = Array.isArray(course.package?.topics) ? course.package.topics : [];
            let initialTopic = -1;
            let initialStep = -1;
            topics.forEach((topic, index) => {
                if (initialTopic >= 0) {
                    return;
                }
                const steps = Array.isArray(topic.steps) ? topic.steps : [];
                if (steps.length > 0) {
                    initialTopic = index;
                    initialStep = 0;
                }
            });

            if (initialTopic >= 0) {
                selectStep(initialTopic, initialStep);
            } else {
                setPlaceholder("Lessons will appear here once they're published.");
            }
        };

        const loadCourse = async () => {
            if (!endpoint) {
                showError("Course details are not available right now.");
                return;
            }
            setPlaceholder("Loading course details...");
            try {
                const payload = await apiRequest(endpoint, { method: "GET" });
                if (payload?.course) {
                    setCourse(payload.course);
                } else {
                    showError("Course details are not available right now.");
                }
            } catch (error) {
                if (error && error.status === 401) {
                    redirectToLogin();
                    return;
                }
                showError(error?.message || "Failed to load course details.");
            }
        };

        const initialData = parseInitialData(document.getElementById("course-player-data"));
        if (initialData) {
            setCourse(initialData);
        } else {
            loadCourse();
        }

        root.addEventListener("click", (event) => {
            const button = event.target.closest("[data-course-player-step]");
            if (!button || !root.contains(button)) {
                return;
            }
            const topicIndex = normalizeNumber(button.dataset.topicIndex);
            const stepIndex = normalizeNumber(button.dataset.stepIndex);
            if (topicIndex == null || stepIndex == null) {
                return;
            }
            event.preventDefault();
            selectStep(topicIndex, stepIndex);
        });
    });
})();

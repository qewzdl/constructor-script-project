
(() => {
    const utils = window.AdminUtils;

    const adminRoot = document.querySelector('[data-page="admin"]');
    const coursePickerCache = {
        promise: null,
        courses: null,
    };

    const normaliseValue = (value) =>
        utils && typeof utils.normaliseString === 'function'
            ? utils.normaliseString(value)
            : String(value || '');

    const parseCourseSelections = (raw) => {
        if (!raw) {
            return [];
        }
        const source = Array.isArray(raw)
            ? raw
            : String(raw).split(/[,;\n\r]/);
        return source
            .map((item) => normaliseValue(item).trim())
            .filter(Boolean);
    };

    const courseIdentifierFromPackage = (course) => {
        if (!course) {
            return '';
        }
        const slug = normaliseValue(course.slug || '').trim();
        if (slug) {
            return slug;
        }
        if (typeof course.id === 'number' && Number.isFinite(course.id)) {
            return String(course.id);
        }
        if (course.id) {
            return String(course.id);
        }
        return '';
    };

    const fetchCoursePackages = async () => {
        if (coursePickerCache.courses) {
            return coursePickerCache.courses;
        }
        if (coursePickerCache.promise) {
            return coursePickerCache.promise;
        }
        const endpoint =
            adminRoot && adminRoot.dataset
                ? adminRoot.dataset.endpointCoursesPackages
                : '';
        if (!endpoint) {
            alert('Course packages endpoint is not configured.');
            return [];
        }
        const dashboard = window.AdminDashboard;
        const apiClient =
            dashboard && dashboard.apiClient && typeof dashboard.apiClient.request === 'function'
                ? dashboard.apiClient
                : null;

        const fetchPromise = apiClient
            ? apiClient.request(endpoint)
            : fetch(endpoint, { credentials: 'include' }).then((response) => {
                  if (!response.ok) {
                      throw new Error(`Failed to load courses: ${response.status}`);
                  }
                  const contentType = response.headers.get('content-type') || '';
                  const isJson = contentType.includes('application/json');
                  return isJson ? response.json() : null;
              });

        coursePickerCache.promise = fetchPromise
            .then((data) => {
                const packages = Array.isArray(data?.packages) ? data.packages : [];
                coursePickerCache.courses = packages;
                return packages;
            })
            .catch((error) => {
                console.error(error);
                alert('Failed to load course packages.');
                return [];
            })
            .finally(() => {
                coursePickerCache.promise = null;
            });
        return coursePickerCache.promise;
    };

    const openAnchorPicker = (inputElement) => {
        if (!inputElement || !utils) {
            return;
        }

        // Get all sections on the page
        const sections = Array.from(document.querySelectorAll('[data-section-client]'));
        if (sections.length === 0) {
            alert('No sections found on this page');
            return;
        }

        // Create modal overlay
        const overlay = utils.createElement('div', {
            className: 'anchor-picker-overlay'
        });

        // Create modal content
        const modal = utils.createElement('div', {
            className: 'anchor-picker-modal'
        });

        // Header
        const header = utils.createElement('div', {
            className: 'anchor-picker-header'
        });
        const title = utils.createElement('h3', {
            className: 'anchor-picker-title',
            textContent: 'Select a section to link'
        });
        header.appendChild(title);

        // Section list
        const sectionList = utils.createElement('div', {
            className: 'anchor-picker-list'
        });

        sections.forEach((section, index) => {
            const sectionClientId = section.dataset.sectionClient;
            const sectionId = section.dataset.sectionId;
            const titleInput = section.querySelector('[data-field="section-title"]');
            const sectionTitle = titleInput ? titleInput.value.trim() : '';
            const sectionType = section.dataset.sectionType || 'unknown';
            
            const displayTitle = sectionTitle || `Section ${index + 1}`;
            const anchorId = sectionId || sectionClientId;

            const button = utils.createElement('button', {
                className: 'anchor-picker-item',
                type: 'button'
            });
            
            const buttonTitle = utils.createElement('div', {
                className: 'anchor-picker-item-title',
                textContent: displayTitle
            });
            
            const buttonMeta = utils.createElement('div', {
                className: 'anchor-picker-item-meta'
            });
            
            const typeSpan = utils.createElement('span', {
                textContent: `Type: ${sectionType}`
            });
            
            const anchorSpan = utils.createElement('code', {
                className: 'anchor-picker-item-code',
                textContent: `#section-${anchorId}`
            });
            
            buttonMeta.append(typeSpan, anchorSpan);
            button.append(buttonTitle, buttonMeta);

            button.addEventListener('click', () => {
                inputElement.value = `#section-${anchorId}`;
                inputElement.dispatchEvent(new Event('input', { bubbles: true }));
                inputElement.dispatchEvent(new Event('change', { bubbles: true }));
                document.body.removeChild(overlay);
            });

            sectionList.appendChild(button);
        });

        // Footer
        const footer = utils.createElement('div', {
            className: 'anchor-picker-footer'
        });
        const cancelButton = utils.createElement('button', {
            className: 'admin-builder__button',
            textContent: 'Cancel',
            type: 'button'
        });
        footer.appendChild(cancelButton);

        modal.append(header, sectionList, footer);
        overlay.appendChild(modal);

        // Close handlers
        const closeModal = () => {
            if (document.body.contains(overlay)) {
                document.body.removeChild(overlay);
            }
        };

        cancelButton.addEventListener('click', closeModal);
        overlay.addEventListener('click', (e) => {
            if (e.target === overlay) {
                closeModal();
            }
        });

        document.body.appendChild(overlay);
    };

    const openCoursePicker = async (inputElement) => {
        if (!inputElement || !utils) {
            return;
        }

        const courses = await fetchCoursePackages();
        if (!courses || courses.length === 0) {
            alert('No course packages found.');
            return;
        }

        const selected = new Set(
            parseCourseSelections(inputElement.value).map((item) =>
                item.toLowerCase()
            )
        );

        const overlay = utils.createElement('div', {
            className: 'anchor-picker-overlay',
        });

        const modal = utils.createElement('div', {
            className: 'anchor-picker-modal',
        });

        const header = utils.createElement('div', {
            className: 'anchor-picker-header',
        });
        const title = utils.createElement('h3', {
            className: 'anchor-picker-title',
            textContent: 'Select courses to display',
        });
        header.appendChild(title);

        const list = utils.createElement('div', {
            className: 'anchor-picker-list',
        });

        courses.forEach((course, index) => {
            const value = courseIdentifierFromPackage(course);
            if (!value) {
                return;
            }
            const lowerValue = value.toLowerCase();
            const item = utils.createElement('label', {
                className: 'anchor-picker-item anchor-picker-item--selectable',
            });
            const checkbox = utils.createElement('input', {
                className: 'anchor-picker-checkbox',
                type: 'checkbox',
            });
            checkbox.checked = selected.has(lowerValue);
            checkbox.dataset.value = value;

            const itemBody = utils.createElement('div');
            const itemTitle = utils.createElement('div', {
                className: 'anchor-picker-item-title',
                textContent:
                    course?.title ||
                    course?.name ||
                    course?.slug ||
                    `Course ${index + 1}`,
            });
            itemBody.append(itemTitle);

            const meta = utils.createElement('div', {
                className: 'anchor-picker-item-meta',
            });
            if (course?.slug) {
                meta.append(
                    utils.createElement('code', {
                        className: 'anchor-picker-item-code',
                        textContent: course.slug,
                    })
                );
            }
            if (course?.id) {
                meta.append(
                    utils.createElement('span', {
                        textContent: `ID: ${course.id}`,
                    })
                );
            }
            if (meta.childElementCount) {
                itemBody.append(meta);
            }

            item.append(checkbox, itemBody);
            list.append(item);
        });

        if (!list.childElementCount) {
            alert('No course packages available for selection.');
            return;
        }

        const footer = utils.createElement('div', {
            className: 'anchor-picker-footer',
        });
        const cancelButton = utils.createElement('button', {
            className: 'admin-builder__button',
            textContent: 'Cancel',
            type: 'button',
        });
        const applyButton = utils.createElement('button', {
            className: 'admin-builder__button admin-builder__button--primary',
            textContent: 'Apply',
            type: 'button',
        });
        footer.append(cancelButton, applyButton);

        modal.append(header, list, footer);
        overlay.append(modal);

        const closeModal = () => {
            if (document.body.contains(overlay)) {
                document.body.removeChild(overlay);
            }
        };

        cancelButton.addEventListener('click', closeModal);
        overlay.addEventListener('click', (event) => {
            if (event.target === overlay) {
                closeModal();
            }
        });

        applyButton.addEventListener('click', () => {
            const selectedValues = Array.from(
                list.querySelectorAll('.anchor-picker-checkbox:checked')
            )
                .map((input) => input.dataset.value || '')
                .map((value) => value.trim())
                .filter(Boolean);
            inputElement.value = selectedValues.join(', ');
            inputElement.dispatchEvent(new Event('input', { bubbles: true }));
            inputElement.dispatchEvent(new Event('change', { bubbles: true }));
            closeModal();
        });

        document.body.append(overlay);
    };

    const createEvents = ({
        listElement,
        onSectionRemove,
        onSectionMove,
        onElementRemove,
        onElementMove,
        onElementAdd,
        onGroupImageAdd,
        onGroupImageRemove,
        onGroupFileAdd,
        onGroupFileRemove,
        onSectionFieldChange,
        onElementFieldChange,
    }) => {
        if (!listElement) {
            return { destroy: () => {} };
        }

        const handleClick = (event) => {
            const target = event.target;
            if (!(target instanceof HTMLElement)) {
                return;
            }
            const sectionNode = target.closest('[data-section-client]');
            if (!sectionNode) {
                return;
            }
            const sectionClientId = sectionNode.dataset.sectionClient;
            if (!sectionClientId) {
                return;
            }

            if (target.matches('[data-action="section-move"]')) {
                event.preventDefault();
                const direction = target.dataset.direction || '';
                onSectionMove?.(sectionClientId, direction);
                return;
            }

            if (target.matches('[data-action="section-remove"]')) {
                event.preventDefault();
                onSectionRemove?.(sectionClientId);
                return;
            }

            if (target.matches('[data-action="element-remove"]')) {
                event.preventDefault();
                const elementNode = target.closest('[data-element-client]');
                if (!elementNode) {
                    return;
                }
                onElementRemove?.(sectionClientId, elementNode.dataset.elementClient);
                return;
            }

            if (target.matches('[data-action="element-move"]')) {
                event.preventDefault();
                const elementNode = target.closest('[data-element-client]');
                if (!elementNode) {
                    return;
                }
                const direction = target.dataset.direction || '';
                onElementMove?.(sectionClientId, elementNode.dataset.elementClient, direction);
                return;
            }

            if (target.matches('[data-action="element-add"]')) {
                event.preventDefault();
                const type = target.dataset.elementType || 'paragraph';
                onElementAdd?.(sectionClientId, type);
                return;
            }

            if (target.matches('[data-action="group-image-add"]')) {
                event.preventDefault();
                const elementNode = target.closest('[data-element-client]');
                if (!elementNode) {
                    return;
                }
                onGroupImageAdd?.(sectionClientId, elementNode.dataset.elementClient);
                return;
            }

            if (target.matches('[data-action="group-image-remove"]')) {
                event.preventDefault();
                const elementNode = target.closest('[data-element-client]');
                const imageNode = target.closest('[data-group-image-client]');
                if (!elementNode || !imageNode) {
                    return;
                }
                onGroupImageRemove?.(
                    sectionClientId,
                    elementNode.dataset.elementClient,
                    imageNode.dataset.groupImageClient
                );
                return;
            }

            if (target.matches('[data-action="group-file-add"]')) {
                event.preventDefault();
                const elementNode = target.closest('[data-element-client]');
                if (!elementNode) {
                    return;
                }
                onGroupFileAdd?.(sectionClientId, elementNode.dataset.elementClient);
                return;
            }

            if (target.matches('[data-action="group-file-remove"]')) {
                event.preventDefault();
                const elementNode = target.closest('[data-element-client]');
                const fileNode = target.closest('[data-group-file-client]');
                if (!elementNode || !fileNode) {
                    return;
                }
                onGroupFileRemove?.(
                    sectionClientId,
                    elementNode.dataset.elementClient,
                    fileNode.dataset.groupFileClient
                );
                return;
            }

            if (target.matches('[data-action="open-anchor-picker"]')) {
                event.preventDefault();
                const targetInputId = target.dataset.anchorTarget;
                if (!targetInputId || !targetInputId.startsWith('#')) {
                    return;
                }
                const inputElement = document.querySelector(targetInputId);
                if (!inputElement) {
                    return;
                }
                openAnchorPicker(inputElement);
            }

            if (target.matches('[data-action="open-course-picker"]')) {
                event.preventDefault();
                const targetInputId = target.dataset.courseTarget;
                if (!targetInputId || !targetInputId.startsWith('#')) {
                    return;
                }
                const inputElement = document.querySelector(targetInputId);
                if (!inputElement) {
                    return;
                }
                openCoursePicker(inputElement);
                return;
            }
        };

        const handleInput = (event) => {
            const target = event.target;
            if (!(target instanceof HTMLElement)) {
                return;
            }
            const sectionNode = target.closest('[data-section-client]');
            if (!sectionNode) {
                return;
            }
            const sectionClientId = sectionNode.dataset.sectionClient;
            if (!sectionClientId) {
                return;
            }

            const field = target.dataset.field;
            if (!field) {
                return;
            }

            let value = target.type === 'checkbox' ? target.checked : target.value;
            if (
                field === 'section-padding-vertical' ||
                field === 'section-margin-vertical'
            ) {
                const options = target.dataset.options
                    ? target.dataset.options
                          .split(',')
                          .map((option) => Number.parseInt(option.trim(), 10))
                          .filter((option) => Number.isFinite(option))
                    : [];
                const rawIndex = Number.parseInt(String(value), 10);
                const maxIndex = options.length - 1;
                const clampedIndex = Number.isFinite(rawIndex)
                    ? Math.min(Math.max(rawIndex, 0), maxIndex >= 0 ? maxIndex : 0)
                    : 0;
                if (Number.isFinite(rawIndex) && clampedIndex !== rawIndex) {
                    target.value = String(clampedIndex);
                }
                const actualValue = options[clampedIndex] ?? 0;
                value = actualValue;
                if (options.length) {
                    target.setAttribute('aria-valuenow', String(actualValue));
                    target.setAttribute('aria-valuetext', `${actualValue} pixels`);
                }
                const displayRole =
                    field === 'section-padding-vertical'
                        ? 'section-padding-value'
                        : 'section-margin-value';
                const displayNode = target.parentElement?.querySelector(
                    `[data-role="${displayRole}"]`
                );
                if (displayNode) {
                    displayNode.textContent = `${actualValue}px`;
                }
            }
            const elementNode = target.closest('[data-element-client]');
            if (elementNode) {
                const elementClientId = elementNode.dataset.elementClient;
                const imageNode = target.closest('[data-group-image-client]');
                const fileNode = target.closest('[data-group-file-client]');
                const nestedClientId = imageNode
                    ? imageNode.dataset.groupImageClient
                    : fileNode?.dataset.groupFileClient;
                onElementFieldChange?.(
                    sectionClientId,
                    elementClientId,
                    field,
                    value,
                    nestedClientId
                );
                return;
            }

            onSectionFieldChange?.(sectionClientId, field, value);
        };

        listElement.addEventListener('click', handleClick);
        listElement.addEventListener('input', handleInput);
        listElement.addEventListener('change', handleInput);

        const destroy = () => {
            listElement.removeEventListener('click', handleClick);
            listElement.removeEventListener('input', handleInput);
            listElement.removeEventListener('change', handleInput);
        };

        return { destroy };
    };

    window.AdminSectionEvents = {
        bind: createEvents,
    };
})();

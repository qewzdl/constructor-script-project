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

    const randomId = () => {
        if (window.crypto && typeof window.crypto.randomUUID === "function") {
            return window.crypto.randomUUID();
        }
        return `id-${Math.random().toString(36).slice(2, 11)}`;
    };

    const normaliseString = (value) => {
        if (typeof value === "string") {
            return value;
        }
        if (value === null || value === undefined) {
            return "";
        }
        if (typeof value === "number" || typeof value === "boolean") {
            return String(value);
        }
        return "";
    };

    const createSectionBuilder = (form) => {
        if (!form) {
            return null;
        }

        const builderRoot = form.querySelector("[data-section-builder]");
        if (!builderRoot) {
            return null;
        }

        const sectionList = builderRoot.querySelector("[data-section-list]");
        const emptyState = builderRoot.querySelector("[data-section-empty]");
        const addSectionButton = builderRoot.querySelector('[data-action="section-add"]');

        if (!sectionList || !addSectionButton) {
            return null;
        }

        const listeners = new Set();
        let sections = [];

        const ensureArray = (value) => (Array.isArray(value) ? value : []);

        const createImageState = (image = {}) => ({
            clientId: randomId(),
            url: normaliseString(image.url ?? image.URL ?? ""),
            alt: normaliseString(image.alt ?? image.Alt ?? ""),
            caption: normaliseString(image.caption ?? image.Caption ?? ""),
        });

        const createElementState = (element = {}) => {
            const type = normaliseString(element.type ?? element.Type ?? "").toLowerCase() || "paragraph";
            const id = normaliseString(element.id ?? element.ID ?? "");
            const rawContent = element.content ?? element.Content ?? {};

            if (type === "paragraph") {
                return {
                    clientId: randomId(),
                    id,
                    type,
                    content: {
                        text: normaliseString(rawContent.text ?? rawContent.Text ?? ""),
                    },
                };
            }

            if (type === "image") {
                return {
                    clientId: randomId(),
                    id,
                    type,
                    content: {
                        url: normaliseString(rawContent.url ?? rawContent.URL ?? ""),
                        alt: normaliseString(rawContent.alt ?? rawContent.Alt ?? ""),
                        caption: normaliseString(rawContent.caption ?? rawContent.Caption ?? ""),
                    },
                };
            }

            if (type === "image_group") {
                const images = ensureArray(rawContent.images ?? rawContent.Images).map(createImageState);
                return {
                    clientId: randomId(),
                    id,
                    type,
                    content: {
                        layout: normaliseString(rawContent.layout ?? rawContent.Layout ?? "grid"),
                        images,
                    },
                };
            }

            return {
                clientId: randomId(),
                id,
                type,
                content: rawContent || {},
                unsupported: true,
            };
        };

        const createSectionState = (section = {}) => {
            const elementsSource = ensureArray(section.elements ?? section.Elements);
            return {
                clientId: randomId(),
                id: normaliseString(section.id ?? section.ID ?? ""),
                title: normaliseString(section.title ?? section.Title ?? ""),
                image: normaliseString(section.image ?? section.Image ?? ""),
                elements: elementsSource.map(createElementState),
            };
        };

        const elementLabel = (type) => {
            switch (type) {
                case "paragraph":
                    return "Paragraph";
                case "image":
                    return "Image";
                case "image_group":
                    return "Image group";
                default:
                    return type || "Block";
            }
        };

        const emitChange = () => {
            const snapshot = getSections();
            listeners.forEach((listener) => {
                try {
                    listener(snapshot);
                } catch (error) {
                    console.error("Section builder listener failed", error);
                }
            });
        };

        const focusField = (selector) => {
            if (!selector) {
                return;
            }
            requestAnimationFrame(() => {
                const field = sectionList.querySelector(selector);
                if (field && typeof field.focus === "function") {
                    field.focus();
                }
            });
        };

        const render = () => {
            sectionList.innerHTML = "";

            if (!sections.length) {
                if (emptyState) {
                    emptyState.hidden = false;
                }
                return;
            }

            if (emptyState) {
                emptyState.hidden = true;
            }

            sections.forEach((section, index) => {
                const sectionItem = createElement("li", { className: "admin-builder__section" });
                sectionItem.dataset.sectionClient = section.clientId;
                sectionItem.dataset.sectionIndex = String(index);

                const sectionHeader = createElement("div", { className: "admin-builder__section-header" });
                const sectionTitle = createElement("h3", {
                    className: "admin-builder__section-title",
                    textContent: `Section ${index + 1}`,
                });
                const removeButton = createElement("button", {
                    className: "admin-builder__remove",
                    textContent: "Remove section",
                });
                removeButton.type = "button";
                removeButton.dataset.action = "section-remove";
                sectionHeader.append(sectionTitle, removeButton);
                sectionItem.append(sectionHeader);

                const titleField = createElement("label", { className: "admin-builder__field" });
                titleField.append(
                    createElement("span", {
                        className: "admin-builder__label",
                        textContent: "Section title",
                    })
                );
                const titleInput = createElement("input", { className: "admin-builder__input" });
                titleInput.type = "text";
                titleInput.placeholder = "e.g. Getting started";
                titleInput.value = section.title;
                titleInput.dataset.field = "section-title";
                titleField.append(titleInput);
                sectionItem.append(titleField);

                const imageField = createElement("label", { className: "admin-builder__field" });
                imageField.append(
                    createElement("span", {
                        className: "admin-builder__label",
                        textContent: "Optional header image URL",
                    })
                );
                const imageInput = createElement("input", { className: "admin-builder__input" });
                imageInput.type = "url";
                imageInput.placeholder = "https://example.com/cover.jpg";
                imageInput.value = section.image;
                imageInput.dataset.field = "section-image";
                imageField.append(imageInput);
                sectionItem.append(imageField);

                const elementsContainer = createElement("div", { className: "admin-builder__section-elements" });

                if (!section.elements.length) {
                    elementsContainer.append(
                        createElement("p", {
                            className: "admin-builder__element-empty",
                            textContent: "No content blocks yet. Add one below.",
                        })
                    );
                } else {
                    section.elements.forEach((element, elementIndex) => {
                        const elementNode = createElement("div", { className: "admin-builder__element" });
                        elementNode.dataset.elementClient = element.clientId;
                        elementNode.dataset.elementType = element.type;
                        elementNode.dataset.elementIndex = String(elementIndex);

                        const elementHeader = createElement("div", { className: "admin-builder__element-header" });
                        const elementTitle = createElement("span", {
                            className: "admin-builder__element-title",
                            textContent: `${elementLabel(element.type)} ${elementIndex + 1}`,
                        });
                        const elementActions = createElement("div", { className: "admin-builder__element-actions" });
                        const removeElementButton = createElement("button", {
                            className: "admin-builder__element-remove",
                            textContent: "Remove",
                        });
                        removeElementButton.type = "button";
                        removeElementButton.dataset.action = "element-remove";
                        elementActions.append(removeElementButton);
                        elementHeader.append(elementTitle, elementActions);
                        elementNode.append(elementHeader);

                        if (element.type === "paragraph") {
                            const paragraphField = createElement("label", { className: "admin-builder__field" });
                            paragraphField.append(
                                createElement("span", {
                                    className: "admin-builder__label",
                                    textContent: "Paragraph text",
                                })
                            );
                            const paragraphTextarea = createElement("textarea", { className: "admin-builder__textarea" });
                            paragraphTextarea.placeholder = "Write the narrative for this part of the section…";
                            paragraphTextarea.value = element.content?.text || "";
                            paragraphTextarea.dataset.field = "paragraph-text";
                            paragraphField.append(paragraphTextarea);
                            elementNode.append(paragraphField);
                        } else if (element.type === "image") {
                            const urlField = createElement("label", { className: "admin-builder__field" });
                            urlField.append(
                                createElement("span", {
                                    className: "admin-builder__label",
                                    textContent: "Image URL",
                                })
                            );
                            const urlInput = createElement("input", { className: "admin-builder__input" });
                            urlInput.type = "url";
                            urlInput.placeholder = "https://example.com/visual.png";
                            urlInput.value = element.content?.url || "";
                            urlInput.dataset.field = "image-url";
                            urlField.append(urlInput);
                            elementNode.append(urlField);

                            const altField = createElement("label", { className: "admin-builder__field" });
                            altField.append(
                                createElement("span", {
                                    className: "admin-builder__label",
                                    textContent: "Accessible alt text",
                                })
                            );
                            const altInput = createElement("input", { className: "admin-builder__input" });
                            altInput.type = "text";
                            altInput.placeholder = "Describe the visual for screen readers";
                            altInput.value = element.content?.alt || "";
                            altInput.dataset.field = "image-alt";
                            altField.append(altInput);
                            elementNode.append(altField);

                            const captionField = createElement("label", { className: "admin-builder__field" });
                            captionField.append(
                                createElement("span", {
                                    className: "admin-builder__label",
                                    textContent: "Optional caption",
                                })
                            );
                            const captionInput = createElement("input", { className: "admin-builder__input" });
                            captionInput.type = "text";
                            captionInput.placeholder = "Add context that appears under the image";
                            captionInput.value = element.content?.caption || "";
                            captionInput.dataset.field = "image-caption";
                            captionField.append(captionInput);
                            elementNode.append(captionField);
                        } else if (element.type === "image_group") {
                            const groupContainer = createElement("div", { className: "admin-builder__group" });

                            const layoutField = createElement("label", { className: "admin-builder__field" });
                            layoutField.append(
                                createElement("span", {
                                    className: "admin-builder__label",
                                    textContent: "Layout preset",
                                })
                            );
                            const layoutInput = createElement("input", { className: "admin-builder__input" });
                            layoutInput.type = "text";
                            layoutInput.placeholder = "e.g. grid, carousel, mosaic";
                            layoutInput.value = element.content?.layout || "";
                            layoutInput.dataset.field = "image-group-layout";
                            layoutField.append(layoutInput);
                            groupContainer.append(layoutField);

                            const groupList = createElement("div", { className: "admin-builder__group-list" });

                            if (!element.content?.images?.length) {
                                groupList.append(
                                    createElement("p", {
                                        className: "admin-builder__element-empty",
                                        textContent: "No images in this group yet.",
                                    })
                                );
                            } else {
                                element.content.images.forEach((image) => {
                                    const groupItem = createElement("div", { className: "admin-builder__group-item" });
                                    groupItem.dataset.groupImageClient = image.clientId;

                                    const groupUrlField = createElement("label", { className: "admin-builder__field" });
                                    groupUrlField.append(
                                        createElement("span", {
                                            className: "admin-builder__label",
                                            textContent: "Image URL",
                                        })
                                    );
                                    const groupUrlInput = createElement("input", { className: "admin-builder__input" });
                                    groupUrlInput.type = "url";
                                    groupUrlInput.placeholder = "https://example.com/gallery-image.jpg";
                                    groupUrlInput.value = image.url || "";
                                    groupUrlInput.dataset.field = "group-image-url";
                                    groupUrlField.append(groupUrlInput);
                                    groupItem.append(groupUrlField);

                                    const groupAltField = createElement("label", { className: "admin-builder__field" });
                                    groupAltField.append(
                                        createElement("span", {
                                            className: "admin-builder__label",
                                            textContent: "Alt text",
                                        })
                                    );
                                    const groupAltInput = createElement("input", { className: "admin-builder__input" });
                                    groupAltInput.type = "text";
                                    groupAltInput.placeholder = "Describe this image";
                                    groupAltInput.value = image.alt || "";
                                    groupAltInput.dataset.field = "group-image-alt";
                                    groupAltField.append(groupAltInput);
                                    groupItem.append(groupAltField);

                                    const groupCaptionField = createElement("label", { className: "admin-builder__field" });
                                    groupCaptionField.append(
                                        createElement("span", {
                                            className: "admin-builder__label",
                                            textContent: "Caption",
                                        })
                                    );
                                    const groupCaptionInput = createElement("input", { className: "admin-builder__input" });
                                    groupCaptionInput.type = "text";
                                    groupCaptionInput.placeholder = "Optional caption";
                                    groupCaptionInput.value = image.caption || "";
                                    groupCaptionInput.dataset.field = "group-image-caption";
                                    groupCaptionField.append(groupCaptionInput);
                                    groupItem.append(groupCaptionField);

                                    const groupActions = createElement("div", { className: "admin-builder__group-actions" });
                                    const removeImageButton = createElement("button", {
                                        className: "admin-builder__element-remove",
                                        textContent: "Remove image",
                                    });
                                    removeImageButton.type = "button";
                                    removeImageButton.dataset.action = "group-image-remove";
                                    groupActions.append(removeImageButton);
                                    groupItem.append(groupActions);

                                    groupList.append(groupItem);
                                });
                            }

                            groupContainer.append(groupList);

                            const addGroupImageButton = createElement("button", {
                                className: "admin-builder__button admin-builder__button--ghost",
                                textContent: "Add image to group",
                            });
                            addGroupImageButton.type = "button";
                            addGroupImageButton.dataset.action = "group-image-add";
                            groupContainer.append(addGroupImageButton);

                            elementNode.append(groupContainer);
                        } else if (element.unsupported) {
                            elementNode.append(
                                createElement("p", {
                                    className: "admin-builder__note",
                                    textContent:
                                        "This block type isn't supported in the visual builder yet, but it will be kept intact when you save.",
                                })
                            );
                        }

                        elementsContainer.append(elementNode);
                    });
                }

                sectionItem.append(elementsContainer);

                const sectionActions = createElement("div", { className: "admin-builder__section-actions" });

                const addParagraphButton = createElement("button", {
                    className: "admin-builder__button admin-builder__button--ghost",
                    textContent: "Add paragraph",
                });
                addParagraphButton.type = "button";
                addParagraphButton.dataset.action = "element-add";
                addParagraphButton.dataset.elementType = "paragraph";
                sectionActions.append(addParagraphButton);

                const addImageButton = createElement("button", {
                    className: "admin-builder__button admin-builder__button--ghost",
                    textContent: "Add image",
                });
                addImageButton.type = "button";
                addImageButton.dataset.action = "element-add";
                addImageButton.dataset.elementType = "image";
                sectionActions.append(addImageButton);

                const addImageGroupButton = createElement("button", {
                    className: "admin-builder__button admin-builder__button--ghost",
                    textContent: "Add image group",
                });
                addImageGroupButton.type = "button";
                addImageGroupButton.dataset.action = "element-add";
                addImageGroupButton.dataset.elementType = "image_group";
                sectionActions.append(addImageGroupButton);

                sectionItem.append(sectionActions);

                sectionList.append(sectionItem);
            });
        };

        const findSection = (clientId) => sections.find((section) => section.clientId === clientId);
        const findElement = (section, clientId) =>
            section?.elements?.find((element) => element.clientId === clientId);

        const addSection = () => {
            const section = createSectionState({});
            sections.push(section);
            render();
            emitChange();
            focusField(`[data-section-client="${section.clientId}"] [data-field="section-title"]`);
        };

        const removeSection = (clientId) => {
            sections = sections.filter((section) => section.clientId !== clientId);
            render();
            emitChange();
        };

        const addElementToSection = (sectionClientId, type) => {
            const section = findSection(sectionClientId);
            if (!section) {
                return;
            }
            const element = createElementState({ type });
            if (type === "image_group" && (!element.content || !element.content.images.length)) {
                element.content.images = [createImageState({})];
            }
            section.elements.push(element);
            render();
            emitChange();
            focusField(
                type === "paragraph"
                    ? `[data-section-client="${sectionClientId}"] [data-element-client="${element.clientId}"] textarea`
                    : `[data-section-client="${sectionClientId}"] [data-element-client="${element.clientId}"] [data-field]`
            );
        };

        const removeElementFromSection = (sectionClientId, elementClientId) => {
            const section = findSection(sectionClientId);
            if (!section) {
                return;
            }
            section.elements = section.elements.filter((element) => element.clientId !== elementClientId);
            render();
            emitChange();
        };

        const addGroupImage = (sectionClientId, elementClientId) => {
            const section = findSection(sectionClientId);
            if (!section) {
                return;
            }
            const element = findElement(section, elementClientId);
            if (!element || element.type !== "image_group") {
                return;
            }
            const image = createImageState({});
            element.content.images.push(image);
            render();
            emitChange();
            focusField(
                `[data-section-client="${sectionClientId}"] [data-element-client="${elementClientId}"] [data-group-image-client="${image.clientId}"] [data-field="group-image-url"]`
            );
        };

        const removeGroupImage = (sectionClientId, elementClientId, imageClientId) => {
            const section = findSection(sectionClientId);
            if (!section) {
                return;
            }
            const element = findElement(section, elementClientId);
            if (!element || element.type !== "image_group") {
                return;
            }
            element.content.images = element.content.images.filter((image) => image.clientId !== imageClientId);
            render();
            emitChange();
        };

        const updateSectionField = (sectionClientId, field, value) => {
            const section = findSection(sectionClientId);
            if (!section) {
                return;
            }
            if (field === "section-title") {
                section.title = value;
            } else if (field === "section-image") {
                section.image = value;
            }
        };

        const updateElementField = (sectionClientId, elementClientId, field, value, imageClientId) => {
            const section = findSection(sectionClientId);
            if (!section) {
                return;
            }
            const element = findElement(section, elementClientId);
            if (!element) {
                return;
            }
            switch (field) {
                case "paragraph-text":
                    element.content.text = value;
                    break;
                case "image-url":
                    element.content.url = value;
                    break;
                case "image-alt":
                    element.content.alt = value;
                    break;
                case "image-caption":
                    element.content.caption = value;
                    break;
                case "image-group-layout":
                    element.content.layout = value;
                    break;
                case "group-image-url":
                case "group-image-alt":
                case "group-image-caption":
                    if (!element.content.images) {
                        element.content.images = [];
                    }
                    {
                        const image = element.content.images.find((img) => img.clientId === imageClientId);
                        if (!image) {
                            return;
                        }
                        if (field === "group-image-url") {
                            image.url = value;
                        }
                        if (field === "group-image-alt") {
                            image.alt = value;
                        }
                        if (field === "group-image-caption") {
                            image.caption = value;
                        }
                    }
                    break;
                default:
                    break;
            }
        };

        const elementHasContent = (element) => {
            if (!element) {
                return false;
            }
            if (element.type === "paragraph") {
                return Boolean(element.content?.text?.trim());
            }
            if (element.type === "image") {
                return Boolean(element.content?.url?.trim());
            }
            if (element.type === "image_group") {
                return (
                    Array.isArray(element.content?.images) &&
                    element.content.images.some((image) => image.url && image.url.trim())
                );
            }
            return true;
        };

        const sanitiseElement = (element, index) => {
            if (!elementHasContent(element)) {
                return null;
            }
            if (element.type === "paragraph") {
                return {
                    id: element.id || "",
                    type: "paragraph",
                    order: index + 1,
                    content: {
                        text: element.content.text.trim(),
                    },
                };
            }
            if (element.type === "image") {
                const payload = {
                    url: element.content.url.trim(),
                };
                if (element.content.alt && element.content.alt.trim()) {
                    payload.alt = element.content.alt.trim();
                }
                if (element.content.caption && element.content.caption.trim()) {
                    payload.caption = element.content.caption.trim();
                }
                return {
                    id: element.id || "",
                    type: "image",
                    order: index + 1,
                    content: payload,
                };
            }
            if (element.type === "image_group") {
                const images = (element.content.images || [])
                    .map((image) => {
                        const url = (image.url || "").trim();
                        if (!url) {
                            return null;
                        }
                        const payload = { url };
                        if (image.alt && image.alt.trim()) {
                            payload.alt = image.alt.trim();
                        }
                        if (image.caption && image.caption.trim()) {
                            payload.caption = image.caption.trim();
                        }
                        return payload;
                    })
                    .filter(Boolean);

                if (!images.length) {
                    return null;
                }

                const payload = { images };
                if (element.content.layout && element.content.layout.trim()) {
                    payload.layout = element.content.layout.trim();
                }

                return {
                    id: element.id || "",
                    type: "image_group",
                    order: index + 1,
                    content: payload,
                };
            }
            return {
                id: element.id || "",
                type: element.type,
                order: index + 1,
                content: element.content,
            };
        };

        const getSections = () =>
            sections
                .map((section, index) => {
                    const elements = section.elements
                        .map((element, elementIndex) => sanitiseElement(element, elementIndex))
                        .filter(Boolean);

                    const image = section.image.trim();
                    const title = section.title.trim();
                    const hasContent = Boolean(title || image || elements.length > 0);

                    if (!hasContent) {
                        return null;
                    }

                    const payload = {
                        id: section.id || "",
                        title,
                        order: index + 1,
                        elements,
                    };

                    if (image) {
                        payload.image = image;
                    }

                    return payload;
                })
                .filter(Boolean);

        addSectionButton.addEventListener("click", () => {
            addSection();
        });

        sectionList.addEventListener("click", (event) => {
            const target = event.target;
            if (!(target instanceof HTMLElement)) {
                return;
            }
            const sectionNode = target.closest("[data-section-client]");
            if (!sectionNode) {
                return;
            }
            const sectionClientId = sectionNode.dataset.sectionClient;
            if (!sectionClientId) {
                return;
            }

            if (target.matches('[data-action="section-remove"]')) {
                event.preventDefault();
                removeSection(sectionClientId);
                return;
            }

            if (target.matches('[data-action="element-remove"]')) {
                event.preventDefault();
                const elementNode = target.closest("[data-element-client]");
                if (!elementNode) {
                    return;
                }
                removeElementFromSection(sectionClientId, elementNode.dataset.elementClient);
                return;
            }

            if (target.matches('[data-action="element-add"]')) {
                event.preventDefault();
                const type = target.dataset.elementType || "paragraph";
                addElementToSection(sectionClientId, type);
                return;
            }

            if (target.matches('[data-action="group-image-add"]')) {
                event.preventDefault();
                const elementNode = target.closest("[data-element-client]");
                if (!elementNode) {
                    return;
                }
                addGroupImage(sectionClientId, elementNode.dataset.elementClient);
                return;
            }

            if (target.matches('[data-action="group-image-remove"]')) {
                event.preventDefault();
                const elementNode = target.closest("[data-element-client]");
                const imageNode = target.closest("[data-group-image-client]");
                if (!elementNode || !imageNode) {
                    return;
                }
                removeGroupImage(sectionClientId, elementNode.dataset.elementClient, imageNode.dataset.groupImageClient);
            }
        });

        sectionList.addEventListener("input", (event) => {
            const target = event.target;
            if (!(target instanceof HTMLElement)) {
                return;
            }
            const sectionNode = target.closest("[data-section-client]");
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

            const elementNode = target.closest("[data-element-client]");
            if (elementNode) {
                const elementClientId = elementNode.dataset.elementClient;
                const imageNode = target.closest("[data-group-image-client]");
                const imageClientId = imageNode ? imageNode.dataset.groupImageClient : undefined;
                updateElementField(sectionClientId, elementClientId, field, target.value, imageClientId);
            } else {
                updateSectionField(sectionClientId, field, target.value);
            }
            emitChange();
        });

        const setSections = (nextSections) => {
            sections = ensureArray(nextSections).map(createSectionState);
            render();
            emitChange();
        };

        const reset = () => {
            sections = [];
            render();
            emitChange();
        };

        const onChange = (listener) => {
            if (typeof listener !== "function") {
                return () => {};
            }
            listeners.add(listener);
            return () => listeners.delete(listener);
        };

        render();

        return {
            setSections,
            reset,
            getSections,
            onChange,
        };
    };

    const generateContentPreview = (sections) => {
        if (!Array.isArray(sections) || sections.length === 0) {
            return "";
        }
        const parts = [];
        sections.forEach((section) => {
            if (section.title) {
                parts.push(section.title);
            }
            if (Array.isArray(section.elements)) {
                section.elements.forEach((element) => {
                    if (element.type === "paragraph" && element.content?.text) {
                        parts.push(element.content.text);
                    }
                });
            }
        });
        return parts.join("\n\n");
    };

    document.addEventListener("DOMContentLoaded", () => {
        const root = document.querySelector('[data-page="admin"]');
        if (!root) {
            return;
        }

        const app = window.App || {};
        const auth = app.auth;
        const fallbackApiRequest = async (url, options = {}) => {
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
                const message =
                    payload && typeof payload === "object" && payload.error
                        ? payload.error
                        : typeof payload === "string"
                        ? payload
                        : "Request failed";
                const error = new Error(message);
                error.status = response.status;
                error.payload = payload;
                throw error;
            }

            return payload;
        };

        const apiRequest =
            typeof app.apiRequest === "function" ? app.apiRequest : fallbackApiRequest;
        if (typeof app.apiRequest !== "function") {
            console.warn(
                "Admin dashboard is using fallback API client because App.apiRequest is unavailable."
            );
        }
        const setAlert = typeof app.setAlert === "function" ? app.setAlert : null;
        const toggleFormDisabled =
            typeof app.toggleFormDisabled === "function" ? app.toggleFormDisabled : null;

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
            siteSettings: root.dataset.endpointSiteSettings,
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
        const settingsForm = root.querySelector("#admin-settings-form");
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

        const postContentField = postForm?.querySelector('textarea[name="content"]');
        let postContentManuallyEdited = false;
        postContentField?.addEventListener("input", () => {
            if (!postContentField) {
                return;
            }
            postContentManuallyEdited = postContentField.value.trim().length > 0;
        });

        const sectionBuilder = createSectionBuilder(postForm);
        if (sectionBuilder) {
            sectionBuilder.onChange((sections) => {
                if (!postContentField || postContentManuallyEdited) {
                    return;
                }
                postContentField.value = generateContentPreview(sections);
            });
        }

        const state = {
            metrics: {},
            posts: [],
            pages: [],
            categories: [],
            comments: [],
            tags: [],
            defaultCategoryId: "",
            site: null,
        };

        const validateSections = (sections) => {
            if (!Array.isArray(sections)) {
                return "";
            }
            for (let index = 0; index < sections.length; index += 1) {
                const section = sections[index];
                if (!section) {
                    continue;
                }
                if (!section.title) {
                    return `Section ${index + 1} needs a title.`;
                }
                if (!Array.isArray(section.elements)) {
                    continue;
                }
                for (let elementIndex = 0; elementIndex < section.elements.length; elementIndex += 1) {
                    const element = section.elements[elementIndex];
                    if (!element) {
                        continue;
                    }
                    if (element.type === "paragraph" && !element.content?.text) {
                        return `Paragraph ${elementIndex + 1} in section "${section.title}" is empty.`;
                    }
                    if (element.type === "image" && !element.content?.url) {
                        return `Image ${elementIndex + 1} in section "${section.title}" is missing a URL.`;
                    }
                    if (element.type === "image_group") {
                        const images = Array.isArray(element.content?.images) ? element.content.images : [];
                        if (!images.length) {
                            return `The image group in section "${section.title}" needs at least one image.`;
                        }
                        const missing = images.findIndex((img) => !img?.url);
                        if (missing !== -1) {
                            return `Image ${missing + 1} in the group for section "${section.title}" is missing a URL.`;
                        }
                    }
                }
            }
            return "";
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
            if (postContentField) {
                const existingContent = post.content || "";
                postContentField.value = existingContent;
                postContentManuallyEdited = Boolean(existingContent.trim());
            }
            if (sectionBuilder) {
                const postSections = post.sections || post.Sections || [];
                sectionBuilder.setSections(postSections);
            }
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
            postContentManuallyEdited = false;
            if (sectionBuilder) {
                sectionBuilder.reset();
            }
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

        const populateSiteSettingsForm = (site) => {
            if (!settingsForm) {
                return;
            }

            const entries = [
                ["name", site?.name],
                ["description", site?.description],
                ["url", site?.url],
                ["favicon", site?.favicon],
                ["logo", site?.logo],
            ];

            entries.forEach(([key, value]) => {
                const field = settingsForm.querySelector(`[name="${key}"]`);
                if (!field) {
                    return;
                }
                field.value = value || "";
            });
        };

        const loadSiteSettings = async () => {
            if (!endpoints.siteSettings) {
                return;
            }
            try {
                const payload = await apiRequest(endpoints.siteSettings);
                state.site = payload?.site || null;
                populateSiteSettingsForm(state.site);
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

        const handleSiteSettingsSubmit = async (event) => {
            event.preventDefault();
            if (!settingsForm || !endpoints.siteSettings) {
                return;
            }

            const getValue = (name) => {
                const field = settingsForm.querySelector(`[name="${name}"]`);
                return field ? field.value.trim() : "";
            };

            const payload = {
                name: getValue("name"),
                description: getValue("description"),
                url: getValue("url"),
                favicon: getValue("favicon"),
                logo: getValue("logo"),
            };

            if (!payload.name) {
                showAlert("Please provide a site name.", "error");
                return;
            }

            if (!payload.url) {
                showAlert("Please provide the primary site URL.", "error");
                return;
            }

            disableForm(settingsForm, true);
            clearAlert();

            try {
                const response = await apiRequest(endpoints.siteSettings, {
                    method: "PUT",
                    body: JSON.stringify(payload),
                });
                state.site = response?.site || payload;
                populateSiteSettingsForm(state.site);
                showAlert("Site settings updated successfully.", "success");
            } catch (error) {
                handleRequestError(error);
            } finally {
                disableForm(settingsForm, false);
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
            const content = postContentField ? postContentField.value.trim() : "";
            const publishedField = postForm.querySelector('input[name="published"]');
            const payload = {
                title,
                description,
                content,
                published: Boolean(publishedField?.checked),
            };
            if (sectionBuilder) {
                const sections = sectionBuilder.getSections();
                const sectionError = validateSections(sections);
                if (sectionError) {
                    showAlert(sectionError, "error");
                    return;
                }
                payload.sections = sections;
            }
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
        settingsForm?.addEventListener("submit", handleSiteSettingsSubmit);
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
        loadSiteSettings();
    });
})();
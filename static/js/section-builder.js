(() => {
    const builders = new WeakMap();

    const generateId = () =>
        window.crypto && typeof window.crypto.randomUUID === "function"
            ? window.crypto.randomUUID()
            : `section-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;

    const cloneDeep = (value) => {
        if (typeof window.structuredClone === "function") {
            try {
                return window.structuredClone(value);
            } catch (error) {
                // fall back to JSON clone
            }
        }
        try {
            return JSON.parse(JSON.stringify(value));
        } catch (error) {
            if (Array.isArray(value)) {
                return value.slice();
            }
            if (value && typeof value === "object") {
                return Object.assign({}, value);
            }
            return value;
        }
    };

    const capitalise = (value) => {
        if (typeof value !== "string" || !value.length) {
            return "";
        }
        return value.charAt(0).toUpperCase() + value.slice(1);
    };

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
        if (options.type) {
            element.type = options.type;
        }
        if (options.value !== undefined) {
            element.value = options.value;
        }
        if (options.attrs) {
            Object.entries(options.attrs).forEach(([key, value]) => {
                if (value !== undefined && value !== null) {
                    element.setAttribute(key, value);
                }
            });
        }
        return element;
    };

    const normaliseImageContent = (content = {}) => ({
        url: content.url || content.URL || "",
        alt: content.alt || content.Alt || "",
        caption: content.caption || content.Caption || "",
    });

    const normaliseImageGroupContent = (content = {}) => {
        const imagesSource = content.images || content.Images;
        const images = Array.isArray(imagesSource) ? imagesSource : [];
        return {
            layout: content.layout || content.Layout || "grid",
            images: images.map((image) => normaliseImageContent(image)),
        };
    };

    const normaliseElement = (element) => {
        if (!element || typeof element !== "object") {
            return {
                id: generateId(),
                type: "paragraph",
                content: { text: "" },
            };
        }
        const type = element.type || element.Type || "paragraph";
        let content = element.content || element.Content || {};
        if (type === "paragraph") {
            content = { text: content.text || content.Text || "" };
        } else if (type === "image") {
            content = normaliseImageContent(content);
        } else if (type === "image_group") {
            content = normaliseImageGroupContent(content);
        } else {
            content = cloneDeep(content);
        }
        return {
            id: element.id || element.ID || generateId(),
            type,
            content,
        };
    };

    const normaliseSection = (section) => {
        if (!section || typeof section !== "object") {
            return {
                id: generateId(),
                title: "",
                image: "",
                elements: [],
            };
        }
        const elementsSource = section.elements || section.Elements || [];
        const elements = Array.isArray(elementsSource)
            ? elementsSource.map((item) => normaliseElement(item))
            : [];
        return {
            id: section.id || section.ID || generateId(),
            title: section.title || section.Title || "",
            image: section.image || section.Image || "",
            elements,
        };
    };

    const createEmptySection = () => ({
        id: generateId(),
        title: "",
        image: "",
        elements: [],
    });

    const createElementByType = (type) => {
        if (type === "image") {
            return { id: generateId(), type: "image", content: { url: "", alt: "", caption: "" } };
        }
        if (type === "image_group") {
            return { id: generateId(), type: "image_group", content: { layout: "grid", images: [] } };
        }
        return { id: generateId(), type: "paragraph", content: { text: "" } };
    };

    const serialiseElementContent = (element) => {
        if (!element || typeof element !== "object") {
            return null;
        }
        if (element.type === "paragraph") {
            const text = element.content?.text || "";
            return text.trim() ? { text: text } : null;
        }
        if (element.type === "image") {
            const url = element.content?.url || "";
            if (!url.trim()) {
                return null;
            }
            const payload = { url: url.trim() };
            const alt = element.content?.alt || "";
            if (alt.trim()) {
                payload.alt = alt.trim();
            }
            const caption = element.content?.caption || "";
            if (caption.trim()) {
                payload.caption = caption.trim();
            }
            return payload;
        }
        if (element.type === "image_group") {
            const layout = element.content?.layout || "";
            const imagesSource = element.content?.images;
            const images = Array.isArray(imagesSource) ? imagesSource : [];
            const serialisedImages = images
                .map((image) => {
                    if (!image) {
                        return null;
                    }
                    const url = image.url || image.URL || "";
                    if (!url.trim()) {
                        return null;
                    }
                    const payload = { url: url.trim() };
                    const alt = image.alt || image.Alt || "";
                    if (alt.trim()) {
                        payload.alt = alt.trim();
                    }
                    const caption = image.caption || image.Caption || "";
                    if (caption.trim()) {
                        payload.caption = caption.trim();
                    }
                    return payload;
                })
                .filter(Boolean);
            if (!serialisedImages.length) {
                return null;
            }
            return {
                layout: (layout || "grid").trim() || "grid",
                images: serialisedImages,
            };
        }
        return cloneDeep(element.content);
    };

    const buildSubmitSections = (sections) => {
        if (!Array.isArray(sections)) {
            return [];
        }
        return sections.map((section, index) => {
            const elements = Array.isArray(section.elements)
                ? section.elements
                      .map((element, elementIndex) => {
                          const content = serialiseElementContent(element);
                          if (!content) {
                              return null;
                          }
                          return {
                              id: element.id || generateId(),
                              type: element.type,
                              order: elementIndex + 1,
                              content,
                          };
                      })
                      .filter(Boolean)
                : [];
            return {
                id: section.id || generateId(),
                title: (section.title || "").trim(),
                image: (section.image || "").trim(),
                order: index + 1,
                elements,
            };
        });
    };

    const validateSections = (sections) => {
        if (!Array.isArray(sections) || !sections.length) {
            return null;
        }
        for (let i = 0; i < sections.length; i += 1) {
            const section = sections[i];
            const title = section?.title || "";
            if (!title.trim()) {
                return `Section ${i + 1} requires a title.`;
            }
            if (!Array.isArray(section.elements) || !section.elements.length) {
                return `Section ${i + 1} must include at least one element.`;
            }
            for (let j = 0; j < section.elements.length; j += 1) {
                const element = section.elements[j];
                if (!element || !element.type) {
                    return `Section ${i + 1}, element ${j + 1} is invalid.`;
                }
                if (element.type === "paragraph") {
                    const text = element.content?.text || "";
                    if (!text.trim()) {
                        return `Section ${i + 1}, paragraph ${j + 1} requires content.`;
                    }
                } else if (element.type === "image") {
                    const url = element.content?.url || "";
                    if (!url.trim()) {
                        return `Section ${i + 1}, image ${j + 1} requires an image URL.`;
                    }
                } else if (element.type === "image_group") {
                    const images = Array.isArray(element.content?.images) ? element.content.images : [];
                    if (!images.length) {
                        return `Section ${i + 1}, image group ${j + 1} requires at least one image.`;
                    }
                    const missingIndex = images.findIndex((image) => !(image?.url || "").trim());
                    if (missingIndex !== -1) {
                        return `Section ${i + 1}, image group ${j + 1} image ${missingIndex + 1} requires an image URL.`;
                    }
                }
            }
        }
        return null;
    };

    const elementTypeLabel = (type) => capitalise(String(type || "Paragraph").replace(/_/g, " "));

    const createBuilder = (root) => {
        if (!root) {
            return null;
        }
        const list = root.querySelector('[data-role="section-list"]');
        const empty = root.querySelector('[data-role="section-empty"]');
        const addButton = root.querySelector('[data-role="section-add"]');
        if (!list || !empty) {
            return null;
        }

        const state = {
            sections: [],
        };

        let draggingIndex = null;

        const clearDropIndicators = () => {
            list
                .querySelectorAll(".section-card--drop-before, .section-card--drop-after")
                .forEach((card) => {
                    card.classList.remove("section-card--drop-before", "section-card--drop-after");
                    delete card.dataset.dropPosition;
                });
        };

        const endDrag = () => {
            list.classList.remove("section-builder__list--dragging");
            clearDropIndicators();
            draggingIndex = null;
            list.querySelectorAll(".section-card").forEach((card) => {
                card.classList.remove("section-card--dragging");
                card.draggable = false;
            });
        };

        const moveSection = (fromIndex, toIndex) => {
            if (fromIndex === toIndex) {
                return;
            }
            if (fromIndex < 0 || fromIndex >= state.sections.length) {
                return;
            }
            const [removed] = state.sections.splice(fromIndex, 1);
            const boundedIndex = Math.max(0, Math.min(toIndex, state.sections.length));
            state.sections.splice(boundedIndex, 0, removed);
            render();
            endDrag();
        };

        const removeSection = (index) => {
            state.sections.splice(index, 1);
            render();
        };

        const moveElement = (sectionIndex, fromIndex, toIndex) => {
            const section = state.sections[sectionIndex];
            if (!section || !Array.isArray(section.elements)) {
                return;
            }
            if (toIndex < 0 || toIndex >= section.elements.length) {
                return;
            }
            const [removed] = section.elements.splice(fromIndex, 1);
            section.elements.splice(toIndex, 0, removed);
            render();
        };

        const removeElement = (sectionIndex, elementIndex) => {
            const section = state.sections[sectionIndex];
            if (!section || !Array.isArray(section.elements)) {
                return;
            }
            section.elements.splice(elementIndex, 1);
            render();
        };

        const moveGroupImage = (sectionIndex, elementIndex, fromIndex, toIndex) => {
            const section = state.sections[sectionIndex];
            const element = section?.elements?.[elementIndex];
            const images = Array.isArray(element?.content?.images) ? element.content.images : null;
            if (!images || toIndex < 0 || toIndex >= images.length) {
                return;
            }
            const [removed] = images.splice(fromIndex, 1);
            images.splice(toIndex, 0, removed);
            render();
        };

        const removeGroupImage = (sectionIndex, elementIndex, imageIndex) => {
            const section = state.sections[sectionIndex];
            const element = section?.elements?.[elementIndex];
            const images = Array.isArray(element?.content?.images) ? element.content.images : null;
            if (!images) {
                return;
            }
            images.splice(imageIndex, 1);
            render();
        };

        const createSectionCard = (section, index) => {
            const item = createElement("li", { className: "section-card" });
            item.dataset.sectionId = section.id;
            item.dataset.index = index;
            item.draggable = false;

            const header = createElement("div", { className: "section-card__header" });

            const dragHandle = createElement("button", {
                className: "section-card__drag-handle",
                type: "button",
                attrs: {
                    "aria-label": "Reorder section",
                    title: "Drag to reorder section",
                },
                html: "<span aria-hidden=\"true\">⋮⋮</span>",
            });

            dragHandle.addEventListener("pointerdown", () => {
                item.draggable = true;
            });

            const resetDraggable = () => {
                if (!item.classList.contains("section-card--dragging")) {
                    item.draggable = false;
                }
            };

            dragHandle.addEventListener("pointerup", resetDraggable);
            dragHandle.addEventListener("pointercancel", resetDraggable);

            header.appendChild(dragHandle);
            header.appendChild(
                createElement("span", {
                    className: "section-card__title",
                    textContent: `Section ${index + 1}`,
                })
            );

            const controls = createElement("div", { className: "section-card__controls" });
            const moveUp = createElement("button", {
                className: "section-card__control",
                textContent: "Move up",
                type: "button",
            });
            moveUp.disabled = index === 0;
            moveUp.addEventListener("click", () => moveSection(index, index - 1));
            controls.appendChild(moveUp);

            const moveDown = createElement("button", {
                className: "section-card__control",
                textContent: "Move down",
                type: "button",
            });
            moveDown.disabled = index === state.sections.length - 1;
            moveDown.addEventListener("click", () => moveSection(index, index + 1));
            controls.appendChild(moveDown);

            const remove = createElement("button", {
                className: "section-card__control",
                textContent: "Remove",
                type: "button",
            });
            remove.addEventListener("click", () => removeSection(index));
            controls.appendChild(remove);

            header.appendChild(controls);
            item.appendChild(header);

            item.addEventListener("dragstart", (event) => {
                draggingIndex = index;
                list.classList.add("section-builder__list--dragging");
                item.classList.add("section-card--dragging");
                try {
                    event.dataTransfer.effectAllowed = "move";
                    event.dataTransfer.setData("text/plain", String(index));
                } catch (error) {
                    // ignore when dataTransfer is unavailable
                }
            });

            item.addEventListener("dragover", (event) => {
                if (draggingIndex === null || draggingIndex === index) {
                    return;
                }
                event.preventDefault();
                clearDropIndicators();
                const rect = item.getBoundingClientRect();
                const offset = event.clientY - rect.top;
                const insertBefore = offset < rect.height / 2;
                item.dataset.dropPosition = insertBefore ? "before" : "after";
                item.classList.add(
                    insertBefore ? "section-card--drop-before" : "section-card--drop-after"
                );
                try {
                    event.dataTransfer.dropEffect = "move";
                } catch (error) {
                    // ignore when dataTransfer is unavailable
                }
            });

            item.addEventListener("drop", (event) => {
                if (draggingIndex === null) {
                    return;
                }
                event.preventDefault();
                event.stopPropagation();
                const fromIndex = draggingIndex;
                const dropPosition = item.dataset.dropPosition === "before" ? "before" : "after";
                let destination = dropPosition === "before" ? index : index + 1;
                if (fromIndex < destination) {
                    destination -= 1;
                }
                clearDropIndicators();
                if (fromIndex === destination) {
                    endDrag();
                    return;
                }
                moveSection(fromIndex, destination);
            });

            item.addEventListener("dragend", () => {
                endDrag();
            });

            const body = createElement("div", { className: "section-card__body" });
            const titleId = `${section.id}-title`;
            const titleField = createElement("div", { className: "section-field" });
            const titleLabel = createElement("label", { textContent: "Title", attrs: { for: titleId } });
            const titleInput = createElement("input", { type: "text" });
            titleInput.id = titleId;
            titleInput.value = section.title || "";
            titleInput.placeholder = "Section heading";
            titleInput.addEventListener("input", (event) => {
                section.title = event.target.value;
            });
            titleField.appendChild(titleLabel);
            titleField.appendChild(titleInput);
            body.appendChild(titleField);

            const imageId = `${section.id}-image`;
            const imageField = createElement("div", { className: "section-field" });
            const imageLabel = createElement("label", { textContent: "Featured image (optional)", attrs: { for: imageId } });
            const imageInput = createElement("input", { type: "url" });
            imageInput.id = imageId;
            imageInput.placeholder = "https://example.com/image.jpg";
            imageInput.value = section.image || "";
            imageInput.addEventListener("input", (event) => {
                section.image = event.target.value;
            });
            imageField.appendChild(imageLabel);
            imageField.appendChild(imageInput);
            body.appendChild(imageField);

            const elementsWrapper = createElement("div", { className: "section-elements" });
            if (!section.elements.length) {
                elementsWrapper.appendChild(
                    createElement("p", {
                        className: "section-elements__empty",
                        textContent: "No elements added yet.",
                    })
                );
            } else {
                section.elements.forEach((element, elementIndex) => {
                    elementsWrapper.appendChild(
                        createElementCard(section, index, element, elementIndex)
                    );
                });
            }

            const elementActions = createElement("div", { className: "section-elements__actions" });
            const addParagraph = createElement("button", {
                className: "section-elements__button",
                textContent: "Add paragraph",
                type: "button",
            });
            addParagraph.addEventListener("click", () => {
                section.elements.push(createElementByType("paragraph"));
                render();
            });
            elementActions.appendChild(addParagraph);

            const addImage = createElement("button", {
                className: "section-elements__button",
                textContent: "Add image",
                type: "button",
            });
            addImage.addEventListener("click", () => {
                section.elements.push(createElementByType("image"));
                render();
            });
            elementActions.appendChild(addImage);

            const addGallery = createElement("button", {
                className: "section-elements__button",
                textContent: "Add image group",
                type: "button",
            });
            addGallery.addEventListener("click", () => {
                section.elements.push(createElementByType("image_group"));
                render();
            });
            elementActions.appendChild(addGallery);

            elementsWrapper.appendChild(elementActions);
            body.appendChild(elementsWrapper);

            item.appendChild(body);
            return item;
        };

        const createElementCard = (section, sectionIndex, element, elementIndex) => {
            const wrapper = createElement("div", { className: "section-element" });
            wrapper.dataset.elementId = element.id;

            const header = createElement("div", { className: "section-element__header" });
            header.appendChild(
                createElement("span", {
                    className: "section-element__title",
                    textContent: `${elementTypeLabel(element.type)} ${elementIndex + 1}`,
                })
            );
            const controls = createElement("div", { className: "section-element__controls" });
            const moveUp = createElement("button", {
                className: "section-element__control",
                textContent: "Up",
                type: "button",
            });
            moveUp.disabled = elementIndex === 0;
            moveUp.addEventListener("click", () => moveElement(sectionIndex, elementIndex, elementIndex - 1));
            controls.appendChild(moveUp);

            const moveDown = createElement("button", {
                className: "section-element__control",
                textContent: "Down",
                type: "button",
            });
            moveDown.disabled = elementIndex === section.elements.length - 1;
            moveDown.addEventListener("click", () => moveElement(sectionIndex, elementIndex, elementIndex + 1));
            controls.appendChild(moveDown);

            const remove = createElement("button", {
                className: "section-element__control",
                textContent: "Remove",
                type: "button",
            });
            remove.addEventListener("click", () => removeElement(sectionIndex, elementIndex));
            controls.appendChild(remove);

            header.appendChild(controls);
            wrapper.appendChild(header);

            const body = createElement("div", { className: "section-element__body" });
            if (element.type === "paragraph") {
                const contentId = `${section.id}-${element.id}-content`;
                const field = createElement("div", { className: "section-field" });
                const label = createElement("label", {
                    textContent: "HTML content",
                    attrs: { for: contentId },
                });
                const textarea = document.createElement("textarea");
                textarea.id = contentId;
                textarea.placeholder = "<p>Describe this section…</p>";
                textarea.value = element.content?.text || "";
                textarea.addEventListener("input", (event) => {
                    element.content.text = event.target.value;
                });
                field.appendChild(label);
                field.appendChild(textarea);
                body.appendChild(field);
            } else if (element.type === "image") {
                const urlId = `${section.id}-${element.id}-url`;
                const urlField = createElement("div", { className: "section-field" });
                const urlLabel = createElement("label", {
                    textContent: "Image URL",
                    attrs: { for: urlId },
                });
                const urlInput = createElement("input", { type: "url" });
                urlInput.id = urlId;
                urlInput.placeholder = "https://example.com/image.jpg";
                urlInput.value = element.content?.url || "";
                urlInput.addEventListener("input", (event) => {
                    element.content.url = event.target.value;
                });
                urlField.appendChild(urlLabel);
                urlField.appendChild(urlInput);
                body.appendChild(urlField);

                const altId = `${section.id}-${element.id}-alt`;
                const altField = createElement("div", { className: "section-field" });
                const altLabel = createElement("label", {
                    textContent: "Alt text",
                    attrs: { for: altId },
                });
                const altInput = createElement("input", { type: "text" });
                altInput.id = altId;
                altInput.placeholder = "Describe the image for accessibility";
                altInput.value = element.content?.alt || "";
                altInput.addEventListener("input", (event) => {
                    element.content.alt = event.target.value;
                });
                altField.appendChild(altLabel);
                altField.appendChild(altInput);
                body.appendChild(altField);

                const captionId = `${section.id}-${element.id}-caption`;
                const captionField = createElement("div", { className: "section-field" });
                const captionLabel = createElement("label", {
                    textContent: "Caption (optional)",
                    attrs: { for: captionId },
                });
                const captionInput = createElement("input", { type: "text" });
                captionInput.id = captionId;
                captionInput.placeholder = "Add a supporting caption";
                captionInput.value = element.content?.caption || "";
                captionInput.addEventListener("input", (event) => {
                    element.content.caption = event.target.value;
                });
                captionField.appendChild(captionLabel);
                captionField.appendChild(captionInput);
                body.appendChild(captionField);
            } else if (element.type === "image_group") {
                if (!Array.isArray(element.content?.images)) {
                    element.content.images = [];
                }
                const layoutId = `${section.id}-${element.id}-layout`;
                const layoutField = createElement("div", { className: "section-field" });
                const layoutLabel = createElement("label", {
                    textContent: "Layout",
                    attrs: { for: layoutId },
                });
                const layoutInput = createElement("input", { type: "text" });
                layoutInput.id = layoutId;
                layoutInput.placeholder = "grid | columns | masonry";
                layoutInput.value = element.content?.layout || "grid";
                layoutInput.addEventListener("input", (event) => {
                    element.content.layout = event.target.value;
                });
                layoutField.appendChild(layoutLabel);
                layoutField.appendChild(layoutInput);
                body.appendChild(layoutField);

                const imageList = createElement("div", { className: "image-group-list" });
                if (!element.content.images.length) {
                    imageList.appendChild(
                        createElement("p", {
                            className: "section-elements__empty",
                            textContent: "No images added to this group.",
                        })
                    );
                } else {
                    element.content.images.forEach((image, imageIndex) => {
                        const item = createElement("div", { className: "image-group-item" });
                        const urlId = `${section.id}-${element.id}-image-${imageIndex}-url`;
                        const urlField = createElement("div", { className: "section-field" });
                        const urlLabel = createElement("label", {
                            textContent: `Image ${imageIndex + 1} URL`,
                            attrs: { for: urlId },
                        });
                        const urlInput = createElement("input", { type: "url" });
                        urlInput.id = urlId;
                        urlInput.placeholder = "https://example.com/image.jpg";
                        urlInput.value = image.url || image.URL || "";
                        urlInput.addEventListener("input", (event) => {
                            image.url = event.target.value;
                        });
                        urlField.appendChild(urlLabel);
                        urlField.appendChild(urlInput);
                        item.appendChild(urlField);

                        const altId = `${section.id}-${element.id}-image-${imageIndex}-alt`;
                        const altField = createElement("div", { className: "section-field" });
                        const altLabel = createElement("label", {
                            textContent: "Alt text",
                            attrs: { for: altId },
                        });
                        const altInput = createElement("input", { type: "text" });
                        altInput.id = altId;
                        altInput.placeholder = "Describe the image";
                        altInput.value = image.alt || image.Alt || "";
                        altInput.addEventListener("input", (event) => {
                            image.alt = event.target.value;
                        });
                        altField.appendChild(altLabel);
                        altField.appendChild(altInput);
                        item.appendChild(altField);

                        const captionId = `${section.id}-${element.id}-image-${imageIndex}-caption`;
                        const captionField = createElement("div", { className: "section-field" });
                        const captionLabel = createElement("label", {
                            textContent: "Caption",
                            attrs: { for: captionId },
                        });
                        const captionInput = createElement("input", { type: "text" });
                        captionInput.id = captionId;
                        captionInput.placeholder = "Optional caption";
                        captionInput.value = image.caption || image.Caption || "";
                        captionInput.addEventListener("input", (event) => {
                            image.caption = event.target.value;
                        });
                        captionField.appendChild(captionLabel);
                        captionField.appendChild(captionInput);
                        item.appendChild(captionField);

                        const imageControls = createElement("div", { className: "image-group-item__controls" });
                        const imageUp = createElement("button", {
                            className: "image-group-item__control",
                            textContent: "Up",
                            type: "button",
                        });
                        imageUp.disabled = imageIndex === 0;
                        imageUp.addEventListener("click", () =>
                            moveGroupImage(sectionIndex, elementIndex, imageIndex, imageIndex - 1)
                        );
                        imageControls.appendChild(imageUp);

                        const imageDown = createElement("button", {
                            className: "image-group-item__control",
                            textContent: "Down",
                            type: "button",
                        });
                        imageDown.disabled = imageIndex === element.content.images.length - 1;
                        imageDown.addEventListener("click", () =>
                            moveGroupImage(sectionIndex, elementIndex, imageIndex, imageIndex + 1)
                        );
                        imageControls.appendChild(imageDown);

                        const imageRemove = createElement("button", {
                            className: "image-group-item__control",
                            textContent: "Remove",
                            type: "button",
                        });
                        imageRemove.addEventListener("click", () =>
                            removeGroupImage(sectionIndex, elementIndex, imageIndex)
                        );
                        imageControls.appendChild(imageRemove);

                        item.appendChild(imageControls);
                        imageList.appendChild(item);
                    });
                }

                const addImageButton = createElement("button", {
                    className: "section-elements__button",
                    textContent: "Add image to group",
                    type: "button",
                });
                addImageButton.addEventListener("click", () => {
                    element.content.images.push({ url: "", alt: "", caption: "" });
                    render();
                });
                imageList.appendChild(addImageButton);
                body.appendChild(imageList);
            }

            wrapper.appendChild(body);
            return wrapper;
        };

        list.addEventListener("dragover", (event) => {
            if (draggingIndex === null) {
                return;
            }
            event.preventDefault();
            if (event.target === list) {
                clearDropIndicators();
            }
        });

        list.addEventListener("drop", (event) => {
            if (draggingIndex === null) {
                return;
            }
            event.preventDefault();
            event.stopPropagation();
            clearDropIndicators();
            const fromIndex = draggingIndex;
            const destination = state.sections.length - 1;
            if (fromIndex === destination) {
                endDrag();
                return;
            }
            moveSection(fromIndex, destination);
        });

        const render = () => {
            list.innerHTML = "";
            if (!state.sections.length) {
                empty.hidden = false;
                return;
            }
            empty.hidden = true;
            state.sections.forEach((section, index) => {
                list.appendChild(createSectionCard(section, index));
            });
        };

        addButton?.addEventListener("click", () => {
            state.sections.push(createEmptySection());
            render();
            window.requestAnimationFrame(() => {
                list.lastElementChild?.scrollIntoView({ behavior: "smooth", block: "center" });
            });
        });

        render();

        return {
            root,
            getSections: () => state.sections,
            setSections: (sections) => {
                if (Array.isArray(sections)) {
                    state.sections = sections.map((section) => normaliseSection(section));
                } else {
                    state.sections = [];
                }
                render();
            },
            reset: () => {
                state.sections = [];
                render();
            },
            validate: () => validateSections(state.sections),
            serialize: () => buildSubmitSections(state.sections),
        };
    };

    const init = (root) => {
        if (!root) {
            return null;
        }
        if (builders.has(root)) {
            return builders.get(root);
        }
        const builder = createBuilder(root);
        if (builder) {
            builders.set(root, builder);
        }
        return builder;
    };

    window.SectionBuilder = {
        init,
    };
})();
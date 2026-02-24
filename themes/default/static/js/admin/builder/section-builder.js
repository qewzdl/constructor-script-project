(() => {
    const utils = window.AdminUtils;
    const registry = window.AdminElementRegistry;
    const sectionRegistry = window.AdminSectionRegistry;
    const stateModule = window.AdminSectionState;
    const viewModule = window.AdminSectionView;
    const eventsModule = window.AdminSectionEvents;

    if (
        !utils ||
        !registry ||
        !sectionRegistry ||
        !stateModule ||
        !viewModule ||
        !eventsModule
    ) {
        return;
    }
    const createSectionBuilder = (form, options = {}) => {
        if (!form) {
            return null;
        }

        const builderRoot = form.querySelector('[data-section-builder]');
        if (!builderRoot) {
            return null;
        }

        const queryWithFallback = (root, selectors) => {
            if (!root) {
                return null;
            }
            for (const selector of selectors) {
                if (!selector) {
                    continue;
                }
                const node = root.querySelector(selector);
                if (node) {
                    return node;
                }
            }
            return null;
        };

        const sectionList = queryWithFallback(builderRoot, [
            '[data-section-list]',
            '[data-role="section-list"]',
        ]);
        const emptyState = queryWithFallback(builderRoot, [
            '[data-section-empty]',
            '[data-role="section-empty"]',
        ]);
        const addSectionButton = queryWithFallback(builderRoot, [
            '[data-action="section-add"]',
            '[data-role="section-add"]',
        ]);

        if (!sectionList || !addSectionButton) {
            return null;
        }

        const definitions = registry.getDefinitions();
        const orderedTypes = registry.getOrderedTypes();
        const sectionDefinitions = sectionRegistry.getDefinitions();
        const orderedSectionTypes = sectionRegistry.getOrderedTypes();
        const sectionTypeOrder = Array.isArray(orderedSectionTypes)
            ? orderedSectionTypes
            : Object.keys(sectionDefinitions || {});
        const normaliseTypeValue = (value) =>
            utils.normaliseString(value).trim().toLowerCase();
        const normaliseBackgroundGroupToken = (value) =>
            utils.normaliseString(value)
                .trim()
                .toLowerCase()
                .replace(/[^a-z0-9_-]+/g, '-')
                .replace(/-{2,}/g, '-')
                .replace(/^-+|-+$/g, '');
        const normaliseBackgroundStyleToken = (value) =>
            utils.normaliseString(value)
                .trim()
                .toLowerCase()
                .replace(/[^a-z0-9_-]+/g, '-')
                .replace(/-{2,}/g, '-')
                .replace(/^-+|-+$/g, '');
        const getTypeVariations = (sectionType) => {
            const definition = sectionDefinitions?.[sectionType];
            if (!definition || !Array.isArray(definition.variations)) {
                return [];
            }

            const seen = new Set();
            return definition.variations
                .map((variation) => {
                    const value = normaliseTypeValue(
                        variation?.value ?? variation?.id ?? variation?.type
                    );
                    if (!value || seen.has(value)) {
                        return null;
                    }
                    seen.add(value);
                    return {
                        value,
                        label: utils.normaliseString(variation?.label).trim() || value,
                        description: utils.normaliseString(
                            variation?.description
                        ).trim(),
                        isDefault:
                            variation?.isDefault === true ||
                            variation?.is_default === true ||
                            variation?.default === true,
                    };
                })
                .filter(Boolean);
        };
        const resolveTypeVariation = (sectionType, value) => {
            const variations = getTypeVariations(sectionType);
            if (!variations.length) {
                return '';
            }

            let requested = normaliseTypeValue(value);
            if (sectionType === 'features' && requested === 'icon-text') {
                requested = 'glyph';
            }
            if (requested) {
                const match = variations.find(
                    (variation) => variation.value === requested
                );
                if (match) {
                    return match.value;
                }
            }

            const fallback =
                variations.find((variation) => variation.isDefault)?.value ||
                variations[0]?.value;
            return fallback || '';
        };
        let selectedSectionType = sectionRegistry.getDefaultType?.()
            || sectionTypeOrder?.[0]
            || 'standard';
        let selectedSectionVariation = resolveTypeVariation(
            selectedSectionType,
            ''
        );
        let openTypeSelectionFlowForAdd = null;
        if (addSectionButton.parentElement) {
            let variationField = null;
            let variationSelect = null;
            let variationHint = null;
            let variationMetaField = null;
            let variationSummaryLabel = null;
            let variationSummaryCode = null;
            let variationMetaHint = null;
            let changeVariationButton = null;
            const updateVariationSelector = () => {
                const variations = getTypeVariations(selectedSectionType);
                selectedSectionVariation = resolveTypeVariation(
                    selectedSectionType,
                    selectedSectionVariation
                );
                const selectedVariation = variations.find(
                    (variation) => variation.value === selectedSectionVariation
                );
                const variationLabel = utils
                    .normaliseString(selectedVariation?.label)
                    .trim();
                const variationDescription = utils
                    .normaliseString(selectedVariation?.description)
                    .trim();

                if (variationField && variationSelect) {
                    variationSelect.innerHTML = '';
                    const hasVariationChoices = variations.length > 1;
                    if (!hasVariationChoices) {
                        variationField.hidden = true;
                        if (variationHint) {
                            variationHint.hidden = true;
                            variationHint.textContent = '';
                        }
                    } else {
                        variations.forEach((variation) => {
                            const option = utils.createElement('option', {
                                value: variation.value,
                                textContent: variation.label,
                            });
                            if (variation.value === selectedSectionVariation) {
                                option.selected = true;
                            }
                            variationSelect.append(option);
                        });
                        variationField.hidden = false;
                        if (variationHint) {
                            variationHint.textContent = variationDescription;
                            variationHint.hidden = !variationDescription;
                        }
                    }
                }

                if (variationMetaField) {
                    const hasVariations = variations.length > 1;
                    variationMetaField.hidden = !hasVariations;
                    if (changeVariationButton) {
                        changeVariationButton.hidden = !hasVariations;
                    }
                    if (!hasVariations) {
                        if (variationSummaryLabel) {
                            variationSummaryLabel.textContent = '';
                        }
                        if (variationSummaryCode) {
                            variationSummaryCode.textContent = '';
                        }
                        if (variationMetaHint) {
                            variationMetaHint.hidden = true;
                            variationMetaHint.textContent = '';
                        }
                        return;
                    }
                    if (variationSummaryLabel) {
                        variationSummaryLabel.textContent =
                            variationLabel || selectedSectionVariation;
                    }
                    if (variationSummaryCode) {
                        variationSummaryCode.textContent = selectedSectionVariation;
                    }
                    if (variationMetaHint) {
                        variationMetaHint.textContent = variationDescription;
                        variationMetaHint.hidden = !variationDescription;
                    }
                }
            };

            const typePickerModule = window.AdminSectionTypePicker;
            if (typePickerModule?.open && sectionTypeOrder.length) {
                const typeSelector = utils.createElement('div', {
                    className: 'section-builder__type-selector',
                });
                const typeMetaStack = utils.createElement('div', {
                    className: 'admin-builder__meta-stack',
                });
                const typeMetaGroup = utils.createElement('div', {
                    className: 'admin-builder__meta-group',
                });
                const typeControl = utils.createElement('div', {
                    className: 'admin-builder__meta-row',
                });
                const typeSummary = utils.createElement('div', {
                    className: 'admin-builder__meta-summary',
                });
                const typeSummaryLabel = utils.createElement('span', {
                    className: 'admin-builder__meta-summary-label',
                });
                const typeSummaryCode = utils.createElement('code', {
                    className: 'admin-builder__meta-summary-code',
                });
                typeSummary.append(typeSummaryLabel, typeSummaryCode);
                const changeTypeButton = utils.createElement('button', {
                    className:
                        'admin-builder__button admin-builder__button--ghost admin-builder__type-button',
                    type: 'button',
                    textContent: 'Change type',
                });
                typeControl.append(typeSummary, changeTypeButton);
                typeMetaGroup.append(typeControl);
                const typeHint = utils.createElement('span', {
                    className:
                        'admin-builder__hint admin-builder__meta-hint section-builder__type-hint',
                });
                typeHint.hidden = true;
                typeMetaGroup.append(typeHint);

                variationMetaField = utils.createElement('div', {
                    className: 'admin-builder__meta-group',
                });
                const variationControl = utils.createElement('div', {
                    className: 'admin-builder__meta-row',
                });
                const variationSummary = utils.createElement('div', {
                    className: 'admin-builder__meta-summary',
                });
                variationSummaryLabel = utils.createElement('span', {
                    className: 'admin-builder__meta-summary-label',
                });
                variationSummaryCode = utils.createElement('code', {
                    className: 'admin-builder__meta-summary-code',
                });
                variationSummary.append(
                    variationSummaryLabel,
                    variationSummaryCode
                );
                changeVariationButton = utils.createElement('button', {
                    className:
                        'admin-builder__button admin-builder__button--ghost admin-builder__type-button',
                    type: 'button',
                    textContent: 'Change variation',
                });
                variationControl.append(
                    variationSummary,
                    changeVariationButton
                );
                variationMetaField.append(variationControl);
                variationMetaHint = utils.createElement('span', {
                    className:
                        'admin-builder__hint admin-builder__meta-hint section-builder__type-hint',
                });
                variationMetaHint.hidden = true;
                variationMetaField.append(variationMetaHint);
                variationMetaField.hidden = true;
                typeMetaStack.append(typeMetaGroup, variationMetaField);
                typeSelector.append(typeMetaStack);

                const updateSelector = () => {
                    const definition = sectionDefinitions?.[selectedSectionType] || {};
                    const safeType =
                        typeof selectedSectionType === 'string' ? selectedSectionType : '';
                    const typeValue = safeType || 'unknown';
                    const labelText = utils.normaliseString(definition.label).trim();
                    typeSummaryLabel.textContent = labelText || typeValue;
                    typeSummaryCode.textContent = typeValue;
                    const description = utils.normaliseString(definition.description).trim();
                    if (description) {
                        typeHint.textContent = description;
                        typeHint.hidden = false;
                    } else {
                        typeHint.textContent = '';
                        typeHint.hidden = true;
                    }
                    updateVariationSelector();
                };

                const applyTypeSelection = (nextType, nextVariation) => {
                    const normalisedType = normaliseTypeValue(nextType);
                    if (!normalisedType) {
                        return;
                    }
                    selectedSectionType = normalisedType;
                    selectedSectionVariation = resolveTypeVariation(
                        normalisedType,
                        nextVariation
                    );
                    updateSelector();
                };

                const openVariationPickerForType = (
                    nextType,
                    { onSelect, onCancel } = {}
                ) => {
                    if (typeof typePickerModule?.openVariations !== 'function') {
                        return false;
                    }
                    const variations = getTypeVariations(nextType);
                    if (variations.length <= 1) {
                        return false;
                    }

                    const activeVariation =
                        nextType === selectedSectionType
                            ? selectedSectionVariation
                            : '';
                    typePickerModule.openVariations({
                        sectionType: nextType,
                        sectionDefinitions,
                        activeVariation: resolveTypeVariation(
                            nextType,
                            activeVariation
                        ),
                        onSelect: (nextVariation) => {
                            const resolvedVariation = resolveTypeVariation(
                                nextType,
                                nextVariation
                            );
                            if (typeof onSelect === 'function') {
                                onSelect(resolvedVariation);
                            }
                        },
                        onCancel,
                    });
                    return true;
                };

                updateSelector();

                changeTypeButton.addEventListener('click', (event) => {
                    event.preventDefault();
                    typePickerModule.open({
                        orderedSectionTypes: sectionTypeOrder,
                        sectionDefinitions,
                        activeType: selectedSectionType,
                        onSelect: (nextType) => {
                            const normalisedType = normaliseTypeValue(nextType);
                            if (!normalisedType) {
                                return;
                            }
                            if (
                                openVariationPickerForType(normalisedType, {
                                    onSelect: (nextVariation) => {
                                        applyTypeSelection(
                                            normalisedType,
                                            nextVariation
                                        );
                                    },
                                })
                            ) {
                                return;
                            }
                            if (normalisedType === selectedSectionType) {
                                return;
                            }
                            applyTypeSelection(normalisedType, '');
                        },
                    });
                });
                changeVariationButton.addEventListener('click', (event) => {
                    event.preventDefault();
                    openVariationPickerForType(selectedSectionType, {
                        onSelect: (nextVariation) => {
                            applyTypeSelection(
                                selectedSectionType,
                                nextVariation
                            );
                        },
                    });
                });

                openTypeSelectionFlowForAdd = (onSelect) => {
                    typePickerModule.open({
                        orderedSectionTypes: sectionTypeOrder,
                        sectionDefinitions,
                        activeType: selectedSectionType,
                        title: 'Choose section to add',
                        onSelect: (nextType) => {
                            const normalisedType = normaliseTypeValue(nextType);
                            if (!normalisedType) {
                                return;
                            }

                            const commitSelection = (nextVariation) => {
                                const resolvedVariation = resolveTypeVariation(
                                    normalisedType,
                                    nextVariation
                                );
                                applyTypeSelection(
                                    normalisedType,
                                    resolvedVariation
                                );
                                if (typeof onSelect === 'function') {
                                    onSelect(
                                        normalisedType,
                                        resolvedVariation
                                    );
                                }
                            };

                            if (
                                openVariationPickerForType(normalisedType, {
                                    onSelect: (nextVariation) => {
                                        commitSelection(nextVariation);
                                    },
                                })
                            ) {
                                return;
                            }

                            commitSelection('');
                        },
                    });
                };

                addSectionButton.parentElement.insertBefore(
                    typeSelector,
                    addSectionButton
                );
            } else {
                const typePicker = utils.createElement('select', {
                    className: 'admin-builder__type-picker',
                });
                typePicker.setAttribute('aria-label', 'Section type');
                sectionTypeOrder.forEach((type) => {
                    const definition = sectionDefinitions?.[type] || {};
                    const option = utils.createElement('option', {
                        textContent: definition.label || type,
                    });
                    option.value = type;
                    if (type === selectedSectionType) {
                        option.selected = true;
                    }
                    typePicker.append(option);
                });
                typePicker.addEventListener('change', (event) => {
                    if (event.target && event.target.value) {
                        selectedSectionType = event.target.value;
                        updateVariationSelector();
                    }
                });

                variationField = utils.createElement('label', {
                    className: 'admin-builder__field',
                });
                variationField.append(
                    utils.createElement('span', {
                        className: 'admin-builder__hint',
                        textContent: 'Variation',
                    })
                );
                variationSelect = utils.createElement('select', {
                    className: 'admin-builder__type-picker',
                });
                variationSelect.setAttribute('aria-label', 'Section variation');
                variationField.append(variationSelect);
                variationHint = utils.createElement('span', {
                    className: 'admin-builder__hint section-builder__type-hint',
                });
                variationHint.hidden = true;
                variationField.append(variationHint);
                variationField.hidden = true;
                variationSelect.addEventListener('change', (event) => {
                    selectedSectionVariation = resolveTypeVariation(
                        selectedSectionType,
                        event?.target?.value
                    );
                    updateVariationSelector();
                });

                const pickerWrapper = utils.createElement('div', {
                    className: 'section-builder__type-selector',
                });
                pickerWrapper.append(typePicker, variationField);
                addSectionButton.parentElement.insertBefore(
                    pickerWrapper,
                    addSectionButton
                );
                updateVariationSelector();
            }
        }
        const state = stateModule.createManager(definitions, sectionDefinitions);
        const { onApplyPaddingToAllSections } = options || {};
        const collapsedSectionIds = new Set();

        const syncCollapsedSections = () => {
            const activeIds = new Set(
                state
                    .getState()
                    .map((section) => utils.normaliseString(section?.clientId))
                    .filter(Boolean)
            );
            Array.from(collapsedSectionIds).forEach((sectionClientId) => {
                if (!activeIds.has(sectionClientId)) {
                    collapsedSectionIds.delete(sectionClientId);
                }
            });
            return activeIds;
        };

        const areAllSectionsCollapsed = () => {
            const sections = state.getState();
            if (!sections.length) {
                return false;
            }
            return sections.every((section) => collapsedSectionIds.has(section.clientId));
        };

        const setSectionCollapsed = (sectionClientId, collapsed) => {
            const normalisedId = utils.normaliseString(sectionClientId);
            if (!normalisedId) {
                return false;
            }
            const section = state
                .getState()
                .find((item) => item && item.clientId === normalisedId);
            if (!section) {
                return false;
            }
            if (collapsed) {
                collapsedSectionIds.add(normalisedId);
            } else {
                collapsedSectionIds.delete(normalisedId);
            }
            return true;
        };

        const toggleSectionCollapsed = (sectionClientId) => {
            const normalisedId = utils.normaliseString(sectionClientId);
            if (!normalisedId) {
                return false;
            }
            const shouldCollapse = !collapsedSectionIds.has(normalisedId);
            return setSectionCollapsed(normalisedId, shouldCollapse);
        };

        const setAllSectionsCollapsed = (collapsed) => {
            const sections = state.getState();
            if (!sections.length) {
                collapsedSectionIds.clear();
                return;
            }
            if (!collapsed) {
                collapsedSectionIds.clear();
                return;
            }
            sections.forEach((section) => {
                if (section?.clientId) {
                    collapsedSectionIds.add(section.clientId);
                }
            });
        };

        let collapseAllButton = null;
        let sectionSearchInput = null;
        let sectionHideDisabledButton = null;
        let sectionFilterSummary = null;
        let filteredSectionsEmptyState = null;
        let sectionSearchQuery = '';
        let hideDisabledSections = false;

        const buildSectionSearchValue = (sectionNode) => {
            if (!sectionNode) {
                return '';
            }
            return [
                utils.normaliseString(sectionNode.dataset.sectionTitle)
                    .trim()
                    .toLowerCase(),
                utils.normaliseString(sectionNode.dataset.sectionTypeLabel)
                    .trim()
                    .toLowerCase(),
                utils.normaliseString(sectionNode.dataset.sectionType)
                    .trim()
                    .toLowerCase(),
                utils.normaliseString(sectionNode.dataset.sectionVariation)
                    .trim()
                    .toLowerCase(),
            ]
                .filter(Boolean)
                .join(' ');
        };

        const getFilteredSectionsEmptyStateMessage = () => {
            const hasQuery = Boolean(
                utils.normaliseString(sectionSearchQuery).trim()
            );
            if (hasQuery && hideDisabledSections) {
                return 'No sections match the current search and enabled-only filter.';
            }
            if (hasQuery) {
                return 'No sections match this search.';
            }
            if (hideDisabledSections) {
                return 'All sections are disabled. Use "Show disabled" to view them.';
            }
            return 'No sections match the current filters.';
        };

        const updateSectionFilterControls = ({
            totalSections = 0,
            visibleSections = 0,
            disabledSections = 0,
        } = {}) => {
            if (sectionHideDisabledButton) {
                const toggleLabel = hideDisabledSections
                    ? disabledSections > 0
                        ? `Show disabled (${disabledSections})`
                        : 'Show disabled'
                    : disabledSections > 0
                    ? `Hide disabled (${disabledSections})`
                    : 'Hide disabled';
                sectionHideDisabledButton.textContent = toggleLabel;
                sectionHideDisabledButton.setAttribute(
                    'aria-pressed',
                    hideDisabledSections ? 'true' : 'false'
                );
                sectionHideDisabledButton.disabled =
                    totalSections === 0 ||
                    (!hideDisabledSections && disabledSections === 0);
            }

            if (!sectionFilterSummary) {
                return;
            }
            if (totalSections === 0) {
                sectionFilterSummary.textContent = 'No sections yet';
                return;
            }
            if (
                visibleSections === totalSections &&
                !utils.normaliseString(sectionSearchQuery).trim() &&
                !hideDisabledSections
            ) {
                sectionFilterSummary.textContent =
                    totalSections === 1
                        ? '1 section total'
                        : `${totalSections} sections total`;
                return;
            }
            sectionFilterSummary.textContent =
                visibleSections === 1
                    ? 'Showing 1 section'
                    : `Showing ${visibleSections} of ${totalSections}`;
        };

        const applySectionFilters = () => {
            const sectionItems = Array.from(
                sectionList.querySelectorAll('[data-section-client]')
            );
            const normalisedQuery = utils
                .normaliseString(sectionSearchQuery)
                .trim()
                .toLowerCase();
            let visibleSections = 0;
            let disabledSections = 0;

            sectionItems.forEach((sectionItem) => {
                const isDisabled =
                    utils.normaliseString(sectionItem.dataset.sectionDisabled) ===
                    'true';
                if (isDisabled) {
                    disabledSections += 1;
                }
                const searchValue = buildSectionSearchValue(sectionItem);
                const matchesQuery =
                    !normalisedQuery || searchValue.includes(normalisedQuery);
                const matchesDisabledFilter =
                    !hideDisabledSections || !isDisabled;
                const shouldShow = matchesQuery && matchesDisabledFilter;
                sectionItem.hidden = !shouldShow;
                if (shouldShow) {
                    visibleSections += 1;
                }
            });

            updateSectionFilterControls({
                totalSections: sectionItems.length,
                visibleSections,
                disabledSections,
            });

            if (!filteredSectionsEmptyState) {
                return;
            }
            const shouldShowFilteredState =
                sectionItems.length > 0 && visibleSections === 0;
            filteredSectionsEmptyState.hidden = !shouldShowFilteredState;
            if (!shouldShowFilteredState) {
                return;
            }
            filteredSectionsEmptyState.textContent =
                getFilteredSectionsEmptyStateMessage();
            sectionList.append(filteredSectionsEmptyState);
        };

        const view = viewModule.createView({
            listElement: sectionList,
            emptyState,
            definitions,
            orderedTypes,
            sectionDefinitions,
            orderedSectionTypes,
            isSectionCollapsed: (sectionClientId) =>
                collapsedSectionIds.has(sectionClientId),
            applyPaddingToAllSections: onApplyPaddingToAllSections,
            applyBackgroundGroupToSections: ({
                sectionClientIds,
                groupValue,
                previousGroup,
                styleValue,
            } = {}) => {
                const selectedIds = new Set(
                    Array.isArray(sectionClientIds)
                        ? sectionClientIds
                              .map((id) => utils.normaliseString(id))
                              .filter(Boolean)
                        : []
                );
                if (!selectedIds.size) {
                    return '';
                }

                const normalisedGroup =
                    normaliseBackgroundGroupToken(groupValue);
                const normalisedPreviousGroup =
                    normaliseBackgroundGroupToken(previousGroup);
                const normalisedStyle =
                    normaliseBackgroundStyleToken(styleValue);

                state.getState().forEach((section) => {
                    if (!section || !section.clientId) {
                        return;
                    }

                    const hasSettings =
                        section.settings &&
                        typeof section.settings === 'object';
                    if (!hasSettings) {
                        section.settings = {};
                    }

                    const existingGroup = normaliseBackgroundGroupToken(
                        section.settings?.background_group ??
                            section.settings?.backgroundGroup
                    );
                    const isSelected = selectedIds.has(section.clientId);

                    if (isSelected) {
                        if (normalisedGroup) {
                            section.settings.background_group = normalisedGroup;
                        } else {
                            delete section.settings.background_group;
                        }
                        if (section.settings.backgroundGroup !== undefined) {
                            delete section.settings.backgroundGroup;
                        }
                        if (normalisedStyle) {
                            section.settings.background_style = normalisedStyle;
                        } else if (
                            section.settings.background_style !== undefined
                        ) {
                            delete section.settings.background_style;
                        }
                        if (section.settings.backgroundStyle !== undefined) {
                            delete section.settings.backgroundStyle;
                        }
                        return;
                    }

                    if (
                        normalisedPreviousGroup &&
                        existingGroup === normalisedPreviousGroup
                    ) {
                        delete section.settings.background_group;
                        if (section.settings.backgroundGroup !== undefined) {
                            delete section.settings.backgroundGroup;
                        }
                        delete section.settings.background_style;
                        if (section.settings.backgroundStyle !== undefined) {
                            delete section.settings.backgroundStyle;
                        }
                    }
                });

                render();
                emitChange();
                return normalisedGroup;
            },
            applyBackgroundStyleToGroup: ({ groupValue, styleValue } = {}) => {
                const normalisedGroup =
                    normaliseBackgroundGroupToken(groupValue);
                if (!normalisedGroup) {
                    return '';
                }

                const normalisedStyle =
                    normaliseBackgroundStyleToken(styleValue);

                state.getState().forEach((section) => {
                    if (!section) {
                        return;
                    }
                    const hasSettings =
                        section.settings &&
                        typeof section.settings === 'object';
                    if (!hasSettings) {
                        section.settings = {};
                    }

                    const existingGroup = normaliseBackgroundGroupToken(
                        section.settings?.background_group ??
                            section.settings?.backgroundGroup
                    );
                    if (existingGroup !== normalisedGroup) {
                        return;
                    }

                    if (normalisedStyle) {
                        section.settings.background_style = normalisedStyle;
                    } else {
                        delete section.settings.background_style;
                    }
                    if (section.settings.backgroundStyle !== undefined) {
                        delete section.settings.backgroundStyle;
                    }
                });

                emitChange();
                return normalisedStyle;
            },
        });

        const updateCollapseAllButton = () => {
            if (!collapseAllButton) {
                return;
            }
            const totalSections = state.getState().length;
            collapseAllButton.disabled = totalSections === 0;
            collapseAllButton.textContent = areAllSectionsCollapsed()
                ? 'Expand all sections'
                : 'Collapse all sections';
        };

        const render = () => {
            syncCollapsedSections();
            view.render(state.getState());
            updateCollapseAllButton();
            applySectionFilters();
        };

        const emitChange = () => {
            state.notify();
        };

        let pageId = null;

        const setPageId = (id) => {
            pageId = id;
        };

        const setSections = (nextSections) => {
            state.setSections(nextSections);
            collapsedSectionIds.clear();
            render();
            emitChange();
        };

        const reset = () => {
            state.reset();
            collapsedSectionIds.clear();
            render();
            emitChange();
        };

        const addSection = (
            sectionType = selectedSectionType,
            variation = selectedSectionVariation
        ) => {
            const section = state.addSection(sectionType, variation);
            collapsedSectionIds.delete(section.clientId);
            render();
            emitChange();
            view.focusField(
                `[data-section-client="${section.clientId}"] [data-field="section-title"]`
            );
        };

        const removeSection = (sectionClientId) => {
            collapsedSectionIds.delete(utils.normaliseString(sectionClientId));
            state.removeSection(sectionClientId);
            render();
            emitChange();
        };

        const duplicateSection = (sectionClientId) => {
            const duplicatedSection = state.duplicateSection(sectionClientId);
            if (!duplicatedSection) {
                return;
            }
            collapsedSectionIds.delete(duplicatedSection.clientId);
            render();
            emitChange();
            view.focusField(
                `[data-section-client="${duplicatedSection.clientId}"] [data-field="section-title"]`
            );
        };

        const toggleSectionCollapse = (sectionClientId) => {
            if (!toggleSectionCollapsed(sectionClientId)) {
                return;
            }
            render();
            const normalisedId = utils.normaliseString(sectionClientId);
            const isCollapsed = collapsedSectionIds.has(normalisedId);
            if (!isCollapsed) {
                view.focusField(
                    `[data-section-client="${normalisedId}"] [data-field="section-title"]`
                );
                return;
            }
            view.focusField(
                `[data-section-client="${normalisedId}"] [data-action="section-collapse"]`
            );
        };

        const moveSection = (sectionClientId, direction) => {
            const nextIndex = state.moveSection(sectionClientId, direction);
            if (nextIndex < 0) {
                return;
            }
            render();
            emitChange();
            let focusRole = '';
            if (direction === 'up') {
                focusRole = nextIndex <= 0 ? 'section-move-down' : 'section-move-up';
            } else if (direction === 'down') {
                const lastIndex = state.getState().length - 1;
                focusRole = nextIndex >= lastIndex ? 'section-move-up' : 'section-move-down';
            }
            if (focusRole) {
                view.focusField(
                    `[data-section-client="${sectionClientId}"] [data-role="${focusRole}"]`
                );
            } else {
                view.focusField(
                    `[data-section-client="${sectionClientId}"] [data-field="section-title"]`
                );
            }
        };

        const addElementToSection = (sectionClientId, type) => {
            const element = state.addElementToSection(sectionClientId, type);
            if (!element) {
                return;
            }
            render();
            emitChange();
            const definition = definitions[type];
            const focusSelector = definition?.initialFocusSelector
                ? ` ${definition.initialFocusSelector}`
                : type === 'paragraph'
                ? ' textarea'
                : ' [data-field]';
            view.focusField(
                `[data-section-client="${sectionClientId}"] [data-element-client="${element.clientId}"]${focusSelector}`
            );
        };

        const removeElementFromSection = (sectionClientId, elementClientId) => {
            state.removeElementFromSection(sectionClientId, elementClientId);
            render();
            emitChange();
        };

        const moveElementInSection = (sectionClientId, elementClientId, direction) => {
            const nextIndex = state.moveElementInSection(sectionClientId, elementClientId, direction);
            if (nextIndex < 0) {
                return;
            }
            render();
            emitChange();
            const focusRole = direction === 'up' ? 'element-move-up' : 'element-move-down';
            view.focusField(
                `[data-section-client="${sectionClientId}"] [data-element-client="${elementClientId}"] [data-role="${focusRole}"]`
            );
        };

        const addGroupImage = (sectionClientId, elementClientId) => {
            const image = state.addGroupImage(sectionClientId, elementClientId);
            if (!image) {
                return;
            }
            render();
            emitChange();
            view.focusField(
                `[data-section-client="${sectionClientId}"] [data-element-client="${elementClientId}"] [data-group-image-client="${image.clientId}"] [data-field="group-image-url"]`
            );
        };

        const removeGroupImage = (
            sectionClientId,
            elementClientId,
            imageClientId
        ) => {
            state.removeGroupImage(sectionClientId, elementClientId, imageClientId);
            render();
            emitChange();
        };

        const addGroupFile = (sectionClientId, elementClientId) => {
            const file = state.addGroupFile(sectionClientId, elementClientId);
            if (!file) {
                return;
            }
            render();
            emitChange();
            view.focusField(
                `[data-section-client="${sectionClientId}"] [data-element-client="${elementClientId}"] [data-group-file-client="${file.clientId}"] [data-field="group-file-url"]`
            );
        };

        const removeGroupFile = (
            sectionClientId,
            elementClientId,
            fileClientId
        ) => {
            state.removeGroupFile(sectionClientId, elementClientId, fileClientId);
            render();
            emitChange();
        };

        const updateSectionField = (sectionClientId, field, value) => {
            state.updateSectionField(sectionClientId, field, value);
            return (
                field === 'section-type' ||
                field === 'section-disabled' ||
                field === 'section-variation'
            );
        };

        const syncSectionFilterMetadata = (sectionClientId, field, value) => {
            const sectionClient = utils.normaliseString(sectionClientId).trim();
            if (!sectionClient || field !== 'section-title') {
                return;
            }
            const sectionNode = sectionList.querySelector(
                `[data-section-client="${sectionClient}"]`
            );
            if (!sectionNode) {
                return;
            }
            const nextTitle = utils.normaliseString(value).trim();
            if (nextTitle) {
                sectionNode.dataset.sectionTitle = nextTitle;
                return;
            }
            delete sectionNode.dataset.sectionTitle;
        };

        const updateElementField = (
            sectionClientId,
            elementClientId,
            field,
            value,
            nestedClientId
        ) => {
            state.updateElementField(
                sectionClientId,
                elementClientId,
                field,
                value,
                nestedClientId
            );
        };

        if (addSectionButton.parentElement) {
            const actionsContainer = addSectionButton.parentElement;
            const filters = utils.createElement('div', {
                className: 'section-builder__filters',
            });
            sectionSearchInput = utils.createElement('input', {
                className: 'section-builder__search',
                type: 'search',
                placeholder: 'Search sections',
                autocomplete: 'off',
                attributes: {
                    'aria-label': 'Search sections by title, type, or variation',
                },
            });
            sectionHideDisabledButton = utils.createElement('button', {
                className: 'section-builder__add section-builder__add--secondary',
                type: 'button',
                textContent: 'Hide disabled',
                attributes: {
                    'aria-pressed': 'false',
                },
            });
            sectionFilterSummary = utils.createElement('span', {
                className: 'section-builder__stats',
                attributes: {
                    'aria-live': 'polite',
                },
            });
            filters.append(
                sectionSearchInput,
                sectionHideDisabledButton,
                sectionFilterSummary
            );

            collapseAllButton = utils.createElement('button', {
                className: 'section-builder__add section-builder__add--secondary',
                type: 'button',
                textContent: 'Collapse all sections',
            });
            collapseAllButton.addEventListener('click', (event) => {
                event.preventDefault();
                const shouldCollapse = !areAllSectionsCollapsed();
                setAllSectionsCollapsed(shouldCollapse);
                render();
            });
            filteredSectionsEmptyState = utils.createElement('li', {
                className: 'section-builder__empty section-builder__empty--filtered',
            });
            filteredSectionsEmptyState.hidden = true;

            sectionSearchInput.addEventListener('input', (event) => {
                sectionSearchQuery = utils.normaliseString(event.target?.value);
                applySectionFilters();
            });
            sectionHideDisabledButton.addEventListener('click', (event) => {
                event.preventDefault();
                hideDisabledSections = !hideDisabledSections;
                applySectionFilters();
            });

            builderRoot.insertBefore(filters, sectionList);
            actionsContainer.insertBefore(collapseAllButton, addSectionButton);
        }

        addSectionButton.addEventListener('click', (event) => {
            if (typeof openTypeSelectionFlowForAdd === 'function') {
                event.preventDefault();
                openTypeSelectionFlowForAdd((sectionType, variation) => {
                    addSection(sectionType, variation);
                });
                return;
            }
            addSection();
        });

        const events = eventsModule.bind({
            listElement: sectionList,
            onSectionRemove: removeSection,
            onSectionMove: moveSection,
            onSectionDuplicate: duplicateSection,
            onSectionCollapse: toggleSectionCollapse,
            onElementRemove: removeElementFromSection,
            onElementMove: moveElementInSection,
            onElementAdd: addElementToSection,
            onGroupImageAdd: addGroupImage,
            onGroupImageRemove: removeGroupImage,
            onGroupFileAdd: addGroupFile,
            onGroupFileRemove: removeGroupFile,
            onSectionFieldChange: (sectionClientId, field, value) => {
                const needsRender = updateSectionField(sectionClientId, field, value);
                if (needsRender) {
                    render();
                } else {
                    syncSectionFilterMetadata(sectionClientId, field, value);
                    applySectionFilters();
                }
                emitChange();
            },
            onElementFieldChange: (
                sectionClientId,
                elementClientId,
                field,
                value,
                imageClientId
            ) => {
                updateElementField(
                    sectionClientId,
                    elementClientId,
                    field,
                    value,
                    imageClientId
                );
                emitChange();
            },
        });

        const onChange = (listener) => {
            if (typeof listener !== 'function') {
                return () => {};
            }
            state.subscribe(listener);
            return () => state.unsubscribe(listener);
        };

        render();

        return {
            setSections,
            reset,
            getSections: () => state.getSections(),
            setPageId,
            onChange,
            destroy: () => {
                events.destroy();
            },
        };
    };

    window.AdminSectionBuilder = {
        create: createSectionBuilder,
    };
})();

(() => {
    const panelDefinitions = [];
    const quickActionDefinitions = [];
    const panelInstances = [];
    const readyCallbacks = [];

    let initialised = false;
    let rootElement = null;
    let panelContainer = null;
    let quickActionsContainer = null;
    let layoutContext = null;

    const parseOrder = (value, fallback = 0) => {
        if (typeof value === 'number' && Number.isFinite(value)) {
            return value;
        }
        if (typeof value === 'string') {
            const trimmed = value.trim();
            if (trimmed) {
                const parsed = Number.parseFloat(trimmed);
                if (Number.isFinite(parsed)) {
                    return parsed;
                }
            }
        }
        return fallback;
    };

    const createFromMarkup = (markup) => {
        if (typeof document === 'undefined') {
            return null;
        }
        const template = document.createElement('template');
        template.innerHTML = markup.trim();
        return template.content.firstElementChild || null;
    };

    const shouldRender = (definition, context) => {
        if (typeof definition?.shouldRender === 'function') {
            return Boolean(definition.shouldRender(context));
        }
        return true;
    };

    const insertPanelInstance = (instance) => {
        if (!panelContainer || !instance?.element) {
            return;
        }
        const { element, order } = instance;
        let insertBefore = null;
        for (let index = 0; index < panelInstances.length; index += 1) {
            const current = panelInstances[index];
            if (order < current.order) {
                insertBefore = current.element;
                panelInstances.splice(index, 0, instance);
                panelContainer.insertBefore(element, insertBefore);
                return;
            }
        }
        panelInstances.push(instance);
        panelContainer.appendChild(element);
    };

    const buildPanel = (definition, context) => {
        if (!shouldRender(definition, context)) {
            return null;
        }
        let element = null;
        if (typeof definition.create === 'function') {
            element = definition.create(context);
        } else if (definition.template) {
            element = createFromMarkup(definition.template);
        }
        if (!element) {
            return null;
        }
        if (!(element instanceof Element)) {
            return null;
        }
        const order = parseOrder(
            definition.order,
            Number.isFinite(definition.__index) ? definition.__index : panelDefinitions.length
        );
        element.dataset.panelOrder = String(order);
        return { element, order };
    };

    const buildQuickActionItem = (definition, context) => {
        if (!shouldRender(definition, context)) {
            return null;
        }
        if (typeof document === 'undefined') {
            return null;
        }
        const label = definition.label || '';
        if (!label) {
            return null;
        }
        const item = document.createElement('li');
        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'admin__shortcut';
        button.textContent = label;
        if (definition.navTarget) {
            button.dataset.navTarget = String(definition.navTarget);
        }
        if (definition.panelAction) {
            button.dataset.panelAction = String(definition.panelAction);
        }
        item.appendChild(button);
        return { element: item, order: parseOrder(definition.order, quickActionDefinitions.length) };
    };

    const rebuildQuickActions = () => {
        if (!quickActionsContainer) {
            return;
        }
        quickActionsContainer.innerHTML = '';
        const context = layoutContext;
        const items = quickActionDefinitions
            .map((definition) => buildQuickActionItem(definition, context))
            .filter((entry) => entry && entry.element);
        items.sort((a, b) => a.order - b.order);
        items.forEach((entry) => {
            quickActionsContainer.appendChild(entry.element);
        });
        if (rootElement) {
            rootElement.dispatchEvent(
                new CustomEvent('admin:quick-actions-changed', {
                    bubbles: false,
                    detail: { items: items.map((entry) => entry.element) },
                })
            );
        }
    };

    const notifyPanelsChanged = () => {
        if (!rootElement) {
            return;
        }
        const panels = Array.from(panelContainer?.querySelectorAll('.admin-panel') || []);
        rootElement.dispatchEvent(
            new CustomEvent('admin:panels-changed', {
                bubbles: false,
                detail: { panels },
            })
        );
    };

    const notifyReady = () => {
        if (!layoutContext) {
            return;
        }
        while (readyCallbacks.length) {
            const callback = readyCallbacks.shift();
            try {
                callback(layoutContext);
            } catch (error) {
                console.error('AdminLayout ready callback failed', error);
            }
        }
        if (rootElement) {
            rootElement.dispatchEvent(
                new CustomEvent('admin:layout-ready', {
                    bubbles: false,
                    detail: layoutContext,
                })
            );
        }
    };

    const init = () => {
        if (initialised) {
            return layoutContext;
        }
        if (typeof document === 'undefined') {
            return null;
        }
        rootElement = document.querySelector('.admin[data-page="admin"]');
        if (!rootElement) {
            return null;
        }
        panelContainer = rootElement.querySelector('[data-role="panel-container"]');
        quickActionsContainer = rootElement.querySelector('[data-role="admin-quick-actions"]');

        layoutContext = {
            root: rootElement,
            dataset: rootElement.dataset,
            blogEnabled: rootElement.dataset.blogEnabled !== 'false',
            forumEnabled:
                rootElement.dataset.forumEnabled !== 'false' &&
                rootElement.dataset.forumEnabled !== undefined,
            languageFeatureEnabled:
                rootElement.dataset.languageFeatureEnabled !== 'false' &&
                rootElement.dataset.languageFeatureEnabled !== undefined,
        };

        const definitions = panelDefinitions.slice();
        definitions.sort((a, b) => {
            const orderA = parseOrder(
                a.order,
                Number.isFinite(a.__index) ? a.__index : panelDefinitions.indexOf(a)
            );
            const orderB = parseOrder(
                b.order,
                Number.isFinite(b.__index) ? b.__index : panelDefinitions.indexOf(b)
            );
            if (orderA !== orderB) {
                return orderA - orderB;
            }
            const labelA = a.id || '';
            const labelB = b.id || '';
            return labelA.localeCompare(labelB);
        });

        definitions.forEach((definition) => {
            const instance = buildPanel(definition, layoutContext);
            if (!instance) {
                return;
            }
            insertPanelInstance(instance);
        });

        rebuildQuickActions();

        initialised = true;
        notifyPanelsChanged();
        notifyReady();
        return layoutContext;
    };

    const registerPanel = (definition) => {
        if (!definition) {
            return;
        }
        const entry = Object.assign({}, definition);
        entry.__index = panelDefinitions.length;
        panelDefinitions.push(entry);
        if (initialised) {
            const instance = buildPanel(entry, layoutContext);
            if (!instance) {
                return;
            }
            insertPanelInstance(instance);
            notifyPanelsChanged();
        }
    };

    const registerQuickAction = (definition) => {
        if (!definition) {
            return;
        }
        quickActionDefinitions.push(Object.assign({}, definition));
        if (initialised) {
            rebuildQuickActions();
        }
    };

    const whenReady = (callback) => {
        if (typeof callback !== 'function') {
            return;
        }
        if (layoutContext) {
            callback(layoutContext);
            return;
        }
        readyCallbacks.push(callback);
    };

    window.AdminLayout = {
        registerPanel,
        registerQuickAction,
        init,
        whenReady,
        getContext: () => layoutContext,
        getRoot: () => rootElement,
    };
})();

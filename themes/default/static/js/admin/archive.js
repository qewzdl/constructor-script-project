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
              const controls = form.querySelectorAll('input, textarea, button, select');
              controls.forEach((element) => {
                  element.disabled = disabled;
              });
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

    const coerceNumber = (value) => {
        const numeric = Number(value);
        return Number.isFinite(numeric) ? numeric : null;
    };

    const normalizeEndpoint = (value) => {
        if (!value) {
            return '';
        }
        return value.replace(/\/+$/, '');
    };

    layout.whenReady((context) => {
        if (!context || !context.archiveEnabled) {
            return;
        }

        const root = context.root || document;
        const panel = root.querySelector('#admin-panel-archive');
        if (!panel) {
            return;
        }

        const directoriesEndpointRaw = (context.dataset?.endpointArchiveDirectories || '').trim();
        const filesEndpointRaw = (context.dataset?.endpointArchiveFiles || '').trim();
        if (!directoriesEndpointRaw || !filesEndpointRaw) {
            return;
        }

        const directoriesEndpoint = normalizeEndpoint(directoriesEndpointRaw);
        const filesEndpoint = normalizeEndpoint(filesEndpointRaw);
        const treeEndpointRaw = (context.dataset?.endpointArchiveTree || '').trim();
        const treeEndpoint = treeEndpointRaw ? treeEndpointRaw : `${directoriesEndpoint}?tree=1`;

        const directoryTreeContainer = panel.querySelector('[data-role="archive-directory-tree"]');
        const directoryTreeList = panel.querySelector('[data-role="archive-directory-list"]');
        const directoryTreeEmpty = panel.querySelector('[data-role="archive-directory-empty"]');
        const directoryForm = panel.querySelector('#admin-archive-directory-form');
        const directoryStatus = panel.querySelector('[data-role="archive-directory-status"]');
        const directoryParentSelect = panel.querySelector('[data-role="archive-directory-parent"]');
        const directoryDeleteButton = panel.querySelector('[data-role="archive-directory-delete"]');
        const directorySubmitButton = panel.querySelector('[data-role="archive-directory-submit"]');
        const directoryResetButton = panel.querySelector('[data-action="archive-directory-reset"]');

        const filesTable = panel.querySelector('#admin-archive-files-table');
        const fileForm = panel.querySelector('#admin-archive-file-form');
        const fileStatus = panel.querySelector('[data-role="archive-file-status"]');
        const fileDirectorySelect = panel.querySelector('[data-role="archive-file-directory"]');
        const fileDeleteButton = panel.querySelector('[data-role="archive-file-delete"]');
        const fileSubmitButton = panel.querySelector('[data-role="archive-file-submit"]');
        const fileResetButton = panel.querySelector('[data-action="archive-file-reset"]');

        if (!directoryTreeContainer || !directoryTreeList || !directoryForm || !filesTable || !fileForm) {
            return;
        }

        const state = {
            tree: [],
            directoryList: [],
            directoryMap: new Map(),
            directoryDescendants: new Map(),
            selectedDirectoryId: '',
            files: [],
            selectedFileId: '',
        };

        const showAlert = (message, type = 'info') => {
            if (!message) {
                setAlert(globalAlertId, '', type);
                return;
            }
            setAlert(globalAlertId, message, type);
        };

        const buildDirectoryMaps = (entries) => {
            state.directoryList = [];
            state.directoryMap = new Map();
            state.directoryDescendants = new Map();

            const childrenMap = new Map();

            const traverse = (items, depth = 0, parentId = null) => {
                if (!Array.isArray(items)) {
                    return;
                }
                items.forEach((item) => {
                    if (!item || typeof item !== 'object') {
                        return;
                    }
                    const id = String(item.id || item.ID || '');
                    if (!id) {
                        return;
                    }
                    const normalized = {
                        id,
                        name: item.name || item.Name || '',
                        path: item.path || item.Path || '',
                        parentId,
                        published: item.published !== false,
                        order: coerceNumber(item.order) ?? 0,
                        updatedAt: item.updated_at || item.updatedAt || item.UpdatedAt || null,
                        children: Array.isArray(item.children) ? item.children : [],
                    };
                    state.directoryMap.set(id, normalized);
                    if (!childrenMap.has(parentId)) {
                        childrenMap.set(parentId, []);
                    }
                    childrenMap.get(parentId).push(id);
                    state.directoryList.push({
                        id,
                        name: normalized.name,
                        path: normalized.path,
                        published: normalized.published,
                        depth,
                    });
                    if (normalized.children.length > 0) {
                        traverse(normalized.children, depth + 1, id);
                    }
                });
            };

            traverse(entries, 0, null);

            const computeDescendants = (id) => {
                if (state.directoryDescendants.has(id)) {
                    return state.directoryDescendants.get(id);
                }
                const childIds = childrenMap.get(id) || [];
                const descendantSet = new Set();
                childIds.forEach((childId) => {
                    descendantSet.add(childId);
                    const childDescendants = computeDescendants(childId);
                    childDescendants.forEach((value) => descendantSet.add(value));
                });
                state.directoryDescendants.set(id, descendantSet);
                return descendantSet;
            };

            state.directoryMap.forEach((_value, key) => {
                computeDescendants(key);
            });
        };

        const renderDirectoryTree = () => {
            if (!directoryTreeList) {
                return;
            }
            directoryTreeList.innerHTML = '';
            const fragment = document.createDocumentFragment();

            const createNode = (entry) => {
                const item = document.createElement('li');
                item.className = 'admin-archive__item';
                const button = document.createElement('button');
                button.type = 'button';
                button.className = 'admin-archive__directory';
                button.dataset.id = entry.id;
                button.dataset.role = 'archive-directory-node';
                const label = entry.name || entry.path || `Directory ${entry.id}`;
                button.textContent = label;
                if (!entry.published) {
                    button.classList.add('is-unpublished');
                }
                if (state.selectedDirectoryId === entry.id) {
                    button.classList.add('is-active');
                }
                item.appendChild(button);

                const normalized = state.directoryMap.get(entry.id);
                const children = normalized?.children || [];
                if (children.length > 0) {
                    const childList = document.createElement('ul');
                    childList.className = 'admin-archive__list';
                    children.forEach((child) => {
                        const childId = String(child.id || child.ID || '');
                        if (!childId) {
                            return;
                        }
                        const childEntry = state.directoryMap.get(childId) || {
                            id: childId,
                            name: child.name || child.Name || '',
                            path: child.path || child.Path || '',
                            published: child.published !== false,
                        };
                        childList.appendChild(createNode(childEntry));
                    });
                    item.appendChild(childList);
                }
                return item;
            };

            if (state.tree.length > 0) {
                state.tree.forEach((entry) => {
                    const id = String(entry.id || entry.ID || '');
                    if (!id) {
                        return;
                    }
                    const normalized = state.directoryMap.get(id) || {
                        id,
                        name: entry.name || entry.Name || '',
                        path: entry.path || entry.Path || '',
                        published: entry.published !== false,
                        children: entry.children || [],
                    };
                    fragment.appendChild(createNode(normalized));
                });
            }

            directoryTreeList.appendChild(fragment);
            if (directoryTreeEmpty) {
                directoryTreeEmpty.hidden = state.tree.length > 0;
            }
        };

        const renderParentOptions = (selectedId, parentId) => {
            if (!directoryParentSelect) {
                return;
            }
            const selectedKey = selectedId ? String(selectedId) : '';
            const parentKey = parentId ? String(parentId) : '';
            const excluded = selectedKey ? state.directoryDescendants.get(selectedKey) || new Set() : new Set();
            directoryParentSelect.innerHTML = '';
            const defaultOption = document.createElement('option');
            defaultOption.value = '';
            defaultOption.textContent = 'No parent';
            directoryParentSelect.appendChild(defaultOption);

            state.directoryList.forEach((entry) => {
                const option = document.createElement('option');
                option.value = entry.id;
                const indent = entry.depth > 0 ? `${'\u2014 '.repeat(entry.depth)}` : '';
                option.textContent = `${indent}${entry.name || entry.path || `Directory ${entry.id}`}`;
                if (entry.id === selectedKey || excluded.has(entry.id)) {
                    option.disabled = true;
                }
                if (entry.id === parentKey) {
                    option.selected = true;
                }
                directoryParentSelect.appendChild(option);
            });
        };

        const renderFileDirectoryOptions = (selectedDirectoryId, fileDirectoryId) => {
            if (!fileDirectorySelect) {
                return;
            }
            const preferred = fileDirectoryId ? String(fileDirectoryId) : selectedDirectoryId ? String(selectedDirectoryId) : '';
            fileDirectorySelect.innerHTML = '';
            if (state.directoryList.length === 0) {
                const placeholder = document.createElement('option');
                placeholder.value = '';
                placeholder.textContent = 'No directories available';
                fileDirectorySelect.appendChild(placeholder);
                fileDirectorySelect.disabled = true;
                return;
            }
            fileDirectorySelect.disabled = false;
            state.directoryList.forEach((entry) => {
                const option = document.createElement('option');
                option.value = entry.id;
                const indent = entry.depth > 0 ? `${'\u2014 '.repeat(entry.depth)}` : '';
                option.textContent = `${indent}${entry.name || entry.path || `Directory ${entry.id}`}`;
                if (entry.id === preferred) {
                    option.selected = true;
                }
                fileDirectorySelect.appendChild(option);
            });
        };

        const resetDirectoryForm = (defaultParentId = '') => {
            directoryForm.reset();
            if (directoryStatus) {
                directoryStatus.hidden = true;
                directoryStatus.textContent = '';
            }
            if (directoryDeleteButton) {
                directoryDeleteButton.hidden = true;
                directoryDeleteButton.disabled = true;
            }
            if (directorySubmitButton) {
                directorySubmitButton.textContent = 'Create directory';
            }
            renderParentOptions('', defaultParentId || '');
        };

        const resetFileForm = (options = {}) => {
            const { directoryId = '', preserveDirectory = false } = options;
            fileForm.reset();
            if (fileStatus) {
                fileStatus.hidden = true;
                fileStatus.textContent = '';
            }
            if (fileDeleteButton) {
                fileDeleteButton.hidden = true;
                fileDeleteButton.disabled = true;
            }
            if (fileSubmitButton) {
                fileSubmitButton.textContent = 'Create file';
            }
            const preferredDirectory = preserveDirectory ? fileDirectorySelect?.value : directoryId;
            renderFileDirectoryOptions(state.selectedDirectoryId, preferredDirectory);
        };

        const setFileFormEnabled = (enabled) => {
            if (!fileForm) {
                return;
            }
            toggleFormDisabled(fileForm, !enabled);
            if (fileResetButton) {
                fileResetButton.disabled = !enabled;
            }
            if (fileDeleteButton) {
                fileDeleteButton.disabled = !enabled;
            }
            if (!enabled) {
                fileForm.classList.add('is-disabled');
            } else {
                fileForm.classList.remove('is-disabled');
            }
        };

        const renderFilesTable = (entries) => {
            filesTable.innerHTML = '';
            if (!Array.isArray(entries) || entries.length === 0) {
                const row = document.createElement('tr');
                row.className = 'admin-table__placeholder';
                const cell = document.createElement('td');
                cell.colSpan = 3;
                cell.textContent = state.selectedDirectoryId
                    ? 'No files found in this directory yet.'
                    : 'Select a directory to view files.';
                row.appendChild(cell);
                filesTable.appendChild(row);
                return;
            }

            const fragment = document.createDocumentFragment();
            entries.forEach((file) => {
                if (!file || typeof file !== 'object') {
                    return;
                }
                const id = String(file.id || file.ID || '');
                if (!id) {
                    return;
                }
                const row = document.createElement('tr');
                row.dataset.id = id;
                row.dataset.role = 'archive-file-row';
                row.className = 'admin-archive__file-row';
                if (state.selectedFileId === id) {
                    row.classList.add('is-active');
                }

                const nameCell = document.createElement('td');
                nameCell.textContent = file.name || file.Name || file.slug || file.Slug || `File ${id}`;
                row.appendChild(nameCell);

                const typeCell = document.createElement('td');
                typeCell.textContent = file.file_type || file.FileType || file.mime_type || file.MimeType || '—';
                row.appendChild(typeCell);

                const updatedCell = document.createElement('td');
                updatedCell.textContent = formatDateTime(file.updated_at || file.updatedAt || file.UpdatedAt);
                row.appendChild(updatedCell);

                fragment.appendChild(row);
            });

            filesTable.appendChild(fragment);
        };

        const loadFiles = async (directoryId, { preserveSelection = false } = {}) => {
            if (!directoryId) {
                state.files = [];
                state.selectedFileId = '';
                renderFilesTable([]);
                resetFileForm({ directoryId: '', preserveDirectory: false });
                return;
            }
            const idValue = Number(directoryId);
            if (!Number.isFinite(idValue)) {
                return;
            }
            try {
                const url = `${filesEndpoint}?directory_id=${encodeURIComponent(idValue)}`;
                const response = await apiClient(url);
                const files = Array.isArray(response?.files) ? response.files : [];
                state.files = files;
                if (!preserveSelection || !state.selectedFileId) {
                    state.selectedFileId = '';
                } else {
                    const exists = files.some((file) => String(file.id || file.ID || '') === state.selectedFileId);
                    if (!exists) {
                        state.selectedFileId = '';
                    }
                }
                renderFilesTable(files);
                resetFileForm({ directoryId: String(directoryId), preserveDirectory: true });
                if (state.selectedFileId) {
                    await selectFile(state.selectedFileId, { preserveForm: true });
                }
            } catch (error) {
                showAlert(error.message || 'Failed to load files', 'error');
                state.files = [];
                state.selectedFileId = '';
                renderFilesTable([]);
            }
        };

        const populateDirectoryForm = (directory) => {
            if (!directory) {
                return;
            }
            const nameInput = directoryForm.querySelector('[name="name"]');
            const slugInput = directoryForm.querySelector('[name="slug"]');
            const orderInput = directoryForm.querySelector('[name="order"]');
            const publishedInput = directoryForm.querySelector('[name="published"]');

            if (nameInput) {
                nameInput.value = directory.name || '';
            }
            if (slugInput) {
                slugInput.value = directory.slug || '';
            }
            if (orderInput) {
                const value = coerceNumber(directory.order);
                orderInput.value = value === null ? '' : String(value);
            }
            if (publishedInput) {
                publishedInput.checked = directory.published !== false;
            }
            const parentId = directory.parent_id ?? directory.parentId ?? null;
            renderParentOptions(String(directory.id || directory.ID || ''), parentId ? String(parentId) : '');
            if (directoryStatus) {
                const path = directory.path || directory.Path || '';
                const published = directory.published !== false ? 'Published' : 'Hidden';
                directoryStatus.textContent = path ? `${published} • /archive/${path}` : published;
                directoryStatus.hidden = false;
            }
            if (directoryDeleteButton) {
                directoryDeleteButton.hidden = false;
                directoryDeleteButton.disabled = false;
            }
            if (directorySubmitButton) {
                directorySubmitButton.textContent = 'Save directory';
            }
        };

        const populateFileForm = (file, { preserveDirectory = false } = {}) => {
            if (!file) {
                return;
            }
            const nameInput = fileForm.querySelector('[name="name"]');
            const slugInput = fileForm.querySelector('[name="slug"]');
            const directoryInput = fileForm.querySelector('[name="directory_id"]');
            const descriptionInput = fileForm.querySelector('[name="description"]');
            const fileUrlInput = fileForm.querySelector('[name="file_url"]');
            const previewUrlInput = fileForm.querySelector('[name="preview_url"]');
            const mimeInput = fileForm.querySelector('[name="mime_type"]');
            const typeInput = fileForm.querySelector('[name="file_type"]');
            const sizeInput = fileForm.querySelector('[name="file_size"]');
            const orderInput = fileForm.querySelector('[name="order"]');
            const publishedInput = fileForm.querySelector('[name="published"]');

            if (nameInput) {
                nameInput.value = file.name || '';
            }
            if (slugInput) {
                slugInput.value = file.slug || '';
            }
            if (descriptionInput) {
                descriptionInput.value = file.description || '';
            }
            if (fileUrlInput) {
                fileUrlInput.value = file.file_url || file.fileURL || file.FileURL || '';
            }
            if (previewUrlInput) {
                previewUrlInput.value = file.preview_url || file.previewURL || file.PreviewURL || '';
            }
            if (mimeInput) {
                mimeInput.value = file.mime_type || file.mimeType || file.MimeType || '';
            }
            if (typeInput) {
                typeInput.value = file.file_type || file.fileType || file.FileType || '';
            }
            if (sizeInput) {
                const size = coerceNumber(file.file_size ?? file.fileSize ?? file.FileSize);
                sizeInput.value = size === null ? '' : String(size);
            }
            if (orderInput) {
                const orderValue = coerceNumber(file.order);
                orderInput.value = orderValue === null ? '' : String(orderValue);
            }
            if (publishedInput) {
                publishedInput.checked = file.published !== false;
            }
            const directoryId = file.directory_id ?? file.directoryId ?? file.DirectoryID ?? null;
            renderFileDirectoryOptions(
                state.selectedDirectoryId,
                preserveDirectory ? (directoryInput ? directoryInput.value : directoryId) : directoryId
            );
            if (directoryInput && directoryId) {
                directoryInput.value = String(directoryId);
            }
            if (fileStatus) {
                const path = file.path || file.Path || '';
                const updated = formatDateTime(file.updated_at || file.updatedAt || file.UpdatedAt);
                const statusParts = [];
                if (path) {
                    statusParts.push(`/archive/files/${path}`);
                }
                if (updated !== '—') {
                    statusParts.push(`Updated ${updated}`);
                }
                fileStatus.textContent = statusParts.join(' • ');
                fileStatus.hidden = statusParts.length === 0;
            }
            if (fileDeleteButton) {
                fileDeleteButton.hidden = false;
                fileDeleteButton.disabled = false;
            }
            if (fileSubmitButton) {
                fileSubmitButton.textContent = 'Save file';
            }
        };

        const selectFile = async (fileId, { preserveForm = false } = {}) => {
            const id = Number(fileId);
            if (!Number.isFinite(id)) {
                return;
            }
            try {
                toggleFormDisabled(fileForm, true);
                const response = await apiClient(`${filesEndpoint}/${id}`);
                const file = response?.file;
                if (!file) {
                    throw new Error('File not found');
                }
                state.selectedFileId = String(file.id || file.ID || id);
                populateFileForm(file, { preserveDirectory: preserveForm });
                renderFilesTable(state.files);
                setFileFormEnabled(true);
            } catch (error) {
                showAlert(error.message || 'Failed to load file', 'error');
            } finally {
                toggleFormDisabled(fileForm, false);
            }
        };

        const selectDirectory = async (directoryId, { preserveForm = false } = {}) => {
            const id = Number(directoryId);
            if (!Number.isFinite(id)) {
                return;
            }
            try {
                toggleFormDisabled(directoryForm, true);
                setFileFormEnabled(false);
                const response = await apiClient(`${directoriesEndpoint}/${id}`);
                const directory = response?.directory;
                if (!directory) {
                    throw new Error('Directory not found');
                }
                state.selectedDirectoryId = String(directory.id || directory.ID || id);
                populateDirectoryForm(directory);
                renderDirectoryTree();
                await loadFiles(directory.id || id, { preserveSelection: preserveForm });
                setFileFormEnabled(true);
            } catch (error) {
                showAlert(error.message || 'Failed to load directory', 'error');
            } finally {
                toggleFormDisabled(directoryForm, false);
            }
        };

        const loadTree = async ({ preserveSelection = false } = {}) => {
            try {
                const response = await apiClient(treeEndpoint);
                const directories = Array.isArray(response?.directories) ? response.directories : [];
                state.tree = directories;
                buildDirectoryMaps(directories);
                renderDirectoryTree();
                renderParentOptions(state.selectedDirectoryId, '');
                renderFileDirectoryOptions(state.selectedDirectoryId, state.selectedDirectoryId);
                if (preserveSelection && state.selectedDirectoryId) {
                    const exists = state.directoryMap.has(state.selectedDirectoryId);
                    if (exists) {
                        await selectDirectory(state.selectedDirectoryId, { preserveForm: true });
                        return;
                    }
                }
                state.selectedDirectoryId = '';
                state.selectedFileId = '';
                resetDirectoryForm();
                renderFilesTable([]);
                resetFileForm();
                setFileFormEnabled(false);
            } catch (error) {
                showAlert(error.message || 'Failed to load archive tree', 'error');
                state.tree = [];
                state.directoryList = [];
                renderDirectoryTree();
                resetDirectoryForm();
                renderFilesTable([]);
                resetFileForm();
                setFileFormEnabled(false);
            }
        };

        directoryTreeContainer.addEventListener('click', (event) => {
            const target = event.target;
            if (!(target instanceof HTMLElement)) {
                return;
            }
            const id = target.dataset.id;
            if (!id) {
                return;
            }
            if (target.dataset.role === 'archive-directory-node') {
                event.preventDefault();
                selectDirectory(id, { preserveForm: false });
            }
        });

        filesTable.addEventListener('click', (event) => {
            const target = event.target;
            if (!(target instanceof HTMLElement)) {
                return;
            }
            const row = target.closest('tr[data-role="archive-file-row"]');
            if (!row) {
                return;
            }
            const id = row.dataset.id;
            if (!id) {
                return;
            }
            event.preventDefault();
            selectFile(id);
        });

        if (directoryResetButton) {
            directoryResetButton.addEventListener('click', (event) => {
                event.preventDefault();
                const parentId = state.selectedDirectoryId || '';
                state.selectedDirectoryId = '';
                resetDirectoryForm(parentId);
                renderDirectoryTree();
                showAlert('Ready to create a new directory.', 'info');
                setFileFormEnabled(false);
                renderFilesTable([]);
                resetFileForm({ directoryId: parentId, preserveDirectory: true });
            });
        }

        if (fileResetButton) {
            fileResetButton.addEventListener('click', (event) => {
                event.preventDefault();
                state.selectedFileId = '';
                resetFileForm({ directoryId: state.selectedDirectoryId, preserveDirectory: true });
                renderFilesTable(state.files);
                showAlert('Ready to create a new file.', 'info');
            });
        }

        directoryForm.addEventListener('submit', async (event) => {
            event.preventDefault();
            const formData = new FormData(directoryForm);
            const payload = {
                name: (formData.get('name') || '').toString().trim(),
                published: Boolean(formData.get('published')),
            };
            const slug = (formData.get('slug') || '').toString().trim();
            if (slug) {
                payload.slug = slug;
            }
            const orderValue = coerceNumber(formData.get('order'));
            if (orderValue !== null) {
                payload.order = orderValue;
            }
            const parentValueRaw = (formData.get('parent_id') || '').toString().trim();
            if (parentValueRaw === '') {
                payload.parent_id = null;
            } else {
                const parentValue = coerceNumber(parentValueRaw);
                if (parentValue !== null) {
                    payload.parent_id = parentValue;
                }
            }

            if (!payload.name) {
                showAlert('Directory name is required.', 'error');
                return;
            }

            const isUpdate = Boolean(state.selectedDirectoryId);
            const targetId = Number(state.selectedDirectoryId);
            const requestInit = {
                method: isUpdate ? 'PUT' : 'POST',
                body: JSON.stringify(payload),
            };

            try {
                toggleFormDisabled(directoryForm, true);
                const url = isUpdate
                    ? `${directoriesEndpoint}/${encodeURIComponent(targetId)}`
                    : directoriesEndpoint;
                const response = await apiClient(url, requestInit);
                const directory = response?.directory;
                showAlert(isUpdate ? 'Directory updated successfully.' : 'Directory created successfully.', 'success');
                if (directory) {
                    state.selectedDirectoryId = String(directory.id || directory.ID || targetId);
                }
                await loadTree({ preserveSelection: true });
            } catch (error) {
                showAlert(error.message || 'Failed to save directory', 'error');
            } finally {
                toggleFormDisabled(directoryForm, false);
            }
        });

        if (directoryDeleteButton) {
            directoryDeleteButton.addEventListener('click', async (event) => {
                event.preventDefault();
                if (!state.selectedDirectoryId) {
                    return;
                }
                const id = Number(state.selectedDirectoryId);
                if (!Number.isFinite(id)) {
                    return;
                }
                const confirmed = window.confirm(
                    'Deleting this directory will also remove its nested files. Are you sure you want to continue?'
                );
                if (!confirmed) {
                    return;
                }
                try {
                    toggleFormDisabled(directoryForm, true);
                    await apiClient(`${directoriesEndpoint}/${encodeURIComponent(id)}`, {
                        method: 'DELETE',
                    });
                    showAlert('Directory deleted.', 'success');
                    state.selectedDirectoryId = '';
                    state.selectedFileId = '';
                    await loadTree({ preserveSelection: false });
                } catch (error) {
                    showAlert(error.message || 'Failed to delete directory', 'error');
                } finally {
                    toggleFormDisabled(directoryForm, false);
                }
            });
        }

        fileForm.addEventListener('submit', async (event) => {
            event.preventDefault();
            const formData = new FormData(fileForm);
            const payload = {
                name: (formData.get('name') || '').toString().trim(),
                description: (formData.get('description') || '').toString().trim(),
                file_url: (formData.get('file_url') || '').toString().trim(),
                preview_url: (formData.get('preview_url') || '').toString().trim(),
                mime_type: (formData.get('mime_type') || '').toString().trim(),
                file_type: (formData.get('file_type') || '').toString().trim(),
                published: Boolean(formData.get('published')),
            };
            const slug = (formData.get('slug') || '').toString().trim();
            if (slug) {
                payload.slug = slug;
            }
            const directoryValue = coerceNumber(formData.get('directory_id'));
            if (directoryValue === null) {
                showAlert('Select a directory for this file.', 'error');
                return;
            }
            payload.directory_id = directoryValue;

            const sizeValue = coerceNumber(formData.get('file_size'));
            if (sizeValue !== null) {
                payload.file_size = sizeValue;
            }
            const orderValue = coerceNumber(formData.get('order'));
            if (orderValue !== null) {
                payload.order = orderValue;
            }

            if (!payload.name) {
                showAlert('File name is required.', 'error');
                return;
            }
            if (!payload.file_url) {
                showAlert('File URL is required.', 'error');
                return;
            }

            const isUpdate = Boolean(state.selectedFileId);
            const targetId = Number(state.selectedFileId);
            const requestInit = {
                method: isUpdate ? 'PUT' : 'POST',
                body: JSON.stringify(payload),
            };

            try {
                toggleFormDisabled(fileForm, true);
                const url = isUpdate
                    ? `${filesEndpoint}/${encodeURIComponent(targetId)}`
                    : filesEndpoint;
                const response = await apiClient(url, requestInit);
                const file = response?.file;
                showAlert(isUpdate ? 'File updated successfully.' : 'File created successfully.', 'success');
                if (file) {
                    state.selectedFileId = String(file.id || file.ID || targetId);
                }
                await loadFiles(payload.directory_id, { preserveSelection: true });
                await loadTree({ preserveSelection: true });
            } catch (error) {
                showAlert(error.message || 'Failed to save file', 'error');
            } finally {
                toggleFormDisabled(fileForm, false);
            }
        });

        if (fileDeleteButton) {
            fileDeleteButton.addEventListener('click', async (event) => {
                event.preventDefault();
                if (!state.selectedFileId) {
                    return;
                }
                const id = Number(state.selectedFileId);
                if (!Number.isFinite(id)) {
                    return;
                }
                const confirmed = window.confirm('Are you sure you want to delete this file?');
                if (!confirmed) {
                    return;
                }
                try {
                    toggleFormDisabled(fileForm, true);
                    await apiClient(`${filesEndpoint}/${encodeURIComponent(id)}`, {
                        method: 'DELETE',
                    });
                    showAlert('File deleted.', 'success');
                    state.selectedFileId = '';
                    await loadFiles(state.selectedDirectoryId, { preserveSelection: false });
                    await loadTree({ preserveSelection: true });
                } catch (error) {
                    showAlert(error.message || 'Failed to delete file', 'error');
                } finally {
                    toggleFormDisabled(fileForm, false);
                }
            });
        }

        resetDirectoryForm();
        resetFileForm();
        setFileFormEnabled(false);
        loadTree();
    });
})();

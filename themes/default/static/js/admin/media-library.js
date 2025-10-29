(() => {
    class MediaLibrary {
        constructor(options = {}) {
            const {
                fetchUploads,
                uploadFile,
                renameUpload,
                onOpen,
                onClose,
                texts = {},
            } = options;

            this.fetchUploads = typeof fetchUploads === 'function' ? fetchUploads : null;
            this.uploadFile = typeof uploadFile === 'function' ? uploadFile : null;
            this.renameUpload = typeof renameUpload === 'function' ? renameUpload : null;
            this.onOpen = typeof onOpen === 'function' ? onOpen : null;
            this.onClose = typeof onClose === 'function' ? onClose : null;
            this.texts = {
                title: 'Select image',
                upload: 'Upload image',
                refresh: 'Refresh',
                rename: 'Rename image',
                renamePrompt: 'Enter a new name for the image',
                renameSuccess: 'Image renamed successfully.',
                renameError: 'Failed to rename image.',
                renameEmpty: 'Image name cannot be empty.',
                empty: 'No uploads found yet.',
                error: 'Failed to load uploads. Try again shortly.',
                close: 'Close',
                cancel: 'Cancel',
                choose: 'Use image',
                uploading: 'Uploading…',
                ...texts,
            };

            this.root = null;
            this.grid = null;
            this.status = null;
            this.chooseButton = null;
            this.fileInput = null;
            this.renameButton = null;
            this.currentSelection = null;
            this.currentUploads = [];
            this.pendingFetch = null;
            this.pendingUpload = null;
            this.pendingRename = null;
            this.resolveClose = null;
        }

        ensureElements() {
            if (this.root) {
                return;
            }

            const root = document.createElement('div');
            root.className = 'admin-media-library';
            root.hidden = true;

            const dialog = document.createElement('div');
            dialog.className = 'admin-media-library__dialog';
            dialog.setAttribute('role', 'dialog');
            dialog.setAttribute('aria-modal', 'true');
            root.append(dialog);

            const header = document.createElement('header');
            header.className = 'admin-media-library__header';
            const title = document.createElement('h2');
            title.className = 'admin-media-library__title';
            title.textContent = this.texts.title;
            header.append(title);

            const closeButton = document.createElement('button');
            closeButton.type = 'button';
            closeButton.className = 'admin-media-library__close';
            closeButton.setAttribute('aria-label', this.texts.close);
            closeButton.textContent = '×';
            closeButton.dataset.action = 'media-library-close';
            header.append(closeButton);

            dialog.append(header);

            const body = document.createElement('div');
            body.className = 'admin-media-library__body';

            const actions = document.createElement('div');
            actions.className = 'admin-media-library__actions';

            const refreshButton = document.createElement('button');
            refreshButton.type = 'button';
            refreshButton.className = 'admin-media-library__action-button';
            refreshButton.dataset.action = 'media-library-refresh';
            refreshButton.textContent = this.texts.refresh;
            actions.append(refreshButton);

            const renameButton = document.createElement('button');
            renameButton.type = 'button';
            renameButton.className = 'admin-media-library__action-button';
            renameButton.dataset.action = 'media-library-rename';
            renameButton.textContent = this.texts.rename;
            if (!this.renameUpload) {
                renameButton.hidden = true;
            } else {
                renameButton.disabled = true;
            }
            actions.append(renameButton);
            this.renameButton = renameButton;

            const uploadLabel = document.createElement('label');
            uploadLabel.className = 'admin-media-library__upload-label';
            uploadLabel.textContent = this.texts.upload;
            uploadLabel.dataset.action = 'media-library-upload';

            if (this.uploadFile) {
                const fileInput = document.createElement('input');
                fileInput.type = 'file';
                fileInput.accept = 'image/*';
                fileInput.hidden = true;
                uploadLabel.append(fileInput);
                this.fileInput = fileInput;
            } else {
                uploadLabel.classList.add('is-disabled');
                uploadLabel.setAttribute('aria-disabled', 'true');
            }

            actions.append(uploadLabel);

            body.append(actions);

            const status = document.createElement('div');
            status.className = 'admin-media-library__status';
            status.hidden = true;
            body.append(status);
            this.status = status;

            const grid = document.createElement('div');
            grid.className = 'admin-media-library__grid';
            body.append(grid);
            this.grid = grid;

            dialog.append(body);

            const footer = document.createElement('footer');
            footer.className = 'admin-media-library__footer';

            const cancelButton = document.createElement('button');
            cancelButton.type = 'button';
            cancelButton.dataset.action = 'media-library-cancel';
            cancelButton.textContent = this.texts.cancel;
            footer.append(cancelButton);

            const chooseButton = document.createElement('button');
            chooseButton.type = 'button';
            chooseButton.dataset.action = 'media-library-choose';
            chooseButton.textContent = this.texts.choose;
            chooseButton.disabled = true;
            footer.append(chooseButton);
            this.chooseButton = chooseButton;

            dialog.append(footer);

            root.addEventListener('click', (event) => {
                const target = event.target;
                if (!(target instanceof HTMLElement)) {
                    return;
                }
                if (target.dataset.action === 'media-library-close') {
                    event.preventDefault();
                    this.close();
                    return;
                }
                if (target.dataset.action === 'media-library-refresh') {
                    event.preventDefault();
                    this.refresh();
                    return;
                }
                if (target.dataset.action === 'media-library-rename') {
                    event.preventDefault();
                    this.handleRename();
                    return;
                }
                if (target.dataset.action === 'media-library-cancel') {
                    event.preventDefault();
                    this.close();
                    return;
                }
                if (target.dataset.action === 'media-library-choose') {
                    event.preventDefault();
                    this.confirmSelection();
                    return;
                }
                const itemNode = target.closest('[data-media-index]');
                if (itemNode && this.grid && this.grid.contains(itemNode)) {
                    event.preventDefault();
                    const index = Number.parseInt(itemNode.dataset.mediaIndex || '', 10);
                    if (Number.isFinite(index)) {
                        this.setSelection(index);
                    }
                }
                if (target.dataset.action === 'media-library-upload') {
                    event.preventDefault();
                    if (!this.uploadFile) {
                        return;
                    }
                    if (this.fileInput && !this.isUploading()) {
                        this.fileInput.click();
                    }
                }
            });

            root.addEventListener('dblclick', (event) => {
                const target = event.target;
                if (!(target instanceof HTMLElement)) {
                    return;
                }
                const itemNode = target.closest('[data-media-index]');
                if (!itemNode || !this.grid || !this.grid.contains(itemNode)) {
                    return;
                }
                event.preventDefault();
                const index = Number.parseInt(itemNode.dataset.mediaIndex || '', 10);
                if (!Number.isFinite(index)) {
                    return;
                }
                this.setSelection(index);
                if (this.currentSelection) {
                    this.confirmSelection();
                }
            });

            root.addEventListener('keydown', (event) => {
                if (event.key === 'Escape') {
                    event.preventDefault();
                    this.close();
                }
            });

            this.fileInput?.addEventListener('change', (event) => {
                const input = event.target;
                if (!(input instanceof HTMLInputElement) || !input.files?.length) {
                    return;
                }
                const file = input.files[0];
                input.value = '';
                if (file) {
                    this.handleUpload(file);
                }
            });

            document.body.append(root);
            this.root = root;
        }

        setLoading(isLoading) {
            if (!this.grid) {
                return;
            }
            this.grid.classList.toggle('is-loading', Boolean(isLoading));
        }

        showStatus(message, type = 'info') {
            if (!this.status) {
                return;
            }
            if (!message) {
                this.status.hidden = true;
                this.status.textContent = '';
                this.status.classList.remove(
                    'admin-media-library__status--error',
                    'admin-media-library__status--success'
                );
                return;
            }
            this.status.hidden = false;
            this.status.textContent = message;
            this.status.classList.remove(
                'admin-media-library__status--error',
                'admin-media-library__status--success'
            );
            if (type === 'error') {
                this.status.classList.add('admin-media-library__status--error');
            } else if (type === 'success') {
                this.status.classList.add('admin-media-library__status--success');
            }
        }

        async refresh() {
            if (!this.fetchUploads || this.pendingFetch) {
                return;
            }
            this.setLoading(true);
            this.showStatus('');
            this.pendingFetch = this.fetchUploads()
                .then((uploads) => {
                    this.currentUploads = Array.isArray(uploads) ? uploads : [];
                    this.renderUploads();
                    if (!this.currentUploads.length) {
                        this.showStatus(this.texts.empty, 'info');
                    }
                })
                .catch((error) => {
                    const message =
                        error && typeof error.message === 'string'
                            ? error.message
                            : this.texts.error;
                    this.showStatus(message, 'error');
                })
                .finally(() => {
                    this.pendingFetch = null;
                    this.setLoading(false);
                });
            await this.pendingFetch;
        }

        renderUploads() {
            if (!this.grid) {
                return;
            }
            this.grid.innerHTML = '';
            this.currentSelection = null;
            this.updateChooseState();
            const uploads = this.currentUploads;
            if (!uploads.length) {
                return;
            }
            uploads.forEach((upload, index) => {
                const item = document.createElement('button');
                item.type = 'button';
                item.className = 'admin-media-library__item';
                item.dataset.mediaIndex = String(index);

                const image = document.createElement('img');
                image.className = 'admin-media-library__thumb';
                image.alt = upload.filename || 'Uploaded image';
                image.src = upload.url || '';
                item.append(image);

                const meta = document.createElement('div');
                meta.className = 'admin-media-library__meta';

                const name = document.createElement('p');
                name.className = 'admin-media-library__name';
                name.textContent = upload.filename || 'Image';
                meta.append(name);

                const details = document.createElement('p');
                details.className = 'admin-media-library__details';
                const sizeLabel = this.formatSize(upload.size);
                const dateLabel = this.formatDate(upload.mod_time || upload.modTime);
                details.textContent = [sizeLabel, dateLabel].filter(Boolean).join(' • ');
                meta.append(details);

                item.append(meta);
                this.grid.append(item);
            });
        }

        formatSize(size) {
            const value = Number(size);
            if (!Number.isFinite(value) || value <= 0) {
                return '';
            }
            const units = ['B', 'KB', 'MB', 'GB'];
            let unitIndex = 0;
            let display = value;
            while (display >= 1024 && unitIndex < units.length - 1) {
                display /= 1024;
                unitIndex += 1;
            }
            return `${display.toFixed(display >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
        }

        formatDate(value) {
            if (!value) {
                return '';
            }
            const date = new Date(value);
            if (Number.isNaN(date.getTime())) {
                return '';
            }
            try {
                return new Intl.DateTimeFormat(undefined, {
                    dateStyle: 'medium',
                    timeStyle: 'short',
                }).format(date);
            } catch (error) {
                return date.toLocaleString();
            }
        }

        setSelection(index) {
            if (!this.grid) {
                return;
            }
            const uploads = this.currentUploads || [];
            if (!uploads[index]) {
                this.currentSelection = null;
                this.updateChooseState();
                return;
            }
            this.currentSelection = uploads[index];
            this.updateChooseState();
            this.grid
                .querySelectorAll('[data-media-index]')
                .forEach((node) => node.classList.toggle('is-selected', false));
            const selectedNode = this.grid.querySelector(`[data-media-index="${index}"]`);
            if (selectedNode) {
                selectedNode.classList.add('is-selected');
                selectedNode.focus();
            }
        }

        updateChooseState() {
            const hasSelection = Boolean(this.currentSelection);
            if (this.chooseButton) {
                this.chooseButton.disabled = !hasSelection;
            }
            if (this.renameButton && this.renameUpload) {
                const shouldDisable =
                    !hasSelection || this.isUploading() || this.isRenaming();
                this.renameButton.disabled = shouldDisable;
            }
        }

        confirmSelection() {
            if (!this.currentSelection || !this.resolveClose) {
                return;
            }
            const payload = this.currentSelection;
            this.close(payload);
        }

        async handleUpload(file) {
            if (!this.uploadFile || this.pendingUpload) {
                return;
            }

            let preferredName = '';
            if (file && typeof file.name === 'string') {
                preferredName = file.name.replace(/\.[^/.]+$/, '').trim();
            }
            if (typeof window !== 'undefined' && typeof window.prompt === 'function') {
                const promptValue = window.prompt('Enter image name (optional)', preferredName);
                if (typeof promptValue === 'string') {
                    preferredName = promptValue.trim();
                }
            }

            const uploadOptions = preferredName ? { name: preferredName } : {};

            this.pendingUpload = this.uploadFile(file, uploadOptions)
                .then((result) => {
                    let url = '';
                    if (result && typeof result === 'object') {
                        url = result.url || result.URL || '';
                    }
                    if (url) {
                        this.showStatus('Image uploaded successfully.', 'success');
                    }
                    return this.refresh().then(() => url);
                })
                .then((url) => {
                    if (url && this.resolveClose) {
                        this.close({ url });
                    }
                })
                .catch((error) => {
                    const message =
                        error && typeof error.message === 'string'
                            ? error.message
                            : 'Failed to upload image.';
                    this.showStatus(message, 'error');
                })
                .finally(() => {
                    this.pendingUpload = null;
                    this.updateChooseState();
                });
            this.updateChooseState();
            await this.pendingUpload;
        }

        async handleRename() {
            if (!this.renameUpload || !this.currentSelection || this.isRenaming()) {
                return;
            }

            const current = this.currentSelection;
            const currentName =
                (current && typeof current.filename === 'string' && current.filename) ||
                (current && typeof current.Filename === 'string' && current.Filename) ||
                '';
            const defaultName = currentName.replace(/\.[^/.]+$/, '');

            let newName = defaultName;
            if (typeof window !== 'undefined' && typeof window.prompt === 'function') {
                const promptValue = window.prompt(this.texts.renamePrompt, defaultName);
                if (promptValue === null) {
                    return;
                }
                newName = promptValue.trim();
            } else {
                newName = (newName || '').trim();
            }

            if (!newName) {
                this.showStatus(this.texts.renameEmpty, 'error');
                return;
            }

            this.pendingRename = this.renameUpload(current, newName)
                .then((updated) => {
                    this.showStatus(this.texts.renameSuccess, 'success');
                    return this.refresh().then(() => updated);
                })
                .then((updated) => {
                    if (!updated) {
                        return;
                    }
                    const updatedUrl =
                        (updated && typeof updated.url === 'string' && updated.url) ||
                        (updated && typeof updated.URL === 'string' && updated.URL) ||
                        '';
                    const updatedFilename =
                        (updated &&
                            typeof updated.filename === 'string' &&
                            updated.filename) ||
                        (updated &&
                            typeof updated.Filename === 'string' &&
                            updated.Filename) ||
                        '';
                    let index = -1;
                    if (updatedUrl) {
                        index = this.currentUploads.findIndex(
                            (upload) => upload.url === updatedUrl
                        );
                    }
                    if (index < 0 && updatedFilename) {
                        index = this.currentUploads.findIndex(
                            (upload) => upload.filename === updatedFilename
                        );
                    }
                    if (index >= 0) {
                        this.setSelection(index);
                    }
                })
                .catch((error) => {
                    const message =
                        error && typeof error.message === 'string'
                            ? error.message
                            : this.texts.renameError;
                    this.showStatus(message, 'error');
                })
                .finally(() => {
                    this.pendingRename = null;
                    this.updateChooseState();
                });
            this.updateChooseState();
            await this.pendingRename;
        }

        isUploading() {
            return Boolean(this.pendingUpload);
        }

        isRenaming() {
            return Boolean(this.pendingRename);
        }

        async open(options = {}) {
            const { currentUrl = '', onSelect } = options;
            if (!this.fetchUploads) {
                throw new Error('Media library is not configured with fetchUploads');
            }
            this.ensureElements();
            if (!this.root) {
                return Promise.reject(new Error('Media library initialisation failed'));
            }
            this.root.hidden = false;
            document.body.style.overflow = 'hidden';
            this.currentSelection = null;
            this.updateChooseState();
            this.showStatus('');
            if (typeof this.onOpen === 'function') {
                this.onOpen();
            }

            const promise = new Promise((resolve) => {
                this.resolveClose = (selection) => {
                    if (selection && onSelect) {
                        if (selection.url) {
                            onSelect(selection.url);
                        } else if (typeof selection === 'string') {
                            onSelect(selection);
                        }
                    }
                    resolve(selection || null);
                };
            });

            this.refresh().then(() => {
                if (currentUrl) {
                    const index = this.currentUploads.findIndex((upload) => upload.url === currentUrl);
                    if (index >= 0) {
                        this.setSelection(index);
                    }
                }
            });

            return promise;
        }

        close(selection = null) {
            if (!this.root) {
                return;
            }
            this.root.hidden = true;
            document.body.style.overflow = '';
            if (typeof this.onClose === 'function') {
                this.onClose();
            }
            const resolver = this.resolveClose;
            this.resolveClose = null;
            if (resolver) {
                resolver(selection);
            }
        }
    }

    window.AdminMediaLibrary = {
        create: (options = {}) => new MediaLibrary(options),
    };
})();

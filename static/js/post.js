(function () {
    const copyButton = document.querySelector('[data-copy-link-button]');
    if (copyButton) {
        const label = copyButton.querySelector('[data-copy-link-label]');
        const feedback = document.querySelector('[data-copy-link-feedback]');
        const defaultLabel = label
            ? label.textContent.trim()
            : copyButton.textContent.trim();
        let labelResetTimeout = null;
        let feedbackResetTimeout = null;

        const updateLabel = (value) => {
            if (label) {
                label.textContent = value;
                return;
            }

            copyButton.textContent = value;
        };

        const resetLabel = () => {
            updateLabel(defaultLabel);
        };

        const showFeedback = (message, isError) => {
            if (!feedback) {
                return;
            }

            feedback.textContent = message;
            feedback.hidden = false;
            feedback.classList.toggle(
                'post__share-feedback--error',
                Boolean(isError)
            );

            if (feedbackResetTimeout) {
                window.clearTimeout(feedbackResetTimeout);
            }

            feedbackResetTimeout = window.setTimeout(() => {
                feedback.hidden = true;
                feedback.textContent = '';
                feedback.classList.remove('post__share-feedback--error');
            }, 4000);
        };

        const legacyCopy = (text) => {
            const textArea = document.createElement('textarea');
            textArea.value = text;
            textArea.setAttribute('readonly', '');
            textArea.style.position = 'absolute';
            textArea.style.left = '-9999px';
            textArea.style.top = '0';
            document.body.appendChild(textArea);

            const selection = document.getSelection();
            const selectedRange =
                selection && selection.rangeCount > 0
                    ? selection.getRangeAt(0)
                    : null;

            textArea.select();

            let succeeded = false;
            try {
                succeeded = document.execCommand('copy');
            } catch (error) {
                succeeded = false;
            }

            document.body.removeChild(textArea);

            if (selectedRange && selection) {
                selection.removeAllRanges();
                selection.addRange(selectedRange);
            }

            return succeeded;
        };

        const copyToClipboard = async (text) => {
            if (
                navigator.clipboard &&
                typeof navigator.clipboard.writeText === 'function'
            ) {
                try {
                    await navigator.clipboard.writeText(text);
                    return true;
                } catch (error) {
                    // Continue with fallback
                }
            }

            if (typeof document.execCommand === 'function') {
                return legacyCopy(text);
            }

            return false;
        };

        copyButton.addEventListener('click', async () => {
            const targetUrl =
                copyButton.getAttribute('data-copy-link-url') ||
                window.location.href;

            if (labelResetTimeout) {
                window.clearTimeout(labelResetTimeout);
            }

            try {
                const copied = await copyToClipboard(targetUrl);
                if (!copied) {
                    throw new Error('Copy command failed');
                }

                updateLabel('Copied!');
                showFeedback('Link copied to clipboard');
            } catch (error) {
                updateLabel('Copy failed');
                showFeedback('Unable to copy link to clipboard', true);
            } finally {
                labelResetTimeout = window.setTimeout(() => {
                    resetLabel();
                }, 2000);
            }
        });
    }

    const modal = document.querySelector('[data-post-image-modal]');
    if (!modal) {
        return;
    }

    const modalImage = modal.querySelector('[data-post-image-modal-image]');
    const closeButton = modal.querySelector('[data-post-image-close]');
    const prevButton = modal.querySelector('[data-post-image-prev]');
    const nextButton = modal.querySelector('[data-post-image-next]');
    if (!modalImage || !closeButton) {
        return;
    }

    const imageCandidates = document.querySelectorAll(
        '.post__image-wrapper img, .post__content img'
    );
    if (!imageCandidates.length) {
        return;
    }

    const images = Array.from(new Set(imageCandidates));

    let previousOverflow = '';
    let lastFocusedElement = null;
    let currentIndex = -1;

    const canNavigate = () => images.length > 1 && prevButton && nextButton;

    const updateNavigationVisibility = () => {
        if (!prevButton || !nextButton) {
            return;
        }

        const shouldShow = images.length > 1;
        prevButton.hidden = !shouldShow;
        nextButton.hidden = !shouldShow;
    };

    const showImageAtIndex = (index) => {
        if (!images.length) {
            return;
        }

        const normalizedIndex = ((index % images.length) + images.length) % images.length;
        const image = images[normalizedIndex];
        const source = image.currentSrc || image.src;

        if (!source) {
            return;
        }

        modalImage.src = source;
        modalImage.alt = image.alt || '';
        currentIndex = normalizedIndex;
    };

    const openModal = (img) => {
        lastFocusedElement = document.activeElement;
        const index = images.indexOf(img);

        showImageAtIndex(index === -1 ? 0 : index);
        updateNavigationVisibility();

        modal.removeAttribute('hidden');
        modal.classList.add('post-image-modal--active');
        previousOverflow = document.body.style.overflow;
        document.body.style.overflow = 'hidden';
        closeButton.focus();
    };

    const closeModal = () => {
        modal.classList.remove('post-image-modal--active');
        modal.setAttribute('hidden', '');
        document.body.style.overflow = previousOverflow;
        modalImage.src = '';
        modalImage.alt = '';
        currentIndex = -1;
        if (lastFocusedElement && typeof lastFocusedElement.focus === 'function') {
            lastFocusedElement.focus();
        }
    };

    const showNextImage = () => {
        if (!canNavigate() || currentIndex === -1) {
            return;
        }

        showImageAtIndex(currentIndex + 1);
    };

    const showPreviousImage = () => {
        if (!canNavigate() || currentIndex === -1) {
            return;
        }

        showImageAtIndex(currentIndex - 1);
    };

    images.forEach((img) => {
        img.style.cursor = 'zoom-in';
        img.addEventListener('click', (event) => {
            event.preventDefault();
            openModal(img);
        });
    });

    closeButton.addEventListener('click', () => {
        closeModal();
    });

    if (prevButton) {
        prevButton.addEventListener('click', (event) => {
            event.preventDefault();
            showPreviousImage();
        });
    }

    if (nextButton) {
        nextButton.addEventListener('click', (event) => {
            event.preventDefault();
            showNextImage();
        });
    }

    modal.addEventListener('click', (event) => {
        if (event.target === modal) {
            closeModal();
        }
    });

    document.addEventListener('keydown', (event) => {
        if (!modal.classList.contains('post-image-modal--active')) {
            return;
        }

        if (event.key === 'Escape') {
            closeModal();
            return;
        }

        if (event.key === 'ArrowRight') {
            event.preventDefault();
            showNextImage();
            return;
        }

        if (event.key === 'ArrowLeft') {
            event.preventDefault();
            showPreviousImage();
        }
    });
})();
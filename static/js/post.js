(function () {
    const modal = document.querySelector('[data-post-image-modal]');
    if (!modal) {
        return;
    }

    const modalImage = modal.querySelector('[data-post-image-modal-image]');
    const closeButton = modal.querySelector('[data-post-image-close]');
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

    const openModal = (img) => {
        lastFocusedElement = document.activeElement;
        const source = img.currentSrc || img.src;
        if (!source) {
            return;
        }

        modalImage.src = source;
        modalImage.alt = img.alt || '';
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
        if (lastFocusedElement && typeof lastFocusedElement.focus === 'function') {
            lastFocusedElement.focus();
        }
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

    modal.addEventListener('click', (event) => {
        if (event.target === modal) {
            closeModal();
        }
    });

    document.addEventListener('keydown', (event) => {
        if (event.key === 'Escape' && modal.classList.contains('post-image-modal--active')) {
            closeModal();
        }
    });
})();
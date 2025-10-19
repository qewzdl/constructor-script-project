(function () {
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
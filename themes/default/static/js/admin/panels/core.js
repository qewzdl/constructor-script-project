(() => {
    const layout = window.AdminLayout;
    if (!layout) {
        console.error('AdminLayout is not available.');
        return;
    }

    const { registerPanel, registerQuickAction, init } = layout;

    const createElementFromMarkup = (markup) => {
        if (typeof document === 'undefined') {
            return null;
        }
        const template = document.createElement('template');
        template.innerHTML = markup.trim();
        return template.content.firstElementChild;
    };

    const registerPanelMarkup = (definition) => {
        if (!definition || typeof definition.markup !== "string") {
            return;
        }
        const { markup, ...rest } = definition;
        registerPanel(
            Object.assign({}, rest, {
                create: () => createElementFromMarkup(markup),
            })
        );
    };

    registerQuickAction({
        id: 'quick-create-post',
        label: 'Create new post',
        navTarget: 'posts',
        panelAction: 'post-reset',
        order: 0,
        shouldRender: (context) => Boolean(context?.blogEnabled),
    });

    registerQuickAction({
        id: 'quick-create-page',
        label: 'Create new page',
        navTarget: 'pages',
        panelAction: 'page-reset',
        order: 10,
    });

    registerQuickAction({
        id: 'quick-create-forum-question',
        label: 'Start forum discussion',
        navTarget: 'forum',
        panelAction: 'forum-question-reset',
        order: 15,
        shouldRender: (context) => Boolean(context?.forumEnabled),
    });

    registerQuickAction({
        id: 'quick-update-settings',
        label: 'Update site identity',
        navTarget: 'settings',
        order: 20,
    });

    registerQuickAction({
        id: 'quick-upload-course-video',
        label: 'Upload course video',
        navTarget: 'courses',
        panelAction: 'course-video-reset',
        order: 30,
        shouldRender: (context) =>
            Boolean(context?.dataset?.endpointCoursesVideos),
    });

    registerQuickAction({
        id: 'quick-create-archive-directory',
        label: 'Create archive directory',
        navTarget: 'archive',
        panelAction: 'archive-directory-reset',
        order: 35,
        shouldRender: (context) =>
            Boolean(
                context?.archiveEnabled &&
                    context?.dataset?.endpointArchiveDirectories
            ),
    });

    registerPanelMarkup({
        id: 'metrics',
        order: 0,
        markup: String.raw`
<section
                id="admin-panel-metrics"
                class="admin-panel is-active"
                data-panel="metrics"
                data-nav-group="overview"
                data-nav-group-label="Overview"
                data-nav-group-order="0"
                data-nav-label="Metrics"
                data-nav-order="1"
                role="tabpanel"
            >
                <header class="admin-panel__header">
                    <div>
                        <h2 class="admin-panel__title">Site metrics</h2>
                        <p class="admin-panel__description">
                            Review key totals and recent activity trends to monitor site health.
                        </p>
                    </div>
                </header>
                <div class="admin-panel__body admin-panel__body--single">
                    <section class="admin-card" aria-labelledby="admin-metrics-title">
                        <div class="admin-card__header">
                            <h3 id="admin-metrics-title" class="admin-card__title">At a glance</h3>
                            <p class="admin-card__description">
                                Keep track of totals alongside how many items were created in the last day and week.
                            </p>
                        </div>
                        <div class="admin__metrics" id="admin-metrics">
                            <article class="admin__metric" data-metric="total_posts">
                                <p class="admin__metric-label">Posts</p>
                                <p class="admin__metric-value">—</p>
                            </article>
                            <article class="admin__metric" data-metric="published_posts">
                                <p class="admin__metric-label">Published posts</p>
                                <p class="admin__metric-value">—</p>
                            </article>
                            <article class="admin__metric" data-metric="total_users">
                                <p class="admin__metric-label">Users</p>
                                <p class="admin__metric-value">—</p>
                            </article>
                            <article class="admin__metric" data-metric="total_categories">
                                <p class="admin__metric-label">Categories</p>
                                <p class="admin__metric-value">—</p>
                            </article>
                            <article class="admin__metric" data-metric="total_comments">
                                <p class="admin__metric-label">Comments</p>
                                <p class="admin__metric-value">—</p>
                            </article>
                            <article class="admin__metric" data-metric="total_tags">
                                <p class="admin__metric-label">Tags</p>
                                <p class="admin__metric-value">—</p>
                            </article>
                            <article class="admin__metric" data-metric="total_views">
                                <p class="admin__metric-label">Total views</p>
                                <p class="admin__metric-value">—</p>
                            </article>
                            <article class="admin__metric" data-metric="posts_last_24_hours">
                                <p class="admin__metric-label">Posts (24 hours)</p>
                                <p class="admin__metric-value">—</p>
                            </article>
                            <article class="admin__metric" data-metric="posts_last_7_days">
                                <p class="admin__metric-label">Posts (7 days)</p>
                                <p class="admin__metric-value">—</p>
                            </article>
                            <article class="admin__metric" data-metric="comments_last_24_hours">
                                <p class="admin__metric-label">Comments (24 hours)</p>
                                <p class="admin__metric-value">—</p>
                            </article>
                            <article class="admin__metric" data-metric="comments_last_7_days">
                                <p class="admin__metric-label">Comments (7 days)</p>
                                <p class="admin__metric-value">—</p>
                            </article>
                            <article class="admin__metric" data-metric="users_last_7_days">
                                <p class="admin__metric-label">New users (7 days)</p>
                                <p class="admin__metric-value">—</p>
                            </article>
                        </div>
                    </section>
                    <section class="admin-card" aria-labelledby="admin-analytics-title">
                        <div class="admin-card__header">
                            <h3 id="admin-analytics-title" class="admin-card__title">Publishing trend</h3>
                            <p class="admin-card__description">
                                Review daily posts, comments, views, and new users over the last month to understand how activity is evolving.
                            </p>
                        </div>
                        <div
                            class="admin__chart"
                            data-role="metrics-chart"
                            role="img"
                            aria-labelledby="admin-analytics-title admin-analytics-caption"
                        >
                            <div class="admin-chart__viewport">
                                <svg class="admin-chart__svg" viewBox="0 0 600 260" preserveAspectRatio="none" focusable="false"></svg>
                            </div>
                            <div class="admin-chart__footer">
                                <ul class="admin-chart__legend" data-role="chart-legend"></ul>
                                <ol
                                    class="admin-chart__summary"
                                    id="admin-analytics-caption"
                                    data-role="chart-summary"
                                    aria-live="polite"
                                ></ol>
                            </div>
                            <p class="admin-chart__empty" data-role="chart-empty" hidden>
                                Not enough recent activity to display a trend yet.
                            </p>
                        </div>
                    </section>
                </div>
            </section>
`,
    });

    registerPanelMarkup({
        id: 'posts',
        order: 10,
        shouldRender: (context) => Boolean(context?.blogEnabled),
        markup: String.raw`
<section
                id="admin-panel-posts"
                class="admin-panel"
                data-panel="posts"
                data-nav-group="content"
                data-nav-group-label="Content"
                data-nav-group-order="1"
                data-nav-label="Posts"
                data-nav-order="1"
                role="tabpanel"
                aria-labelledby="admin-tab-posts"
                hidden
            >
            <header class="admin-panel__header">
                <div>
                    <h2 class="admin-panel__title">Blog posts</h2>
                    <p class="admin-panel__description">
                        Draft, publish and revise long-form content. Selecting an item loads it for editing.
                    </p>
                </div>
                <div class="admin-panel__actions">
                    <label class="admin-search" for="admin-posts-search">
                        <span class="admin-search__label">Search posts</span>
                        <input
                            id="admin-posts-search"
                            type="search"
                            class="admin-search__input"
                            placeholder="Search posts…"
                            autocomplete="off"
                            data-role="post-search"
                        />
                    </label>
                    <button type="button" class="admin-panel__reset" data-action="post-reset">New post</button>
                </div>
            </header>
            <div class="admin-panel__body admin-panel__body--split">
                <div class="admin-panel__list" aria-live="polite">
                    <table class="admin-table">
                        <thead>
                            <tr>
                                <th scope="col">Title</th>
                                <th scope="col">Category</th>
                                <th scope="col">Tags</th>
                                <th scope="col">Publication</th>
                                <th scope="col">Updated</th>
                            </tr>
                        </thead>
                        <tbody id="admin-posts-table">
                            <tr class="admin-table__placeholder">
                                <td colspan="5">Loading posts…</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
                <div class="admin-panel__details">
                    <form id="admin-post-form" class="admin-form" novalidate>
                        <fieldset class="admin-card admin-form__fieldset">
                            <legend class="admin-card__title admin-form__legend">Post details</legend>
                            <label class="admin-form__label">
                                Title
                                <input type="text" name="title" required class="admin-form__input" />
                            </label>
                            <label class="admin-form__label">
                                Description
                                <textarea name="description" rows="2" class="admin-form__input"></textarea>
                            </label>
                            <label class="admin-form__label">
                                Featured image URL
                                <input
                                    type="url"
                                    name="featured_img"
                                    id="admin-post-featured-image"
                                    class="admin-form__input"
                                    placeholder="https://example.com/image.jpg"
                                />
                                <div class="admin-form__upload-actions">
                                    <button
                                        type="button"
                                        class="admin-form__upload-button"
                                        data-action="open-media-library"
                                        data-media-target="#admin-post-featured-image"
                                        data-media-allowed-types="image"
                                    >
                                        Browse uploads
                                    </button>
                                </div>
                                <small class="admin-card__description admin-form__hint">
                                    Optional cover image displayed on listings and the post page.
                                </small>
                            </label>
                            <input type="hidden" name="content" />
                            <fieldset class="admin-card admin-form__fieldset admin-form__fieldset--sections">
                                <legend class="admin-card__title admin-form__legend">Structured sections</legend>
                                <p class="admin-card__description admin-form__hint">
                                    Build rich layouts by combining reusable sections. Use the controls to reorder sections and
                                    adjust their content before publishing.
                                </p>
                                <div class="section-builder" data-section-builder="post">
                                    <ol class="section-builder__list" data-role="section-list">
                                        <li class="section-builder__empty" data-role="section-empty">
                                            No sections added yet.
                                        </li>
                                    </ol>
                                    <div class="section-builder__actions">
                                        <button type="button" class="section-builder__add" data-role="section-add">
                                            Add section
                                        </button>
                                    </div>
                                </div>
                            </fieldset>
                            <label class="admin-form__label">
                                Category
                                <select name="category_id" class="admin-form__input" id="admin-post-category"></select>
                            </label>
                            <label class="admin-form__label">
                                Tags
                                <input
                                    type="text"
                                    name="tags"
                                    class="admin-form__input"
                                    id="admin-post-tags"
                                    placeholder="e.g. go, backend, database"
                                    list="admin-post-tags-list"
                                    autocomplete="off"
                                />
                                <small class="admin-card__description admin-form__hint">Separate tags with commas. Existing tags appear as suggestions.</small>
                                <datalist id="admin-post-tags-list"></datalist>
                            </label>
                            <label class="admin-form__label">
                                Publish at
                                <input
                                    type="datetime-local"
                                    name="publish_at"
                                    class="admin-form__input"
                                    step="60"
                                />
                                <small class="admin-card__description admin-form__hint">
                                    Leave empty to publish immediately. Future times schedule the post for automatic release.
                                </small>
                            </label>
                            <p class="admin-card__description admin-form__hint" data-role="post-published-at" hidden></p>
                            <div class="admin-form__actions">
                                <button
                                    type="submit"
                                    class="admin-form__submit"
                                    data-role="post-submit-publish"
                                    data-intent="publish"
                                >
                                    Save &amp; publish
                                </button>
                                <button
                                    type="submit"
                                    class="admin-form__submit admin-form__submit--secondary"
                                    data-role="post-submit-draft"
                                    data-intent="draft"
                                >
                                    Save as draft
                                </button>
                                <button type="button" class="admin-form__delete" data-role="post-delete" hidden>
                                    Delete post
                                </button>
                            </div>
                        </fieldset>
                    </form>
                    <section class="admin-panel__aside admin-post-analytics" aria-labelledby="admin-post-analytics-title">
                        <header class="admin-post-analytics__header">
                            <h3 id="admin-post-analytics-title" class="admin-post-analytics__title">Post analytics</h3>
                            <p class="admin-post-analytics__description">
                                Track how each post performs over time to spot growth or declines in engagement.
                            </p>
                        </header>
                        <div class="admin-analytics" data-role="post-analytics" hidden>
                            <ul class="admin-analytics__summary" data-role="post-analytics-summary">
                                <li class="admin-analytics__summary-item" data-metric="views">
                                    <p class="admin-analytics__summary-label">Views</p>
                                    <p class="admin-analytics__summary-value" data-role="summary-value">—</p>
                                    <p class="admin-analytics__summary-subvalue" data-role="summary-subvalue">—</p>
                                    <p class="admin-analytics__summary-delta" data-role="summary-delta" hidden></p>
                                </li>
                                <li class="admin-analytics__summary-item" data-metric="comments">
                                    <p class="admin-analytics__summary-label">Comments</p>
                                    <p class="admin-analytics__summary-value" data-role="summary-value">—</p>
                                    <p class="admin-analytics__summary-subvalue" data-role="summary-subvalue">—</p>
                                    <p class="admin-analytics__summary-delta" data-role="summary-delta" hidden></p>
                                </li>
                                <li class="admin-analytics__summary-item" data-metric="engagement">
                                    <p class="admin-analytics__summary-label">Engagement</p>
                                    <p class="admin-analytics__summary-value" data-role="summary-value">—</p>
                                    <p class="admin-analytics__summary-subvalue" data-role="summary-subvalue">—</p>
                                    <p class="admin-analytics__summary-delta" data-role="summary-delta" hidden></p>
                                </li>
                            </ul>
                            <div
                                class="admin__chart admin__chart--compact"
                                data-role="post-analytics-chart"
                                role="img"
                                aria-labelledby="admin-post-analytics-title admin-post-analytics-caption"
                            >
                                <div class="admin-chart__viewport">
                                    <svg class="admin-chart__svg" viewBox="0 0 600 260" preserveAspectRatio="none" focusable="false"></svg>
                                </div>
                                <div class="admin-chart__footer">
                                    <ul class="admin-chart__legend" data-role="chart-legend"></ul>
                                    <ol
                                        class="admin-chart__summary"
                                        id="admin-post-analytics-caption"
                                        data-role="chart-summary"
                                        aria-live="polite"
                                    ></ol>
                                </div>
                                <p class="admin-chart__empty" data-role="chart-empty" hidden>
                                    Not enough recent data to display a trend yet.
                                </p>
                            </div>
                            <dl class="admin-analytics__comparisons" data-role="post-analytics-comparisons"></dl>
                            <p class="admin-analytics__message" data-role="post-analytics-comparisons-empty" hidden>
                                Not enough data for comparisons yet.
                            </p>
                        </div>
                        <p class="admin-analytics__message" data-role="post-analytics-loading" hidden>Loading analytics…</p>
                        <p class="admin-analytics__message" data-role="post-analytics-empty">
                            Select a published post to view analytics.
                        </p>
                    </section>
                    <section class="admin-panel__aside admin-tags" aria-labelledby="admin-tags-title">
                        <header class="admin-tags__header">
                            <h3 id="admin-tags-title" class="admin-tags__title">Tags</h3>
                            <p class="admin-tags__description">
                                Remove unused tags to keep the suggestions list tidy. Deleting a tag also detaches it from
                                any posts.
                            </p>
                        </header>
                        <ul class="admin-tags__list" id="admin-tags-list">
                            <li class="admin-tags__item admin-tags__item--empty">No tags available.</li>
                        </ul>
                    </section>
                </div>
            </div>
        </section>
`,
    });

    registerPanelMarkup({
        id: 'pages',
        order: 20,
        markup: String.raw`
<section
                id="admin-panel-pages"
                class="admin-panel"
                data-panel="pages"
                data-nav-group="content"
                data-nav-group-label="Content"
                data-nav-group-order="1"
                data-nav-label="Pages"
                data-nav-order="2"
                role="tabpanel"
                aria-labelledby="admin-tab-pages"
                hidden
            >
            <header class="admin-panel__header">
                <div>
                    <h2 class="admin-panel__title">Static pages</h2>
                    <p class="admin-panel__description">
                        Create landing pages and important evergreen resources for your visitors.
                    </p>
                </div>
                <div class="admin-panel__actions">
                    <label class="admin-search" for="admin-pages-search">
                        <span class="admin-search__label">Search pages</span>
                        <input
                            id="admin-pages-search"
                            type="search"
                            class="admin-search__input"
                            placeholder="Search pages…"
                            autocomplete="off"
                            data-role="page-search"
                        />
                    </label>
                    <button type="button" class="admin-panel__reset" data-action="page-reset">New page</button>
                </div>
            </header>
            <div class="admin-panel__body admin-panel__body--split">
                <div class="admin-panel__list" aria-live="polite">
                    <table class="admin-table">
                        <thead>
                            <tr>
                                <th scope="col">Title</th>
                                <th scope="col">Path</th>
                                <th scope="col">Slug</th>
                                <th scope="col">Publication</th>
                                <th scope="col">Updated</th>
                            </tr>
                        </thead>
                        <tbody id="admin-pages-table">
                            <tr class="admin-table__placeholder">
                                <td colspan="5">Loading pages…</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
                <div class="admin-panel__details">
                    <form id="admin-page-form" class="admin-form" novalidate>
                        <fieldset class="admin-card admin-form__fieldset">
                            <legend class="admin-card__title admin-form__legend">Page details</legend>
                            <label class="admin-form__label">
                                Title
                                <input type="text" name="title" required class="admin-form__input" />
                            </label>
                            <label class="admin-form__label">
                                Public path
                                <input type="text" name="path" class="admin-form__input" placeholder="e.g. /about" />
                                <small class="admin-card__description admin-form__hint">Paths should start with a forward slash. Leave blank to use the default based on the slug.</small>
                            </label>
                            <label class="admin-form__label">
                                Custom slug
                                <input type="text" name="slug" class="admin-form__input" placeholder="optional" />
                            </label>
                            <label class="admin-form__label">
                                Description
                                <textarea name="description" rows="3" class="admin-form__input"></textarea>
                            </label>
                            <fieldset class="admin-card admin-form__fieldset admin-form__fieldset--sections">
                                <legend class="admin-card__title admin-form__legend">Structured sections</legend>
                                <p class="admin-card__description admin-form__hint">
                                    Combine headings, paragraphs, lists, and media into reusable sections. Sections render on the
                                    published page in the order shown below.
                                </p>
                                <div class="section-builder" data-section-builder="page">
                                    <ol class="section-builder__list" data-role="section-list">
                                        <li class="section-builder__empty" data-role="section-empty">
                                            No sections added yet.
                                        </li>
                                    </ol>
                                    <div class="section-builder__actions">
                                        <button type="button" class="section-builder__add" data-role="section-add">
                                            Add section
                                        </button>
                                    </div>
                                </div>
                            </fieldset>
                            <label class="admin-form__label">
                                Display order
                                <input type="number" name="order" value="0" class="admin-form__input" />
                            </label>
                            <label class="admin-form__checkbox checkbox">
                                <input type="checkbox" name="hide_header" class="checkbox__input" />
                                <span class="checkbox__label">Hide page header</span>
                            </label>
                            <label class="admin-form__label">
                                Publish at
                                <input
                                    type="datetime-local"
                                    name="publish_at"
                                    class="admin-form__input"
                                    step="60"
                                />
                                <small class="admin-card__description admin-form__hint">
                                    Leave empty to publish immediately. Future times schedule the page for automatic release.
                                </small>
                            </label>
                            <p class="admin-card__description admin-form__hint" data-role="page-published-at" hidden></p>
                            <div class="admin-form__actions">
                                <button
                                    type="submit"
                                    class="admin-form__submit"
                                    data-role="page-submit-publish"
                                    data-intent="publish"
                                >
                                    Save &amp; publish
                                </button>
                                <button
                                    type="submit"
                                    class="admin-form__submit admin-form__submit--secondary"
                                    data-role="page-submit-draft"
                                    data-intent="draft"
                                >
                                    Save as draft
                                </button>
                                <button type="button" class="admin-form__delete" data-role="page-delete" hidden>
                                    Delete page
                                </button>
                            </div>
                        </fieldset>
                    </form>
                </div>
            </div>
        </section>
`,
    });

    registerPanelMarkup({
        id: 'categories',
        order: 30,
        shouldRender: (context) => Boolean(context?.blogEnabled),
        markup: String.raw`
<section
                id="admin-panel-categories"
                class="admin-panel"
                data-panel="categories"
                data-nav-group="content"
                data-nav-group-label="Content"
                data-nav-group-order="1"
                data-nav-label="Categories"
                data-nav-order="3"
                role="tabpanel"
                aria-labelledby="admin-tab-categories"
                hidden
            >
            <header class="admin-panel__header">
                <div>
                    <h2 class="admin-panel__title">Categories</h2>
                    <p class="admin-panel__description">
                        Organise posts into themes. Changes are reflected immediately across the site.
                    </p>
                </div>
                <div class="admin-panel__actions">
                    <label class="admin-search" for="admin-categories-search">
                        <span class="admin-search__label">Search categories</span>
                        <input
                            id="admin-categories-search"
                            type="search"
                            class="admin-search__input"
                            placeholder="Search categories…"
                            autocomplete="off"
                            data-role="category-search"
                        />
                    </label>
                    <button type="button" class="admin-panel__reset" data-action="category-reset">New category</button>
                </div>
            </header>
            <div class="admin-panel__body admin-panel__body--split">
                <div class="admin-panel__list" aria-live="polite">
                    <table class="admin-table">
                        <thead>
                            <tr>
                                <th scope="col">Name</th>
                                <th scope="col">Slug</th>
                                <th scope="col">Updated</th>
                            </tr>
                        </thead>
                        <tbody id="admin-categories-table">
                            <tr class="admin-table__placeholder">
                                <td colspan="3">Loading categories…</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
                <div class="admin-panel__details">
                    <form id="admin-category-form" class="admin-form" novalidate>
                        <fieldset class="admin-card admin-form__fieldset">
                            <legend class="admin-card__title admin-form__legend">Category details</legend>
                            <label class="admin-form__label">
                                Name
                                <input type="text" name="name" required class="admin-form__input" />
                            </label>
                            <label class="admin-form__label">
                                Description
                                <textarea name="description" rows="3" class="admin-form__input"></textarea>
                            </label>
                            <div class="admin-form__actions">
                                <button type="submit" class="admin-form__submit" data-role="category-submit">
                                    Create category
                                </button>
                                <button type="button" class="admin-form__delete" data-role="category-delete" hidden>
                                    Delete category
                                </button>
                            </div>
                        </fieldset>
                    </form>
                </div>
            </div>
        </section>
`,
    });

    registerPanelMarkup({
        id: 'forum',
        order: 30,
        shouldRender: (context) => Boolean(context?.forumEnabled),
        markup: String.raw`
<section
                id="admin-panel-forum"
                class="admin-panel"
                data-panel="forum"
                data-nav-group="community"
                data-nav-group-label="Community"
                data-nav-group-order="2"
                data-nav-label="Forum"
                data-nav-order="0"
                role="tabpanel"
                aria-labelledby="admin-tab-forum"
                hidden
            >
            <header class="admin-panel__header">
                <div>
                    <h2 class="admin-panel__title">Forum discussions</h2>
                    <p class="admin-panel__description">
                        Moderate questions, update their content, and manage community answers.
                    </p>
                </div>
            </header>
            <div class="admin-panel__body admin-panel__body--stacked">
                <section
                    class="admin-card admin__section"
                    id="admin-forum-questions-section"
                    aria-labelledby="admin-forum-questions-title"
                    data-nav-child-of="forum"
                    data-nav-child-label="Questions"
                    data-nav-child-order="1"
                >
                    <header class="admin-card__header">
                        <div>
                            <h3 id="admin-forum-questions-title" class="admin-card__title">Moderate questions</h3>
                            <p class="admin-card__description">
                                Update question content, review activity, and curate community replies.
                            </p>
                        </div>
                        <div class="admin-panel__actions">
                            <label class="admin-search" for="admin-forum-search">
                                <span class="admin-search__label">Search questions</span>
                                <input
                                    id="admin-forum-search"
                                    type="search"
                                    class="admin-search__input"
                                    placeholder="Search questions…"
                                    autocomplete="off"
                                    data-role="forum-question-search"
                                />
                            </label>
                            <button type="button" class="admin-panel__reset" data-action="forum-question-reset">
                                New question
                            </button>
                        </div>
                    </header>
                    <div class="admin-card__body">
                        <div class="admin-panel__body admin-panel__body--split">
                            <div class="admin-panel__list" aria-live="polite">
                                <table class="admin-table">
                                    <thead>
                                        <tr>
                                            <th scope="col">Question</th>
                                            <th scope="col">Category</th>
                                            <th scope="col">Author</th>
                                            <th scope="col">Answers</th>
                                            <th scope="col">Rating</th>
                                            <th scope="col">Updated</th>
                                        </tr>
                                    </thead>
                                    <tbody id="admin-forum-questions-table">
                                        <tr class="admin-table__placeholder">
                                            <td colspan="6">Loading questions…</td>
                                        </tr>
                                    </tbody>
                                </table>
                            </div>
                            <div class="admin-panel__details">
                                <form id="admin-forum-question-form" class="admin-form" novalidate>
                                    <fieldset class="admin-card admin-form__fieldset">
                                        <legend class="admin-card__title admin-form__legend">Question details</legend>
                                        <p class="admin-card__description admin-form__hint" data-role="forum-question-status" hidden></p>
                                        <label class="admin-form__label">
                                            Title
                                            <input type="text" name="title" class="admin-form__input" required />
                                        </label>
                                        <label class="admin-form__label">
                                            Content
                                            <textarea name="content" rows="6" class="admin-form__input" required></textarea>
                                        </label>
                                        <label class="admin-form__label">
                                            Category
                                            <select
                                                name="category_id"
                                                class="admin-form__input"
                                                data-role="forum-question-category"
                                            >
                                                <option value="">No category</option>
                                            </select>
                                        </label>
                                        <div class="admin-form__actions">
                                            <button type="submit" class="admin-form__submit" data-role="forum-question-submit">
                                                Save question
                                            </button>
                                            <button type="button" class="admin-form__delete" data-role="forum-question-delete" hidden>
                                                Delete question
                                            </button>
                                        </div>
                                    </fieldset>
                                </form>
                                <section
                                    class="admin-card admin-panel__aside"
                                    id="admin-forum-answers"
                                    aria-labelledby="admin-forum-answers-title"
                                >
                                    <div class="admin-card__header">
                                        <h3 id="admin-forum-answers-title" class="admin-card__title">Answers</h3>
                                        <p class="admin-card__description">
                                            View responses and curate community contributions.
                                        </p>
                                    </div>
                                    <div class="admin-forum-answers" data-role="forum-answer-container">
                                        <p class="admin-card__description" data-role="forum-answer-empty">
                                            Select a question to see submitted answers.
                                        </p>
                                        <ul class="admin-forum-answers__list" data-role="forum-answer-list"></ul>
                                    </div>
                                    <form id="admin-forum-answer-form" class="admin-form" novalidate>
                                        <label class="admin-form__label">
                                            Answer content
                                            <textarea
                                                name="content"
                                                rows="4"
                                                class="admin-form__input"
                                                required
                                                data-role="forum-answer-content"
                                            ></textarea>
                                        </label>
                                        <div class="admin-form__actions">
                                            <button type="submit" class="admin-form__submit" data-role="forum-answer-submit">
                                                Save answer
                                            </button>
                                            <button
                                                type="button"
                                                class="admin-form__submit admin-form__submit--secondary"
                                                data-role="forum-answer-cancel"
                                                hidden
                                            >
                                                Cancel edit
                                            </button>
                                            <button type="button" class="admin-form__delete" data-role="forum-answer-delete" hidden>
                                                Delete answer
                                            </button>
                                        </div>
                                    </form>
                                </section>
                            </div>
                        </div>
                    </div>
                </section>
                <section
                    class="admin-card admin__section"
                    id="admin-forum-categories-section"
                    aria-labelledby="admin-forum-categories-title"
                    data-nav-child-of="forum"
                    data-nav-child-label="Categories"
                    data-nav-child-order="2"
                >
                    <header class="admin-card__header">
                        <div>
                            <h3 id="admin-forum-categories-title" class="admin-card__title">Question categories</h3>
                            <p class="admin-card__description">
                                Group related questions so members can browse focused discussions.
                            </p>
                        </div>
                        <div class="admin-panel__actions">
                            <label class="admin-search" for="admin-forum-categories-search">
                                <span class="admin-search__label">Search categories</span>
                                <input
                                    id="admin-forum-categories-search"
                                    type="search"
                                    class="admin-search__input"
                                    placeholder="Search categories…"
                                    autocomplete="off"
                                    data-role="forum-category-search"
                                />
                            </label>
                            <button type="button" class="admin-panel__reset" data-action="forum-category-reset">
                                New category
                            </button>
                        </div>
                    </header>
                    <div class="admin-card__body">
                        <div class="admin-panel__body admin-panel__body--split">
                            <div class="admin-panel__list" aria-live="polite">
                                <table class="admin-table">
                                    <thead>
                                        <tr>
                                            <th scope="col">Name</th>
                                            <th scope="col">Questions</th>
                                            <th scope="col">Updated</th>
                                        </tr>
                                    </thead>
                                    <tbody id="admin-forum-categories-table">
                                        <tr class="admin-table__placeholder">
                                            <td colspan="3">Loading categories…</td>
                                        </tr>
                                    </tbody>
                                </table>
                            </div>
                            <div class="admin-panel__details">
                                <form id="admin-forum-category-form" class="admin-form" novalidate>
                                    <fieldset class="admin-card admin-form__fieldset">
                                        <legend class="admin-card__title admin-form__legend">Category details</legend>
                                        <p class="admin-card__description admin-form__hint" data-role="forum-category-status" hidden></p>
                                        <label class="admin-form__label">
                                            Name
                                            <input type="text" name="name" required class="admin-form__input" />
                                        </label>
                                        <div class="admin-form__actions">
                                            <button type="submit" class="admin-form__submit" data-role="forum-category-submit">
                                                Create category
                                            </button>
                                            <button type="button" class="admin-form__delete" data-role="forum-category-delete" hidden>
                                                Delete category
                                            </button>
                                        </div>
                                    </fieldset>
                                </form>
                            </div>
                        </div>
                    </div>
                </section>
            </div>
        </section>
`,
    });

    registerPanelMarkup({
        id: 'users',
        order: 40,
        markup: String.raw`
<section
                id="admin-panel-users"
                class="admin-panel"
                data-panel="users"
                data-nav-group="community"
                data-nav-group-label="Community"
                data-nav-group-order="2"
                data-nav-label="Users"
                data-nav-order="1"
                role="tabpanel"
                aria-labelledby="admin-tab-users"
                hidden
            >
            <header class="admin-panel__header">
                <div>
                    <h2 class="admin-panel__title">User accounts</h2>
                    <p class="admin-panel__description">
                        Review who has access to the community and adjust roles or account status as needed.
                    </p>
                </div>
                <div class="admin-panel__actions">
                    <label class="admin-search" for="admin-users-search">
                        <span class="admin-search__label">Search users</span>
                        <input
                            id="admin-users-search"
                            type="search"
                            class="admin-search__input"
                            placeholder="Search by name or email…"
                            autocomplete="off"
                            data-role="user-search"
                        />
                    </label>
                    <button type="button" class="admin-panel__reset" data-action="user-reset">
                        Clear selection
                    </button>
                </div>
            </header>
            <div class="admin-panel__body admin-panel__body--split">
                <div class="admin-panel__list" aria-live="polite">
                    <table class="admin-table">
                        <thead>
                            <tr>
                                <th scope="col">Username</th>
                                <th scope="col">Email</th>
                                <th scope="col">Role</th>
                                <th scope="col">Status</th>
                                <th scope="col">Joined</th>
                            </tr>
                        </thead>
                        <tbody id="admin-users-table">
                            <tr class="admin-table__placeholder">
                                <td colspan="5">Loading users…</td>
                            </tr>
                        </tbody>
                    </table>
                </div>
                <div class="admin-panel__details">
                    <form id="admin-user-form" class="admin-form" novalidate>
                        <fieldset class="admin-card admin-form__fieldset">
                            <legend class="admin-card__title admin-form__legend">Account overview</legend>
                            <p class="admin-card__description admin-form__hint" data-role="user-hint">
                                Select a user from the list to view their account details.
                            </p>
                            <label class="admin-form__label">
                                Username
                                <input type="text" name="username" class="admin-form__input" readonly />
                            </label>
                            <label class="admin-form__label">
                                Email
                                <input type="email" name="email" class="admin-form__input" readonly />
                            </label>
                            <label class="admin-form__label">
                                Role
                                <select name="role" class="admin-form__input" data-role="user-role" required>
                                    <option value="admin">Administrator</option>
                                    <option value="user">User</option>
                                </select>
                            </label>
                            <label class="admin-form__label">
                                Status
                                <select name="status" class="admin-form__input" data-role="user-status" required>
                                    <option value="active">Active</option>
                                    <option value="inactive">Inactive</option>
                                    <option value="suspended">Suspended</option>
                                    <option value="pending">Pending</option>
                                </select>
                            </label>
                            <div class="admin-form__actions">
                                <button type="submit" class="admin-form__submit" data-role="user-submit" disabled>
                                    Update user
                                </button>
                                <button type="button" class="admin-form__delete" data-role="user-delete" hidden>
                                    Delete user
                                </button>
                            </div>
                        </fieldset>
                    </form>
                </div>
            </div>
        </section>
`,
    });

    registerPanelMarkup({
        id: 'comments',
        order: 50,
        shouldRender: (context) => Boolean(context?.blogEnabled),
        markup: String.raw`
<section
                id="admin-panel-comments"
                class="admin-panel"
                data-panel="comments"
                data-nav-group="community"
                data-nav-group-label="Community"
                data-nav-group-order="2"
                data-nav-label="Comments"
                data-nav-order="2"
                role="tabpanel"
                aria-labelledby="admin-tab-comments"
                hidden
            >
            <header class="admin-panel__header">
                <div>
                    <h2 class="admin-panel__title">Recent comments</h2>
                    <p class="admin-panel__description">
                        Approve new feedback or remove unwanted submissions. Older comments remain accessible via the API.
                    </p>
                </div>
            </header>
            <div class="admin-panel__body admin-panel__body--single">
                <ul id="admin-comments-list" class="admin-comment-list" aria-live="polite">
                    <li class="admin-comment-list__item admin-comment-list__item--empty">Loading comments…</li>
                </ul>
            </div>
        </section>
`,
    });

    registerPanelMarkup({
        id: 'settings',
        order: 60,
        markup: String.raw`
<section
                id="admin-panel-settings"
                class="admin-panel"
                data-panel="settings"
                data-nav-group="configuration"
                data-nav-group-label="Configuration"
                data-nav-group-order="3"
                data-nav-label="Site settings"
                data-nav-order="1"
                role="tabpanel"
                aria-labelledby="admin-tab-settings"
                hidden
            >
            <header class="admin-panel__header">
                <div>
                    <h2 class="admin-panel__title">Site identity</h2>
                    <p class="admin-panel__description">
                        Update how your brand appears across templates, including the name, tagline, URL, and imagery.
                    </p>
                </div>
            </header>
            <div class="admin-panel__body">
                <form id="admin-settings-form" class="admin-form" novalidate>
                    <fieldset class="admin-card admin-form__fieldset">
                        <legend class="admin-card__title admin-form__legend">Site details</legend>
                        <label class="admin-form__label">
                            Site name
                            <input type="text" name="name" required class="admin-form__input" />
                        </label>
                        <label class="admin-form__label">
                            Tagline
                            <textarea name="description" rows="2" class="admin-form__input"></textarea>
                        </label>
                        <label class="admin-form__label">
                            Site URL
                            <input type="url" name="url" required class="admin-form__input" placeholder="https://example.com" />
                            <small class="admin-card__description admin-form__hint">Used to generate canonical links and social sharing metadata.</small>
                        </label>
                        <label class="admin-form__label">
                            Unused tag retention (hours)
                            <input
                                type="number"
                                name="unused_tag_retention_hours"
                                min="1"
                                step="1"
                                required
                                class="admin-form__input"
                            />
                            <small class="admin-card__description admin-form__hint">
                                Tags that are not attached to any posts will be removed after the specified number of hours.
                            </small>
                        </label>
                        <div class="admin-form__actions">
                            <button type="submit" class="admin-form__submit" data-role="settings-submit">
                                Save changes
                            </button>
                        </div>
                    </fieldset>
                    <fieldset class="admin-card admin-form__fieldset">
                        <legend class="admin-card__title admin-form__legend">Brand assets</legend>
                        <label class="admin-form__label">
                            Favicon
                            <input
                                type="url"
                                name="favicon"
                                id="admin-settings-favicon"
                                class="admin-form__input"
                                placeholder="/favicon.ico"
                            />
                            <div class="admin-form__upload-actions">
                                <input
                                    type="file"
                                    accept=".ico,.png,.jpg,.jpeg,.gif,.webp"
                                    data-role="favicon-file"
                                    hidden
                                />
                                <button type="button" class="admin-form__upload-button" data-role="favicon-upload">
                                    Upload favicon
                                </button>
                                <button
                                    type="button"
                                    class="admin-form__upload-button"
                                    data-action="open-media-library"
                                    data-media-target="#admin-settings-favicon"
                                    data-media-allowed-types="image"
                                >
                                    Browse uploads
                                </button>
                            </div>
                            <small class="admin-card__description admin-form__hint">
                                Accepts relative or absolute URLs. Leave blank to use the default icon. Uploads support ICO,
                                PNG, JPG, GIF or WEBP files.
                            </small>
                            <div class="admin-form__favicon-preview" data-role="favicon-preview" hidden>
                                <span class="admin-form__hint admin-form__hint--label">Current favicon:</span>
                                <img
                                    src=""
                                    alt="Current favicon"
                                    class="admin-form__favicon-image"
                                    data-role="favicon-preview-image"
                                />
                            </div>
                        </label>
                        <label class="admin-form__label">
                            Logo
                            <input
                                type="url"
                                name="logo"
                                id="admin-settings-logo"
                                class="admin-form__input"
                                placeholder="/static/icons/logo.svg"
                            />
                            <div class="admin-form__upload-actions">
                                <input
                                    type="file"
                                    accept="image/*"
                                    data-role="logo-file"
                                    hidden
                                />
                                <button type="button" class="admin-form__upload-button" data-role="logo-upload">
                                    Upload logo
                                </button>
                                <button
                                    type="button"
                                    class="admin-form__upload-button"
                                    data-action="open-media-library"
                                    data-media-target="#admin-settings-logo"
                                    data-media-allowed-types="image"
                                >
                                    Browse uploads
                                </button>
                            </div>
                            <small class="admin-card__description admin-form__hint">
                                Accepts relative or absolute URLs. Leave blank to use the default logo. Uploads support common image formats such as SVG, PNG, JPG and WEBP.
                            </small>
                            <div class="admin-form__logo-preview" data-role="logo-preview" hidden>
                                <span class="admin-form__hint admin-form__hint--label">Current logo:</span>
                                <img
                                    src=""
                                    alt="Current logo"
                                    class="admin-form__logo-image"
                                    data-role="logo-preview-image"
                                />
                            </div>
                        </label>
                    </fieldset>
                </form>
            </div>
            </section>
`,
    });

    registerPanelMarkup({
        id: 'payments',
        order: 65,
        markup: String.raw`
<section
        id="admin-panel-payments"
        class="admin-panel"
        data-panel="payments"
        data-nav-group="configuration"
        data-nav-group-label="Configuration"
        data-nav-group-order="3"
        data-nav-label="Payments"
        data-nav-order="1.2"
        role="tabpanel"
        aria-labelledby="admin-tab-payments"
        hidden
    >
        <header class="admin-panel__header">
            <div>
                <h2 class="admin-panel__title">Payments</h2>
                <p class="admin-panel__description">
                    Manage Stripe credentials and checkout preferences for selling course packages.
                </p>
            </div>
        </header>
        <div class="admin-panel__body admin-panel__body--single">
            <form id="admin-payments-form" class="admin-form" novalidate>
                <fieldset class="admin-card admin-form__fieldset">
                    <legend class="admin-card__title admin-form__legend">Stripe configuration</legend>
                    <p class="admin-card__description admin-form__hint">
                        Provide Stripe credentials and checkout preferences for selling course packages.
                    </p>
                    <label class="admin-form__label">
                        Stripe publishable key
                        <input type="text" name="stripe_publishable_key" class="admin-form__input" autocomplete="off" />
                    </label>
                    <label class="admin-form__label">
                        Stripe secret key
                        <input type="password" name="stripe_secret_key" class="admin-form__input" autocomplete="off" />
                        <small class="admin-card__description admin-form__hint">
                            Use a restricted key with permissions to create Checkout Sessions.
                        </small>
                    </label>
                    <label class="admin-form__label">
                        Stripe webhook secret
                        <input type="password" name="stripe_webhook_secret" class="admin-form__input" autocomplete="off" />
                        <small class="admin-card__description admin-form__hint">
                            Optional. Required if Stripe webhooks are configured for post-payment processing.
                        </small>
                    </label>
                    <label class="admin-form__label">
                        Checkout success URL
                        <input
                            type="url"
                            name="course_checkout_success_url"
                            class="admin-form__input"
                            placeholder="https://example.com/courses/checkout/success"
                        />
                    </label>
                    <label class="admin-form__label">
                        Checkout cancel URL
                        <input
                            type="url"
                            name="course_checkout_cancel_url"
                            class="admin-form__input"
                            placeholder="https://example.com/courses/checkout/cancel"
                        />
                    </label>
                    <label class="admin-form__label">
                        Checkout currency
                        <input
                            type="text"
                            name="course_checkout_currency"
                            class="admin-form__input"
                            maxlength="3"
                            placeholder="usd"
                            autocomplete="off"
                        />
                        <small class="admin-card__description admin-form__hint">
                            Specify a three-letter ISO currency code (for example, <code>usd</code> or <code>eur</code>).
                        </small>
                    </label>
                    <div class="admin-form__actions">
                        <button type="submit" class="admin-form__submit" data-role="payments-submit">
                            Save payment settings
                        </button>
                    </div>
                </fieldset>
            </form>
        </div>
    </section>
`,
    });

    registerPanelMarkup({
        id: 'courses',
        order: 30,
        shouldRender: (context) =>
            Boolean(
                context?.dataset?.endpointCoursesVideos ||
                    context?.dataset?.endpointCoursesTopics ||
                    context?.dataset?.endpointCoursesPackages
            ),
        markup: String.raw`
<section
        id="admin-panel-courses"
        class="admin-panel"
        data-panel="courses"
        data-nav-group="content"
        data-nav-group-label="Content"
        data-nav-group-order="1"
        data-nav-label="Courses"
        data-nav-order="3"
        role="tabpanel"
        aria-labelledby="admin-tab-courses"
        hidden
    >
        <header class="admin-panel__header">
            <div>
                <h2 class="admin-panel__title">Course library</h2>
                <p class="admin-panel__description">
                    Manage course videos, topics, and packages. Upload lessons, assemble topics, and build purchasable bundles.
                </p>
            </div>
        </header>
        <div class="admin-panel__body admin-panel__body--stacked">
            <section
                class="admin-card admin-courses admin__section"
                aria-labelledby="admin-course-videos-title"
                id="admin-course-videos-section"
                data-nav-child-of="courses"
                data-nav-child-label="Videos"
                data-nav-child-order="1"
            >
                <header class="admin-card__header admin-courses__header">
                    <div>
                        <h3 id="admin-course-videos-title" class="admin-card__title">Course videos</h3>
                        <p class="admin-card__description">
                            Upload lesson videos. Duration is detected automatically during upload.
                        </p>
                    </div>
                    <div class="admin-panel__actions admin-courses__actions">
                        <label class="admin-search" for="admin-course-videos-search">
                            <span class="admin-search__label">Search videos</span>
                            <input
                                id="admin-course-videos-search"
                                type="search"
                                class="admin-search__input"
                                placeholder="Search videos…"
                                autocomplete="off"
                                data-role="course-video-search"
                            />
                        </label>
                        <button type="button" class="admin-panel__reset" data-action="course-video-reset">
                            New video
                        </button>
                    </div>
                </header>
                <div class="admin-card__body">
                    <div class="admin-panel__list admin-courses__list" aria-live="polite">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th scope="col">Title</th>
                                    <th scope="col">Duration</th>
                                    <th scope="col">Updated</th>
                                </tr>
                            </thead>
                            <tbody id="admin-course-videos-table">
                                <tr class="admin-table__placeholder">
                                    <td colspan="3">Loading videos…</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                    <div class="admin-panel__details">
                        <form id="admin-course-video-form" class="admin-form admin-courses__form" novalidate>
                            <label class="admin-form__label">
                                Title
                                <input type="text" name="title" required class="admin-form__input" />
                            </label>
                            <label class="admin-form__label">
                                Description
                                <textarea name="description" rows="3" class="admin-form__input"></textarea>
                            </label>
                            <fieldset class="admin-card admin-form__fieldset admin-form__fieldset--sections">
                                <legend class="admin-card__title admin-form__legend">Lesson content</legend>
                                <p class="admin-card__description admin-form__hint">
                                    Use the visual builder to add structured content that appears before the video. Combine
                                    text, images, and other elements to introduce each lesson.
                                </p>
                                <div class="section-builder" data-section-builder="course-video">
                                    <ol class="section-builder__list" data-role="section-list">
                                        <li class="section-builder__empty" data-role="section-empty">
                                            No sections added yet.
                                        </li>
                                    </ol>
                                    <div class="section-builder__actions">
                                        <button type="button" class="section-builder__add" data-role="section-add">
                                            Add section
                                        </button>
                                    </div>
                                </div>
                            </fieldset>
                            <fieldset class="admin-form__fieldset admin-courses__fieldset">
                                <legend class="admin-form__legend">Downloadable files</legend>
                                <p class="admin-card__description admin-form__hint">
                                    Attach supporting documents that learners can download alongside this lesson.
                                </p>
                                <div class="admin-courses__picker">
                                    <button
                                        type="button"
                                        class="admin-navigation__button"
                                        data-role="course-video-attachment-add"
                                    >
                                        Attach file
                                    </button>
                                </div>
                                <ul
                                    class="admin-courses__selection-list"
                                    data-role="course-video-attachment-list"
                                    aria-live="polite"
                                >
                                    <li
                                        class="admin-courses__selection-empty"
                                        data-role="course-video-attachment-empty"
                                    >
                                        No files attached yet.
                                    </li>
                                </ul>
                            </fieldset>
                            <fieldset
                                class="admin-form__fieldset admin-courses__fieldset"
                                data-role="course-video-subtitle-fieldset"
                            >
                                <legend class="admin-form__legend">Subtitles</legend>
                                <p class="admin-card__description admin-form__hint">
                                    Review and edit the subtitle track attached to this lesson. Updates are saved as a WebVTT
                                    download for learners.
                                </p>
                                <p
                                    class="admin-card__description admin-form__hint"
                                    data-role="course-video-subtitle-empty"
                                >
                                    Subtitles will appear here after they are generated or uploaded.
                                </p>
                                <div
                                    class="admin-form__group admin-form__group--stacked"
                                    data-role="course-video-subtitle-editor"
                                    hidden
                                >
                                    <label class="admin-form__label">
                                        Subtitle title
                                        <input
                                            type="text"
                                            class="admin-form__input"
                                            data-role="course-video-subtitle-title"
                                            placeholder="Subtitle download name"
                                        />
                                    </label>
                                    <label class="admin-form__label">
                                        Subtitle content
                                        <textarea
                                            class="admin-form__input admin-form__input--monospace"
                                            rows="10"
                                            data-role="course-video-subtitle-content"
                                            spellcheck="false"
                                        ></textarea>
                                    </label>
                                    <div class="admin-form__actions admin-form__actions--inline">
                                        <button
                                            type="button"
                                            class="admin-navigation__button"
                                            data-role="course-video-subtitle-save"
                                        >
                                            Save subtitles
                                        </button>
                                        <button
                                            type="button"
                                            class="admin-navigation__button admin-navigation__button--secondary"
                                            data-role="course-video-subtitle-reset"
                                        >
                                            Discard changes
                                        </button>
                                    </div>
                                    <p
                                        class="admin-card__description admin-form__hint"
                                        data-role="course-video-subtitle-status"
                                        hidden
                                    ></p>
                                </div>
                            </fieldset>
                            <div class="admin-form__group" data-role="course-video-upload-group">
                                <label class="admin-form__label">
                                    Video file
                                    <input
                                        type="file"
                                        name="video"
                                        accept="video/mp4,video/m4v,video/quicktime"
                                        class="admin-form__input"
                                        required
                                    />
                                    <small class="admin-card__description admin-form__hint">
                                        Accepted formats: MP4, M4V, MOV. Duration is stored automatically.
                                    </small>
                                </label>
                            </div>
                            <p
                                class="admin-card__description admin-form__hint"
                                data-role="course-video-upload-hint"
                                hidden
                            >
                                Uploads are only available when creating a new video. Delete and recreate the entry to replace the file.
                            </p>
                            <p class="admin-courses__meta">
                                Duration:
                                <span data-role="course-video-duration">—</span>
                            </p>
                            <figure
                                class="admin-courses__preview"
                                data-role="course-video-preview-wrapper"
                                hidden
                            >
                                <video
                                    class="admin-courses__preview-media"
                                    controls
                                    preload="metadata"
                                    data-role="course-video-preview"
                                >
                                    Your browser does not support HTML5 video.
                                </video>
                                <figcaption class="admin-card__description admin-courses__preview-caption">
                                    Video preview
                                </figcaption>
                            </figure>
                            <div class="admin-form__actions">
                                <button type="submit" class="admin-form__submit" data-role="course-video-submit">
                                    Upload video
                                </button>
                                <button type="button" class="admin-form__delete" data-role="course-video-delete" hidden>
                                    Delete video
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            </section>
            <section
                class="admin-card admin-courses admin__section"
                aria-labelledby="admin-course-topics-title"
                id="admin-course-topics-section"
                data-nav-child-of="courses"
                data-nav-child-label="Topics"
                data-nav-child-order="2"
            >
                <header class="admin-card__header admin-courses__header">
                    <div>
                        <h3 id="admin-course-topics-title" class="admin-card__title">Course topics</h3>
                        <p class="admin-card__description">
                            Combine lessons and assessments into structured learning paths. Arrange the steps exactly how learners
                            should progress.
                        </p>
                    </div>
                    <div class="admin-panel__actions admin-courses__actions">
                        <label class="admin-search" for="admin-course-topics-search">
                            <span class="admin-search__label">Search topics</span>
                            <input
                                id="admin-course-topics-search"
                                type="search"
                                class="admin-search__input"
                                placeholder="Search topics…"
                                autocomplete="off"
                                data-role="course-topic-search"
                            />
                        </label>
                        <button type="button" class="admin-panel__reset" data-action="course-topic-reset">
                            New topic
                        </button>
                    </div>
                </header>
                <div class="admin-card__body">
                    <div class="admin-panel__list admin-courses__list" aria-live="polite">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th scope="col">Title</th>
                                    <th scope="col">Steps</th>
                                    <th scope="col">Updated</th>
                                </tr>
                            </thead>
                            <tbody id="admin-course-topics-table">
                                <tr class="admin-table__placeholder">
                                    <td colspan="3">Loading topics…</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                    <div class="admin-panel__details">
                        <form id="admin-course-topic-form" class="admin-form admin-courses__form" novalidate>
                            <label class="admin-form__label">
                                Title
                                <input type="text" name="title" required class="admin-form__input" />
                            </label>
                            <label class="admin-form__label">
                                Slug
                                <input
                                    type="text"
                                    name="slug"
                                    required
                                    pattern="[a-z0-9-]+"
                                    class="admin-form__input"
                                    placeholder="e.g. introduction"
                                />
                                <small class="admin-card__description admin-form__hint">
                                    Used in links and references. Lowercase letters, numbers, and hyphens only.
                                </small>
                            </label>
                            <label class="admin-form__label">
                                Summary
                                <textarea name="summary" rows="2" class="admin-form__input"></textarea>
                                <small class="admin-card__description admin-form__hint">
                                    Provide a concise overview shown in course outlines.
                                </small>
                            </label>
                            <label class="admin-form__label">
                                Detailed description
                                <textarea name="description" rows="3" class="admin-form__input"></textarea>
                            </label>
                            <label class="admin-form__label">
                                Meta title
                                <input
                                    type="text"
                                    name="meta_title"
                                    class="admin-form__input"
                                    placeholder="Optional SEO title"
                                />
                            </label>
                            <label class="admin-form__label">
                                Meta description
                                <textarea name="meta_description" rows="2" class="admin-form__input"></textarea>
                            </label>
                            <fieldset class="admin-form__fieldset admin-courses__fieldset">
                                <legend class="admin-form__legend">Topic steps</legend>
                                <p class="admin-card__description admin-form__hint">
                                    Mix videos and tests to build the sequence students will follow. Reorder steps to match the
                                    desired flow.
                                </p>
                                <div class="admin-courses__picker">
                                    <label class="admin-form__label">
                                        Add step
                                        <select class="admin-form__input" data-role="course-topic-step-select">
                                            <option value="">Select a step…</option>
                                        </select>
                                    </label>
                                    <button type="button" class="admin-navigation__button" data-role="course-topic-step-add">
                                        Add step
                                    </button>
                                </div>
                                <ul
                                    class="admin-courses__selection-list"
                                    data-role="course-topic-step-list"
                                    aria-live="polite"
                                >
                                    <li class="admin-courses__selection-empty" data-role="course-topic-step-empty">
                                        No steps added yet.
                                    </li>
                                </ul>
                            </fieldset>
                            <div class="admin-form__actions">
                                <button type="submit" class="admin-form__submit" data-role="course-topic-submit">
                                    Save topic
                                </button>
                                <button type="button" class="admin-form__delete" data-role="course-topic-delete" hidden>
                                    Delete topic
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            </section>
            <section
                class="admin-card admin-courses admin__section"
                aria-labelledby="admin-course-tests-title"
                id="admin-course-tests-section"
                data-nav-child-of="courses"
                data-nav-child-label="Tests"
                data-nav-child-order="3"
            >
                <header class="admin-card__header admin-courses__header">
                    <div>
                        <h3 id="admin-course-tests-title" class="admin-card__title">Course tests</h3>
                        <p class="admin-card__description">
                            Build quizzes with detailed explanations. Scores are stored automatically and feedback appears after
                            submission.
                        </p>
                    </div>
                    <div class="admin-panel__actions admin-courses__actions">
                        <label class="admin-search" for="admin-course-tests-search">
                            <span class="admin-search__label">Search tests</span>
                            <input
                                id="admin-course-tests-search"
                                type="search"
                                class="admin-search__input"
                                placeholder="Search tests…"
                                autocomplete="off"
                                data-role="course-test-search"
                            />
                        </label>
                        <button type="button" class="admin-panel__reset" data-action="course-test-reset">
                            New test
                        </button>
                    </div>
                </header>
                <div class="admin-card__body">
                    <div class="admin-panel__list admin-courses__list" aria-live="polite">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th scope="col">Title</th>
                                    <th scope="col">Questions</th>
                                    <th scope="col">Updated</th>
                                </tr>
                            </thead>
                            <tbody id="admin-course-tests-table">
                                <tr class="admin-table__placeholder">
                                    <td colspan="3">Loading tests…</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                    <div class="admin-panel__details">
                        <form id="admin-course-test-form" class="admin-form admin-courses__form" novalidate>
                            <label class="admin-form__label">
                                Title
                                <input type="text" name="title" required class="admin-form__input" />
                            </label>
                            <label class="admin-form__label">
                                Description <span class="admin-form__hint">Optional</span>
                                <textarea name="description" rows="3" class="admin-form__input"></textarea>
                            </label>
                            <fieldset class="admin-form__fieldset admin-courses__fieldset">
                                <legend class="admin-form__legend">Questions</legend>
                                <p class="admin-card__description admin-form__hint">
                                    Use text answers, single-choice, or multiple-choice questions. Learners see explanations after
                                    completing the test.
                                </p>
                                <div class="admin-course-test-questions__actions">
                                    <button
                                        type="button"
                                        class="admin-navigation__button"
                                        data-role="course-test-question-add"
                                    >
                                        Add question
                                    </button>
                                </div>
                                <div
                                    class="admin-course-test-questions"
                                    data-role="course-test-question-list"
                                    aria-live="polite"
                                ></div>
                                <p class="admin-course-test-questions__empty" data-role="course-test-question-empty">
                                    No questions added yet.
                                </p>
                            </fieldset>
                            <div class="admin-form__actions">
                                <button type="submit" class="admin-form__submit" data-role="course-test-submit">
                                    Save test
                                </button>
                                <button type="button" class="admin-form__delete" data-role="course-test-delete" hidden>
                                    Delete test
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            </section>
            <section
                class="admin-card admin-courses admin__section"
                aria-labelledby="admin-course-packages-title"
                id="admin-course-packages-section"
                data-nav-child-of="courses"
                data-nav-child-label="Packages"
                data-nav-child-order="4"
            >
                <header class="admin-card__header admin-courses__header">
                    <div>
                        <h3 id="admin-course-packages-title" class="admin-card__title">Course packages</h3>
                        <p class="admin-card__description">
                            Bundle topics together, define pricing, and add a promotional image for storefront use.
                        </p>
                    </div>
                    <div class="admin-panel__actions admin-courses__actions">
                        <label class="admin-search" for="admin-course-packages-search">
                            <span class="admin-search__label">Search packages</span>
                            <input
                                id="admin-course-packages-search"
                                type="search"
                                class="admin-search__input"
                                placeholder="Search packages…"
                                autocomplete="off"
                                data-role="course-package-search"
                            />
                        </label>
                        <button type="button" class="admin-panel__reset" data-action="course-package-reset">
                            New package
                        </button>
                    </div>
                </header>
                <div class="admin-card__body">
                    <div class="admin-panel__list admin-courses__list" aria-live="polite">
                        <table class="admin-table">
                            <thead>
                                <tr>
                                    <th scope="col">Title</th>
                                    <th scope="col">Price</th>
                                    <th scope="col">Topics</th>
                                    <th scope="col">Updated</th>
                                </tr>
                            </thead>
                            <tbody id="admin-course-packages-table">
                                <tr class="admin-table__placeholder">
                                    <td colspan="4">Loading packages…</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                    <div class="admin-panel__details">
                        <form id="admin-course-package-form" class="admin-form admin-courses__form" novalidate>
                            <label class="admin-form__label">
                                Title
                                <input type="text" name="title" required class="admin-form__input" />
                            </label>
                            <label class="admin-form__label">
                                Slug
                                <input
                                    type="text"
                                    name="slug"
                                    required
                                    pattern="[a-z0-9-]+"
                                    class="admin-form__input"
                                    placeholder="e.g. go-basics"
                                />
                                <small class="admin-card__description admin-form__hint">
                                    Used in URLs. Lowercase letters, numbers, and hyphens only.
                                </small>
                            </label>
                            <label class="admin-form__label">
                                Summary
                                <textarea name="summary" rows="2" class="admin-form__input"></textarea>
                                <small class="admin-card__description admin-form__hint">
                                    Shown in listings and share previews. Keep it concise.
                                </small>
                            </label>
                            <label class="admin-form__label">
                                Detailed description
                                <textarea name="description" rows="3" class="admin-form__input"></textarea>
                            </label>
                            <label class="admin-form__label">
                                Meta title
                                <input
                                    type="text"
                                    name="meta_title"
                                    class="admin-form__input"
                                    placeholder="Optional SEO title"
                                />
                                <small class="admin-card__description admin-form__hint">
                                    Overrides the default page title shown in browser tabs and search results.
                                </small>
                            </label>
                            <label class="admin-form__label">
                                Meta description
                                <textarea name="meta_description" rows="2" class="admin-form__input"></textarea>
                                <small class="admin-card__description admin-form__hint">
                                    Optional search and social description. Defaults to the summary when blank.
                                </small>
                            </label>
                            <label class="admin-form__label">
                                Price
                                <input
                                    type="number"
                                    name="price"
                                    step="0.01"
                                    min="0"
                                    class="admin-form__input"
                                    placeholder="e.g. 1990.00"
                                    required
                                />
                                <small class="admin-card__description admin-form__hint">
                                    Enter the amount in your storefront currency. Values are stored with cent precision.
                                </small>
                            </label>
                            <label class="admin-form__label">
                                Image URL
                                <input
                                    type="text"
                                    name="image_url"
                                    id="admin-course-package-image"
                                    class="admin-form__input"
                                    placeholder="/uploads/course-preview.jpg"
                                />
                                <div class="admin-form__upload-actions">
                                    <button
                                        type="button"
                                        class="admin-form__upload-button"
                                        data-action="open-media-library"
                                        data-media-target="#admin-course-package-image"
                                        data-media-allowed-types="image"
                                    >
                                        Browse uploads
                                    </button>
                                </div>
                                <small class="admin-card__description admin-form__hint">
                                    Provide a preview image to feature this package in marketing pages.
                                </small>
                            </label>
                            <fieldset class="admin-form__fieldset admin-courses__fieldset">
                                <legend class="admin-form__legend">Included topics</legend>
                                <p class="admin-card__description admin-form__hint">
                                    Choose which topics belong to the package and adjust their order.
                                </p>
                                <div class="admin-courses__picker">
                                    <label class="admin-form__label">
                                        Add topic
                                        <select class="admin-form__input" data-role="course-package-topic-select">
                                            <option value="">Select a topic…</option>
                                        </select>
                                    </label>
                                    <button type="button" class="admin-navigation__button" data-role="course-package-topic-add">
                                        Add topic
                                    </button>
                                </div>
                                <ul
                                    class="admin-courses__selection-list"
                                    data-role="course-package-topic-list"
                                    aria-live="polite"
                                >
                                    <li class="admin-courses__selection-empty" data-role="course-package-topic-empty">
                                        No topics selected yet.
                                    </li>
                                </ul>
                            </fieldset>
                            <fieldset class="admin-form__fieldset admin-courses__fieldset admin-courses__grant">
                                <legend class="admin-form__legend">Grant access</legend>
                                <p class="admin-card__description admin-form__hint">
                                    Issue this package to a user immediately. Search by name or email and optionally set an
                                    expiration date.
                                </p>
                                <div class="admin-courses__grant-search">
                                    <label class="admin-form__label">
                                        Search user
                                        <input
                                            type="search"
                                            class="admin-form__input"
                                            data-role="course-package-grant-search"
                                            placeholder="Start typing a name or email…"
                                            autocomplete="off"
                                        />
                                    </label>
                                    <p
                                        class="admin-card__description admin-form__hint"
                                        data-role="course-package-grant-results-status"
                                    >
                                        Start typing to find a user.
                                    </p>
                                    <ul
                                        class="admin-courses__grant-results"
                                        data-role="course-package-grant-results"
                                        hidden
                                    ></ul>
                                </div>
                                <div
                                    class="admin-courses__grant-selection"
                                    data-role="course-package-grant-selection"
                                    hidden
                                >
                                    <p class="admin-card__description admin-form__hint">
                                        Granting access to
                                        <span data-role="course-package-grant-selection-label"></span>.
                                    </p>
                                    <button
                                        type="button"
                                        class="admin-navigation__button admin-navigation__button--ghost"
                                        data-role="course-package-grant-clear-user"
                                    >
                                        Change user
                                    </button>
                                </div>
                                <input type="hidden" data-role="course-package-grant-user" />
                                <label class="admin-form__label">
                                    Expiration
                                    <input
                                        type="datetime-local"
                                        class="admin-form__input"
                                        data-role="course-package-grant-expires"
                                    />
                                    <small class="admin-card__description admin-form__hint">
                                        Leave blank to keep the current expiration.
                                    </small>
                                </label>
                                <label class="admin-form__checkbox checkbox">
                                    <input type="checkbox" data-role="course-package-grant-clear-expiration" />
                                    <span class="checkbox__label">Clear any existing expiration</span>
                                </label>
                                <div class="admin-form__actions">
                                    <button
                                        type="button"
                                        class="admin-form__submit admin-form__submit--secondary"
                                        data-role="course-package-grant-submit"
                                    >
                                        Grant package to user
                                    </button>
                                </div>
                                <p
                                    class="admin-card__description admin-form__hint"
                                    data-role="course-package-grant-status"
                                    hidden
                                ></p>
                            </fieldset>
                            <div class="admin-form__actions">
                                <button type="submit" class="admin-form__submit" data-role="course-package-submit">
                                    Save package
                                </button>
                                <button type="button" class="admin-form__delete" data-role="course-package-delete" hidden>
                                    Delete package
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            </section>
        </div>
    </section>
`,
    });

    registerPanelMarkup({
        id: 'archive',
        order: 40,
        shouldRender: (context) =>
            Boolean(
                context?.archiveEnabled &&
                    context?.dataset?.endpointArchiveDirectories
            ),
        markup: String.raw`
<section
        id="admin-panel-archive"
        class="admin-panel"
        data-panel="archive"
        data-nav-group="content"
        data-nav-group-label="Content"
        data-nav-group-order="1"
        data-nav-label="Archive"
        data-nav-order="4"
        role="tabpanel"
        aria-labelledby="admin-tab-archive"
        hidden
    >
        <header class="admin-panel__header">
            <div>
                <h2 class="admin-panel__title">Resource archive</h2>
                <p class="admin-panel__description">
                    Organize directories and files that appear in the public archive. Update structure and download links.
                </p>
            </div>
        </header>
        <div class="admin-panel__body admin-panel__body--stacked">
            <section
                class="admin-card admin__section"
                id="admin-archive-directories-section"
                aria-labelledby="admin-archive-directories-title"
                data-nav-child-of="archive"
                data-nav-child-label="Directories"
                data-nav-child-order="1"
            >
                <header class="admin-card__header">
                    <div>
                        <h3 id="admin-archive-directories-title" class="admin-card__title">Directories</h3>
                        <p class="admin-card__description">
                            Build the directory tree and control visibility for each section of the archive.
                        </p>
                    </div>
                    <div class="admin-panel__actions">
                        <label class="admin-search" for="admin-archive-directories-search">
                            <span class="admin-search__label">Search directories</span>
                            <input
                                id="admin-archive-directories-search"
                                type="search"
                                class="admin-search__input"
                                placeholder="Search directories…"
                                autocomplete="off"
                                data-role="archive-directory-search"
                            />
                        </label>
                        <button type="button" class="admin-panel__reset" data-action="archive-directory-reset">
                            New directory
                        </button>
                    </div>
                </header>
                <div class="admin-card__body">
                    <div class="admin-panel__body admin-panel__body--split">
                        <div class="admin-panel__list" aria-live="polite">
                            <table class="admin-table">
                                <thead>
                                    <tr>
                                        <th scope="col">Directory</th>
                                        <th scope="col">Path</th>
                                        <th scope="col">Status</th>
                                        <th scope="col">Updated</th>
                                    </tr>
                                </thead>
                                <tbody id="admin-archive-directories-table">
                                    <tr class="admin-table__placeholder">
                                        <td colspan="4">Create a directory to start building your archive structure.</td>
                                    </tr>
                                </tbody>
                            </table>
                        </div>
                        <div class="admin-panel__details">
                            <form id="admin-archive-directory-form" class="admin-form" novalidate>
                                <fieldset class="admin-card admin-form__fieldset">
                                    <legend class="admin-card__title admin-form__legend">Directory details</legend>
                                    <p class="admin-card__description admin-form__hint" data-role="archive-directory-status" hidden></p>
                                    <label class="admin-form__label">
                                        Name
                                        <input type="text" name="name" required class="admin-form__input" />
                                    </label>
                                    <label class="admin-form__label">
                                        Slug
                                        <input type="text" name="slug" class="admin-form__input" placeholder="Auto-generated if left blank" />
                                    </label>
                                    <label class="admin-form__label">
                                        Parent directory
                                        <select name="parent_id" class="admin-form__input" data-role="archive-directory-parent"></select>
                                    </label>
                                    <label class="admin-form__label">
                                        Display order
                                        <input type="number" name="order" class="admin-form__input" inputmode="numeric" />
                                    </label>
                                    <label class="admin-form__checkbox checkbox">
                                        <input type="checkbox" name="published" value="true" checked />
                                        <span class="checkbox__label">Published</span>
                                    </label>
                                    <div class="admin-form__actions">
                                        <button type="submit" class="admin-form__submit" data-role="archive-directory-submit">
                                            Save directory
                                        </button>
                                        <button type="button" class="admin-form__delete" data-role="archive-directory-delete" hidden>
                                            Delete directory
                                        </button>
                                    </div>
                                </fieldset>
                            </form>
                        </div>
                    </div>
                </div>
            </section>
            <section
                class="admin-card admin__section"
                id="admin-archive-files-section"
                aria-labelledby="admin-archive-files-title"
                data-nav-child-of="archive"
                data-nav-child-label="Files"
                data-nav-child-order="2"
            >
                <header class="admin-card__header">
                    <div>
                        <h3 id="admin-archive-files-title" class="admin-card__title">Files</h3>
                        <p class="admin-card__description">
                            Upload or link files within the selected directory. Provide friendly names and optional preview URLs.
                        </p>
                    </div>
                    <div class="admin-panel__actions">
                        <label class="admin-search" for="admin-archive-files-search">
                            <span class="admin-search__label">Search files</span>
                            <input
                                id="admin-archive-files-search"
                                type="search"
                                class="admin-search__input"
                                placeholder="Search files…"
                                autocomplete="off"
                                data-role="archive-file-search"
                            />
                        </label>
                        <button type="button" class="admin-panel__reset" data-action="archive-file-reset">
                            New file
                        </button>
                    </div>
                </header>
                <div class="admin-card__body">
                    <div class="admin-panel__body admin-panel__body--split">
                        <div class="admin-panel__list" aria-live="polite">
                            <table class="admin-table">
                                <thead>
                                    <tr>
                                        <th scope="col">File</th>
                                        <th scope="col">Type</th>
                                        <th scope="col">Updated</th>
                                    </tr>
                                </thead>
                                <tbody id="admin-archive-files-table">
                                    <tr class="admin-table__placeholder">
                                        <td colspan="3">Select a directory to view files.</td>
                                    </tr>
                                </tbody>
                            </table>
                        </div>
                        <div class="admin-panel__details">
                            <form id="admin-archive-file-form" class="admin-form" novalidate>
                                <fieldset class="admin-card admin-form__fieldset">
                                    <legend class="admin-card__title admin-form__legend">File details</legend>
                                    <p class="admin-card__description admin-form__hint" data-role="archive-file-status" hidden></p>
                                    <label class="admin-form__label">
                                        Name
                                        <input type="text" name="name" required class="admin-form__input" />
                                    </label>
                                    <label class="admin-form__label">
                                        Slug
                                        <input type="text" name="slug" class="admin-form__input" placeholder="Auto-generated if left blank" />
                                    </label>
                                    <label class="admin-form__label">
                                        Directory
                                        <select name="directory_id" class="admin-form__input" data-role="archive-file-directory" required></select>
                                    </label>
                                    <label class="admin-form__label">
                                        Description
                                        <textarea name="description" rows="3" class="admin-form__input"></textarea>
                                    </label>
                                    <label class="admin-form__label">
                                        File URL
                                        <input type="url" name="file_url" required class="admin-form__input" placeholder="https://example.com/file.pdf" />
                                    </label>
                                    <label class="admin-form__label">
                                        Preview URL
                                        <input type="url" name="preview_url" class="admin-form__input" placeholder="Optional preview link" />
                                    </label>
                                    <label class="admin-form__label">
                                        MIME type
                                        <input type="text" name="mime_type" class="admin-form__input" placeholder="application/pdf" />
                                    </label>
                                    <label class="admin-form__label">
                                        File type
                                        <input type="text" name="file_type" class="admin-form__input" placeholder="Document, image, video…" />
                                    </label>
                                    <label class="admin-form__label">
                                        File size (bytes)
                                        <input type="number" name="file_size" class="admin-form__input" inputmode="numeric" min="0" />
                                    </label>
                                    <label class="admin-form__label">
                                        Display order
                                        <input type="number" name="order" class="admin-form__input" inputmode="numeric" />
                                    </label>
                                    <label class="admin-form__checkbox checkbox">
                                        <input type="checkbox" name="published" value="true" checked />
                                        <span class="checkbox__label">Published</span>
                                    </label>
                                    <div class="admin-form__actions">
                                        <button type="submit" class="admin-form__submit" data-role="archive-file-submit">
                                            Save file
                                        </button>
                                        <button type="button" class="admin-form__delete" data-role="archive-file-delete" hidden>
                                            Delete file
                                        </button>
                                    </div>
                                </fieldset>
                            </form>
                        </div>
                    </div>
                </div>
            </section>
        </div>
    </section>
`,
    });

    registerPanelMarkup({
        id: 'fonts',
        order: 70,
        markup: String.raw`
<section
                id="admin-panel-fonts"
                class="admin-panel"
                data-panel="fonts"
                data-nav-group="configuration"
                data-nav-group-label="Configuration"
                data-nav-group-order="3"
                data-nav-label="Fonts"
                data-nav-order="1.1"
                role="tabpanel"
                aria-labelledby="admin-tab-fonts"
                hidden
            >
                <header class="admin-panel__header">
                    <div>
                        <h2 class="admin-panel__title">Typography</h2>
                        <p class="admin-panel__description">
                            Connect external font providers, manage loading order, and control which families are active across the site.
                        </p>
                    </div>
                </header>
                <div class="admin-panel__body admin-panel__body--single">
                    <section class="admin-card admin-fonts" aria-labelledby="admin-fonts-title">
                        <div class="admin-card__header">
                            <h3 id="admin-fonts-title" class="admin-card__title">Font library</h3>
                            <p class="admin-card__description">
                                Enabled fonts are injected into the <code>&lt;head&gt;</code> of every page before custom stylesheets.
                            </p>
                        </div>
                        <div class="admin-card__body">
                            <p class="admin-fonts__hint">
                                Drag the handles or use the move buttons to fine-tune the order in which fonts load.
                            </p>
                            <ul class="admin-fonts__list" data-role="font-list" aria-live="polite"></ul>
                            <p class="admin-fonts__empty" data-role="font-empty" hidden>
                                No fonts configured yet. Add your first font using the form on the right.
                            </p>
                        </div>
                    </section>

                    <section class="admin-card admin-fonts__form-card" aria-labelledby="admin-fonts-form-title">
                        <div class="admin-card__header">
                            <h3 id="admin-fonts-form-title" class="admin-card__title">Add or edit font</h3>
                            <p class="admin-card__description">
                                Provide a descriptive name and paste the embed code exactly as supplied by your font provider.
                            </p>
                        </div>
                        <form id="admin-font-form" class="admin-form admin-fonts__form" novalidate>
                            <input type="hidden" name="id" />
                            <label class="admin-form__label">
                                Display name
                                <input type="text" name="name" class="admin-form__input" required />
                            </label>
                            <label class="admin-form__label">
                                Embed code
                                <textarea
                                    name="snippet"
                                    rows="3"
                                    class="admin-form__input"
                                    required
                                    placeholder="&lt;link href=\&quot;https://fonts.googleapis.com/...\&quot; rel=\&quot;stylesheet\&quot;&gt;"
                                ></textarea>
                                <small class="admin-card__description admin-form__hint">
                                    Paste the <code>&lt;link&gt;</code>, <code>&lt;style&gt;</code>, or loader snippet exactly as instructed by the provider.
                                </small>
                            </label>
                            <label class="admin-form__label">
                                Preconnect domains
                                <input
                                    type="text"
                                    name="preconnects"
                                    class="admin-form__input"
                                    placeholder="https://fonts.googleapis.com, https://fonts.gstatic.com"
                                />
                                <small class="admin-card__description admin-form__hint">
                                    Separate multiple URLs with commas. Preconnect hints speed up font delivery for repeat visitors.
                                </small>
                            </label>
                            <label class="admin-form__checkbox">
                                <input type="checkbox" name="enabled" checked />
                                <span class="checkbox__label">Enable font</span>
                            </label>
                            <label class="admin-form__label">
                                Notes <span class="admin-form__hint">Optional</span>
                                <textarea
                                    name="notes"
                                    rows="2"
                                    class="admin-form__input"
                                    placeholder="Usage guidance, pairing suggestions, or fallbacks."
                                ></textarea>
                            </label>
                            <div class="admin-form__actions admin-fonts__form-actions">
                                <button type="submit" class="admin-form__submit" data-role="font-submit">
                                    Save font
                                </button>
                                <button
                                    type="button"
                                    class="admin-form__cancel"
                                    data-role="font-cancel"
                                    hidden
                                >
                                    Cancel
                                </button>
                            </div>
                        </form>
                    </section>
                </div>
            </section>
`,
    });

    registerPanelMarkup({
        id: 'languages',
        order: 75,
        shouldRender: (context) => Boolean(context?.languageFeatureEnabled),
        markup: String.raw`
<section
                    id="admin-panel-languages"
                    class="admin-panel"
                    data-panel="languages"
                    data-nav-group="configuration"
                    data-nav-group-label="Configuration"
                    data-nav-group-order="3"
                    data-nav-label="Languages"
                    data-nav-order="1.25"
                    role="tabpanel"
                    aria-labelledby="admin-tab-languages"
                    hidden
                >
                    <header class="admin-panel__header">
                        <div>
                            <h2 class="admin-panel__title">Languages</h2>
                            <p class="admin-panel__description">
                                Configure the default interface language and manage which locales are available to visitors.
                            </p>
                        </div>
                    </header>
                    <div class="admin-panel__body admin-panel__body--single">
                        <form id="admin-language-form" class="admin-form" novalidate>
                            <fieldset class="admin-card admin-form__fieldset">
                                <legend class="admin-card__title admin-form__legend">Language configuration</legend>
                                <label class="admin-form__label" for="admin-default-language">
                                    Default language
                                    <input
                                        id="admin-default-language"
                                        type="text"
                                        name="default_language"
                                        required
                                        class="admin-form__input"
                                        placeholder="en"
                                        pattern="^[a-z]{2,8}(-[A-Za-z]{2,3})?$"
                                        list="admin-language-suggestions"
                                        data-role="language-default"
                                    />
                                    <datalist id="admin-language-suggestions" data-role="language-suggestions"></datalist>
                                    <small class="admin-card__description admin-form__hint">
                                        Use a BCP&nbsp;47 language tag such as <code>en</code> or <code>en-GB</code>.
                                    </small>
                                </label>
                                <div
                                    class="admin-form__label"
                                    role="group"
                                    aria-labelledby="admin-supported-languages-label"
                                >
                                    <span id="admin-supported-languages-label">Supported languages</span>
                                    <div class="admin-languages" data-role="language-manager">
                                        <p class="admin-languages__empty" data-role="language-empty" hidden>
                                            Only the default language is currently available.
                                        </p>
                                        <ul class="admin-languages__list" data-role="language-list"></ul>
                                        <div class="admin-languages__controls">
                                            <input
                                                type="text"
                                                class="admin-form__input admin-languages__input"
                                                placeholder="Add language code (e.g. fr or de-AT)"
                                                data-role="language-input"
                                                pattern="^[a-z]{2,8}(-[A-Za-z]{2,3})?$"
                                                aria-label="Add supported language"
                                                list="admin-language-suggestions"
                                            />
                                            <button type="button" class="admin-languages__add-button" data-role="language-add">
                                                Add language
                                            </button>
                                        </div>
                                        <small class="admin-card__description admin-form__hint">
                                            The default language is always supported. Add additional language codes to offer
                                            translations.
                                        </small>
                                    </div>
                                    <input type="hidden" name="supported_languages" data-role="language-hidden" />
                                </div>
                                <div class="admin-form__actions">
                                    <button type="submit" class="admin-form__submit" data-role="language-submit">
                                        Save languages
                                    </button>
                                </div>
                            </fieldset>
                        </form>
                    </div>
                </section>
`,
    });

    registerPanelMarkup({
        id: 'homepage',
        order: 80,
        markup: String.raw`
<section
                id="admin-panel-homepage"
                class="admin-panel"
                data-panel="homepage"
                data-nav-group="configuration"
                data-nav-group-label="Configuration"
                data-nav-group-order="3"
                data-nav-label="Homepage"
                data-nav-order="1.5"
                role="tabpanel"
                aria-labelledby="admin-tab-homepage"
                hidden
            >
                <header class="admin-panel__header">
                    <div>
                        <h2 class="admin-panel__title">Homepage</h2>
                        <p class="admin-panel__description">
                            Choose which published page visitors see when they land on the root URL.
                        </p>
                    </div>
                </header>
                <div class="admin-panel__body">
                    <form id="admin-homepage-form" class="admin-form" novalidate>
                        <fieldset class="admin-card admin-form__fieldset">
                            <legend class="admin-card__title admin-form__legend">Homepage selection</legend>
                            <p class="admin-card__description admin-form__hint">
                                Select a published page to use as the homepage. Leave blank to continue using the page
                                assigned to the "/" path.
                            </p>
                            <label class="admin-form__label">
                                Homepage page
                                <select class="admin-form__input" name="page_id" data-role="homepage-select">
                                    <option value="">Use page assigned to "/" path</option>
                                </select>
                            </label>
                            <p class="admin-card__description admin-form__hint" data-role="homepage-status"></p>
                            <div class="admin-form__actions">
                                <button type="submit" class="admin-form__submit" data-role="homepage-submit">
                                    Save homepage
                                </button>
                            </div>
                        </fieldset>
                    </form>
                    <section class="admin-card" aria-labelledby="admin-homepage-options-title">
                        <div class="admin-card__header">
                            <h3 id="admin-homepage-options-title" class="admin-card__title">Available pages</h3>
                            <p class="admin-card__description">
                                Review each page’s status before selecting it as the homepage.
                            </p>
                        </div>
                        <div class="admin-homepage__options" data-role="homepage-options">
                            <p class="admin-homepage__empty" data-role="homepage-empty">
                                No pages available yet. Create and publish a page to select it as the homepage.
                            </p>
                        </div>
                    </section>
                </div>
            </section>
`,
    });

    registerPanelMarkup({
        id: 'backups',
        order: 90,
        markup: String.raw`
<section
                id="admin-panel-backups"
                class="admin-panel"
                data-panel="backups"
                data-nav-group="configuration"
                data-nav-group-label="Configuration"
                data-nav-group-order="3"
                data-nav-label="Backups"
                data-nav-order="2"
                role="tabpanel"
                aria-labelledby="admin-tab-backups"
                hidden
            >
                <header class="admin-panel__header">
                    <div>
                        <h2 class="admin-panel__title">Site backups</h2>
                        <p class="admin-panel__description">
                            Create a full export of database records and managed uploads or restore a previous snapshot when needed.
                        </p>
                    </div>
                </header>
                <div class="admin-panel__body admin-panel__body--stacked">
                    <section
                        id="admin-backup-download-section"
                        class="admin-card admin__section"
                        aria-labelledby="admin-backup-download-title"
                        data-nav-child-of="backups"
                        data-nav-child-label="Download"
                        data-nav-child-order="1"
                    >
                        <div class="admin-card__header">
                            <h3 id="admin-backup-download-title" class="admin-card__title">Download full backup</h3>
                            <p class="admin-card__description">
                                Generates a ZIP archive containing all site data and uploaded media so you can store it securely.
                            </p>
                        </div>
                        <p class="admin-card__description admin-form__hint">
                            Backups include users, content, configuration, and files managed through the media uploader.
                        </p>
                        <div class="admin-form__actions">
                            <button type="button" class="admin-form__submit" data-role="backup-download">
                                Download backup
                            </button>
                        </div>
                        <p class="admin-card__description admin-form__hint" data-role="backup-summary" hidden></p>
                    </section>
                    <section
                        id="admin-backup-auto-section"
                        class="admin-card admin__section"
                        aria-labelledby="admin-backup-auto-title"
                        data-nav-child-of="backups"
                        data-nav-child-label="Automatic"
                        data-nav-child-order="2"
                    >
                        <div class="admin-card__header">
                            <h3 id="admin-backup-auto-title" class="admin-card__title">Automatic backups</h3>
                            <p class="admin-card__description">
                                Schedule recurring backups that are stored on the server for additional protection.
                            </p>
                        </div>
                        <form id="admin-backup-settings-form" class="admin-form" novalidate>
                            <fieldset class="admin-card admin-form__fieldset">
                                <legend class="admin-card__title admin-form__legend">Schedule</legend>
                                <label class="admin-form__checkbox checkbox" for="backup-auto-enabled">
                                    <input
                                        type="checkbox"
                                        id="backup-auto-enabled"
                                        name="auto_enabled"
                                        class="checkbox__input"
                                    />
                                    <span class="checkbox__label">Enable automatic backups</span>
                                </label>
                                <label class="admin-form__label" for="backup-auto-interval">
                                    Backup interval (hours)
                                    <input
                                        type="number"
                                        id="backup-auto-interval"
                                        name="interval_hours"
                                        class="admin-form__input"
                                        min="1"
                                        max="168"
                                        value="24"
                                    />
                                </label>
                                <small class="admin-card__description admin-form__hint">
                                    Choose how often to run automatic backups. The maximum interval is 168 hours (7 days).
                                </small>
                                <p class="admin-card__description admin-form__hint" data-role="backup-settings-status" hidden></p>
                                <div class="admin-form__actions">
                                    <button type="submit" class="admin-form__submit" data-role="backup-settings-submit">
                                        Save backup settings
                                    </button>
                                </div>
                            </fieldset>
                        </form>
                    </section>
                    <section
                        id="admin-backup-restore-section"
                        class="admin-card admin__section"
                        aria-labelledby="admin-backup-restore-title"
                        data-nav-child-of="backups"
                        data-nav-child-label="Restore"
                        data-nav-child-order="3"
                    >
                        <div class="admin-card__header">
                            <h3 id="admin-backup-restore-title" class="admin-card__title">Restore from backup</h3>
                            <p class="admin-card__description">
                                Upload a previously generated archive to replace all content and media with the snapshot it contains.
                            </p>
                        </div>
                        <form id="admin-backup-import-form" class="admin-form" novalidate>
                            <fieldset class="admin-card admin-form__fieldset">
                                <legend class="admin-card__title admin-form__legend">Backup archive</legend>
                                <label class="admin-form__label">
                                    Archive file
                                    <input
                                        type="file"
                                        name="backup_file"
                                        accept="application/zip,.zip"
                                        required
                                        class="admin-form__input"
                                    />
                                    <small class="admin-card__description admin-form__hint">
                                        Restoring a backup overwrites the current database and uploads. Ensure you have a recent export before continuing.
                                    </small>
                                </label>
                                <div class="admin-form__actions">
                                    <button type="submit" class="admin-form__submit" data-role="backup-upload-submit">
                                        Restore backup
                                    </button>
                                </div>
                            </fieldset>
                        </form>
                    </section>
                </div>
            </section>
`,
    });

    registerPanelMarkup({
        id: 'advertising',
        order: 100,
        markup: String.raw`
<section
                id="admin-panel-advertising"
                class="admin-panel"
                data-panel="advertising"
                data-nav-group="configuration"
                data-nav-group-label="Configuration"
                data-nav-group-order="3"
                data-nav-label="Monetization"
                data-nav-order="3"
                role="tabpanel"
                aria-labelledby="admin-tab-advertising"
                hidden
            >
                <header class="admin-panel__header">
                    <div>
                        <h2 class="admin-panel__title">Advertising</h2>
                        <p class="admin-panel__description">
                            Configure Google AdSense placements and control how advertisements appear across the site.
                        </p>
                    </div>
                </header>
                <div class="admin-panel__body">
                    <form id="admin-ads-form" class="admin-form" novalidate>
                        <fieldset class="admin-card admin-form__fieldset">
                            <legend class="admin-card__title admin-form__legend">General settings</legend>
                            <label class="admin-form__checkbox checkbox">
                                <input
                                    type="checkbox"
                                    name="enabled"
                                    data-role="ads-enabled"
                                    class="checkbox__input"
                                />
                                <span class="checkbox__label">Enable advertising</span>
                            </label>
                            <label class="admin-form__label">
                                Provider
                                <select name="provider" class="admin-form__input" data-role="ads-provider"></select>
                            </label>
                            <small class="admin-card__description admin-form__hint">
                                Select the advertising provider to manage. New providers can be added without changing templates.
                            </small>
                        </fieldset>
                        <fieldset class="admin-card admin-form__fieldset" data-role="ads-provider-fields" data-provider="google_ads">
                            <legend class="admin-card__title admin-form__legend">Google AdSense</legend>
                            <label class="admin-form__label">
                                Publisher ID
                                <input
                                    type="text"
                                    name="google_ads.publisher_id"
                                    class="admin-form__input"
                                    data-role="ads-google-publisher"
                                    placeholder="ca-pub-0000000000000000"
                                    autocomplete="off"
                                />
                            </label>
                            <small class="admin-card__description admin-form__hint">
                                Use the publisher ID from your AdSense account (format: ca-pub-XXXXXXXXXXXXXXXX).
                            </small>
                            <label class="admin-form__checkbox checkbox">
                                <input
                                    type="checkbox"
                                    name="google_ads.auto_ads"
                                    data-role="ads-google-auto"
                                    class="checkbox__input"
                                />
                                <span class="checkbox__label">Enable Auto Ads</span>
                            </label>
                            <p class="admin-card__description admin-form__hint">
                                Auto Ads allow Google to automatically place ads across your pages. You can still define manual slots.
                            </p>
                            <div class="admin-ads__slots" data-role="ads-slots" aria-live="polite"></div>
                            <button type="button" class="admin-form__link-button" data-role="ads-slot-add">
                                Add placement
                            </button>
                            <small class="admin-card__description admin-form__hint">
                                Each placement maps an AdSense ad unit to a specific location such as the header, sidebar, or footer.
                            </small>
                        </fieldset>
                        <div class="admin-form__actions">
                            <button type="submit" class="admin-form__submit" data-role="ads-submit">
                                Save changes
                            </button>
                        </div>
                    </form>
                </div>
            </section>
`,
    });

    registerPanelMarkup({
        id: 'plugins',
        order: 110,
        markup: String.raw`
<section
                id="admin-panel-plugins"
                class="admin-panel"
                data-panel="plugins"
                data-nav-group="configuration"
                data-nav-group-label="Configuration"
                data-nav-group-order="3"
                data-nav-label="Plugins"
                data-nav-order="1"
                role="tabpanel"
                aria-labelledby="admin-tab-plugins"
                hidden
            >
                <header class="admin-panel__header">
                    <div>
                        <h2 class="admin-panel__title">Plugins</h2>
                        <p class="admin-panel__description">
                            Install, activate, and deactivate plugins to extend the site's capabilities without modifying the core codebase.
                        </p>
                    </div>
                </header>
                <div class="admin-panel__body admin-panel__body--stacked">
                    <section
                        id="admin-plugins-installed-section"
                        class="admin-card admin-plugins admin__section"
                        aria-labelledby="admin-plugins-title"
                        data-nav-child-of="plugins"
                        data-nav-child-label="Installed"
                        data-nav-child-order="1"
                    >
                        <header class="admin-card__header admin-plugins__header">
                            <h3 id="admin-plugins-title" class="admin-card__title admin-form__legend">Installed plugins</h3>
                            <p class="admin-card__description admin-form__hint">
                                Manage installed plugins. Activate a plugin to enable its features or deactivate it to disable functionality while keeping the files available.
                            </p>
                        </header>
                        <div class="admin-card__body admin-plugins__body">
                            <ul class="admin-plugins__list" data-role="plugin-list" aria-live="polite">
                                <li class="admin-plugins__empty" data-role="plugin-empty">No plugins installed yet.</li>
                            </ul>
                        </div>
                    </section>
                    <section
                        id="admin-plugins-install-section"
                        class="admin-card admin-plugins__install admin__section"
                        aria-labelledby="admin-plugin-install-title"
                        data-nav-child-of="plugins"
                        data-nav-child-label="Install"
                        data-nav-child-order="2"
                    >
                        <header class="admin-card__header admin-plugins__install-header">
                            <h3 id="admin-plugin-install-title" class="admin-card__title admin-form__legend">Install plugin</h3>
                            <p class="admin-card__description admin-form__hint">
                                Upload a ZIP archive that contains the plugin files and a <code>plugin.json</code> manifest describing the plugin metadata.
                            </p>
                        </header>
                        <div class="admin-card__body">
                            <form class="admin-form admin-plugins__form" data-role="plugin-install-form" enctype="multipart/form-data" novalidate>
                                <label class="admin-form__label">
                                    Plugin archive (.zip)
                                    <input
                                        type="file"
                                        name="file"
                                        required
                                        accept=".zip"
                                        class="admin-form__input"
                                        data-role="plugin-upload-input"
                                    />
                                    <small class="admin-card__description admin-form__hint">
                                        Install plugins packaged as ZIP files. The plugin will be extracted to the server's plugins directory.
                                    </small>
                                </label>
                                <div class="admin-form__actions">
                                    <button type="submit" class="admin-form__submit" data-role="plugin-install-button">
                                        Install plugin
                                    </button>
                                </div>
                            </form>
                        </div>
                    </section>
                </div>
            </section>
`,
    });

    registerPanelMarkup({
        id: 'themes',
        order: 120,
        markup: String.raw`
<section
                id="admin-panel-themes"
                class="admin-panel"
                data-panel="themes"
                data-nav-group="configuration"
                data-nav-group-label="Configuration"
                data-nav-group-order="3"
                data-nav-label="Themes"
                data-nav-order="2"
                role="tabpanel"
                aria-labelledby="admin-tab-themes"
                hidden
            >
                <header class="admin-panel__header">
                    <div>
                        <h2 class="admin-panel__title">Themes</h2>
                        <p class="admin-panel__description">
                            Switch the active theme to apply a different global layout, stylesheet, and default content structure.
                        </p>
                    </div>
                </header>
                <div class="admin-panel__body admin-panel__body--single">
                    <section class="admin-card admin-theme" aria-labelledby="admin-themes-title">
                        <header class="admin-card__header admin-theme__header">
                            <h3 id="admin-themes-title" class="admin-card__title admin-form__legend">Available themes</h3>
                            <p class="admin-card__description admin-form__hint">
                                Select a theme to change the site's overall appearance. Activate a theme to apply its templates and assets immediately.
                            </p>
                        </header>
                        <div class="admin-card__body admin-theme__body">
                            <ul class="admin-theme__list" data-role="theme-list" aria-live="polite">
                                <li class="admin-theme__empty" data-role="theme-empty">No themes available yet.</li>
                            </ul>
                        </div>
                    </section>
                </div>
            </section>
`,
    });

    registerPanelMarkup({
        id: 'social',
        order: 130,
        markup: String.raw`
<section
                id="admin-panel-social"
                class="admin-panel"
                data-panel="social"
                data-nav-group="configuration"
                data-nav-group-label="Configuration"
                data-nav-group-order="3"
                data-nav-label="Social profiles"
                data-nav-order="3"
                role="tabpanel"
                aria-labelledby="admin-tab-social"
                hidden
            >
                <header class="admin-panel__header">
                    <div>
                        <h2 class="admin-panel__title">Social profiles</h2>
                        <p class="admin-panel__description">
                            Manage the social networks displayed in the site footer. Add a name, destination URL, and an optional icon for each profile.
                        </p>
                    </div>
                </header>
                <div class="admin-panel__body">
                    <section class="admin-card admin-social" aria-labelledby="admin-social-title">
                        <header class="admin-card__header admin-social__header">
                            <h3 id="admin-social-title" class="admin-card__title admin-form__legend">Social profiles</h3>
                            <p class="admin-card__description admin-form__hint">
                                Manage the social networks displayed in the site footer. Add a name, destination URL, and an optional icon for each profile.
                            </p>
                        </header>
                        <div class="admin-card__body admin-social__content">
                            <ul class="admin-social__list" data-role="social-list" aria-live="polite">
                                <li class="admin-social__empty" data-role="social-empty">No social profiles added yet.</li>
                            </ul>
                            <form id="admin-social-form" class="admin-form admin-social__form" novalidate>
                                <input type="hidden" name="id" />
                                <label class="admin-form__label">
                                    Name
                                    <input type="text" name="name" required class="admin-form__input" />
                                </label>
                                <label class="admin-form__label">
                                    Profile URL
                                    <input type="url" name="url" required class="admin-form__input" placeholder="https://example.com" />
                                </label>
                                <label class="admin-form__label">
                                    Icon URL
                                    <input type="text" name="icon" class="admin-form__input" placeholder="/static/icons/social/twitter.svg" />
                                    <small class="admin-card__description admin-form__hint">
                                        Provide an SVG or PNG icon. Leave blank to display the network initials.
                                    </small>
                                </label>
                                <div class="admin-form__actions admin-form__actions--inline">
                                    <button type="submit" class="admin-form__submit" data-role="social-submit">
                                        Save social link
                                    </button>
                                    <button type="button" class="admin-form__cancel" data-role="social-cancel" hidden>
                                        Cancel
                                    </button>
                                </div>
                            </form>
                        </div>
                    </section>
                </div>
            </section>
`,
    });

    registerPanelMarkup({
        id: 'navigation',
        order: 140,
        markup: String.raw`
<section
                id="admin-panel-navigation"
                class="admin-panel"
                data-panel="navigation"
                data-nav-group="configuration"
                data-nav-group-label="Configuration"
                data-nav-group-order="3"
                data-nav-label="Navigation menu"
                data-nav-order="4"
                role="tabpanel"
                aria-labelledby="admin-tab-navigation"
                hidden
            >
                <header class="admin-panel__header">
                    <div>
                        <h2 class="admin-panel__title">Navigation menu</h2>
                        <p class="admin-panel__description">
                            Control the navigation links shown in the site header and footer. Choose a location, add menu items with a label and destination URL, and arrange them in the preferred order.
                        </p>
                    </div>
                </header>
                <div class="admin-panel__body">
                    <section class="admin-card admin-navigation" aria-labelledby="admin-navigation-title">
                        <header class="admin-card__header admin-navigation__header">
                            <h3 id="admin-navigation-title" class="admin-card__title admin-form__legend">Navigation menu</h3>
                            <p class="admin-card__description admin-form__hint">
                                Control the navigation links shown in the site header and footer. Choose a location, add menu
                                items with a label and destination URL, and arrange them in the preferred order.
                            </p>
                        </header>
                        <div class="admin-card__body admin-navigation__content">
                            <ul class="admin-navigation__list" data-role="menu-list" aria-live="polite">
                                <li class="admin-navigation__empty" data-role="menu-empty">
                                    No menu items added for this location yet.
                                </li>
                            </ul>
                            <form id="admin-menu-form" class="admin-form admin-navigation__form" novalidate>
                                <input type="hidden" name="id" />
                                <fieldset class="admin-form__fieldset admin-navigation__step">
                                    <legend class="admin-form__legend admin-navigation__step-title">1. Choose menu location</legend>
                                    <p class="admin-navigation__step-description">
                                        Pick where this menu item should appear on your site.
                                    </p>
                                    <label class="admin-form__label admin-navigation__location">
                                        Menu location
                                        <select
                                            name="location"
                                            required
                                            class="admin-form__input"
                                            data-role="menu-location"
                                        >
                                            <option value="header">Header</option>
                                            <option value="footer:explore">Footer – Explore</option>
                                            <option value="footer:account">Footer – Account</option>
                                            <option value="footer:legal">Footer – Legal</option>
                                            <option value="footer">Footer (general)</option>
                                            <option value="__custom_footer__">Create new footer section…</option>
                                        </select>
                                        <small class="admin-card__description admin-form__hint">
                                            Select the navigation area to update.
                                        </small>
                                    </label>
                                </fieldset>
                                <fieldset class="admin-form__fieldset admin-navigation__step">
                                    <legend class="admin-form__legend admin-navigation__step-title">2. Create a footer section</legend>
                                    <p class="admin-navigation__step-description">
                                        Need a new footer column? Choose "Create new footer section…" above and name it here.
                                    </p>
                                    <p class="admin-navigation__step-note" data-role="menu-custom-location-hint">
                                        Select "Create new footer section…" to enable this field.
                                    </p>
                                    <div class="admin-navigation__custom" data-role="menu-custom-location" hidden>
                                        <label class="admin-form__label">
                                            Footer section name
                                            <input
                                                type="text"
                                                class="admin-form__input"
                                                data-role="menu-location-name"
                                                placeholder="Resources"
                                                autocomplete="off"
                                            />
                                            <small class="admin-card__description admin-form__hint">
                                                We'll create a new footer group using this name and organise
                                                links under it.
                                            </small>
                                        </label>
                                    </div>
                                </fieldset>
                                <fieldset class="admin-form__fieldset admin-navigation__step">
                                    <legend class="admin-form__legend admin-navigation__step-title">3. Add menu item</legend>
                                    <p class="admin-navigation__step-description">
                                        Give the link a label and destination URL for the selected location.
                                    </p>
                                    <label class="admin-form__label">
                                        Label
                                        <input type="text" name="title" required class="admin-form__input" />
                                    </label>
                                    <label class="admin-form__label">
                                        Destination URL
                                        <input type="url" name="url" required class="admin-form__input" placeholder="/blog" />
                                        <small class="admin-card__description admin-form__hint">
                                            Accepts relative or absolute URLs. Relative paths are recommended for internal pages.
                                        </small>
                                    </label>
                                    <div class="admin-form__actions admin-form__actions--inline">
                                        <button type="submit" class="admin-form__submit" data-role="menu-submit">
                                            Save menu item
                                        </button>
                                        <button type="button" class="admin-form__cancel" data-role="menu-cancel" hidden>
                                            Cancel
                                        </button>
                                    </div>
                                </fieldset>
                            </form>
                        </div>
                    </section>
                </div>
            </section>
`,
    });

    init();
})();

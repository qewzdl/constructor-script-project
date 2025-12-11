/**
 * Admin Analytics Module
 * Handles metrics, charts, and analytics display
 */
(() => {
    /**
     * Create analytics manager instance
     */
    const createAnalyticsManager = ({ root, chartLibrary }) => {
        const metricElements = new Map();

        // Collect metric elements
        root.querySelectorAll('.admin__metric').forEach((card) => {
            const key = card.dataset.metric;
            const valueElement = card.querySelector('.admin__metric-value');
            if (key && valueElement) {
                metricElements.set(key, valueElement);
            }
        });

        /**
         * Update metric value
         */
        const updateMetric = (key, value) => {
            const element = metricElements.get(key);
            if (!element) {
                return;
            }
            element.textContent = String(value);
        };

        /**
         * Update multiple metrics
         */
        const updateMetrics = (metrics) => {
            if (!metrics || typeof metrics !== 'object') {
                return;
            }
            Object.entries(metrics).forEach(([key, value]) => {
                updateMetric(key, value);
            });
        };

        /**
         * Create chart instance
         */
        const createChart = (container, series, data) => {
            if (!container || !chartLibrary) {
                return null;
            }

            // Chart library integration would go here
            // This is a placeholder for the actual chart implementation
            return {
                update: (newData) => {
                    // Update chart with new data
                },
                destroy: () => {
                    // Cleanup chart
                },
            };
        };

        return {
            updateMetric,
            updateMetrics,
            createChart,
        };
    };

    /**
     * Create post analytics manager
     */
    const createPostAnalyticsManager = ({ root, apiClient, uiManager }) => {
        const container = root.querySelector('[data-role="post-analytics"]');
        const summary = root.querySelector('[data-role="post-analytics-summary"]');
        const loading = root.querySelector('[data-role="post-analytics-loading"]');
        const empty = root.querySelector('[data-role="post-analytics-empty"]');
        const comparisons = root.querySelector('[data-role="post-analytics-comparisons"]');
        const comparisonsEmpty = root.querySelector('[data-role="post-analytics-comparisons-empty"]');
        const chartContainer = root.querySelector('[data-role="post-analytics-chart"]');

        const summaryItems = new Map();
        if (summary) {
            summary.querySelectorAll('[data-metric]').forEach((item) => {
                const key = item.dataset.metric;
                if (key) {
                    summaryItems.set(key, item);
                }
            });
        }

        /**
         * Show loading state
         */
        const showLoading = () => {
            if (loading) loading.hidden = false;
            if (summary) summary.hidden = true;
            if (empty) empty.hidden = true;
            if (comparisons) comparisons.hidden = true;
        };

        /**
         * Show empty state
         */
        const showEmpty = () => {
            if (loading) loading.hidden = true;
            if (summary) summary.hidden = true;
            if (empty) empty.hidden = false;
            if (comparisons) comparisons.hidden = true;
        };

        /**
         * Update summary item
         */
        const updateSummaryItem = (key, value) => {
            const item = summaryItems.get(key);
            if (!item) {
                return;
            }
            const valueElement = item.querySelector('.admin__metric-value');
            if (valueElement) {
                valueElement.textContent = String(value);
            }
        };

        /**
         * Update post analytics data
         */
        const updateAnalytics = (data) => {
            if (!data || !data.summary) {
                showEmpty();
                return;
            }

            if (loading) loading.hidden = true;
            if (summary) summary.hidden = false;
            if (empty) empty.hidden = true;

            // Update summary metrics
            Object.entries(data.summary).forEach(([key, value]) => {
                updateSummaryItem(key, value);
            });

            // Update comparisons if available
            if (data.comparisons && comparisons) {
                comparisons.hidden = false;
                if (comparisonsEmpty) comparisonsEmpty.hidden = true;
                // Render comparison data
            }
        };

        /**
         * Load analytics for post
         */
        const loadPostAnalytics = async (postId) => {
            if (!postId) {
                showEmpty();
                return;
            }

            showLoading();

            try {
                const data = await apiClient.request(
                    `/api/admin/posts/${postId}/analytics`
                );
                updateAnalytics(data);
            } catch (error) {
                uiManager.handleRequestError(error);
                showEmpty();
            }
        };

        return {
            loadPostAnalytics,
            updateAnalytics,
            showLoading,
            showEmpty,
        };
    };

    // Export to global namespace
    window.AdminAnalytics = {
        create: createAnalyticsManager,
        createPostAnalytics: createPostAnalyticsManager,
    };
})();

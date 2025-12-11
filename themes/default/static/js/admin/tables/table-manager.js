/**
 * Admin Table Manager Base Module
 * Provides common functionality for managing data tables in admin panel
 */
(() => {
    /**
     * Create table manager instance
     * @param {Object} config - Configuration object
     * @param {HTMLElement} config.tableElement - Table element
     * @param {HTMLElement} config.searchInput - Search input element
     * @param {Object} config.apiClient - API client instance
     * @param {Object} config.uiManager - UI manager instance
     * @param {Function} config.renderRow - Function to render table row
     * @param {Function} config.getRowId - Function to get row ID from data
     */
    const createTableManager = (config) => {
        const {
            tableElement,
            searchInput,
            apiClient,
            uiManager,
            renderRow,
            getRowId = (item) => item.id || item.ID,
            emptyMessage = 'No items found',
        } = config;

        let data = [];
        let filteredData = [];
        let searchQuery = '';

        /**
         * Render table with current data
         */
        const render = () => {
            if (!tableElement) {
                return;
            }

            const tbody = tableElement.querySelector('tbody');
            if (!tbody) {
                return;
            }

            tbody.innerHTML = '';

            if (filteredData.length === 0) {
                const tr = document.createElement('tr');
                const td = document.createElement('td');
                td.colSpan = 100;
                td.className = 'admin__table-empty';
                td.textContent = searchQuery ? 'No matching items found' : emptyMessage;
                tr.appendChild(td);
                tbody.appendChild(tr);
                return;
            }

            filteredData.forEach((item) => {
                const row = renderRow(item);
                if (row) {
                    tbody.appendChild(row);
                }
            });
        };

        /**
         * Filter data based on search query
         */
        const applyFilter = (query) => {
            searchQuery = query.toLowerCase().trim();

            if (!searchQuery) {
                filteredData = [...data];
                render();
                return;
            }

            filteredData = data.filter((item) => {
                const searchableText = JSON.stringify(item).toLowerCase();
                return searchableText.includes(searchQuery);
            });

            render();
        };

        /**
         * Set table data
         */
        const setData = (newData) => {
            data = Array.isArray(newData) ? newData : [];
            applyFilter(searchQuery);
        };

        /**
         * Add item to table
         */
        const addItem = (item) => {
            data.push(item);
            applyFilter(searchQuery);
        };

        /**
         * Update item in table
         */
        const updateItem = (updatedItem) => {
            const id = getRowId(updatedItem);
            const index = data.findIndex((item) => getRowId(item) === id);
            if (index !== -1) {
                data[index] = updatedItem;
                applyFilter(searchQuery);
            }
        };

        /**
         * Remove item from table
         */
        const removeItem = (id) => {
            data = data.filter((item) => getRowId(item) !== id);
            applyFilter(searchQuery);
        };

        /**
         * Get item by ID
         */
        const getItem = (id) => {
            return data.find((item) => getRowId(item) === id);
        };

        /**
         * Get all data
         */
        const getData = () => [...data];

        /**
         * Initialize search functionality
         */
        const initSearch = () => {
            if (!searchInput) {
                return;
            }

            let searchTimeoutId = null;

            searchInput.addEventListener('input', (event) => {
                window.clearTimeout(searchTimeoutId);
                searchTimeoutId = window.setTimeout(() => {
                    applyFilter(event.target.value);
                }, 300);
            });
        };

        // Initialize
        initSearch();

        return {
            render,
            setData,
            addItem,
            updateItem,
            removeItem,
            getItem,
            getData,
            applyFilter,
        };
    };

    // Export to global namespace
    window.AdminTableManager = {
        create: createTableManager,
    };
})();

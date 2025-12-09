// Setup Wizard Configuration and Builder
// Этот механизм автоматически строит интерфейс setup на основе конфигурации

// UI Helper Functions
const showAlert = (element, message, type = "error") => {
    element.textContent = message;
    element.className = `setup__alert setup__alert--${type}`;
    element.hidden = false;
    element.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
};

const clearAlert = (element) => {
    element.hidden = true;
    element.textContent = "";
};

const disableForm = (form, disabled) => {
    Array.from(form.elements).forEach((el) => {
        el.disabled = disabled;
    });
};

class SetupStepConfig {
    constructor(id, title, description, fields) {
        this.id = id;
        this.title = title;
        this.description = description;
        this.fields = fields;
    }
}

class SetupFieldConfig {
    constructor(config) {
        this.name = config.name;
        this.label = config.label;
        this.type = config.type || 'text';
        this.placeholder = config.placeholder || '';
        this.required = config.required !== false;
        this.minLength = config.minLength;
        this.maxLength = config.maxLength;
        this.pattern = config.pattern;
        this.helperText = config.helperText;
        this.rows = config.rows; // для textarea
        this.autocomplete = config.autocomplete;
        this.validator = config.validator; // custom validator function
    }

    createFieldHTML() {
        const fieldDiv = document.createElement('div');
        fieldDiv.className = 'form-field';

        const label = document.createElement('label');
        label.className = 'form-field__label';
        label.setAttribute('for', `setup-${this.name}`);
        label.textContent = this.label;
        fieldDiv.appendChild(label);

        let input;
        if (this.type === 'textarea') {
            input = document.createElement('textarea');
            if (this.rows) {
                input.rows = this.rows;
            }
        } else {
            input = document.createElement('input');
            input.type = this.type;
        }

        input.id = `setup-${this.name}`;
        input.name = this.name;
        input.className = 'form-field__input';
        input.placeholder = this.placeholder;

        if (this.required) {
            input.required = true;
        }
        if (this.minLength) {
            input.minLength = this.minLength;
        }
        if (this.maxLength) {
            input.maxLength = this.maxLength;
        }
        if (this.pattern) {
            input.pattern = this.pattern;
        }
        if (this.autocomplete) {
            input.autocomplete = this.autocomplete;
        }

        fieldDiv.appendChild(input);

        if (this.helperText) {
            const helper = document.createElement('small');
            helper.className = 'form-helper';
            helper.innerHTML = this.helperText;
            fieldDiv.appendChild(helper);
        }

        return fieldDiv;
    }

    validate(value) {
        value = typeof value === 'string' ? value.trim() : value;

        if (this.required && !value) {
            return `${this.label} is required`;
        }

        if (value && this.minLength && value.length < this.minLength) {
            return `${this.label} must be at least ${this.minLength} characters`;
        }

        if (value && this.maxLength && value.length > this.maxLength) {
            return `${this.label} must not exceed ${this.maxLength} characters`;
        }

        if (value && this.pattern && !new RegExp(this.pattern).test(value)) {
            return `${this.label} format is invalid`;
        }

        if (this.validator && value) {
            return this.validator(value);
        }

        return null;
    }
}

// Setup Configuration
const SETUP_CONFIGURATION = {
    steps: [
        new SetupStepConfig(
            'site_info',
            'Site Information',
            'Configure your site name, URL, and branding.',
            [
                new SetupFieldConfig({
                    name: 'site_name',
                    label: 'Site name',
                    placeholder: 'My awesome blog',
                    required: true,
                    maxLength: 255
                }),
                new SetupFieldConfig({
                    name: 'site_description',
                    label: 'Tagline',
                    type: 'textarea',
                    rows: 2,
                    placeholder: 'A modern blog built with Go',
                    required: false,
                    maxLength: 1000
                }),
                new SetupFieldConfig({
                    name: 'site_url',
                    label: 'Site URL',
                    type: 'url',
                    placeholder: 'https://example.com',
                    required: true,
                    validator: (value) => {
                        try {
                            new URL(value);
                            return null;
                        } catch {
                            return 'Site URL must be a valid URL';
                        }
                    }
                }),
                new SetupFieldConfig({
                    name: 'site_favicon',
                    label: 'Favicon URL',
                    placeholder: '/favicon.ico',
                    required: false
                }),
                new SetupFieldConfig({
                    name: 'site_logo',
                    label: 'Logo URL',
                    placeholder: '/static/icons/logo.svg',
                    required: false
                })
            ]
        ),
        new SetupStepConfig(
            'admin',
            'Administrator Account',
            'Create the first administrator account.',
            [
                new SetupFieldConfig({
                    name: 'admin_username',
                    label: 'Username',
                    placeholder: 'Admin username',
                    required: true,
                    minLength: 3,
                    maxLength: 50,
                    autocomplete: 'username'
                }),
                new SetupFieldConfig({
                    name: 'admin_email',
                    label: 'Email',
                    type: 'email',
                    placeholder: 'admin@example.com',
                    required: true,
                    autocomplete: 'email',
                    validator: (value) => {
                        const emailRegex = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
                        if (!emailRegex.test(value)) {
                            return 'Please enter a valid email address';
                        }
                        return null;
                    }
                }),
                new SetupFieldConfig({
                    name: 'admin_password',
                    label: 'Password',
                    type: 'password',
                    placeholder: 'Create a strong password',
                    required: true,
                    minLength: 8,
                    maxLength: 128,
                    autocomplete: 'new-password',
                    helperText: 'Must be at least 8 characters'
                })
            ]
        ),
        new SetupStepConfig(
            'languages',
            'Language Settings',
            'Set up language preferences for your site.',
            [
                new SetupFieldConfig({
                    name: 'site_default_language',
                    label: 'Default language',
                    placeholder: 'en',
                    required: true,
                    pattern: '^[a-z]{2,8}(?:-[A-Za-z]{2,3})?$',
                    helperText: 'Provide a language code using the BCP&nbsp;47 format (for example <code>en</code> or <code>en-GB</code>).'
                }),
                new SetupFieldConfig({
                    name: 'site_supported_languages',
                    label: 'Supported languages',
                    placeholder: 'en, en-GB, fr',
                    required: false,
                    helperText: 'Optional comma-separated list of additional language codes. The default language is always available.'
                })
            ]
        )
    ],

    getStepById(stepId) {
        return this.steps.find(s => s.id === stepId);
    },

    getStepIndex(stepId) {
        return this.steps.findIndex(s => s.id === stepId);
    },

    getAllStepIds() {
        return this.steps.map(s => s.id);
    }
};

// Setup Builder - строит UI на основе конфигурации
class SetupBuilder {
    constructor(containerSelector, progressSelector) {
        this.container = document.querySelector(containerSelector);
        this.progressContainer = document.querySelector(progressSelector);
        
        if (!this.container) {
            throw new Error(`Container ${containerSelector} not found`);
        }
    }

    buildProgressIndicator() {
        if (!this.progressContainer) return;

        this.progressContainer.innerHTML = '';

        SETUP_CONFIGURATION.steps.forEach((step, index) => {
            const stepDiv = document.createElement('div');
            stepDiv.className = 'progress-step';
            stepDiv.setAttribute('data-step', step.id);

            const numberDiv = document.createElement('div');
            numberDiv.className = 'progress-step__number';
            numberDiv.textContent = index + 1;
            stepDiv.appendChild(numberDiv);

            const labelDiv = document.createElement('div');
            labelDiv.className = 'progress-step__label';
            labelDiv.textContent = step.title;
            stepDiv.appendChild(labelDiv);

            this.progressContainer.appendChild(stepDiv);
        });
    }

    buildStepForm(stepConfig) {
        const stepDiv = document.createElement('div');
        stepDiv.className = 'setup-step';
        stepDiv.id = `step-${stepConfig.id}`;
        stepDiv.setAttribute('data-step', stepConfig.id);
        stepDiv.hidden = true;

        const fieldset = document.createElement('fieldset');
        fieldset.className = 'form-fieldset';

        const legend = document.createElement('legend');
        legend.className = 'form-legend';
        legend.textContent = stepConfig.title;
        fieldset.appendChild(legend);

        stepConfig.fields.forEach(fieldConfig => {
            const fieldHTML = fieldConfig.createFieldHTML();
            fieldset.appendChild(fieldHTML);
        });

        stepDiv.appendChild(fieldset);
        return stepDiv;
    }

    buildAllSteps() {
        SETUP_CONFIGURATION.steps.forEach(stepConfig => {
            const stepElement = this.buildStepForm(stepConfig);
            this.container.appendChild(stepElement);
        });
    }

    build() {
        this.buildProgressIndicator();
        this.buildAllSteps();
    }
}

// Setup Field Manager - управляет полями формы
class SetupFieldManager {
    static getFieldValue(stepId, fieldName) {
        const field = document.querySelector(`#step-${stepId} [name="${fieldName}"]`);
        return field ? field.value.trim() : '';
    }

    static setFieldValue(stepId, fieldName, value) {
        const field = document.querySelector(`#step-${stepId} [name="${fieldName}"]`);
        if (field && value) {
            field.value = value;
        }
    }

    static getStepData(stepId) {
        const stepConfig = SETUP_CONFIGURATION.getStepById(stepId);
        if (!stepConfig) return {};

        const data = {};
        stepConfig.fields.forEach(fieldConfig => {
            const value = this.getFieldValue(stepId, fieldConfig.name);
            
            if (fieldConfig.name === 'site_supported_languages') {
                data.supported_languages = value ? value.split(',').map(s => s.trim()).filter(Boolean) : [];
            } else if (fieldConfig.name === 'site_default_language') {
                data.default_language = this.normalizeLanguageCode(value);
            } else {
                data[fieldConfig.name] = value;
            }
        });

        return data;
    }

    static normalizeLanguageCode(value) {
        if (typeof value !== 'string') return '';
        const trimmed = value.trim();
        if (!trimmed) return '';
        
        const parts = trimmed.split('-');
        const base = parts[0].toLowerCase();
        if (parts.length === 1) return base;
        
        const region = parts[1]?.toUpperCase();
        return region ? `${base}-${region}` : base;
    }

    static validateStep(stepId) {
        const stepConfig = SETUP_CONFIGURATION.getStepById(stepId);
        if (!stepConfig) return 'Invalid step';

        for (const fieldConfig of stepConfig.fields) {
            const value = this.getFieldValue(stepId, fieldConfig.name);
            const error = fieldConfig.validate(value);
            if (error) {
                return error;
            }
        }

        return null;
    }

    static populateFromProgress(progress) {
        if (!progress) return;

        // Site info
        const siteInfo = progress.site_info || {};
        this.setFieldValue('site_info', 'site_name', siteInfo.name);
        this.setFieldValue('site_info', 'site_description', siteInfo.description);
        this.setFieldValue('site_info', 'site_url', siteInfo.url);
        this.setFieldValue('site_info', 'site_favicon', siteInfo.favicon);
        this.setFieldValue('site_info', 'site_logo', siteInfo.logo);

        // Admin
        const admin = progress.admin || {};
        this.setFieldValue('admin', 'admin_username', admin.username);
        this.setFieldValue('admin', 'admin_email', admin.email);

        // Languages
        const languages = progress.languages || {};
        this.setFieldValue('languages', 'site_default_language', languages.default_language);
        this.setFieldValue('languages', 'site_supported_languages', languages.supported_languages);
    }
}

// Экспорт для использования в основном коде
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { 
        SETUP_CONFIGURATION, 
        SetupBuilder, 
        SetupFieldManager,
        SetupStepConfig,
        SetupFieldConfig
    };
}

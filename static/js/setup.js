// Setup wizard state
class SetupWizard {
    constructor(form, alertElement, statusUrl) {
        this.form = form;
        this.alertElement = alertElement;
        this.statusUrl = statusUrl;
        this.steps = SETUP_CONFIGURATION.getAllStepIds();
        this.currentStepIndex = 0;
        this.progress = null;
        
        // Extract setup key from URL
        const urlParams = new URLSearchParams(window.location.search);
        this.setupKey = urlParams.get('key') || '';
        
        this.prevButton = document.getElementById('prev-button');
        this.nextButton = document.getElementById('next-button');
        this.completeButton = document.getElementById('complete-button');
        
        // UI already exists in HTML, no need to build
        // const builder = new SetupBuilder('#setup-form', '#setup-progress');
        // builder.build();
        
        this.init();
    }
    
    async init() {
        try {
            await this.loadProgress();
            this.setupEventListeners();
            this.updateUI();
        } catch (error) {
            showAlert(this.alertElement, error.message || "Failed to initialize setup");
        }
    }
    
    async loadProgress() {
        // Add setup key to status URL if present
        const statusUrl = this.setupKey 
            ? `${this.statusUrl}?key=${encodeURIComponent(this.setupKey)}`
            : this.statusUrl;
            
        const response = await fetch(statusUrl, {
            headers: this.setupKey ? { 'X-Setup-Key': this.setupKey } : {}
        });
        
        if (!response.ok) {
            throw new Error("Failed to load setup status");
        }
        
        const data = await response.json();
        
        if (!data.setup_required) {
            window.location.href = "/";
            return;
        }
        
        this.progress = data.progress || {};
        
        // Determine current step based on progress
        if (this.progress.current_step) {
            const stepIndex = this.steps.indexOf(this.progress.current_step);
            if (stepIndex >= 0) {
                this.currentStepIndex = stepIndex;
            }
        }
        
        // Populate fields from progress using FieldManager
        SetupFieldManager.populateFromProgress(this.progress);
    }
    
    setupEventListeners() {
        this.prevButton.addEventListener('click', () => this.previousStep());
        this.nextButton.addEventListener('click', () => this.nextStep());
        this.form.addEventListener('submit', (e) => this.handleSubmit(e));
    }
    
    updateUI() {
        const currentStep = this.steps[this.currentStepIndex];
        const currentStepConfig = SETUP_CONFIGURATION.getStepById(currentStep);
        
        // Update step visibility
        document.querySelectorAll('.setup-step').forEach((stepEl) => {
            stepEl.hidden = stepEl.dataset.step !== currentStep;
        });
        
        // Update progress indicators
        document.querySelectorAll('.progress-step').forEach((indicator, index) => {
            indicator.classList.remove('active', 'completed');
            if (index < this.currentStepIndex) {
                indicator.classList.add('completed');
            } else if (index === this.currentStepIndex) {
                indicator.classList.add('active');
            }
        });
        
        // Update buttons
        this.prevButton.hidden = this.currentStepIndex === 0;
        this.nextButton.hidden = this.currentStepIndex === this.steps.length - 1;
        this.completeButton.hidden = this.currentStepIndex !== this.steps.length - 1;
        
        // Update subtitle
        const subtitle = document.getElementById('setup-subtitle');
        if (subtitle && currentStepConfig) {
            subtitle.textContent = currentStepConfig.description;
        }
        
        clearAlert(this.alertElement);
    }
    
    async nextStep() {
        clearAlert(this.alertElement);
        
        const currentStep = this.steps[this.currentStepIndex];
        
        // Validate using FieldManager
        const error = SetupFieldManager.validateStep(currentStep);
        if (error) {
            showAlert(this.alertElement, error, 'error');
            return;
        }
        
        // Get step data using FieldManager
        const stepData = SetupFieldManager.getStepData(currentStep);
        
        // Save step data to server
        try {
            disableForm(this.form, true);
            
            const url = this.setupKey 
                ? `/api/v1/setup/step?key=${encodeURIComponent(this.setupKey)}`
                : '/api/v1/setup/step';
            
            const headers = { 'Content-Type': 'application/json' };
            if (this.setupKey) {
                headers['X-Setup-Key'] = this.setupKey;
            }
            
            const response = await fetch(url, {
                method: 'POST',
                headers: headers,
                body: JSON.stringify({ step: currentStep, ...stepData })
            });
            
            if (!response.ok) {
                const error = await response.json().catch(() => ({ error: 'Failed to save step' }));
                throw new Error(error.error || 'Failed to save step');
            }
            
            const data = await response.json();
            this.progress = data.progress || {};
            
            // Move to next step
            this.currentStepIndex++;
            this.updateUI();
        } catch (error) {
            showAlert(this.alertElement, error.message || 'Failed to save step', 'error');
        } finally {
            disableForm(this.form, false);
        }
    }
    
    previousStep() {
        if (this.currentStepIndex > 0) {
            this.currentStepIndex--;
            this.updateUI();
        }
    }
    
    async handleSubmit(event) {
        event.preventDefault();
        clearAlert(this.alertElement);
        
        try {
            disableForm(this.form, true);
            
            // Save the current (last) step first
            const currentStep = this.steps[this.currentStepIndex];
            const error = SetupFieldManager.validateStep(currentStep);
            if (error) {
                showAlert(this.alertElement, error, 'error');
                return;
            }
            
            const stepData = SetupFieldManager.getStepData(currentStep);
            
            // Save last step
            const saveUrl = this.setupKey 
                ? `/api/v1/setup/step?key=${encodeURIComponent(this.setupKey)}`
                : '/api/v1/setup/step';
            
            const saveHeaders = { 'Content-Type': 'application/json' };
            if (this.setupKey) {
                saveHeaders['X-Setup-Key'] = this.setupKey;
            }
            
            const saveResponse = await fetch(saveUrl, {
                method: 'POST',
                headers: saveHeaders,
                body: JSON.stringify({ step: currentStep, ...stepData })
            });
            
            if (!saveResponse.ok) {
                const error = await saveResponse.json().catch(() => ({ error: 'Failed to save last step' }));
                throw new Error(error.error || 'Failed to save last step');
            }
            
            // Now complete setup
            const completeUrl = this.setupKey 
                ? `/api/v1/setup/complete?key=${encodeURIComponent(this.setupKey)}`
                : '/api/v1/setup/complete';
            
            const completeHeaders = { 'Content-Type': 'application/json' };
            if (this.setupKey) {
                completeHeaders['X-Setup-Key'] = this.setupKey;
            }
            
            const response = await fetch(completeUrl, {
                method: 'POST',
                headers: completeHeaders
            });
            
            if (!response.ok) {
                const error = await response.json().catch(() => ({ error: 'Failed to complete setup' }));
                throw new Error(error.error || 'Failed to complete setup');
            }
            
            showAlert(this.alertElement, 'Setup completed successfully. Redirecting to sign in…', 'success');
            setTimeout(() => {
                window.location.href = '/login';
            }, 1200);
        } catch (error) {
            showAlert(this.alertElement, error.message || 'Failed to complete setup', 'error');
        } finally {
            disableForm(this.form, false);
        }
    }
}

// Initialize when DOM is ready
document.addEventListener("DOMContentLoaded", () => {
    const root = document.querySelector('[data-page="setup"]');
    if (!root) return;

    const form = root.querySelector("#setup-form");
    const alertElement = root.querySelector("#setup-alert");
    const statusUrl = form?.dataset.status;

    if (!form || !statusUrl) {
        console.error('Setup form or status URL not found');
        return;
    }

    new SetupWizard(form, alertElement, statusUrl);
});

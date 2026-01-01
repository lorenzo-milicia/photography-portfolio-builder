/**
 * Photography Portfolio Builder - Client-side JavaScript
 * Handles UI interactions and dynamic behaviors for the builder interface
 */

(function() {
  'use strict';

  // Prevent multiple instantiations
  if (window.PortfolioBuilder) {
    return;
  }

  // =============================================================================
  // TOAST NOTIFICATIONS
  // =============================================================================

  const ToastManager = {
    AUTO_DISMISS_DELAY: 5000, // milliseconds
    container: null,
    clearAllButton: null,
    initialized: false,

    init() {
      if (this.initialized) return;
      
      this.container = document.getElementById('toast-container');
      this.clearAllButton = document.getElementById('toast-clear-all');
      
      if (!this.container) {
        console.warn('Toast container not found');
        return;
      }

      // Set up event delegation for close buttons
      this.container.addEventListener('click', (e) => {
        const closeButton = e.target.closest('.toast-close');
        if (closeButton) {
          const toast = closeButton.closest('[data-toast]');
          if (toast) {
            this.dismissToast(toast);
          }
        }
      });

      // Set up clear all button
      if (this.clearAllButton) {
        this.clearAllButton.addEventListener('click', () => {
          this.clearAll();
        });
      }

      // Set up MutationObserver to handle dynamically added toasts
      this.observeToastAdditions();

      // Initialize any existing toasts on page load
      this.initializeExistingToasts();

      this.initialized = true;
    },

    observeToastAdditions() {
      const observer = new MutationObserver((mutations) => {
        mutations.forEach((mutation) => {
          mutation.addedNodes.forEach((node) => {
            if (node.nodeType === 1 && node.hasAttribute && node.hasAttribute('data-toast')) {
              this.setupToast(node);
            }
          });
        });
        this.updateClearAllButton();
      });

      observer.observe(this.container, {
        childList: true,
        subtree: false
      });
    },

    initializeExistingToasts() {
      const toasts = this.container.querySelectorAll('[data-toast]');
      toasts.forEach(toast => this.setupToast(toast));
      this.updateClearAllButton();
    },

    setupToast(toast) {
      // Skip if already set up
      if (toast.dataset.toastInitialized) return;
      
      toast.dataset.toastInitialized = 'true';

      // Auto-dismiss after delay
      const timeoutId = setTimeout(() => {
        this.dismissToast(toast);
      }, this.AUTO_DISMISS_DELAY);

      // Store timeout ID so we can cancel it if manually dismissed
      toast.dataset.timeoutId = timeoutId;
    },

    dismissToast(toast) {
      // Clear auto-dismiss timeout
      const timeoutId = toast.dataset.timeoutId;
      if (timeoutId) {
        clearTimeout(parseInt(timeoutId));
      }

      // Add fade-out animation
      toast.style.animation = 'slideOut 0.3s ease-in forwards';
      
      // Remove from DOM after animation
      setTimeout(() => {
        toast.remove();
        this.updateClearAllButton();
      }, 300);
    },

    clearAll() {
      const toasts = this.container.querySelectorAll('[data-toast]');
      toasts.forEach(toast => {
        // Clear timeouts
        const timeoutId = toast.dataset.timeoutId;
        if (timeoutId) {
          clearTimeout(parseInt(timeoutId));
        }
        toast.remove();
      });
      this.updateClearAllButton();
    },

    updateClearAllButton() {
      if (!this.clearAllButton) return;
      
      const toastCount = this.container.querySelectorAll('[data-toast]').length;
      this.clearAllButton.style.display = toastCount > 1 ? 'block' : 'none';
    }
  };

  // =============================================================================
  // INITIALIZATION
  // =============================================================================

  function init() {
    // Initialize toast manager
    ToastManager.init();
  }

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  // Expose public API
  window.PortfolioBuilder = {
    ToastManager: ToastManager,
    version: '1.0.0'
  };

})();

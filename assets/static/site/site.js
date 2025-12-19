/**
 * Photography Portfolio - Site JavaScript
 * This file contains all interactive behavior for the static site
 */

(function() {
    'use strict';

    // Mobile menu toggle
    window.toggleMobileMenu = function() {
        const mobileMenuOverlay = document.querySelector('.mobile-menu-overlay');
        const navbarToggle = document.querySelector('.navbar-toggle');
        if (mobileMenuOverlay) {
            mobileMenuOverlay.classList.toggle('open');
        }
        if (navbarToggle) {
            navbarToggle.classList.toggle('active');
        }
    };

    // Scroll to top of page
    window.scrollToTop = function() {
        window.scrollTo({
            top: 0,
            behavior: 'smooth'
        });
    };

    // Initialize all event listeners when DOM is ready
    document.addEventListener('DOMContentLoaded', function() {
        initializeNavbar();
        initializeScrollBehavior();
        initializeRevealAnimations();
    });

    /**
     * Initialize navbar interactions
     */
    function initializeNavbar() {
        // Close mobile menu when clicking outside
        document.addEventListener('click', function(e) {
            const mobileMenuOverlay = document.querySelector('.mobile-menu-overlay');
            const navbarToggle = document.querySelector('.navbar-toggle');
            
            if (mobileMenuOverlay && mobileMenuOverlay.classList.contains('open')) {
                // Check if click is outside the overlay and not on the toggle button
                if (!mobileMenuOverlay.contains(e.target) && !navbarToggle.contains(e.target)) {
                    mobileMenuOverlay.classList.remove('open');
                    navbarToggle.classList.remove('active');
                }
            }
        });

        // Dropdown click handler (for desktop)
        document.addEventListener('click', function(e) {
            const dropdowns = document.querySelectorAll('.dropdown');
            dropdowns.forEach(dropdown => {
                const menu = dropdown.querySelector('.dropdown-menu');
                const toggle = dropdown.querySelector('.dropdown-toggle');
                if (!dropdown.contains(e.target)) {
                    if (menu) menu.classList.remove('open');
                    if (toggle) toggle.classList.remove('active');
                }
            });
        });
    }

    /**
     * Initialize scroll-based navbar project controller reveal
     * (Title is now sticky, so we only need to show the controller)
     */
    function initializeScrollBehavior() {
        const navbarController = document.querySelector('.navbar-controller');
        const projectTitleElement = document.querySelector('.project-title');

        if (!navbarController || !projectTitleElement) {
            return; // Not a project page or no controller
        }

        let ticking = false;

        function updateNavbarController() {
            const titleRect = projectTitleElement.getBoundingClientRect();
            const navbar = document.querySelector('.main-navbar');
            if (!navbar) return;

            const navbarBottom = navbar.getBoundingClientRect().bottom;
            const titleIsAboveNavbar = titleRect.bottom < navbarBottom;

            // Show controller while the title is visible; hide it after the title scrolls past
            if (!titleIsAboveNavbar) {
                navbar.classList.add('navbar-controller-visible');
                navbarController.classList.add('visible');

                if (window.innerWidth <= 768) {
                    morphLogoText('project');
                }
            } else {
                navbar.classList.remove('navbar-controller-visible');
                navbarController.classList.remove('visible');

                if (window.innerWidth <= 768) {
                    morphLogoText('default');
                }
            }
            ticking = false;
        }

        window.addEventListener('scroll', function() {
            if (!ticking) {
                window.requestAnimationFrame(updateNavbarController);
                ticking = true;
            }
        });

        // Initial check
        updateNavbarController();
    }

    /**
     * Morph logo secondary text with fade animation
     * @param {string} mode - 'project' or 'default'
     */
    function morphLogoText(mode) {
        const logoSecondary = document.querySelector('.logo-secondary');
        if (!logoSecondary) return;

        const targetText = mode === 'project' 
            ? logoSecondary.getAttribute('data-project')
            : logoSecondary.getAttribute('data-default');

        if (!targetText || logoSecondary.textContent === targetText) {
            return;
        }

        logoSecondary.style.opacity = '0';
        setTimeout(() => {
            logoSecondary.textContent = targetText;
            logoSecondary.style.opacity = '1';
        }, 230);
    }

    /**
     * Initialize reveal animations for page elements
     */
    function initializeRevealAnimations() {
        // Simple reveal for home page sections
        const simpleRevealSection = document.querySelector('.reveal-item');
        if (simpleRevealSection) {
            // Wait for images inside the hero section to be decoded/loaded
            waitForImagesInContainer(simpleRevealSection, 1000 /*ms timeout*/) 
                .then(() => simpleRevealSection.classList.add('visible'))
                .catch(() => simpleRevealSection.classList.add('visible'));
        }

        // IntersectionObserver-based reveal for gallery items
        setupRevealObserver();
    }

    /**
     * Wait for all images inside a container to be decoded or loaded.
     * Resolves when all images are ready or when the timeout elapses.
     * @param {Element} container
     * @param {number} timeoutMs
     * @returns {Promise}
     */
    function waitForImagesInContainer(container, timeoutMs) {
        const imgs = Array.from(container.querySelectorAll('img'));
        if (imgs.length === 0) return Promise.resolve();

        const decodes = imgs.map(img => {
            // If already complete, resolve immediately
            if (img.complete) return Promise.resolve();

            // Prefer decode() where available
            if (img.decode) {
                return img.decode().catch(() => new Promise(resolve => img.addEventListener('load', resolve, { once: true })));
            }

            // Fallback to load event
            return new Promise(resolve => img.addEventListener('load', resolve, { once: true }));
        });

        // Race the decode promises against a timeout so animation still proceeds
        const timeout = new Promise((resolve) => setTimeout(resolve, timeoutMs));
        return Promise.race([Promise.all(decodes).then(() => {}), timeout]);
    }

    /**
     * Setup IntersectionObserver for scroll-based reveal animations (project galleries)
     */
    function setupRevealObserver() {
        const revealItems = document.querySelectorAll('.reveal-on-scroll');
        if (revealItems.length === 0) return;

        if ('IntersectionObserver' in window) {
            const observerOptions = {
                root: null,
                rootMargin: '-50px 0px -50px 0px',
                threshold: 0
            };

            const observer = new IntersectionObserver((entries) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        const item = entry.target;
                        const img = item.querySelector('img');
                        const pairKey = item.getAttribute('data-pair');

                        function markVisible() {
                            // mark the observed item visible
                            item.classList.add('visible');
                            // If there is a data-pair, mark all paired elements visible too
                            if (pairKey) {
                                document.querySelectorAll('[data-pair="' + pairKey + '"]').forEach(el => el.classList.add('visible'));
                            }
                        }

                        // Use decode API to avoid blocking main thread
                        if (img && !img.complete) {
                            if (img.decode) {
                                img.decode()
                                    .then(() => markVisible())
                                    .catch(() => markVisible());
                            } else {
                                // Fallback for browsers without decode API
                                img.addEventListener('load', () => {
                                    markVisible();
                                }, { once: true });
                            }
                        } else {
                            // Image already loaded or no image
                            markVisible();
                        }

                        observer.unobserve(item);
                    }
                });
            }, observerOptions);

            revealItems.forEach(item => {
                observer.observe(item);
            });
        } else {
            // Fallback for browsers without IntersectionObserver
            revealItems.forEach(item => {
                const img = item.querySelector('img');
                if (img && !img.complete) {
                    if (img.decode) {
                        img.decode()
                            .then(() => item.classList.add('visible'))
                            .catch(() => item.classList.add('visible'));
                    } else {
                        img.addEventListener('load', () => {
                            item.classList.add('visible');
                        }, { once: true });
                    }
                } else {
                    item.classList.add('visible');
                }
            });
        }
    }

})();

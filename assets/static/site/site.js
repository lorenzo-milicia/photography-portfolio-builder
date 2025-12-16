/**
 * Photography Portfolio - Site JavaScript
 * This file contains all interactive behavior for the static site
 */

(function() {
    'use strict';

    // Mobile menu toggle
    window.toggleMobileMenu = function() {
        const navbarActions = document.querySelector('.navbar-actions');
        if (navbarActions) {
            navbarActions.classList.toggle('mobile-open');
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
            const navbar = document.querySelector('.main-navbar');
            const navbarActions = document.querySelector('.navbar-actions');

            if (navbar && navbarActions && 
                !navbar.contains(e.target) && 
                navbarActions.classList.contains('mobile-open')) {
                navbarActions.classList.remove('mobile-open');
            }
        });

        // Dropdown click handler
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
     * Initialize scroll-based navbar project title reveal
     */
    function initializeScrollBehavior() {
        const navbarCenter = document.querySelector('.navbar-center');
        const projectTitleElement = document.querySelector('.project-title');

        if (!navbarCenter || !projectTitleElement) {
            return; // Not a project page
        }

        let ticking = false;

        function updateNavbarTitle() {
            const titleRect = projectTitleElement.getBoundingClientRect();
            const navbar = document.querySelector('.main-navbar');
            if (!navbar) return;

            const navbarBottom = navbar.getBoundingClientRect().bottom;

            if (titleRect.bottom < navbarBottom) {
                navbar.classList.add('navbar-title-visible');
                navbarCenter.classList.add('visible');

                // Mobile only: morph logo secondary with fade
                if (window.innerWidth <= 768) {
                    morphLogoText('project');
                }
            } else {
                navbar.classList.remove('navbar-title-visible');
                navbarCenter.classList.remove('visible');

                // Mobile only: restore logo secondary with fade
                if (window.innerWidth <= 768) {
                    morphLogoText('default');
                }
            }
            ticking = false;
        }

        window.addEventListener('scroll', function() {
            if (!ticking) {
                window.requestAnimationFrame(updateNavbarTitle);
                ticking = true;
            }
        });

        // Initial check
        updateNavbarTitle();
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
            setTimeout(() => {
                simpleRevealSection.classList.add('visible');
            }, 100);
        }

        // IntersectionObserver-based reveal for gallery items
        setupRevealObserver();
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
                        
                        // Use decode API to avoid blocking main thread
                        if (img && !img.complete) {
                            if (img.decode) {
                                img.decode()
                                    .then(() => item.classList.add('visible'))
                                    .catch(() => item.classList.add('visible'));
                            } else {
                                // Fallback for browsers without decode API
                                img.addEventListener('load', () => {
                                    item.classList.add('visible');
                                }, { once: true });
                            }
                        } else {
                            // Image already loaded or no image
                            item.classList.add('visible');
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

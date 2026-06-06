/* ===== Aspira Pay — Animation & Interaction Engine ===== */
/* ===== Apple Music-inspired micro-interactions ===== */

const Animations = {
    initialized: false,

    init() {
        if (this.initialized) return;
        this.initialized = true;

        this.initRippleEffect();
        this.initCardTilt();
        this.initSmoothCounters();
        this.initScrollReveal();
        this.initButtonPressEffect();
        this.initInputFocusEffect();
        this.initHoverGlowEffect();
    },

    /* ===== Ripple Effect on Buttons ===== */
    initRippleEffect() {
        document.addEventListener('click', function (e) {
            const btn = e.target.closest('.btn:not(.btn-ghost), .quick-action');
            if (!btn || btn.disabled) return;

            // Prevent double ripples
            const existing = btn.querySelector('.ripple-effect');
            if (existing) existing.remove();

            const ripple = document.createElement('span');
            ripple.className = 'ripple-effect';

            const rect = btn.getBoundingClientRect();
            const size = Math.max(rect.width, rect.height) * 2;
            const x = e.clientX - rect.left - size / 2;
            const y = e.clientY - rect.top - size / 2;

            ripple.style.width = ripple.style.height = size + 'px';
            ripple.style.left = x + 'px';
            ripple.style.top = y + 'px';

            btn.appendChild(ripple);

            // Clean up after animation
            ripple.addEventListener('animationend', function () {
                ripple.remove();
            });
        });
    },

    /* ===== Card Tilt Effect (3D perspective) ===== */
    initCardTilt() {
        const cards = document.querySelectorAll('.stat-card, .account-card, .login-card');
        const mainContent = document.getElementById('main-content');

        if (!mainContent) return;

        mainContent.addEventListener('mousemove', function (e) {
            cards.forEach(function (card) {
                // Only process visible cards
                if (card.offsetParent === null) return;

                const rect = card.getBoundingClientRect();
                const centerX = rect.left + rect.width / 2;
                const centerY = rect.top + rect.height / 2;

                // Check if mouse is near the card (within 200px)
                const distX = e.clientX - centerX;
                const distY = e.clientY - centerY;
                const dist = Math.sqrt(distX * distX + distY * distY);
                const maxDist = Math.max(rect.width, rect.height) * 1.5;

                if (dist < maxDist) {
                    const intensity = (1 - dist / maxDist) * 8;
                    const rotateY = (distX / (rect.width / 2)) * intensity;
                    const rotateX = -(distY / (rect.height / 2)) * intensity;

                    card.style.transform =
                        'perspective(800px) translateY(-4px) scale(1.02) rotateX(' + rotateX + 'deg) rotateY(' + rotateY + 'deg)';
                    card.style.transition = 'transform 0.1s ease-out';
                } else {
                    card.style.transform = '';
                    card.style.transition = 'transform 0.5s cubic-bezier(0.34, 1.56, 0.64, 1)';
                }
            });
        });

        mainContent.addEventListener('mouseleave', function () {
            cards.forEach(function (card) {
                card.style.transform = '';
                card.style.transition = 'transform 0.5s cubic-bezier(0.34, 1.56, 0.64, 1)';
            });
        });
    },

    /* ===== Smooth Animated Counter ===== */
    animateValue(element, startVal, endVal, duration, formatter) {
        if (!element) return;

        formatter = formatter || function (v) { return Math.round(v).toString(); };
        duration = duration || 800;

        const startTime = performance.now();

        // Easing function: ease-out-expo
        function easeOutExpo(t) {
            return t === 1 ? 1 : 1 - Math.pow(2, -10 * t);
        }

        function update(currentTime) {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            const easedProgress = easeOutExpo(progress);
            const current = startVal + (endVal - startVal) * easedProgress;

            element.textContent = formatter(current);
            element.classList.add('count-up');

            if (progress < 1) {
                requestAnimationFrame(update);
            } else {
                element.textContent = formatter(endVal);
                element.classList.remove('count-up');
            }
        }

        requestAnimationFrame(update);
    },

    /* ===== Scroll Reveal Animation ===== */
    initScrollReveal() {
        const mainContent = document.getElementById('main-content');
        if (!mainContent) return;

        const observer = new IntersectionObserver(function (entries) {
            entries.forEach(function (entry) {
                if (entry.isIntersecting) {
                    entry.target.style.opacity = '1';
                    entry.target.style.transform = 'translateY(0)';
                    observer.unobserve(entry.target);
                }
            });
        }, {
            threshold: 0.1,
            rootMargin: '0px 0px -40px 0px'
        });

        // Observe cards and sections
        const revealTargets = mainContent.querySelectorAll('.card, .stat-card, .account-card, .filter-bar');
        revealTargets.forEach(function (el) {
            el.style.opacity = '0';
            el.style.transform = 'translateY(20px)';
            el.style.transition = 'opacity 0.6s cubic-bezier(0.34, 1.56, 0.64, 1), transform 0.6s cubic-bezier(0.34, 1.56, 0.64, 1)';
            observer.observe(el);
        });
    },

    /* ===== Button Press Effect ===== */
    initButtonPressEffect() {
        document.addEventListener('mousedown', function (e) {
            const btn = e.target.closest('.btn, .quick-action, .nav-item, .header-btn, .modal-close');
            if (!btn || btn.disabled) return;
            btn.style.transform = 'scale(0.95)';
            btn.style.transition = 'transform 0.1s ease';
        });

        document.addEventListener('mouseup', function (e) {
            const btn = e.target.closest('.btn, .quick-action, .nav-item, .header-btn, .modal-close');
            if (!btn || btn.disabled) return;
            btn.style.transform = '';
            btn.style.transition = 'transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1)';
        });

        document.addEventListener('mouseleave', function (e) {
            const btn = e.target.closest('.btn, .quick-action, .nav-item, .header-btn, .modal-close');
            if (!btn || btn.disabled) return;
            btn.style.transform = '';
            btn.style.transition = 'transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1)';
        });
    },

    /* ===== Input Focus Glow Effect ===== */
    initInputFocusEffect() {
        document.addEventListener('focusin', function (e) {
            const input = e.target.closest('.form-input, .form-select, .form-textarea');
            if (!input) return;

            const formGroup = input.closest('.form-group');
            if (formGroup) {
                const label = formGroup.querySelector('label, .filter-label');
                if (label) {
                    label.style.color = 'var(--color-blue)';
                    label.style.transition = 'color 0.2s ease';
                }
            }
        });

        document.addEventListener('focusout', function (e) {
            const input = e.target.closest('.form-input, .form-select, .form-textarea');
            if (!input) return;

            const formGroup = input.closest('.form-group');
            if (formGroup) {
                const label = formGroup.querySelector('label, .filter-label');
                if (label) {
                    label.style.color = '';
                }
            }
        });
    },

    /* ===== Hover Glow Effect on Interactive Cards ===== */
    initHoverGlowEffect() {
        document.addEventListener('mouseover', function (e) {
            const card = e.target.closest('.card.card-interactive, .account-card');
            if (!card) return;
            card.style.borderColor = 'var(--color-border-glow)';
            card.style.boxShadow = '0 4px 20px rgba(0, 0, 0, 0.3), inset 0 0 0 1px rgba(255,255,255,0.04)';
        });

        document.addEventListener('mouseout', function (e) {
            const card = e.target.closest('.card.card-interactive, .account-card');
            if (!card) return;
            card.style.borderColor = '';
            card.style.boxShadow = '';
        });
    },

    /* ===== Page Transition ===== */
    pageTransition(fromPage, toPage, callback) {
        const fromEl = document.getElementById('page-' + fromPage);
        const toEl = document.getElementById('page-' + toPage);

        if (fromEl) {
            fromEl.style.animation = 'fadeOutSlide 0.2s ease forwards';
        }

        setTimeout(function () {
            if (fromEl) {
                fromEl.style.display = 'none';
                fromEl.style.animation = '';
            }

            if (toEl) {
                toEl.style.display = 'block';
                toEl.style.animation = 'pageEnter 0.45s cubic-bezier(0.34, 1.56, 0.64, 1) forwards';
            }

            if (typeof callback === 'function') {
                callback();
            }
        }, 180);
    },

    /* ===== Notification Badge Bounce ===== */
    bounceBadge(badgeEl) {
        if (!badgeEl) return;
        badgeEl.style.animation = 'none';
        badgeEl.offsetHeight; // force reflow
        badgeEl.style.animation = 'badgePop 0.3s cubic-bezier(0.34, 1.56, 0.64, 1)';
    },

    /* ===== Shake Element (for errors) ===== */
    shake(element) {
        if (!element) return;
        element.style.animation = 'none';
        element.offsetHeight;
        element.style.animation = 'shake 0.5s ease';
    },

    /* ===== Pulse Element ===== */
    pulse(element) {
        if (!element) return;
        element.classList.remove('pulse');
        void element.offsetWidth;
        element.classList.add('pulse');
    }
};

/* ===== Add fadeOutSlide keyframe ===== */
const fadeOutSlideStyle = document.createElement('style');
fadeOutSlideStyle.textContent = `
    @keyframes fadeOutSlide {
        from { opacity: 1; transform: translateY(0); }
        to { opacity: 0; transform: translateY(-12px); }
    }
`;
document.head.appendChild(fadeOutSlideStyle);

/* ===== Initialize animations on DOM ready ===== */
document.addEventListener('DOMContentLoaded', function () {
    Animations.init();
});

/* ===== Re-initialize on page changes (for dynamically loaded content) ===== */
const _origShowPage = App ? App.showPage : null;
if (typeof App !== 'undefined' && App.showPage) {
    const origShowPage = App.showPage.bind(App);
    App._origShowPage = origShowPage;

    // We'll hook into showPage from app.js after it's loaded
    // This is a fallback observer for dynamic content
    const mainContent = document.getElementById('main-content');
    if (mainContent) {
        const mutationObserver = new MutationObserver(function () {
            // Re-apply reveal animations for new content
            setTimeout(function () {
                const cards = mainContent.querySelectorAll('.card:not([data-revealed]), .stat-card:not([data-revealed]), .account-card:not([data-revealed])');
                cards.forEach(function (el) {
                    el.setAttribute('data-revealed', 'true');
                    el.style.opacity = '0';
                    el.style.transform = 'translateY(20px)';
                    el.style.transition = 'opacity 0.6s cubic-bezier(0.34, 1.56, 0.64, 1), transform 0.6s cubic-bezier(0.34, 1.56, 0.64, 1)';
                    requestAnimationFrame(function () {
                        el.style.opacity = '1';
                        el.style.transform = 'translateY(0)';
                    });
                });
            }, 50);
        });

        mutationObserver.observe(mainContent, { childList: true, subtree: true });
    }
}

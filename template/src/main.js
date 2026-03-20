import Alpine from 'alpinejs';

// Make Alpine available globally for inline usage and HTMX reinit
window.Alpine = Alpine;

// Register Alpine data components

// Dropdown component
Alpine.data('dropdown', () => ({
  open: false,
  toggle() { this.open = !this.open; },
  close() { this.open = false; }
}));

// Modal component
Alpine.data('modal', () => ({
  open: false,
  toggle() { this.open = !this.open; },
  show() { this.open = true; },
  close() { this.open = false; },
  init() {
    this.$watch('open', (val) => {
      document.body.style.overflow = val ? 'hidden' : '';
    });
  }
}));

// Tooltip component
Alpine.data('tooltip', () => ({
  show: false,
  text: '',
  init() {
    this.text = this.$el.getAttribute('title') || this.$el.getAttribute('aria-label') || '';
    // Remove title to prevent native tooltip
    this.$el.removeAttribute('title');
  }
}));

// Start Alpine
Alpine.start();

// HTMX integration: reinitialize Alpine trees after HTMX swaps
if (!window._htmxAlpineInit) {
  window._htmxAlpineInit = true;
  document.addEventListener('htmx:afterSettle', function(evt) {
    // Re-initialize Alpine components in the swapped content
    if (window.Alpine && evt.detail && evt.detail.elt) {
      window.Alpine.initTree(evt.detail.elt);
    }
  });
}

// Lazy-load images with class="lazy" (kept from original)
(function(){
  if (!window._lazyObserver) {
    if ('IntersectionObserver' in window) {
      window._lazyObserver = new IntersectionObserver(function(entries, obs) {
        entries.forEach(function(entry) {
          if (entry.isIntersecting) {
            var img = entry.target;
            var src = img.getAttribute('data-src');
            if (src) { img.src = src; img.removeAttribute('data-src'); }
            obs.unobserve(img);
          }
        });
      }, { root: null, rootMargin: '1000px', threshold: 0.01 });
    } else {
      window._lazyObserver = null;
    }
  }
  function observeLazy(root) {
    var imgs = (root || document).querySelectorAll('.lazy[data-src]');
    if (window._lazyObserver) {
      imgs.forEach(function(img){ window._lazyObserver.observe(img); });
    } else {
      imgs.forEach(function(img){
        var src = img.getAttribute('data-src');
        if (src) { img.src = src; img.removeAttribute('data-src'); }
      });
    }
  }
  if (!window._htmxLazyInit) {
    window._htmxLazyInit = true;
    document.addEventListener('DOMContentLoaded', function(){ observeLazy(); });
    document.addEventListener('htmx:afterSettle', function(evt){ observeLazy(evt.detail.elt); });
  }
  observeLazy();
})();

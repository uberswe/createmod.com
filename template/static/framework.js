/*! CreateMod Framework JS — lightweight Bootstrap/Tabler JS replacement */
(function () {
  'use strict';

  // =====================================================================
  // DROPDOWN
  // =====================================================================
  function closeAllDropdowns(except) {
    document.querySelectorAll('.dropdown-menu.show').forEach(function (menu) {
      if (menu === except) return;
      menu.classList.remove('show');
      var toggle = menu.closest('.dropdown');
      if (toggle) {
        var btn = toggle.querySelector('[data-cm-toggle="dropdown"]');
        if (btn) btn.setAttribute('aria-expanded', 'false');
      }
    });
  }

  function toggleDropdown(e) {
    var btn = e.currentTarget;
    var parent = btn.closest('.dropdown') || btn.parentElement;
    var menu = parent.querySelector('.dropdown-menu');
    if (!menu) return;

    e.preventDefault();
    e.stopPropagation();

    var isOpen = menu.classList.contains('show');
    closeAllDropdowns();

    if (!isOpen) {
      menu.classList.add('show');
      btn.setAttribute('aria-expanded', 'true');
    }
  }

  document.addEventListener('click', function (e) {
    var toggle = e.target.closest('[data-cm-toggle="dropdown"]');
    if (toggle) {
      toggleDropdown(e);
      return;
    }
    if (!e.target.closest('.dropdown-menu')) {
      closeAllDropdowns();
    }
  });

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') closeAllDropdowns();
  });

  // =====================================================================
  // COLLAPSE
  // =====================================================================
  function toggleCollapse(targetSel) {
    var target = document.querySelector(targetSel);
    if (!target) return;

    if (target.classList.contains('show')) {
      target.style.height = target.scrollHeight + 'px';
      target.offsetHeight; // force reflow
      target.classList.add('collapsing');
      target.classList.remove('show', 'collapse');
      target.style.height = '';
      target.addEventListener('transitionend', function handler() {
        target.removeEventListener('transitionend', handler);
        target.classList.remove('collapsing');
        target.classList.add('collapse');
      }, { once: true });
    } else {
      target.classList.remove('collapse');
      target.classList.add('collapsing');
      target.style.height = '0';
      target.offsetHeight; // force reflow
      target.style.height = target.scrollHeight + 'px';
      target.addEventListener('transitionend', function handler() {
        target.removeEventListener('transitionend', handler);
        target.classList.remove('collapsing');
        target.classList.add('collapse', 'show');
        target.style.height = '';
      }, { once: true });
    }

    // Toggle aria-expanded on trigger
    document.querySelectorAll('[data-cm-target="' + targetSel + '"]').forEach(function (t) {
      var expanded = target.classList.contains('show') || target.classList.contains('collapsing');
      t.setAttribute('aria-expanded', expanded ? 'false' : 'true');
    });
  }

  document.addEventListener('click', function (e) {
    var trigger = e.target.closest('[data-cm-toggle="collapse"]');
    if (!trigger) return;
    e.preventDefault();
    var targetSel = trigger.getAttribute('data-cm-target') || trigger.getAttribute('href');
    if (targetSel) toggleCollapse(targetSel);
  });

  // =====================================================================
  // MODAL
  // =====================================================================
  var activeModal = null;
  var modalBackdrop = null;

  function showModal(modalEl) {
    if (activeModal) hideModal(activeModal);

    modalBackdrop = document.createElement('div');
    modalBackdrop.className = 'modal-backdrop show';
    document.body.appendChild(modalBackdrop);

    modalEl.style.display = 'block';
    modalEl.offsetHeight; // force reflow
    modalEl.classList.add('show');
    modalEl.setAttribute('aria-modal', 'true');
    modalEl.setAttribute('role', 'dialog');
    document.body.style.overflow = 'hidden';
    activeModal = modalEl;

    modalBackdrop.addEventListener('click', function () {
      hideModal(modalEl);
    });

    modalEl.addEventListener('click', function (e) {
      if (e.target === modalEl) hideModal(modalEl);
    });
  }

  function hideModal(modalEl) {
    if (!modalEl) return;
    modalEl.classList.remove('show');
    modalEl.removeAttribute('aria-modal');
    modalEl.removeAttribute('role');

    setTimeout(function () {
      modalEl.style.display = 'none';
    }, 150);

    if (modalBackdrop && modalBackdrop.parentNode) {
      modalBackdrop.parentNode.removeChild(modalBackdrop);
      modalBackdrop = null;
    }

    document.body.style.overflow = '';
    activeModal = null;

    var event = new CustomEvent('hidden.bs.modal');
    modalEl.dispatchEvent(event);
  }

  document.addEventListener('click', function (e) {
    var trigger = e.target.closest('[data-cm-toggle="modal"]');
    if (trigger) {
      e.preventDefault();
      var targetSel = trigger.getAttribute('data-cm-target') || trigger.getAttribute('href');
      var modalEl = document.querySelector(targetSel);
      if (modalEl) showModal(modalEl);
      return;
    }

    var dismiss = e.target.closest('[data-cm-dismiss="modal"]');
    if (dismiss) {
      e.preventDefault();
      var modal = dismiss.closest('.modal');
      if (modal) hideModal(modal);
      return;
    }
  });

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape' && activeModal) hideModal(activeModal);
  });

  // Expose for programmatic use
  window.cmModal = {
    show: function (sel) {
      var el = typeof sel === 'string' ? document.querySelector(sel) : sel;
      if (el) showModal(el);
    },
    hide: function (sel) {
      var el = typeof sel === 'string' ? document.querySelector(sel) : sel;
      if (el) hideModal(el);
    }
  };

  // =====================================================================
  // TABS
  // =====================================================================
  document.addEventListener('click', function (e) {
    var trigger = e.target.closest('[data-cm-toggle="tab"], [data-cm-toggle="pill"]');
    if (!trigger) return;
    e.preventDefault();

    var targetSel = trigger.getAttribute('data-cm-target') || trigger.getAttribute('href');
    if (!targetSel) return;

    // Deactivate current active tab in the same nav
    var nav = trigger.closest('.nav, .card-header-tabs');
    if (nav) {
      nav.querySelectorAll('.nav-link').forEach(function (link) {
        link.classList.remove('active');
        link.setAttribute('aria-selected', 'false');
      });
    }

    // Activate clicked tab
    trigger.classList.add('active');
    trigger.setAttribute('aria-selected', 'true');

    // Switch tab pane
    var targetPane = document.querySelector(targetSel);
    if (targetPane) {
      var tabContent = targetPane.closest('.tab-content');
      if (tabContent) {
        tabContent.querySelectorAll('.tab-pane').forEach(function (pane) {
          pane.classList.remove('show', 'active');
        });
      }
      targetPane.classList.add('show', 'active');
    }
  });

  // =====================================================================
  // ALERT DISMISS
  // =====================================================================
  document.addEventListener('click', function (e) {
    var dismiss = e.target.closest('[data-cm-dismiss="alert"]');
    if (!dismiss) return;
    var alert = dismiss.closest('.alert');
    if (alert) {
      alert.style.opacity = '0';
      alert.style.transition = 'opacity 0.15s ease';
      setTimeout(function () { alert.remove(); }, 150);
    }
  });

  // =====================================================================
  // TOOLTIP (basic — title attribute based)
  // =====================================================================
  var tooltipEl = null;

  function showTooltipEl(target) {
    var title = target.getAttribute('data-cm-title') || target.getAttribute('title');
    if (!title) return;

    // Prevent native tooltip
    if (target.getAttribute('title')) {
      target.setAttribute('data-cm-title', title);
      target.removeAttribute('title');
    }

    tooltipEl = document.createElement('div');
    tooltipEl.className = 'tooltip';
    tooltipEl.textContent = title;
    tooltipEl.style.display = 'block';
    document.body.appendChild(tooltipEl);

    var rect = target.getBoundingClientRect();
    var placement = target.getAttribute('data-cm-placement') || 'top';
    var tt = tooltipEl.getBoundingClientRect();

    var top, left;
    if (placement === 'bottom') {
      top = rect.bottom + 4 + window.scrollY;
      left = rect.left + rect.width / 2 - tt.width / 2 + window.scrollX;
    } else if (placement === 'left') {
      top = rect.top + rect.height / 2 - tt.height / 2 + window.scrollY;
      left = rect.left - tt.width - 4 + window.scrollX;
    } else if (placement === 'right') {
      top = rect.top + rect.height / 2 - tt.height / 2 + window.scrollY;
      left = rect.right + 4 + window.scrollX;
    } else {
      top = rect.top - tt.height - 4 + window.scrollY;
      left = rect.left + rect.width / 2 - tt.width / 2 + window.scrollX;
    }

    tooltipEl.style.top = top + 'px';
    tooltipEl.style.left = left + 'px';
  }

  function hideTooltipEl() {
    if (tooltipEl && tooltipEl.parentNode) {
      tooltipEl.parentNode.removeChild(tooltipEl);
      tooltipEl = null;
    }
  }

  document.addEventListener('mouseenter', function (e) {
    var target = e.target.closest('[data-cm-toggle="tooltip"]');
    if (target) showTooltipEl(target);
  }, true);

  document.addEventListener('mouseleave', function (e) {
    var target = e.target.closest('[data-cm-toggle="tooltip"]');
    if (target) hideTooltipEl();
  }, true);

  document.addEventListener('focusin', function (e) {
    var target = e.target.closest('[data-cm-toggle="tooltip"]');
    if (target) showTooltipEl(target);
  }, true);

  document.addEventListener('focusout', function (e) {
    var target = e.target.closest('[data-cm-toggle="tooltip"]');
    if (target) hideTooltipEl();
  }, true);

  // =====================================================================
  // Re-initialize on HTMX content swaps
  // =====================================================================
  document.addEventListener('htmx:afterSettle', function () {
    closeAllDropdowns();
  });

})();

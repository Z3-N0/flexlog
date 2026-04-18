htmx.config.includeIndicatorStyles = false;
htmx.config.attributesToSettle = ["class", "width", "height"];

addEventListener("DOMContentLoaded", () => {

  // ---- Theme ----
  // Dark is default - :root defines dark values, no attribute needed.
  // Light mode sets data-theme="light" to trigger the override block.
  Alpine.data('themeChanger', () => ({
    isDark: true,

    init() {
      const saved = localStorage.getItem('theme');
      this.isDark = saved !== 'light'; // default to dark if nothing saved
      this.applyTheme();
    },

    toggle() {
      this.isDark = !this.isDark;
      localStorage.setItem('theme', this.isDark ? 'dark' : 'light');
      this.applyTheme();
    },

    applyTheme() {
      if (this.isDark) {
        document.body.removeAttribute('data-theme');
      } else {
        document.body.setAttribute('data-theme', 'light');
      }
    }
  }));

  // ---- Leftbar ----
  Alpine.data('leftbar', () => ({
    expanded: true,
    fullyExpanded: true,
    selectionChanged: false,

    initLeftbar() {
      const el = document.querySelector('#leftbar');
      if (!el) return;

      el.addEventListener('transitionend', () => {
        this.fullyExpanded = this.expanded;
      });
    },

    expand() {
      this.expanded = true;
      this.fullyExpanded = false;
      const el = document.querySelector('#leftbar');
      if (!el) return;
      el.addEventListener('transitionend', () => {
        if (this.expanded) this.fullyExpanded = true;
      }, { once: true });
    },

    collapse() {
      this.expanded = false;
      this.fullyExpanded = false;
    },

    isExpanded() {
      return this.expanded && this.fullyExpanded;
    },

    isCollapsed() {
      return !this.expanded && !this.fullyExpanded;
    },

    onSelectionChange() {
      this.selectionChanged = true;
    },
  }));

  // ---- Log row expand ----
  Alpine.data('logRow', () => ({
    expanded: false,

    toggle() {
      this.expanded = !this.expanded;
    }
  }));

});

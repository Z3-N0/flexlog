htmx.config.includeIndicatorStyles = false;
htmx.config.attributesToSettle = ["class", "width", "height"];

addEventListener("DOMContentLoaded", () => {
  // ---- Theme ----
  // Dark is default - :root defines dark values, no attribute needed.
  // Light mode sets data-theme="light" to trigger the override block.
  Alpine.data("themeChanger", () => ({
    isDark: true,

    init() {
      const saved = localStorage.getItem("theme");
      this.isDark = saved !== "light"; // default to dark if nothing saved
      this.applyTheme();
    },

    toggle() {
      this.isDark = !this.isDark;
      localStorage.setItem("theme", this.isDark ? "dark" : "light");
      this.applyTheme();
    },

    applyTheme() {
      if (this.isDark) {
        document.body.removeAttribute("data-theme");
      } else {
        document.body.setAttribute("data-theme", "light");
      }
    },
  }));

  Alpine.data("changeTracker", () => ({
    init() {
      const inps = this.$root.querySelectorAll("input");
      if (inps.length <= 0) {
        return;
      }

      inps.forEach((inp) =>
        inp.addEventListener("change", () => this.checkChange()),
      );

      if (this.$root.dataset.forceUpdate == "true") {
        this.$root.addEventListener("htmx:afterRequest", () =>
          this.resetOrigValues(),
        );
      }
    },

    checkChange() {
      const applyBtn = document.querySelector(this.$root.dataset.btnId);
      if (!applyBtn) return;

      const inps = this.$root.querySelectorAll("input");
      const anyChanged = Array.from(inps).some((inp) => {
        if (inp.type === "checkbox") {
          return inp.checked !== (inp.dataset.origValue === "true");
        }
        return inp.value !== inp.dataset.origValue;
      });

      anyChanged
        ? this.setApplyState(applyBtn, true)
        : this.setApplyState(applyBtn, false);
    },

    resetOrigValues() {
      this.$root.querySelectorAll("input").forEach((inp) => {
        inp.dataset.origValue =
          inp.type === "checkbox" ? String(inp.checked) : inp.value;
      });
      const applyBtn = document.querySelector(this.$root.dataset.btnId);
      if (applyBtn) this.setApplyState(applyBtn, false);
    },

    setApplyState(applyBtn, enabled) {
      if (enabled) {
        applyBtn.removeAttribute("disabled");
        applyBtn.style.backgroundColor = "var(--accent)";
        applyBtn.style.color = "#fff";
        applyBtn.style.opacity = "1";
        applyBtn.style.cursor = "pointer";
      } else {
        applyBtn.setAttribute("disabled", "");
        applyBtn.style.backgroundColor = "var(--bg-raised)";
        applyBtn.style.color = "var(--text-muted)";
        applyBtn.style.opacity = "0.5";
        applyBtn.style.cursor = "not-allowed";
      }
    },
  }));

  // ---- Leftbar ----
  Alpine.data("leftbar", () => ({
    expanded: true,
    fullyExpanded: true,

    initLeftbar() {
      const el = document.querySelector("#leftbar");
      if (!el) return;

      el.addEventListener("transitionend", () => {
        this.fullyExpanded = this.expanded;
      });
    },

    expand() {
      this.expanded = true;
      this.fullyExpanded = false;
      const el = document.querySelector("#leftbar");
      if (!el) return;
      el.addEventListener(
        "transitionend",
        () => {
          if (this.expanded) this.fullyExpanded = true;
        },
        { once: true },
      );
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
  }));

  // ---- Log row expand ----
  Alpine.data("logRow", () => ({
    expanded: false,

    toggle() {
      this.expanded = !this.expanded;
    },
  }));
});

// ---- Filters ----
function getLevelStyle(lvl) {
  const colors = {
    TRACE: "#6b6b80",
    DEBUG: "#4a9eff",
    INFO: "#34d399",
    WARN: "#fbbf24",
    ERROR: "#f87171",
    FATAL: "#dc2626",
  };
  const c = colors[lvl] || "#6b6b80";
  return `color: ${c}; border-color: ${c};`;
}

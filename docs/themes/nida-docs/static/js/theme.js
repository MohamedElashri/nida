// Nida Docs theme behavior. Everything here is progressive enhancement:
// pages are fully readable without JavaScript.
(function () {
  "use strict";

  var root = document.documentElement;

  // Dark mode toggle. The initial theme is set inline in <head> to avoid a
  // flash; this only handles changes.
  document.querySelectorAll("[data-theme-toggle]").forEach(function (button) {
    button.addEventListener("click", function () {
      var next = root.getAttribute("data-theme") === "dark" ? "light" : "dark";
      root.setAttribute("data-theme", next);
      try {
        localStorage.setItem("nida-theme", next);
      } catch (e) {}
    });
  });

  // Mobile sidebar drawer.
  var navToggle = document.querySelector("[data-nav-toggle]");
  if (navToggle) {
    navToggle.addEventListener("click", function () {
      var open = document.body.classList.toggle("nav-open");
      navToggle.setAttribute("aria-expanded", String(open));
    });
    document.addEventListener("click", function (event) {
      if (!document.body.classList.contains("nav-open")) return;
      if (event.target.closest("#sidebar") || event.target.closest("[data-nav-toggle]")) return;
      document.body.classList.remove("nav-open");
      navToggle.setAttribute("aria-expanded", "false");
    });
  }

  // Keep the active sidebar link in view.
  // Scroll only the sidebar, never the page.
  var sidebar = document.querySelector(".sidebar");
  var activeLink = sidebar && sidebar.querySelector(".nav-link.active");
  if (activeLink) {
    var offset = activeLink.offsetTop - sidebar.clientHeight / 2;
    if (offset > 0) sidebar.scrollTop = offset;
  }

  // Copy buttons.
  function copyText(text, button) {
    var done = function () {
      button.classList.add("copied");
      button.setAttribute("aria-label", copiedLabel);
      setTimeout(function () {
        button.classList.remove("copied");
        button.setAttribute("aria-label", copyLabel);
      }, 1600);
    };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(done, function () {});
    }
  }

  document.querySelectorAll("[data-copy]").forEach(function (button) {
    button.addEventListener("click", function () {
      var source = button.closest("[data-copy-root]").querySelector("[data-copy-source]");
      copyText(source.textContent.trim(), button);
    });
  });

  var copyIcon = document.querySelector("[data-copy] svg");
  var copyLabel = document.body.dataset.copyLabel || "Copy";
  var copiedLabel = document.body.dataset.copiedLabel || "Copied";
  document.querySelectorAll(".prose pre").forEach(function (pre) {
    if (pre.classList.contains("mermaid")) return;
    var wrap = document.createElement("div");
    wrap.className = "code-wrap";
    pre.parentNode.insertBefore(wrap, pre);
    wrap.appendChild(pre);
    var button = document.createElement("button");
    button.type = "button";
    button.className = "icon-button code-copy";
    button.setAttribute("aria-label", copyLabel);
    if (copyIcon) button.appendChild(copyIcon.cloneNode(true));
    else button.textContent = copyLabel;
    button.addEventListener("click", function () {
      var lines = pre.querySelectorAll(".z-cl");
      var text = lines.length
        ? Array.prototype.map.call(lines, function (line) { return line.textContent; }).join("")
        : pre.textContent;
      copyText(text.replace(/\n$/, ""), button);
    });
    wrap.appendChild(button);
  });

  // Tabs from the tabs/tab shortcodes. The chosen label is remembered so
  // every tab group on the site shows the same option, such as "Homebrew".
  var savedTab = null;
  try {
    savedTab = localStorage.getItem("nida-tab");
  } catch (e) {}

  document.querySelectorAll("[data-tabs]").forEach(function (tabs, groupIndex) {
    var panels = Array.prototype.filter.call(tabs.children, function (el) {
      return el.classList.contains("tab-panel");
    });
    if (panels.length === 0) return;
    var list = document.createElement("div");
    list.className = "tab-list";
    list.setAttribute("role", "tablist");
    var buttons = panels.map(function (panel, i) {
      var id = "tab-" + groupIndex + "-" + i;
      panel.id = id + "-panel";
      panel.setAttribute("role", "tabpanel");
      panel.setAttribute("aria-labelledby", id);
      var button = document.createElement("button");
      button.type = "button";
      button.id = id;
      button.className = "tab-button";
      button.setAttribute("role", "tab");
      button.setAttribute("aria-controls", panel.id);
      button.textContent = panel.getAttribute("data-tab-label");
      list.appendChild(button);
      return button;
    });

    function select(index, remember) {
      buttons.forEach(function (button, i) {
        var selected = i === index;
        button.setAttribute("aria-selected", String(selected));
        button.tabIndex = selected ? 0 : -1;
        panels[i].hidden = !selected;
      });
      if (remember) {
        try {
          localStorage.setItem("nida-tab", buttons[index].textContent);
        } catch (e) {}
      }
    }

    buttons.forEach(function (button, i) {
      button.addEventListener("click", function () {
        select(i, true);
      });
      button.addEventListener("keydown", function (event) {
        var delta = event.key === "ArrowRight" ? 1 : event.key === "ArrowLeft" ? -1 : 0;
        if (!delta) return;
        var next = (i + delta + buttons.length) % buttons.length;
        select(next, true);
        buttons[next].focus();
      });
    });

    tabs.insertBefore(list, tabs.firstChild);
    tabs.classList.add("is-ready");
    var initial = buttons.findIndex(function (b) { return b.textContent === savedTab; });
    select(initial >= 0 ? initial : 0, false);
  });

  // Highlight the TOC entry for the heading currently in view.
  var toc = document.querySelector("[data-toc]");
  if (toc && "IntersectionObserver" in window) {
    var links = {};
    toc.querySelectorAll("a[href^='#']").forEach(function (link) {
      links[decodeURIComponent(link.getAttribute("href").slice(1))] = link;
    });
    var headings = Array.prototype.filter.call(document.querySelectorAll(".prose h2[id], .prose h3[id]"), function (h) {
      return links[h.id];
    });
    var visible = new Set();
    var setActive = function (id) {
      Object.keys(links).forEach(function (key) {
        links[key].classList.toggle("active", key === id);
      });
    };
    var observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (entry.isIntersecting) visible.add(entry.target.id);
        else visible.delete(entry.target.id);
      });
      var first = headings.find(function (h) { return visible.has(h.id); });
      if (first) setActive(first.id);
    }, { rootMargin: "-70px 0px -65% 0px" });
    headings.forEach(function (h) { observer.observe(h); });
  }

  // Search dialog. The index is loaded on first open.
  var dialog = document.querySelector("[data-search-dialog]");
  if (!dialog) return;
  var input = dialog.querySelector("[data-search-input]");
  var results = dialog.querySelector("[data-search-results]");
  var empty = dialog.querySelector("[data-search-empty]");
  var basePath = dialog.getAttribute("data-base-path") || "";
  var entries = null;
  var selected = 0;

  function loadIndex(callback) {
    if (entries) return callback();
    var script = document.createElement("script");
    script.src = dialog.getAttribute("data-index");
    script.onload = function () {
      var docs = (window.searchIndex && window.searchIndex.documentStore.docs) || {};
      entries = Object.keys(docs).map(function (url) {
        var doc = docs[url];
        var section = url.split("/").filter(Boolean)[0] || "";
        return {
          url: url,
          section: section.replace(/-/g, " "),
          title: doc.title || url,
          titleLower: (doc.title || "").toLowerCase(),
          description: doc.description || "",
          body: doc.body || "",
          bodyLower: ((doc.description || "") + " " + (doc.body || "")).toLowerCase()
        };
      });
      callback();
    };
    document.head.appendChild(script);
  }

  function escapeHTML(text) {
    return text.replace(/[&<>"]/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c];
    });
  }

  function highlight(text, terms) {
    var html = escapeHTML(text);
    terms.forEach(function (term) {
      var pattern = new RegExp("(" + term.replace(/[.*+?^${}()|[\]\\]/g, "\\$&") + ")", "gi");
      html = html.replace(pattern, "<mark>$1</mark>");
    });
    return html;
  }

  // Prefer the description when it mentions a term; otherwise show the
  // body text around the first match.
  function snippet(entry, terms) {
    var sources = [entry.description, entry.body];
    for (var s = 0; s < sources.length; s++) {
      var text = sources[s];
      if (!text) continue;
      var lower = text.toLowerCase();
      for (var t = 0; t < terms.length; t++) {
        var at = lower.indexOf(terms[t]);
        if (at === -1) continue;
        if (s === 0) return text;
        var start = Math.max(0, at - 50);
        return (start > 0 ? "…" : "") + text.slice(start, start + 160) + (start + 160 < text.length ? "…" : "");
      }
    }
    return entry.description || entry.body.slice(0, 140);
  }

  function occurrences(haystack, needle) {
    var count = 0;
    for (var at = haystack.indexOf(needle); at !== -1 && count < 25; at = haystack.indexOf(needle, at + needle.length)) {
      count++;
    }
    return count;
  }

  // Every term must appear somewhere. Title matches weigh most, then how
  // often the term appears in the body, with diminishing returns.
  function search(query) {
    var terms = query.toLowerCase().split(/\s+/).filter(Boolean);
    if (!terms.length) return [];
    return entries.map(function (entry) {
      var score = 0;
      for (var i = 0; i < terms.length; i++) {
        var term = terms[i];
        var titleAt = entry.titleLower.indexOf(term);
        var hits = occurrences(entry.bodyLower, term);
        if (titleAt === -1 && hits === 0) return null;
        if (titleAt !== -1) score += titleAt === 0 ? 30 : 20;
        score += Math.sqrt(hits) * 3;
      }
      return { entry: entry, score: score, terms: terms };
    }).filter(Boolean).sort(function (a, b) {
      return b.score - a.score || a.entry.title.localeCompare(b.entry.title);
    }).slice(0, 10);
  }

  function render() {
    var matches = search(input.value);
    results.replaceChildren();
    selected = 0;
    matches.forEach(function (match, i) {
      var item = document.createElement("li");
      item.className = "search-result";
      var link = document.createElement("a");
      link.href = basePath + match.entry.url;
      link.setAttribute("role", "option");
      link.innerHTML =
        (match.entry.section ? '<span class="search-result-section">' + escapeHTML(match.entry.section) + "</span>" : "") +
        '<span class="search-result-title">' + highlight(match.entry.title, match.terms) + "</span>" +
        '<span class="search-result-snippet">' + highlight(snippet(match.entry, match.terms), match.terms) + "</span>";
      link.addEventListener("mousemove", function () {
        select(i);
      });
      item.appendChild(link);
      results.appendChild(item);
    });
    empty.hidden = matches.length > 0 || input.value.trim() === "";
    select(0);
  }

  function select(index) {
    var links = results.querySelectorAll("a");
    if (!links.length) return;
    selected = (index + links.length) % links.length;
    links.forEach(function (link, i) {
      link.setAttribute("aria-selected", String(i === selected));
    });
    links[selected].scrollIntoView({ block: "nearest" });
  }

  function open() {
    if (dialog.open) return;
    dialog.showModal();
    input.select();
    loadIndex(render);
  }

  document.querySelectorAll("[data-search-open]").forEach(function (button) {
    button.addEventListener("click", open);
  });

  var isMac = /Mac|iPhone|iPad/.test(navigator.platform || "");
  var keyHint = document.querySelector("[data-search-key]");
  if (keyHint) keyHint.textContent = isMac ? "⌘K" : "Ctrl K";

  document.addEventListener("keydown", function (event) {
    var typing = /INPUT|TEXTAREA|SELECT/.test(document.activeElement.tagName) || document.activeElement.isContentEditable;
    if ((event.key === "k" && (event.metaKey || event.ctrlKey)) || (event.key === "/" && !typing)) {
      event.preventDefault();
      open();
    }
  });

  input.addEventListener("input", function () {
    if (entries) render();
  });
  input.addEventListener("keydown", function (event) {
    if (event.key === "Escape") {
      // A search input would otherwise clear itself instead of closing.
      event.preventDefault();
      dialog.close();
    } else if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      select(selected + (event.key === "ArrowDown" ? 1 : -1));
    } else if (event.key === "Enter") {
      var link = results.querySelectorAll("a")[selected];
      if (link) {
        event.preventDefault();
        window.location.href = link.href;
      }
    }
  });
  dialog.addEventListener("click", function (event) {
    if (event.target === dialog) dialog.close();
  });
})();

// Renders Nida's mermaid blocks, matching the page's dark mode if it has one.
(function () {
  var theme = document.documentElement.getAttribute("data-theme");
  var dark = theme ? theme === "dark" : window.matchMedia("(prefers-color-scheme: dark)").matches;
  mermaid.initialize({ startOnLoad: false, theme: dark ? "dark" : "default" });
  mermaid.run({ querySelector: "pre.mermaid" });
})();

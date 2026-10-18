// Renders Nida's math spans, which wrap TeX in \( \) or \[ \].
document.querySelectorAll(".math").forEach(function (el) {
  var tex = el.textContent.trim().replace(/^\\[([]/, "").replace(/\\[)\]]$/, "");
  katex.render(tex, el, { displayMode: el.classList.contains("math-display"), throwOnError: false });
});

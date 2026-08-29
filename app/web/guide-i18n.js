(() => {
  const language = String(new URLSearchParams(location.search).get("lang") || "pl").toLowerCase().split("-")[0];
  if (language === "pl") return;

  const normalize = value => String(value || "").replace(/\s+/g, " ").trim();

  async function loadTranslations() {
    const response = await fetch(`/locale/guide/${encodeURIComponent(language)}`, { cache: "no-store" });
    if (!response.ok) throw new Error(`Missing guide translation ${language}: HTTP ${response.status}`);
    const translations = await response.json();
    if (!translations || typeof translations !== "object" || Array.isArray(translations)) {
      throw new Error(`Invalid guide translation ${language}`);
    }
    return translations;
  }

  function applyTranslations(translations) {
    const translate = source => translations[normalize(source)] || source;
    const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT, {
      acceptNode(node) {
        const parent = node.parentElement;
        if (!parent || ["SCRIPT", "STYLE", "CODE", "PRE"].includes(parent.tagName)) return NodeFilter.FILTER_REJECT;
        return normalize(node.nodeValue) ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_REJECT;
      }
    });
    const nodes = [];
    while (walker.nextNode()) nodes.push(walker.currentNode);
    for (const node of nodes) {
      const source = normalize(node.nodeValue);
      const translated = translations[source];
      if (!translated) continue;
      const leading = (node.nodeValue.match(/^\s*/) || [""])[0];
      const trailing = (node.nodeValue.match(/\s*$/) || [""])[0];
      node.nodeValue = `${leading}${translated}${trailing}`;
    }

    for (const element of document.querySelectorAll("[title],[aria-label],[alt],[placeholder]")) {
      for (const attribute of ["title", "aria-label", "alt", "placeholder"]) {
        if (!element.hasAttribute(attribute)) continue;
        const source = normalize(element.getAttribute(attribute));
        if (translations[source]) element.setAttribute(attribute, translations[source]);
      }
    }

    document.documentElement.lang = language;
    document.documentElement.dir = ["ar", "he"].includes(language) ? "rtl" : "ltr";
    document.body.classList.toggle("rtl-guide", ["ar", "he"].includes(language));
    const titleSource = normalize(document.title);
    if (translations[titleSource]) document.title = translations[titleSource];
    document.documentElement.dataset.guideI18n = "ready";
  }

  loadTranslations().then(applyTranslations).catch(error => {
    document.documentElement.dataset.guideI18n = "error";
    console.error(error);
    const message = document.createElement("div");
    message.style.cssText = "position:fixed;left:12px;right:12px;bottom:12px;z-index:99999;padding:10px 14px;background:#7f1d1d;color:#fff;border-radius:8px;font:13px system-ui";
    message.textContent = `Translation package error (${language})`;
    document.body.appendChild(message);
  });
})();

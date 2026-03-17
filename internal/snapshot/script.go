package snapshot

const DistillScript = `
() => {
  const limit = 120;
  const viewport = {
    width: window.innerWidth || document.documentElement.clientWidth || 0,
    height: window.innerHeight || document.documentElement.clientHeight || 0
  };

  document.querySelectorAll('[agentium-id]').forEach((node) => node.removeAttribute('agentium-id'));

  const roleFor = (el) => {
    const explicit = (el.getAttribute('role') || '').trim();
    if (explicit) return explicit;
    const tag = el.tagName.toLowerCase();
    if (tag === 'a') return 'link';
    if (tag === 'button') return 'button';
    if (tag === 'input') {
      const type = (el.getAttribute('type') || 'text').toLowerCase();
      if (type === 'checkbox') return 'checkbox';
      return 'input';
    }
    if (tag === 'textarea' || tag === 'select') return 'input';
    if (tag === 'img') return 'img';
    return 'text';
  };

  const interactable = (el, role) => {
    if (el.disabled) return false;
    if (['button', 'link', 'input', 'checkbox'].includes(role)) return true;
    return !!el.onclick || el.tabIndex >= 0;
  };

  const readableText = (el) => {
    const text = (el.innerText || el.textContent || '').replace(/\s+/g, ' ').trim();
    if (text) return text.slice(0, 160);
    const aria = (el.getAttribute('aria-label') || '').trim();
    if (aria) return aria.slice(0, 160);
    const alt = (el.getAttribute('alt') || '').trim();
    return alt.slice(0, 160);
  };

  const attrMap = (el) => {
    const attrs = {};
    ['aria-label', 'type', 'name', 'placeholder', 'href', 'value'].forEach((key) => {
      const value = el.getAttribute(key);
      if (value) attrs[key] = value.slice(0, 200);
    });
    return attrs;
  };

  const isVisible = (el, rect) => {
    const style = window.getComputedStyle(el);
    if (!style) return false;
    if (style.display === 'none') return false;
    if (style.visibility === 'hidden') return false;
    if (Number(style.opacity) === 0) return false;
    if (rect.width < 2 || rect.height < 2) return false;
    if (rect.bottom < 0 || rect.right < 0) return false;
    if (rect.top > viewport.height || rect.left > viewport.width) return false;
    return true;
  };

  const items = [];
  let refId = 1;

  for (const el of document.querySelectorAll('body *')) {
    if (items.length >= limit) break;
    if (!(el instanceof HTMLElement)) continue;

    const rect = el.getBoundingClientRect();
    if (!isVisible(el, rect)) continue;

    const role = roleFor(el);
    const text = readableText(el);
    const canInteract = interactable(el, role);
    if (!text && !canInteract) continue;

    el.setAttribute('agentium-id', String(refId));
    items.push({
      ref_id: refId,
      role,
      text,
      interactable: canInteract,
      attributes: attrMap(el),
      bbox: {
        x: Math.max(0, rect.x),
        y: Math.max(0, rect.y),
        w: Math.max(0, rect.width),
        h: Math.max(0, rect.height)
      }
    });
    refId += 1;
  }

  return {
    url: location.href,
    title: document.title,
    viewport,
    elements: items
  };
}
`

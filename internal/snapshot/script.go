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
  const candidates = [];

  const scoreFor = (el, role, text, canInteract, rect) => {
    let score = 0;
    if (canInteract) score += 1000;
    if (role === 'button' || role === 'link') score += 300;
    if (role === 'input' || role === 'checkbox') score += 250;
    if (text) score += Math.min(text.length, 120);
    if (el.getAttribute('aria-label')) score += 120;
    if (el.tagName.toLowerCase() === 'img') score += 50;
    score += Math.min(rect.width * rect.height, 5000) / 100;

    const centerX = rect.left + rect.width / 2;
    const centerY = rect.top + rect.height / 2;
    const viewportCenterX = viewport.width / 2;
    const viewportCenterY = viewport.height / 2;
    const distance = Math.hypot(centerX - viewportCenterX, centerY - viewportCenterY);
    score -= Math.min(distance / 10, 100);

    return score;
  };

  for (const el of document.querySelectorAll('body *')) {
    if (!(el instanceof HTMLElement)) continue;

    const rect = el.getBoundingClientRect();
    if (!isVisible(el, rect)) continue;

    const role = roleFor(el);
    const text = readableText(el);
    const canInteract = interactable(el, role);
    if (!text && !canInteract) continue;

    candidates.push({
      el,
      score: scoreFor(el, role, text, canInteract, rect),
      data: {
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
      }
    });
  }

  candidates
    .sort((a, b) => b.score - a.score)
    .slice(0, limit)
    .forEach((candidate, index) => {
      const refId = index + 1;
      candidate.el.setAttribute('agentium-id', String(refId));
      items.push({
        ref_id: refId,
        ...candidate.data
      });
    });

  items.sort((a, b) => a.ref_id - b.ref_id);
  for (const item of items) {
    delete item.__score;
  }

  return {
    url: location.href,
    title: document.title,
    viewport,
    elements: items
  };
}
`

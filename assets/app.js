const navToggle = document.querySelector('.nav-toggle');
const sidebar = document.querySelector('.sidebar');
const sidebarBackdrop = document.querySelector('.sidebar-backdrop');

function setSidebar(open) {
  sidebar?.classList.toggle('open', open);
  navToggle?.setAttribute('aria-expanded', String(open));
  document.body.classList.toggle('nav-open', open && window.innerWidth <= 900);
  if (sidebarBackdrop) sidebarBackdrop.hidden = !open;
}

navToggle?.addEventListener('click', () => setSidebar(!sidebar.classList.contains('open')));
sidebarBackdrop?.addEventListener('click', () => setSidebar(false));

sidebar?.addEventListener('click', (event) => {
  if (event.target.closest('a') && window.innerWidth <= 900) setSidebar(false);
});

window.addEventListener('resize', () => {
  if (window.innerWidth > 900) setSidebar(false);
});

document.querySelectorAll('.nav-group-toggle').forEach((button) => {
  button.addEventListener('click', () => {
    const expanded = button.getAttribute('aria-expanded') === 'true';
    button.setAttribute('aria-expanded', String(!expanded));
  });
});

document.querySelectorAll('.copy-button').forEach((button) => {
  button.addEventListener('click', async () => {
    const code = button.closest('.code-block')?.querySelector('code')?.textContent;
    if (!code) return;

    try {
      if (navigator.clipboard) {
        await navigator.clipboard.writeText(code);
      } else {
        copyWithTextArea(code);
      }
      button.textContent = '已复制';
    } catch {
      try {
        copyWithTextArea(code);
        button.textContent = '已复制';
      } catch {
        button.textContent = '复制失败';
      }
    }
    window.setTimeout(() => { button.textContent = '复制'; }, 1400);
  });
});

function copyWithTextArea(text) {
  const textArea = document.createElement('textarea');
  textArea.value = text;
  textArea.setAttribute('readonly', '');
  textArea.style.position = 'fixed';
  textArea.style.opacity = '0';
  document.body.appendChild(textArea);
  textArea.select();
  const copied = document.execCommand('copy');
  textArea.remove();
  if (!copied) throw new Error('Copy command was rejected');
}

document.querySelectorAll('[data-tabs]').forEach((tabs) => {
  const buttons = [...tabs.querySelectorAll('[role="tab"]')];
  const panels = [...tabs.querySelectorAll('[role="tabpanel"]')];

  function selectTab(selectedButton) {
    buttons.forEach((button) => {
      const selected = button === selectedButton;
      button.setAttribute('aria-selected', String(selected));
      button.tabIndex = selected ? 0 : -1;
    });
    panels.forEach((panel) => {
      panel.hidden = panel.id !== selectedButton.getAttribute('aria-controls');
    });
  }

  buttons.forEach((button, index) => {
    button.addEventListener('click', () => selectTab(button));
    button.addEventListener('keydown', (event) => {
      if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
      event.preventDefault();
      let nextIndex = index;
      if (event.key === 'ArrowLeft') nextIndex = (index - 1 + buttons.length) % buttons.length;
      if (event.key === 'ArrowRight') nextIndex = (index + 1) % buttons.length;
      if (event.key === 'Home') nextIndex = 0;
      if (event.key === 'End') nextIndex = buttons.length - 1;
      selectTab(buttons[nextIndex]);
      buttons[nextIndex].focus();
    });
  });

  selectTab(buttons.find((button) => button.getAttribute('aria-selected') === 'true') || buttons[0]);
});

const searchInput = document.querySelector('#doc-search');
const searchResults = document.querySelector('#search-results');
const searchItems = [...document.querySelectorAll('article section[id]')].map((section) => {
  const title = section.dataset.searchTitle || section.querySelector('h1, h2')?.textContent.trim() || section.id;
  const text = section.textContent.replace(/\s+/g, ' ').trim();
  return { id: section.id, title, text, searchable: `${title} ${text}`.toLocaleLowerCase('zh-CN') };
});

function closeSearch() {
  if (!searchResults || !searchInput) return;
  searchResults.hidden = true;
  searchInput.setAttribute('aria-expanded', 'false');
}

function renderSearch(query) {
  if (!searchResults || !searchInput) return;
  const normalized = query.trim().toLocaleLowerCase('zh-CN');
  searchResults.replaceChildren();

  if (!normalized) {
    closeSearch();
    return;
  }

  const matches = searchItems.filter((item) => item.searchable.includes(normalized)).slice(0, 8);
  if (!matches.length) {
    const empty = document.createElement('p');
    empty.className = 'search-empty';
    empty.textContent = '没有找到相关内容';
    searchResults.appendChild(empty);
  } else {
    matches.forEach((item) => {
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'search-result';
      button.setAttribute('role', 'option');

      const title = document.createElement('strong');
      title.textContent = item.title;
      const excerpt = document.createElement('span');
      excerpt.textContent = item.text.length > 76 ? `${item.text.slice(0, 76)}...` : item.text;
      button.append(title, excerpt);
      button.addEventListener('click', () => {
        document.querySelector(`#${CSS.escape(item.id)}`)?.scrollIntoView();
        history.replaceState(null, '', `#${item.id}`);
        closeSearch();
      });
      searchResults.appendChild(button);
    });
  }

  searchResults.hidden = false;
  searchInput.setAttribute('aria-expanded', 'true');
}

searchInput?.addEventListener('input', () => renderSearch(searchInput.value));
searchInput?.addEventListener('keydown', (event) => {
  const options = [...searchResults.querySelectorAll('.search-result')];
  if (event.key === 'Escape') {
    closeSearch();
    searchInput.blur();
  } else if (event.key === 'ArrowDown' && options.length) {
    event.preventDefault();
    options[0].focus();
  } else if (event.key === 'Enter' && options.length) {
    event.preventDefault();
    options[0].click();
  }
});

searchResults?.addEventListener('keydown', (event) => {
  const options = [...searchResults.querySelectorAll('.search-result')];
  const current = options.indexOf(document.activeElement);
  if (event.key === 'ArrowDown') {
    event.preventDefault();
    options[(current + 1) % options.length]?.focus();
  } else if (event.key === 'ArrowUp') {
    event.preventDefault();
    if (current <= 0) searchInput?.focus();
    else options[current - 1]?.focus();
  } else if (event.key === 'Escape') {
    closeSearch();
    searchInput?.focus();
  }
});

document.addEventListener('click', (event) => {
  if (!event.target.closest('.search-shell')) closeSearch();
});

const trackedLinks = [...document.querySelectorAll('.toc a, .nav-group-items a')];
const trackedSections = [...new Set(trackedLinks
  .map((link) => document.querySelector(link.getAttribute('href')))
  .filter(Boolean))];

if ('IntersectionObserver' in window) {
  const observer = new IntersectionObserver((entries) => {
    const visible = entries
      .filter((entry) => entry.isIntersecting)
      .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0];
    if (!visible) return;

    trackedLinks.forEach((link) => {
      link.classList.toggle('active', link.getAttribute('href') === `#${visible.target.id}`);
    });
  }, { rootMargin: '-12% 0px -72% 0px' });

  trackedSections.forEach((section) => observer.observe(section));
}

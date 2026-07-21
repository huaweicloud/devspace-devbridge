const navToggle = document.querySelector('.nav-toggle');
const sidebar = document.querySelector('.sidebar');

navToggle?.addEventListener('click', () => {
  const open = sidebar.classList.toggle('open');
  navToggle.setAttribute('aria-expanded', String(open));
});

sidebar?.addEventListener('click', (event) => {
  if (event.target.closest('a') && window.innerWidth <= 760) {
    sidebar.classList.remove('open');
    navToggle?.setAttribute('aria-expanded', 'false');
  }
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
      window.setTimeout(() => { button.textContent = '复制'; }, 1400);
    } catch {
      try {
        copyWithTextArea(code);
        button.textContent = '已复制';
      } catch {
        button.textContent = '复制失败';
      }
      window.setTimeout(() => { button.textContent = '复制'; }, 1400);
    }
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

const tocLinks = [...document.querySelectorAll('.toc a')];
const sections = tocLinks
  .map((link) => document.querySelector(link.getAttribute('href')))
  .filter(Boolean);

if ('IntersectionObserver' in window) {
  const observer = new IntersectionObserver((entries) => {
    const visible = entries
      .filter((entry) => entry.isIntersecting)
      .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0];
    if (!visible) return;

    tocLinks.forEach((link) => {
      link.classList.toggle('active', link.getAttribute('href') === `#${visible.target.id}`);
    });
  }, { rootMargin: '-15% 0px -70% 0px' });

  sections.forEach((section) => observer.observe(section));
}

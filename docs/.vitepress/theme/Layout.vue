<script setup>
import { ref, onMounted, watch, nextTick, onBeforeUnmount } from "vue";
import { useRoute } from "vitepress";
import DefaultTheme from "vitepress/theme";
import TranslationNotice from "./TranslationNotice.vue";

const route = useRoute();
const target = ref(null);
let mountEl = null;

const collapsed = ref(false);
function toggle() {
  collapsed.value = !collapsed.value;
}
watch(collapsed, (v) => {
  document.documentElement.classList.toggle("sidebar-collapsed", v);
  try {
    localStorage.setItem("devbridge-sidebar-collapsed", String(v));
  } catch {}
});

const asideCollapsed = ref(false);
function toggleAside() {
  asideCollapsed.value = !asideCollapsed.value;
}
watch(asideCollapsed, (v) => {
  document.documentElement.classList.toggle("aside-collapsed", v);
  try {
    localStorage.setItem("devbridge-aside-collapsed", String(v));
  } catch {}
});

const asideToggleEl = ref(null);
function moveAsideToggle() {
  const title = document.querySelector(".VPDocAsideOutline .outline-title");
  if (
    title &&
    asideToggleEl.value &&
    title !== asideToggleEl.value.parentElement
  ) {
    title.appendChild(asideToggleEl.value);
  }
}
onMounted(() => {
  nextTick(moveAsideToggle);
  setTimeout(moveAsideToggle, 300);
});
watch(
  () => route.path,
  () => {
    nextTick(moveAsideToggle);
    setTimeout(moveAsideToggle, 300);
  },
);

function startResize(e) {
  e.preventDefault();
  document.documentElement.classList.add("sidebar-resizing");

  const onMove = (ev) => {
    const x = ev.touches ? ev.touches[0].clientX : ev.clientX;
    const sliderLeftPct = parseFloat(
      getComputedStyle(document.documentElement)
        .getPropertyValue("--vp-slider-left")
        .trim() || "0",
    );
    const sliderLeft = (window.innerWidth * sliderLeftPct) / 100;
    let w = x - sliderLeft;
    const minW = window.innerWidth * 0.1;
    const maxW = window.innerWidth * 0.4;
    w = Math.max(minW, Math.min(maxW, w));
    document.documentElement.style.setProperty("--vp-nav-width", w + "px");
  };
  const onUp = () => {
    document.documentElement.classList.remove("sidebar-resizing");
    document.removeEventListener("mousemove", onMove);
    document.removeEventListener("mouseup", onUp);
    document.removeEventListener("touchmove", onMove);
    document.removeEventListener("touchend", onUp);
    try {
      const w = document.documentElement.style
        .getPropertyValue("--vp-nav-width")
        .trim();
      if (w) localStorage.setItem("devbridge-sidebar-width", w);
    } catch {}
  };
  document.addEventListener("mousemove", onMove);
  document.addEventListener("mouseup", onUp);
  document.addEventListener("touchmove", onMove, { passive: false });
  document.addEventListener("touchend", onUp);
}

function setupTarget() {
  if (mountEl && mountEl.parentNode) {
    mountEl.parentNode.removeChild(mountEl);
    mountEl = null;
  }
  const container = document.querySelector(".content-container");
  if (container) {
    mountEl = document.createElement("div");
    container.insertBefore(mountEl, container.firstChild);
    target.value = mountEl;
  }
}

onMounted(() => {
  nextTick(setupTarget);
  try {
    collapsed.value =
      localStorage.getItem("devbridge-sidebar-collapsed") === "true";
    asideCollapsed.value =
      localStorage.getItem("devbridge-aside-collapsed") === "true";
    const savedWidth = localStorage.getItem("devbridge-sidebar-width");
    if (savedWidth) {
      const px = parseFloat(savedWidth);
      const minW = window.innerWidth * 0.1;
      const maxW = window.innerWidth * 0.4;
      if (px >= minW && px <= maxW) {
        document.documentElement.style.setProperty(
          "--vp-nav-width",
          savedWidth,
        );
      } else {
        localStorage.removeItem("devbridge-sidebar-width");
      }
    }
  } catch {}
});

watch(
  () => route.path,
  () => {
    target.value = null;
    nextTick(() => setTimeout(setupTarget, 0));
  },
);

onBeforeUnmount(() => {
  if (mountEl && mountEl.parentNode) {
    mountEl.parentNode.removeChild(mountEl);
  }
  removeImgClickHandler();
});

/* === 图片点击放大 === */
let imgClickHandler = null;
function setupImgZoom() {
  removeImgClickHandler();
  imgClickHandler = (e) => {
    const img = e.target.closest(".vp-doc img");
    if (!img) return;
    if (img.closest('div[class*="language-"]')) return;
    e.preventDefault();
    const overlay = document.createElement("div");
    overlay.className = "img-lightbox";
    const clone = document.createElement("img");
    clone.src = img.src;
    clone.alt = img.alt || "";
    overlay.appendChild(clone);
    overlay.addEventListener("click", () => {
      overlay.remove();
    });
    document.body.appendChild(overlay);
  };
  document.addEventListener("click", imgClickHandler, true);
}
function removeImgClickHandler() {
  if (imgClickHandler) {
    document.removeEventListener("click", imgClickHandler, true);
    imgClickHandler = null;
  }
}

/* === 最后更新时间移到标题下方 === */
function moveLastUpdated() {
  const editInfo = document.querySelector(".VPDocFooter .edit-info");
  const h1 = document.querySelector(".vp-doc h1");
  if (editInfo && h1 && editInfo.parentElement !== h1.parentElement) {
    h1.insertAdjacentElement("afterend", editInfo);
  }
  const timeEl = document.querySelector(".vp-doc h1 + .edit-info time");
  if (timeEl) {
    const dt = new Date(timeEl.getAttribute("datetime") || timeEl.textContent);
    if (!isNaN(dt)) {
      const pad = (n) => String(n).padStart(2, "0");
      timeEl.textContent = `${dt.getFullYear()}-${pad(dt.getMonth() + 1)}-${pad(dt.getDate())} ${pad(dt.getHours())}:${pad(dt.getMinutes())}`;
    }
  }
}
/* === 代码块：替换复制图标 + 添加深浅色切换按钮 + 修正语法高亮 === */
function setupCodeBlocks() {
  const blocks = document.querySelectorAll('.vp-doc div[class*="language-"]');
  blocks.forEach((block) => {
    if (block.dataset.codeBlockInit) return;
    block.dataset.codeBlockInit = "1";

    const copyBtn = block.querySelector(".copy");
    if (copyBtn) {
      copyBtn.removeAttribute("title");
      copyBtn.innerHTML =
        '<img class="icon-light-default" src="/devspace-devbridge/images/support-doc-revision-copy-m.svg" alt="复制">' +
        '<img class="icon-light-hover" src="/devspace-devbridge/images/support-doc-revision-copy-h.svg" alt="复制">' +
        '<img class="icon-dark-default" src="/devspace-devbridge/images/support-doc-revision-copy-dark-m.svg" alt="复制">' +
        '<img class="icon-dark-hover" src="/devspace-devbridge/images/support-doc-revision-copy-dark-h.svg" alt="复制">' +
        '<span class="code-btn-tooltip">复制</span>';
      copyBtn.addEventListener("click", (e) => {
        e.stopPropagation();
        const pre = block.querySelector("pre");
        const text = pre ? pre.textContent : "";
        if (text) {
          navigator.clipboard.writeText(text).then(() => {
            const tooltip = copyBtn.querySelector(".code-btn-tooltip");
            if (tooltip) {
              tooltip.textContent = "复制成功";
              setTimeout(() => {
                tooltip.textContent = "复制";
              }, 2000);
            }
          });
        }
      });
    }

    const toggleBtn = document.createElement("button");
    toggleBtn.className = "code-theme-toggle";
    toggleBtn.type = "button";
    toggleBtn.innerHTML =
      '<img class="icon-light-default" src="/devspace-devbridge/images/support-doc-revision-cgc-m.svg" alt="切换主题">' +
      '<img class="icon-light-hover" src="/devspace-devbridge/images/support-doc-revision-cgc-h.svg" alt="切换主题">' +
      '<img class="icon-dark-default" src="/devspace-devbridge/images/support-doc-revision-cgc-dark-m.svg" alt="切换主题">' +
      '<img class="icon-dark-hover" src="/devspace-devbridge/images/support-doc-revision-cgc-dark-h.svg" alt="切换主题">' +
      '<span class="code-btn-tooltip">夜间模式</span>';
    toggleBtn.addEventListener("click", () => {
      block.classList.toggle("code-dark");
      const isDark = block.classList.contains("code-dark");
      const tooltip = toggleBtn.querySelector(".code-btn-tooltip");
      if (tooltip) {
        tooltip.textContent = isDark ? "日间模式" : "夜间模式";
      }
    });
    block.appendChild(toggleBtn);

    const isCmdLang = /language-(bash|shell|powershell|console)/.test(
      block.className,
    );
    if (isCmdLang) {
      const spans = block.querySelectorAll(".vp-code span[style]");
      spans.forEach((span) => {
        const style = span.getAttribute("style") || "";
        const text = span.textContent;

        if (style.includes("--shiki-light:#0F5ED4") && text.includes(".")) {
          span.style.setProperty("--shiki-light", "#191919", "important");
          span.style.setProperty("--shiki-dark", "#FFFFFF", "important");
        }

        if (style.includes("--shiki-light:#191919")) {
          const m = text.match(/^(\s*)(irm|iex|devbridge)(\s.*)?$/);
          if (m) {
            const [, leading, cmdName, rest] = m;
            const prev = span.previousSibling;
            const prevText = prev ? prev.textContent : "";
            const inUrl = /[/.\\-]$/.test(prevText);
            if (inUrl) return;
            if (leading) {
              span.textContent = leading;
              const cmdSpan = document.createElement("span");
              cmdSpan.style.setProperty(
                "--shiki-light",
                "#841eb0",
                "important",
              );
              cmdSpan.style.setProperty("--shiki-dark", "#d083f1", "important");
              cmdSpan.textContent = cmdName;
              span.parentNode.insertBefore(cmdSpan, span.nextSibling);
              if (rest) {
                const restSpan = document.createElement("span");
                restSpan.style.setProperty(
                  "--shiki-light",
                  "#191919",
                  "important",
                );
                restSpan.style.setProperty(
                  "--shiki-dark",
                  "#FFFFFF",
                  "important",
                );
                restSpan.textContent = rest;
                span.parentNode.insertBefore(restSpan, cmdSpan.nextSibling);
              }
            } else {
              span.style.setProperty("--shiki-light", "#841eb0", "important");
              span.style.setProperty("--shiki-dark", "#d083f1", "important");
              span.textContent = cmdName;
              if (rest) {
                const restSpan = document.createElement("span");
                restSpan.style.setProperty(
                  "--shiki-light",
                  "#191919",
                  "important",
                );
                restSpan.style.setProperty(
                  "--shiki-dark",
                  "#FFFFFF",
                  "important",
                );
                restSpan.textContent = rest;
                span.parentNode.insertBefore(restSpan, span.nextSibling);
              }
            }
          }
        }
      });
    }
  });
}

onMounted(() => {
  nextTick(() => {
    setupImgZoom();
    moveLastUpdated();
    setupCodeBlocks();
  });
  setTimeout(() => {
    moveLastUpdated();
    setupCodeBlocks();
  }, 300);
});
watch(
  () => route.path,
  () => {
    nextTick(() => {
      setupImgZoom();
      moveLastUpdated();
      setupCodeBlocks();
    });
    setTimeout(() => {
      moveLastUpdated();
      setupCodeBlocks();
    }, 300);
  },
);
</script>

<template>
  <DefaultTheme.Layout>
    <template #doc-top>
      <ClientOnly>
        <Teleport v-if="target" :to="target">
          <TranslationNotice />
        </Teleport>
      </ClientOnly>
    </template>
    <template #sidebar-nav-after>
      <ClientOnly>
        <Teleport to="body">
          <button
            type="button"
            class="sidebar-toggle"
            :class="{ collapsed }"
            :aria-label="collapsed ? '展开目录' : '折叠目录'"
            @click="toggle"
          />
          <div
            class="sidebar-resizer"
            @mousedown="startResize"
            @touchstart="startResize"
          />
        </Teleport>
      </ClientOnly>
    </template>
    <template #aside-outline-before>
      <ClientOnly>
        <button
          ref="asideToggleEl"
          type="button"
          class="aside-toggle"
          :class="{ collapsed: asideCollapsed }"
          :aria-label="asideCollapsed ? '展开导读' : '折叠导读'"
          @click="toggleAside"
        />
      </ClientOnly>
    </template>
  </DefaultTheme.Layout>
</template>

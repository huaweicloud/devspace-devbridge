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

function startResize(e) {
  e.preventDefault();
  document.documentElement.classList.add("sidebar-resizing");
  const onMove = (ev) => {
    const x = ev.touches ? ev.touches[0].clientX : ev.clientX;
    const sliderLeft = parseFloat(
      getComputedStyle(document.documentElement)
        .getPropertyValue("--vp-slider-left")
        .trim() || "0",
    );
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
        document.documentElement.style.setProperty("--vp-nav-width", savedWidth);
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
});
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
        <Teleport to="body">
          <button
            type="button"
            class="aside-toggle"
            :class="{ collapsed: asideCollapsed }"
            :aria-label="asideCollapsed ? '展开导读' : '折叠导读'"
            @click="toggleAside"
          />
        </Teleport>
      </ClientOnly>
    </template>
  </DefaultTheme.Layout>
</template>

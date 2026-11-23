<script setup>
import { computed } from "vue";
import { useRoute, useData, withBase } from "vitepress";

const route = useRoute();
const { theme, localeIndex, site } = useData();

const homeLink = computed(() =>
  withBase(localeIndex.value === "zh" ? "/zh/" : "/"),
);

const homeText = computed(() =>
  localeIndex.value === "zh" ? "开发隧道" : "DevBridge",
);

function normalize(p) {
  if (!p) return "";
  let s = p;
  const base = site.value.base || "/";
  if (base !== "/" && s.startsWith(base)) {
    s = s.slice(base.length);
    if (!s.startsWith("/")) s = "/" + s;
  }
  s = s.replace(/\.html$/, "");
  if (s.length > 1 && s.endsWith("/")) s = s.slice(0, -1);
  return s;
}

function findInSidebar(path, sidebar) {
  const np = normalize(path);
  for (const group of sidebar) {
    if (!group.items) continue;
    for (const item of group.items) {
      if (item.link && np === normalize(item.link)) {
        return [
          { text: group.text, link: withBase(group.items[0].link) },
          { text: item.text, link: null },
        ];
      }
      if (item.items) {
        for (const subItem of item.items) {
          if (subItem.link && np === normalize(subItem.link)) {
            return [
              { text: group.text, link: withBase(group.items[0].link) },
              { text: item.text, link: withBase(item.items[0].link) },
              { text: subItem.text, link: null },
            ];
          }
        }
      }
    }
  }
  return null;
}

const breadcrumbs = computed(() => {
  const path = route.path;
  const np = normalize(path);
  if (np === "/" || np === "/zh") return [];

  let sidebar = theme.value.sidebar || [];
  if (!Array.isArray(sidebar) && typeof sidebar === "object") {
    const key = localeIndex.value === "zh" ? "/zh/" : "/";
    sidebar = sidebar[key] || sidebar["/"] || [];
  }
  return findInSidebar(path, sidebar) || [];
});
</script>

<template>
  <nav v-if="breadcrumbs.length" class="doc-breadcrumb" aria-label="Breadcrumb">
    <a :href="homeLink" class="breadcrumb-link">{{ homeText }}</a>
    <template v-for="(crumb, i) in breadcrumbs" :key="i">
      <span class="breadcrumb-separator">/</span>
      <a v-if="crumb.link" :href="crumb.link" class="breadcrumb-link">{{
        crumb.text
      }}</a>
      <span v-else class="breadcrumb-current">{{ crumb.text }}</span>
    </template>
  </nav>
</template>
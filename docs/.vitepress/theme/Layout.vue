<script setup>
import { ref, onMounted, watch, nextTick, onBeforeUnmount } from "vue";
import { useRoute } from "vitepress";
import DefaultTheme from "vitepress/theme";
import TranslationNotice from "./TranslationNotice.vue";

const route = useRoute();
const target = ref(null);
let mountEl = null;

function setupTarget() {
  // Clean up previous mount point
  if (mountEl && mountEl.parentNode) {
    mountEl.parentNode.removeChild(mountEl);
    mountEl = null;
  }

  const container = document.querySelector(".content-container");
  if (container) {
    // Create a dedicated mount point at the TOP of content-container
    mountEl = document.createElement("div");
    container.insertBefore(mountEl, container.firstChild);
    target.value = mountEl;
  }
}

onMounted(() => {
  nextTick(setupTarget);
});

// Re-setup on route change (new page = new content-container)
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
  </DefaultTheme.Layout>
</template>

<script setup>
import { ref, watch, onMounted } from "vue";
import { useData } from "vitepress";

const { lang } = useData();
const visible = ref(false);

function updateVisibility() {
  // Only show on English pages; dismissal is session-only (not persisted)
  visible.value = lang.value === "en";
}

// React to language toggle
watch(lang, updateVisibility);

// Initial check after mount
onMounted(updateVisibility);

function dismiss() {
  visible.value = false;
}
</script>

<template>
  <div v-if="visible" class="translation-notice">
    <span class="translation-notice-text">
      ⚠ Some content in this documentation may be machine-translated or
      AI-translated.
    </span>
    <button
      class="translation-notice-close"
      aria-label="Dismiss notice"
      @click="dismiss"
    >
      ✕
    </button>
  </div>
</template>

<style scoped>
.translation-notice {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 0 0 16px;
  padding: 8px 16px;
  border-radius: 8px;
  background-color: #fff8e1;
  border: 1px solid #ffc107;
  font-size: 14px;
  line-height: 1.5;
}

.translation-notice-text {
  flex: 1;
  color: #7c5e10;
}

.translation-notice-close {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: #7c5e10;
  font-size: 14px;
  cursor: pointer;
  opacity: 0.6;
  line-height: 1;
  padding: 0;
}

.translation-notice-close:hover {
  opacity: 1;
  background-color: rgba(255, 193, 7, 0.2);
}

/* Dark mode */
.dark .translation-notice {
  background-color: #3d3520;
  border-color: #8a6d0b;
}

.dark .translation-notice-text {
  color: #ffd54f;
}

.dark .translation-notice-close {
  color: #ffd54f;
}

.dark .translation-notice-close:hover {
  background-color: rgba(255, 193, 7, 0.25);
}
</style>

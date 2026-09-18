<script setup>
import { computed } from "vue";
import { withBase } from "vitepress";

const props = defineProps({
  src: { type: String, required: true },
  poster: { type: String, default: "" },
  title: { type: String, required: true },
  caption: { type: Boolean, default: true },
});

// 外链（http/https 开头）直接使用，仓库内路径补 base 前缀
const videoSrc = computed(() =>
  /^https?:\/\//.test(props.src) ? props.src : withBase(props.src),
);

const videoPoster = computed(() => {
  if (!props.poster) return undefined;
  return /^https?:\/\//.test(props.poster)
    ? props.poster
    : withBase(props.poster);
});

// 无封面时预加载元数据以展示视频首帧
const preload = computed(() => (props.poster ? "none" : "metadata"));
</script>

<template>
  <figure class="video-card">
    <video
      class="video-card-media"
      :src="videoSrc"
      :poster="videoPoster"
      :aria-label="title"
      controls
      :preload="preload"
    ></video>
    <figcaption v-if="caption" class="video-card-caption">
      <span class="video-card-title">{{ title }}</span>
    </figcaption>
  </figure>
</template>

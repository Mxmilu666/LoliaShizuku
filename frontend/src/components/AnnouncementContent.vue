<script lang="ts" setup>
import { computed } from "vue";
import { openExternalURL } from "@/utils/browser";
import { renderAnnouncementContent, safeExternalURL } from "@/utils/announcement";

const props = defineProps<{ content: string }>();

const html = computed(() => renderAnnouncementContent(props.content));

// 内容中的链接在系统浏览器中打开，避免应用的 WebView 跳转离开
const handleClick = (event: MouseEvent) => {
  const anchor = (event.target as HTMLElement | null)?.closest("a");
  if (!anchor) return;
  event.preventDefault();
  const url = safeExternalURL(anchor.getAttribute("href"));
  if (url) {
    openExternalURL(url);
  }
};
</script>

<template>
  <!-- 仅插入经 DOMPurify 清洗后的内容 -->
  <!-- eslint-disable-next-line vue/no-v-html -->
  <div class="announcement-content" @click="handleClick" v-html="html" />
</template>

<style scoped>
.announcement-content {
  min-width: 0;
  overflow-wrap: anywhere;
  line-height: 1.65;
}

.announcement-content :deep(p),
.announcement-content :deep(ul),
.announcement-content :deep(ol),
.announcement-content :deep(blockquote),
.announcement-content :deep(pre),
.announcement-content :deep(table),
.announcement-content :deep(details) {
  margin-block: 0.5em;
}

.announcement-content :deep(> :first-child) {
  margin-top: 0;
}

.announcement-content :deep(> :last-child) {
  margin-bottom: 0;
}

.announcement-content :deep(h1),
.announcement-content :deep(h2),
.announcement-content :deep(h3),
.announcement-content :deep(h4) {
  margin-block: 0.8em 0.4em;
  font-weight: 600;
  line-height: 1.4;
}

.announcement-content :deep(h1) { font-size: 1.3em; }
.announcement-content :deep(h2) { font-size: 1.2em; }
.announcement-content :deep(h3) { font-size: 1.1em; }
.announcement-content :deep(ul) { list-style: disc; padding-inline-start: 1.5em; }
.announcement-content :deep(ol) { list-style: decimal; padding-inline-start: 1.5em; }

.announcement-content :deep(a) {
  color: rgb(var(--v-theme-primary));
  text-decoration: underline;
  cursor: pointer;
}

.announcement-content :deep(blockquote) {
  padding-inline-start: 0.8em;
  border-inline-start: 3px solid var(--md-outline-variant);
  color: var(--md-on-surface-variant);
}

.announcement-content :deep(code) {
  padding: 0.1em 0.3em;
  border-radius: 4px;
  font-size: 0.9em;
  background: var(--md-surface-container-highest);
}

.announcement-content :deep(pre) {
  padding: 0.75em 1em;
  border-radius: 8px;
  overflow-x: auto;
  background: var(--md-surface-container-highest);
}

.announcement-content :deep(pre code) {
  padding: 0;
  background: none;
}

.announcement-content :deep(img) {
  max-width: 100%;
  border-radius: 8px;
}

.announcement-content :deep(table) {
  display: block;
  overflow-x: auto;
  border-collapse: collapse;
}

.announcement-content :deep(th),
.announcement-content :deep(td) {
  padding: 4px 8px;
  border: 1px solid var(--md-outline-variant);
}
</style>

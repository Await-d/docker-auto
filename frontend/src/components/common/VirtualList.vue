<template>
  <div
    ref="containerRef"
    class="virtual-list-container"
    :style="{ height: containerHeight + 'px' }"
    @scroll="handleScroll"
  >
    <div
      class="virtual-list-viewport"
      :style="{ height: totalHeight + 'px', position: 'relative' }"
    >
      <div
        class="virtual-list-content"
        :style="{
          transform: `translateY(${offsetY}px)`,
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0
        }"
      >
        <div
          v-for="(item, index) in visibleItems"
          :key="startIndex + index"
          class="virtual-list-item"
          :style="{ height: itemHeight + 'px' }"
        >
          <slot
            :item="item"
            :index="startIndex + index"
            :isVisible="true"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'

interface Props {
  items: any[]
  itemHeight: number
  containerHeight: number
  buffer?: number
  overscan?: number
}

const props = withDefaults(defineProps<Props>(), {
  buffer: 5,
  overscan: 3
})

const emit = defineEmits<{
  scroll: [scrollTop: number, isAtBottom: boolean]
}>()

const containerRef = ref<HTMLElement>()
const scrollTop = ref(0)

// Computed properties
const totalHeight = computed(() => props.items.length * props.itemHeight)

const viewportHeight = computed(() => props.containerHeight)

const visibleCount = computed(() => Math.ceil(viewportHeight.value / props.itemHeight) + props.overscan * 2)

const startIndex = computed(() => {
  const start = Math.floor(scrollTop.value / props.itemHeight) - props.overscan
  return Math.max(0, start)
})

const endIndex = computed(() => {
  const end = startIndex.value + visibleCount.value
  return Math.min(props.items.length - 1, end)
})

const visibleItems = computed(() => {
  return props.items.slice(startIndex.value, endIndex.value + 1)
})

const offsetY = computed(() => startIndex.value * props.itemHeight)

const isAtBottom = computed(() => {
  const containerEl = containerRef.value
  if (!containerEl) return false

  const { scrollTop, scrollHeight, clientHeight } = containerEl
  return scrollTop + clientHeight >= scrollHeight - 10
})

// Methods
function handleScroll(event: Event) {
  const target = event.target as HTMLElement
  scrollTop.value = target.scrollTop

  emit('scroll', scrollTop.value, isAtBottom.value)
}

function scrollToIndex(index: number, align: 'start' | 'center' | 'end' = 'start') {
  if (!containerRef.value) return

  let targetScrollTop = index * props.itemHeight

  if (align === 'center') {
    targetScrollTop -= viewportHeight.value / 2 - props.itemHeight / 2
  } else if (align === 'end') {
    targetScrollTop -= viewportHeight.value - props.itemHeight
  }

  targetScrollTop = Math.max(0, Math.min(targetScrollTop, totalHeight.value - viewportHeight.value))

  containerRef.value.scrollTop = targetScrollTop
}

function scrollToTop() {
  if (containerRef.value) {
    containerRef.value.scrollTop = 0
  }
}

function scrollToBottom() {
  if (containerRef.value) {
    containerRef.value.scrollTop = totalHeight.value
  }
}

function getScrollPosition() {
  return {
    scrollTop: scrollTop.value,
    isAtBottom: isAtBottom.value,
    startIndex: startIndex.value,
    endIndex: endIndex.value
  }
}

// Watch for items changes and maintain scroll position if needed
watch(() => props.items.length, (newLength, oldLength) => {
  if (newLength > oldLength && isAtBottom.value) {
    // Auto-scroll to bottom when new items are added and user is at bottom
    nextTick(() => {
      scrollToBottom()
    })
  }
})

// Expose methods to parent
defineExpose({
  scrollToIndex,
  scrollToTop,
  scrollToBottom,
  getScrollPosition
})

onMounted(() => {
  // Initial scroll position setup if needed
})

onUnmounted(() => {
  // Cleanup if needed
})
</script>

<style scoped>
.virtual-list-container {
  overflow-y: auto;
  overflow-x: hidden;
  width: 100%;
}

.virtual-list-viewport {
  width: 100%;
}

.virtual-list-content {
  width: 100%;
}

.virtual-list-item {
  width: 100%;
  box-sizing: border-box;
}

/* Custom scrollbar for better UX */
.virtual-list-container::-webkit-scrollbar {
  width: 8px;
}

.virtual-list-container::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
}

.virtual-list-container::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.3);
  border-radius: 4px;
}

.virtual-list-container::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.5);
}
</style>
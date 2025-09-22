<template>
  <div class="settings-drawer">
    <!-- Settings drawer placeholder -->
    <el-drawer
v-model="visible" title="设置"
direction="rtl" size="320px"
>
      <div class="settings-content">
        <p>设置面板即将推出...</p>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";

const visible = ref(false);
</script>

<style scoped lang="scss">
.settings-content {
  padding: 20px;
  text-align: center;
  color: var(--el-text-color-secondary);
}
</style>

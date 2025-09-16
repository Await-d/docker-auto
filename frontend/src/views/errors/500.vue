<template>
  <div class="error-page">
    <div class="error-content">
      <div class="error-icon">
        <el-icon :size="120" color="var(--el-color-danger)">
          <Warning />
        </el-icon>
      </div>
      <h1 class="error-title">500</h1>
      <h2 class="error-subtitle">Server Error</h2>
      <p class="error-description">
        Something went wrong on our servers. Please try again later or contact support if the problem persists.
      </p>
      <div class="error-actions">
        <el-button type="primary" @click="reload">
          <el-icon><Refresh /></el-icon>
          Reload Page
        </el-button>
        <el-button @click="goHome">
          <el-icon><House /></el-icon>
          Go Home
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { Warning, Refresh, House } from '@element-plus/icons-vue'

const router = useRouter()

const reload = () => {
  window.location.reload()
}

const goHome = () => {
  router.push('/')
}
</script>

<style scoped lang="scss">
@use '@/styles/variables' as *;

.error-page {
  min-height: 100vh;
  @include flex-center;
  padding: $spacing-lg;
  background: var(--el-bg-color-page);
}

.error-content {
  text-align: center;
  max-width: 500px;

  .error-icon {
    margin-bottom: $spacing-xl;
  }

  .error-title {
    font-size: 4rem;
    font-weight: $font-weight-bold;
    color: var(--el-color-danger);
    margin-bottom: $spacing-base;
  }

  .error-subtitle {
    font-size: 1.5rem;
    font-weight: $font-weight-primary;
    color: var(--el-text-color-primary);
    margin-bottom: $spacing-base;
  }

  .error-description {
    color: var(--el-text-color-secondary);
    margin-bottom: $spacing-xl;
    line-height: 1.6;
  }

  .error-actions {
    display: flex;
    gap: $spacing-base;
    justify-content: center;

    @include max-screen($breakpoint-xs) {
      flex-direction: column;
    }
  }
}
</style>
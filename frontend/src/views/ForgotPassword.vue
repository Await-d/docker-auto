<template>
  <div class="forgot-password-page">
    <div class="forgot-password-container">
      <el-card>
        <template #header>
          <h2>忘记密码</h2>
        </template>
        <p>密码重置功能即将推出...</p>
        <el-button @click="$router.push('/login')"> 返回登录 </el-button>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
// Placeholder component
</script>

<style scoped lang="scss">
.forgot-password-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.forgot-password-container {
  width: 100%;
  max-width: 400px;
  text-align: center;
}
</style>

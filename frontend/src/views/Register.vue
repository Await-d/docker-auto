<template>
  <div class="register-page">
    <div class="register-container">
      <el-card>
        <template #header>
          <h2>注册</h2>
        </template>
        <p>注册功能即将推出...</p>
        <el-button @click="$router.push('/login')"> 返回登录 </el-button>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
// Placeholder component
</script>

<style scoped lang="scss">
.register-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.register-container {
  width: 100%;
  max-width: 400px;
  text-align: center;
}
</style>

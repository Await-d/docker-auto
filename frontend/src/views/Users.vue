<template>
  <div class="users-page">
    <el-page-header @back="$router.back()" content="用户管理">
      <template #extra>
        <el-button type="primary" :icon="Plus" @click="showCreateUser = true">
          新增用户
        </el-button>
      </template>
    </el-page-header>

    <div class="page-content">
      <el-card>
        <template #header>
          <div class="card-header">
            <span>用户列表</span>
          </div>
        </template>

        <el-table :data="users" v-loading="loading" empty-text="暂无用户数据">
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="username" label="用户名" width="150" />
          <el-table-column prop="email" label="邮箱" />
          <el-table-column prop="role" label="角色" width="120">
            <template #default="{ row }">
              <el-tag :type="getRoleType(row.role)">
                {{ getRoleLabel(row.role) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="is_active" label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.is_active ? 'success' : 'info'">
                {{ row.is_active ? '激活' : '禁用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="180">
            <template #default="{ row }">
              {{ formatDate(row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button size="small" type="primary" @click="editUser(row)">
                编辑
              </el-button>
              <el-button
                size="small"
                :type="row.is_active ? 'warning' : 'success'"
                @click="toggleUserStatus(row)">
                {{ row.is_active ? '禁用' : '启用' }}
              </el-button>
              <el-button size="small" type="danger" @click="deleteUser(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>

    <!-- Create User Dialog -->
    <el-dialog v-model="showCreateUser" title="新增用户" width="500px">
      <el-form :model="newUser" label-width="80px">
        <el-form-item label="用户名">
          <el-input v-model="newUser.username" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="newUser.email" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="newUser.password" type="password" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="newUser.role">
            <el-option label="管理员" value="admin" />
            <el-option label="操作员" value="operator" />
            <el-option label="查看者" value="viewer" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateUser = false">取消</el-button>
        <el-button type="primary" @click="createUser">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { Plus } from "@element-plus/icons-vue";
import { userAPI, type CreateUserRequest } from "@/api/user";
import type { User } from "@/types/user";

const users = ref<User[]>([]);
const loading = ref(false);
const showCreateUser = ref(false);
const newUser = ref<CreateUserRequest>({
  username: "",
  email: "",
  password: "",
  role: "viewer",
});

const refreshUsers = async () => {
  loading.value = true;
  try {
    const response = await userAPI.getUserList();
    users.value = response.users || [];
  } catch (error) {
    console.error("Failed to fetch users:", error);
    ElMessage.error("获取用户列表失败");
  } finally {
    loading.value = false;
  }
};

const createUser = async () => {
  try {
    await userAPI.createUser(newUser.value);
    showCreateUser.value = false;
    newUser.value = { username: "", email: "", password: "", role: "viewer" };
    await refreshUsers();
  } catch (error) {
    console.error("Failed to create user:", error);
    ElMessage.error("创建用户失败");
  }
};

const editUser = (user: User) => {
  ElMessage.info(`编辑用户: ${user.username}`);
};

const toggleUserStatus = async (user: User) => {
  try {
    const newStatus = !user.is_active;
    await userAPI.toggleUserStatus(user.id, newStatus);
    await refreshUsers();
  } catch (error) {
    ElMessage.error("更新用户状态失败");
  }
};

const deleteUser = async (user: User) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除用户 "${user.username}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    );

    await userAPI.deleteUser(user.id);
    await refreshUsers();
  } catch (error) {
    if (error !== 'cancel') {
      console.error("Failed to delete user:", error);
      ElMessage.error("删除用户失败");
    }
  }
};

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString();
};

const getRoleType = (role: string) => {
  switch (role) {
    case "admin":
      return "danger";
    case "operator":
      return "warning";
    case "viewer":
      return "info";
    default:
      return "info";
  }
};

const getRoleLabel = (role: string) => {
  switch (role) {
    case "admin":
      return "管理员";
    case "operator":
      return "操作员";
    case "viewer":
      return "查看者";
    default:
      return role;
  }
};

onMounted(() => {
  refreshUsers();
});
</script>

<style scoped>
.users-page {
  padding: 20px;
}

.page-content {
  margin-top: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
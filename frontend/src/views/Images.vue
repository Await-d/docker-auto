<template>
  <div class="images-page">
    <el-page-header @back="$router.back()" content="镜像管理">
      <template #extra>
        <el-button type="primary" :icon="Refresh" @click="refreshImages">
          刷新
        </el-button>
      </template>
    </el-page-header>

    <div class="page-content">
      <el-card>
        <template #header>
          <div class="card-header">
            <span>Docker 镜像列表</span>
          </div>
        </template>

        <el-table :data="images" v-loading="loading" empty-text="暂无镜像数据">
          <el-table-column prop="id" label="镜像ID" width="200" />
          <el-table-column prop="repository" label="仓库" />
          <el-table-column prop="tag" label="标签" width="150" />
          <el-table-column prop="created" label="创建时间" width="180">
            <template #default="{ row }">
              {{ formatDate(row.created) }}
            </template>
          </el-table-column>
          <el-table-column prop="size" label="大小" width="120">
            <template #default="{ row }">
              {{ formatSize(row.size) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button size="small" type="primary" @click="viewImage(row)">
                查看
              </el-button>
              <el-button size="small" type="danger" @click="handleDeleteImage(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { ElMessage } from "element-plus";
import { Refresh } from "@element-plus/icons-vue";
import { getImages, deleteImage, type DockerImage } from "@/api/images";

const images = ref<DockerImage[]>([]);
const loading = ref(false);

const refreshImages = async () => {
  loading.value = true;
  try {
    const response = await getImages();
    images.value = response.data || [];
  } catch (error) {
    console.error("Failed to fetch images:", error);
    ElMessage.error("获取镜像列表失败");
  } finally {
    loading.value = false;
  }
};

const viewImage = (image: DockerImage) => {
  ElMessage.info(`查看镜像: ${image.repository}:${image.tag}`);
};

const handleDeleteImage = async (image: DockerImage) => {
  try {
    await deleteImage(image.id);
    ElMessage.success(`删除镜像: ${image.repository}:${image.tag}`);
    await refreshImages();
  } catch (error) {
    ElMessage.error("删除镜像失败");
  }
};

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleString();
};

const formatSize = (bytes: number) => {
  const sizes = ["Bytes", "KB", "MB", "GB"];
  if (bytes === 0) return "0 Bytes";
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + " " + sizes[i];
};

onMounted(() => {
  refreshImages();
});
</script>

<style scoped>
.images-page {
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
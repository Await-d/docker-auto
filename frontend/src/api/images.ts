/**
 * Docker 镜像管理 API
 */
import { get, post, del } from "@/utils/request";

export interface DockerImage {
  id: string;
  repository: string;
  tag: string;
  created: string;
  size: number;
  digest?: string;
  parentId?: string;
  repoTags?: string[];
  repoDigests?: string[];
  labels?: Record<string, string>;
}

export interface ImageListResponse {
  data: DockerImage[];
  total: number;
  page: number;
  limit: number;
}

export interface ImagePullRequest {
  repository: string;
  tag?: string;
  registryAuth?: {
    username: string;
    password: string;
  };
}

export interface ImageBuildRequest {
  dockerfile: string;
  context: string;
  tag: string;
  buildArgs?: Record<string, string>;
}

/**
 * 获取镜像列表
 */
export const getImages = async (params?: {
  page?: number;
  limit?: number;
  search?: string;
}): Promise<ImageListResponse> => {
  return await get("/api/images", { params });
};

/**
 * 获取镜像详情
 */
export const getImageDetail = async (id: string): Promise<DockerImage> => {
  return await get(`/api/images/${id}`);
};

/**
 * 拉取镜像
 */
export const pullImage = async (data: ImagePullRequest): Promise<void> => {
  return await post("/api/images/pull", data);
};

/**
 * 构建镜像
 */
export const buildImage = async (data: ImageBuildRequest): Promise<void> => {
  return await post("/api/images/build", data);
};

/**
 * 删除镜像
 */
export const deleteImage = async (id: string): Promise<void> => {
  return await del(`/api/images/${id}`);
};

/**
 * 批量删除镜像
 */
export const deleteImages = async (ids: string[]): Promise<void> => {
  return await post("/api/images/batch-delete", { ids });
};

/**
 * 清理无用镜像
 */
export const pruneImages = async (params?: {
  dangling?: boolean;
  until?: string;
  filter?: Record<string, string>;
}): Promise<{ deletedImages: string[]; reclaimedSpace: number }> => {
  return await post("/api/images/prune", params);
};

/**
 * 获取镜像历史
 */
export const getImageHistory = async (id: string): Promise<any[]> => {
  return await get(`/api/images/${id}/history`);
};

/**
 * 导出镜像
 */
export const exportImage = async (id: string): Promise<Blob> => {
  return await get(`/api/images/${id}/export`, {
    responseType: "blob",
  });
};

/**
 * 导入镜像
 */
export const importImage = async (file: File): Promise<void> => {
  const formData = new FormData();
  formData.append("file", file);
  return await post("/api/images/import", formData, {
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });
};
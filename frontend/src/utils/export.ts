/**
 * Export utilities for downloading data in various formats
 */
import { ElMessage, ElNotification } from "element-plus";

export interface ExportOptions {
  filename?: string;
  dateRange?: { start: string; end: string };
  includeHeaders?: boolean;
  showProgress?: boolean;
}

/**
 * Download a blob as a file
 */
export const downloadBlob = (blob: Blob, filename: string): void => {
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;

  // Trigger download
  document.body.appendChild(link);
  link.click();

  // Cleanup
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);

  ElMessage.success(`文件 ${filename} 下载成功`);
};

/**
 * Convert JSON data to CSV format
 */
export const jsonToCsv = (data: any[], options: ExportOptions = {}): string => {
  if (!data || data.length === 0) {
    throw new Error("No data to export");
  }

  const { includeHeaders = true } = options;

  // Get all unique keys from all objects
  const allKeys = new Set<string>();
  data.forEach(item => {
    Object.keys(item).forEach(key => allKeys.add(key));
  });

  const headers = Array.from(allKeys);

  // Helper function to escape CSV values
  const escapeCsvValue = (value: any): string => {
    if (value === null || value === undefined) return "";

    const str = String(value);
    // If value contains comma, newline, or quotes, wrap in quotes and escape quotes
    if (str.includes(",") || str.includes("\n") || str.includes('"')) {
      return `"${str.replace(/"/g, '""')}"`;
    }
    return str;
  };

  const csvRows: string[] = [];

  // Add headers
  if (includeHeaders) {
    csvRows.push(headers.map(header => escapeCsvValue(header)).join(","));
  }

  // Add data rows
  data.forEach(item => {
    const row = headers.map(header => escapeCsvValue(item[header]));
    csvRows.push(row.join(","));
  });

  return csvRows.join("\n");
};

/**
 * Export data as CSV
 */
export const exportToCsv = (
  data: any[],
  filename: string = "export.csv",
  options: ExportOptions = {}
): void => {
  try {
    const csvContent = jsonToCsv(data, options);
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });

    // Add timestamp to filename if not provided
    const finalFilename = filename.includes('.')
      ? filename
      : `${filename}_${new Date().toISOString().split('T')[0]}.csv`;

    downloadBlob(blob, finalFilename);
  } catch (error) {
    console.error("CSV export failed:", error);
    ElMessage.error("CSV导出失败");
    throw error;
  }
};

/**
 * Export data as JSON
 */
export const exportToJson = (
  data: any,
  filename: string = "export.json",
  _options: ExportOptions = {}
): void => {
  try {
    const jsonContent = JSON.stringify(data, null, 2);
    const blob = new Blob([jsonContent], { type: "application/json;charset=utf-8;" });

    // Add timestamp to filename if not provided
    const finalFilename = filename.includes('.')
      ? filename
      : `${filename}_${new Date().toISOString().split('T')[0]}.json`;

    downloadBlob(blob, finalFilename);
  } catch (error) {
    console.error("JSON export failed:", error);
    ElMessage.error("JSON导出失败");
    throw error;
  }
};

/**
 * Create Excel-compatible CSV (with BOM for proper encoding)
 */
export const exportToExcel = (
  data: any[],
  filename: string = "export.xlsx",
  options: ExportOptions = {}
): void => {
  try {
    const csvContent = jsonToCsv(data, options);

    // Add BOM for proper UTF-8 encoding in Excel
    const BOM = "\uFEFF";
    const blob = new Blob([BOM + csvContent], {
      type: "application/vnd.ms-excel;charset=utf-8;"
    });

    // Add timestamp to filename if not provided
    const finalFilename = filename.includes('.')
      ? filename.replace(/\.(csv|json)$/, '.xlsx')
      : `${filename}_${new Date().toISOString().split('T')[0]}.xlsx`;

    downloadBlob(blob, finalFilename);
  } catch (error) {
    console.error("Excel export failed:", error);
    ElMessage.error("Excel导出失败");
    throw error;
  }
};

/**
 * Export data with progress indication
 */
export const exportWithProgress = async (
  exportFn: () => Promise<void>,
  message: string = "正在导出数据..."
): Promise<void> => {
  const notification = ElNotification({
    title: "导出进行中",
    message,
    type: "info",
    duration: 0,
  });

  try {
    await exportFn();
    notification.close();
    ElNotification({
      title: "导出成功",
      message: "文件已成功下载",
      type: "success",
      duration: 3000,
    });
  } catch (error) {
    notification.close();
    ElNotification({
      title: "导出失败",
      message: "导出过程中发生错误",
      type: "error",
      duration: 5000,
    });
    throw error;
  }
};

/**
 * Format data for export by flattening nested objects
 */
export const flattenObjectForExport = (obj: any, prefix: string = ""): any => {
  const flattened: any = {};

  for (const key in obj) {
    if (obj.hasOwnProperty(key)) {
      const newKey = prefix ? `${prefix}.${key}` : key;

      if (obj[key] !== null && typeof obj[key] === "object" && !Array.isArray(obj[key])) {
        // Recursively flatten nested objects
        Object.assign(flattened, flattenObjectForExport(obj[key], newKey));
      } else if (Array.isArray(obj[key])) {
        // Convert arrays to string representation
        flattened[newKey] = obj[key].join(", ");
      } else {
        flattened[newKey] = obj[key];
      }
    }
  }

  return flattened;
};

/**
 * Prepare data for export by flattening all objects in array
 */
export const prepareDataForExport = (data: any[]): any[] => {
  return data.map(item => flattenObjectForExport(item));
};

/**
 * Generate filename with timestamp and format
 */
export const generateFilename = (
  baseName: string,
  format: "csv" | "json" | "xlsx",
  includeTimestamp: boolean = true
): string => {
  const timestamp = includeTimestamp
    ? `_${new Date().toISOString().replace(/[:.]/g, "-").split("T")[0]}`
    : "";

  return `${baseName}${timestamp}.${format}`;
};

/**
 * Validate export data
 */
export const validateExportData = (data: any): boolean => {
  if (!data) {
    ElMessage.warning("没有数据可导出");
    return false;
  }

  if (Array.isArray(data) && data.length === 0) {
    ElMessage.warning("数据为空");
    return false;
  }

  return true;
};

/**
 * Export multiple sheets/datasets as a ZIP file
 */
export const exportMultipleSheetsAsZip = async (
  datasets: Array<{
    name: string;
    data: any[];
    format: "csv" | "json" | "xlsx";
  }>,
  _zipFilename: string = "export.zip"
): Promise<void> => {
  // This would require a ZIP library like JSZip
  // For now, we'll export each dataset separately
  for (const dataset of datasets) {
    const filename = generateFilename(dataset.name, dataset.format);

    switch (dataset.format) {
      case "csv":
        exportToCsv(dataset.data, filename);
        break;
      case "json":
        exportToJson(dataset.data, filename);
        break;
      case "xlsx":
        exportToExcel(dataset.data, filename);
        break;
    }

    // Add small delay between downloads
    await new Promise(resolve => setTimeout(resolve, 500));
  }
};
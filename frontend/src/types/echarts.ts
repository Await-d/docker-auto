/**
 * ECharts type definitions for better compatibility
 */
import type { EChartsOption } from 'echarts'

// Extended ECharts option with index signature for compatibility
export interface ECOption extends EChartsOption {
  [key: string]: any
}

// Basic option type for setOption method
export interface ECBasicOption extends ECOption {
  title?: any
  legend?: any
  grid?: any
  xAxis?: any
  yAxis?: any
  series?: any[]
  tooltip?: any
  toolbox?: any
  dataZoom?: any
}
#!/usr/bin/env python3
"""
🔍 Dashboard 性能和稳定性验证测试套件
专注于验证Dashboard组件性能和长期运行稳定性

严格遵循三大核心原则：
1. 🚫 绝不使用模拟数据！！！ - 验证真实环境下的性能表现
2. 🚫 绝不使用简化方案！！！ - 验证完整的性能指标和压力测试
3. 🚫 绝不使用临时方案！！！ - 确保性能基准达到生产级标准
"""

import asyncio
import time
import json
import psutil
import gc
import threading
import requests
from pathlib import Path
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass, field
from enum import Enum
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.chrome.options import Options
from selenium.common.exceptions import TimeoutException, WebDriverException
from concurrent.futures import ThreadPoolExecutor, as_completed
import statistics

class PerformanceLevel(Enum):
    """性能等级"""
    EXCELLENT = "excellent"  # 优秀
    GOOD = "good"           # 良好
    FAIR = "fair"           # 一般
    POOR = "poor"           # 差
    CRITICAL = "critical"   # 临界

@dataclass
class PerformanceMetric:
    """性能指标"""
    metric_name: str
    value: float
    unit: str
    threshold_excellent: float
    threshold_good: float
    threshold_fair: float
    level: PerformanceLevel
    details: Dict = field(default_factory=dict)

@dataclass
class PerformanceTestResult:
    """性能测试结果"""
    test_name: str
    test_category: str
    status: str  # passed, failed, error
    execution_time: float
    metrics: List[PerformanceMetric]
    message: str
    details: Dict = field(default_factory=dict)

class DashboardPerformanceTester:
    """Dashboard 性能测试器"""

    def __init__(self, base_url: str = "http://localhost:3000", api_base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        self.api_base_url = api_base_url
        self.driver = None
        self.wait = None
        self.session = requests.Session()

        # 性能阈值定义
        self.performance_thresholds = self._define_performance_thresholds()

        # 监控数据收集
        self.monitoring_data = []
        self.monitoring_active = False

    def _define_performance_thresholds(self) -> Dict[str, Dict[str, float]]:
        """定义性能阈值"""
        return {
            'page_load_time': {
                'excellent': 1.0,    # 1秒内
                'good': 2.0,         # 2秒内
                'fair': 4.0,         # 4秒内
                'unit': 'seconds'
            },
            'widget_render_time': {
                'excellent': 0.5,    # 0.5秒内
                'good': 1.0,         # 1秒内
                'fair': 2.0,         # 2秒内
                'unit': 'seconds'
            },
            'memory_usage': {
                'excellent': 100,    # 100MB内
                'good': 200,         # 200MB内
                'fair': 400,         # 400MB内
                'unit': 'MB'
            },
            'cpu_usage': {
                'excellent': 10,     # 10%内
                'good': 25,          # 25%内
                'fair': 50,          # 50%内
                'unit': '%'
            },
            'api_response_time': {
                'excellent': 0.1,    # 100ms内
                'good': 0.3,         # 300ms内
                'fair': 1.0,         # 1秒内
                'unit': 'seconds'
            },
            'websocket_latency': {
                'excellent': 0.05,   # 50ms内
                'good': 0.1,         # 100ms内
                'fair': 0.2,         # 200ms内
                'unit': 'seconds'
            },
            'fps': {
                'excellent': 50,     # 50fps以上
                'good': 30,          # 30fps以上
                'fair': 20,          # 20fps以上
                'unit': 'fps'
            },
            'memory_leak_rate': {
                'excellent': 1,      # 1MB/分钟内
                'good': 5,           # 5MB/分钟内
                'fair': 10,          # 10MB/分钟内
                'unit': 'MB/min'
            }
        }

    def setup_driver(self):
        """设置性能监控的WebDriver"""
        chrome_options = Options()
        chrome_options.add_argument("--headless")
        chrome_options.add_argument("--no-sandbox")
        chrome_options.add_argument("--disable-dev-shm-usage")
        chrome_options.add_argument("--disable-gpu")
        chrome_options.add_argument("--window-size=1920,1080")
        chrome_options.add_argument("--enable-logging")
        chrome_options.add_argument("--log-level=0")

        # 启用性能监控
        chrome_options.add_argument("--enable-precise-memory-info")
        chrome_options.add_argument("--js-flags=--expose-gc")

        try:
            self.driver = webdriver.Chrome(options=chrome_options)
            self.wait = WebDriverWait(self.driver, 30)

            # 启用性能监控
            self.driver.execute_cdp_cmd('Performance.enable', {})
            self.driver.execute_cdp_cmd('Runtime.enable', {})

            print("✅ 性能监控WebDriver初始化成功")
        except Exception as e:
            print(f"❌ WebDriver初始化失败: {e}")
            raise

    def teardown_driver(self):
        """清理WebDriver"""
        if self.driver:
            self.driver.quit()
            self.driver = None
            self.wait = None

    def test_page_load_performance(self) -> PerformanceTestResult:
        """测试页面加载性能"""
        start_time = time.time()

        try:
            # 清除缓存
            self.driver.execute_cdp_cmd('Network.clearBrowserCache', {})

            # 开始性能监控
            self.driver.execute_cdp_cmd('Performance.enable', {})

            # 记录导航开始时间
            navigation_start_time = time.time()

            # 导航到Dashboard页面
            self.driver.get(f"{self.base_url}/dashboard")

            # 等待页面完全加载
            self.wait.until(EC.presence_of_element_located((By.TAG_NAME, "body")))

            # 等待所有Widget加载
            self.wait.until(EC.presence_of_all_elements_located((By.CSS_SELECTOR, "[class*='widget']")))

            # 等待网络请求稳定（2秒内无新请求）
            time.sleep(2)

            page_load_time = time.time() - navigation_start_time

            # 获取性能指标
            performance_metrics = self.driver.execute_script("""
                return {
                    navigation: performance.getEntriesByType('navigation')[0],
                    resources: performance.getEntriesByType('resource').length,
                    memory: performance.memory ? {
                        usedJSHeapSize: performance.memory.usedJSHeapSize,
                        totalJSHeapSize: performance.memory.totalJSHeapSize,
                        jsHeapSizeLimit: performance.memory.jsHeapSizeLimit
                    } : null,
                    timing: {
                        domContentLoaded: performance.timing.domContentLoadedEventEnd - performance.timing.navigationStart,
                        fullyLoaded: performance.timing.loadEventEnd - performance.timing.navigationStart,
                        firstPaint: performance.getEntriesByType('paint').find(entry => entry.name === 'first-paint')?.startTime || 0,
                        firstContentfulPaint: performance.getEntriesByType('paint').find(entry => entry.name === 'first-contentful-paint')?.startTime || 0
                    }
                };
            """)

            # 计算性能指标
            metrics = []

            # 页面加载时间
            load_time_level = self._evaluate_performance_level('page_load_time', page_load_time)
            metrics.append(PerformanceMetric(
                metric_name="页面加载时间",
                value=page_load_time,
                unit="秒",
                threshold_excellent=self.performance_thresholds['page_load_time']['excellent'],
                threshold_good=self.performance_thresholds['page_load_time']['good'],
                threshold_fair=self.performance_thresholds['page_load_time']['fair'],
                level=load_time_level,
                details={"navigation_metrics": performance_metrics['timing']}
            ))

            # 内存使用
            if performance_metrics['memory']:
                memory_usage_mb = performance_metrics['memory']['usedJSHeapSize'] / 1024 / 1024
                memory_level = self._evaluate_performance_level('memory_usage', memory_usage_mb)
                metrics.append(PerformanceMetric(
                    metric_name="内存使用",
                    value=memory_usage_mb,
                    unit="MB",
                    threshold_excellent=self.performance_thresholds['memory_usage']['excellent'],
                    threshold_good=self.performance_thresholds['memory_usage']['good'],
                    threshold_fair=self.performance_thresholds['memory_usage']['fair'],
                    level=memory_level,
                    details=performance_metrics['memory']
                ))

            # 资源加载数量
            metrics.append(PerformanceMetric(
                metric_name="资源加载数量",
                value=performance_metrics['resources'],
                unit="个",
                threshold_excellent=50,
                threshold_good=100,
                threshold_fair=200,
                level=self._evaluate_performance_level_by_range(performance_metrics['resources'], 50, 100, 200, reverse=True),
                details={"resource_count": performance_metrics['resources']}
            ))

            execution_time = time.time() - start_time

            # 判断总体结果
            overall_level = min([m.level for m in metrics], key=lambda x: list(PerformanceLevel).index(x))
            status = "passed" if overall_level in [PerformanceLevel.EXCELLENT, PerformanceLevel.GOOD] else "failed"

            return PerformanceTestResult(
                test_name="页面加载性能",
                test_category="performance",
                status=status,
                execution_time=execution_time,
                metrics=metrics,
                message=f"页面加载完成，总体性能等级: {overall_level.value}",
                details={"performance_metrics": performance_metrics}
            )

        except Exception as e:
            execution_time = time.time() - start_time
            return PerformanceTestResult(
                test_name="页面加载性能",
                test_category="performance",
                status="error",
                execution_time=execution_time,
                metrics=[],
                message=f"页面加载性能测试失败: {str(e)}",
                details={"error": str(e)}
            )

    def test_widget_render_performance(self) -> PerformanceTestResult:
        """测试Widget渲染性能"""
        start_time = time.time()

        try:
            # 导航到Dashboard页面
            self.driver.get(f"{self.base_url}/dashboard")

            # 等待基础页面加载
            self.wait.until(EC.presence_of_element_located((By.TAG_NAME, "body")))

            # 查找所有Widget
            widget_selectors = [
                '.update-activity-widget',
                '.realtime-monitor-widget',
                '.security-dashboard-widget',
                '.health-monitor-widget',
                '.recent-activities-widget',
                '.system-overview-widget',
                '.quick-actions-widget',
                '.notification-center-widget',
                '.resource-charts-widget',
                '.container-stats-widget'
            ]

            widget_performance = []

            for selector in widget_selectors:
                try:
                    widget_start_time = time.time()

                    # 等待Widget出现
                    widget = self.wait.until(
                        EC.presence_of_element_located((By.CSS_SELECTOR, selector))
                    )

                    # 等待Widget完全渲染
                    self.wait.until(
                        EC.visibility_of(widget)
                    )

                    widget_render_time = time.time() - widget_start_time

                    # 检查Widget是否有内容
                    has_content = len(widget.text.strip()) > 0

                    widget_performance.append({
                        'selector': selector,
                        'render_time': widget_render_time,
                        'has_content': has_content,
                        'visible': widget.is_displayed(),
                        'size': widget.size
                    })

                except TimeoutException:
                    widget_performance.append({
                        'selector': selector,
                        'render_time': 30.0,  # 超时时间
                        'has_content': False,
                        'visible': False,
                        'error': 'timeout'
                    })

            # 计算性能指标
            metrics = []

            # 平均渲染时间
            render_times = [w['render_time'] for w in widget_performance if 'error' not in w]
            if render_times:
                avg_render_time = statistics.mean(render_times)
                max_render_time = max(render_times)
                min_render_time = min(render_times)

                avg_level = self._evaluate_performance_level('widget_render_time', avg_render_time)
                max_level = self._evaluate_performance_level('widget_render_time', max_render_time)

                metrics.append(PerformanceMetric(
                    metric_name="平均Widget渲染时间",
                    value=avg_render_time,
                    unit="秒",
                    threshold_excellent=self.performance_thresholds['widget_render_time']['excellent'],
                    threshold_good=self.performance_thresholds['widget_render_time']['good'],
                    threshold_fair=self.performance_thresholds['widget_render_time']['fair'],
                    level=avg_level,
                    details={"min": min_render_time, "max": max_render_time, "count": len(render_times)}
                ))

                metrics.append(PerformanceMetric(
                    metric_name="最慢Widget渲染时间",
                    value=max_render_time,
                    unit="秒",
                    threshold_excellent=self.performance_thresholds['widget_render_time']['excellent'],
                    threshold_good=self.performance_thresholds['widget_render_time']['good'],
                    threshold_fair=self.performance_thresholds['widget_render_time']['fair'],
                    level=max_level,
                    details={"slowest_widget": max(widget_performance, key=lambda x: x.get('render_time', 0))['selector']}
                ))

            # Widget加载成功率
            successful_widgets = len([w for w in widget_performance if w.get('visible', False) and w.get('has_content', False)])
            success_rate = (successful_widgets / len(widget_selectors)) * 100 if widget_selectors else 0

            metrics.append(PerformanceMetric(
                metric_name="Widget加载成功率",
                value=success_rate,
                unit="%",
                threshold_excellent=95,
                threshold_good=85,
                threshold_fair=70,
                level=self._evaluate_performance_level_by_range(success_rate, 95, 85, 70),
                details={"successful_widgets": successful_widgets, "total_widgets": len(widget_selectors)}
            ))

            execution_time = time.time() - start_time

            # 判断总体结果
            overall_level = min([m.level for m in metrics], key=lambda x: list(PerformanceLevel).index(x)) if metrics else PerformanceLevel.CRITICAL
            status = "passed" if overall_level in [PerformanceLevel.EXCELLENT, PerformanceLevel.GOOD] else "failed"

            return PerformanceTestResult(
                test_name="Widget渲染性能",
                test_category="performance",
                status=status,
                execution_time=execution_time,
                metrics=metrics,
                message=f"Widget渲染完成，总体性能等级: {overall_level.value}",
                details={"widget_performance": widget_performance}
            )

        except Exception as e:
            execution_time = time.time() - start_time
            return PerformanceTestResult(
                test_name="Widget渲染性能",
                test_category="performance",
                status="error",
                execution_time=execution_time,
                metrics=[],
                message=f"Widget渲染性能测试失败: {str(e)}",
                details={"error": str(e)}
            )

    def test_api_performance(self) -> PerformanceTestResult:
        """测试API性能"""
        start_time = time.time()

        api_endpoints = [
            '/api/monitoring/metrics',
            '/api/containers',
            '/api/updates',
            '/api/security/status',
            '/api/system/overview',
            '/api/notifications',
            '/api/activities',
            '/api/monitoring/events',
            '/api/containers/health'
        ]

        try:
            api_results = []

            # 并发测试API性能
            with ThreadPoolExecutor(max_workers=5) as executor:
                futures = {
                    executor.submit(self._test_single_api, endpoint): endpoint
                    for endpoint in api_endpoints
                }

                for future in as_completed(futures):
                    endpoint = futures[future]
                    try:
                        result = future.result()
                        result['endpoint'] = endpoint
                        api_results.append(result)
                    except Exception as e:
                        api_results.append({
                            'endpoint': endpoint,
                            'response_time': 30.0,
                            'status_code': 0,
                            'error': str(e)
                        })

            # 计算性能指标
            metrics = []

            successful_requests = [r for r in api_results if 200 <= r.get('status_code', 0) < 300]
            response_times = [r['response_time'] for r in successful_requests]

            if response_times:
                avg_response_time = statistics.mean(response_times)
                max_response_time = max(response_times)
                min_response_time = min(response_times)

                avg_level = self._evaluate_performance_level('api_response_time', avg_response_time)
                max_level = self._evaluate_performance_level('api_response_time', max_response_time)

                metrics.append(PerformanceMetric(
                    metric_name="平均API响应时间",
                    value=avg_response_time,
                    unit="秒",
                    threshold_excellent=self.performance_thresholds['api_response_time']['excellent'],
                    threshold_good=self.performance_thresholds['api_response_time']['good'],
                    threshold_fair=self.performance_thresholds['api_response_time']['fair'],
                    level=avg_level,
                    details={"min": min_response_time, "max": max_response_time}
                ))

                metrics.append(PerformanceMetric(
                    metric_name="最慢API响应时间",
                    value=max_response_time,
                    unit="秒",
                    threshold_excellent=self.performance_thresholds['api_response_time']['excellent'],
                    threshold_good=self.performance_thresholds['api_response_time']['good'],
                    threshold_fair=self.performance_thresholds['api_response_time']['fair'],
                    level=max_level,
                    details={"slowest_api": max(successful_requests, key=lambda x: x['response_time'])['endpoint']}
                ))

            # API成功率
            success_rate = (len(successful_requests) / len(api_endpoints)) * 100 if api_endpoints else 0

            metrics.append(PerformanceMetric(
                metric_name="API成功率",
                value=success_rate,
                unit="%",
                threshold_excellent=95,
                threshold_good=90,
                threshold_fair=80,
                level=self._evaluate_performance_level_by_range(success_rate, 95, 90, 80),
                details={"successful_requests": len(successful_requests), "total_requests": len(api_endpoints)}
            ))

            execution_time = time.time() - start_time

            # 判断总体结果
            overall_level = min([m.level for m in metrics], key=lambda x: list(PerformanceLevel).index(x)) if metrics else PerformanceLevel.CRITICAL
            status = "passed" if overall_level in [PerformanceLevel.EXCELLENT, PerformanceLevel.GOOD] else "failed"

            return PerformanceTestResult(
                test_name="API性能",
                test_category="performance",
                status=status,
                execution_time=execution_time,
                metrics=metrics,
                message=f"API性能测试完成，总体性能等级: {overall_level.value}",
                details={"api_results": api_results}
            )

        except Exception as e:
            execution_time = time.time() - start_time
            return PerformanceTestResult(
                test_name="API性能",
                test_category="performance",
                status="error",
                execution_time=execution_time,
                metrics=[],
                message=f"API性能测试失败: {str(e)}",
                details={"error": str(e)}
            )

    def test_memory_leak_detection(self, duration_minutes: int = 5) -> PerformanceTestResult:
        """内存泄漏检测"""
        start_time = time.time()

        try:
            print(f"🔍 开始 {duration_minutes} 分钟内存泄漏检测...")

            # 导航到Dashboard页面
            self.driver.get(f"{self.base_url}/dashboard")
            self.wait.until(EC.presence_of_element_located((By.TAG_NAME, "body")))

            # 强制垃圾回收获取基线
            self.driver.execute_script("if (window.gc) { window.gc(); }")
            time.sleep(2)

            # 获取初始内存使用
            initial_memory = self._get_browser_memory_usage()
            memory_samples = [initial_memory]
            sample_times = [0]

            # 开始内存监控
            monitoring_duration = duration_minutes * 60  # 转换为秒
            sample_interval = 10  # 每10秒采样一次

            for elapsed in range(sample_interval, int(monitoring_duration) + 1, sample_interval):
                # 执行一些操作模拟用户交互
                self._simulate_user_interaction()

                # 采样内存使用
                current_memory = self._get_browser_memory_usage()
                memory_samples.append(current_memory)
                sample_times.append(elapsed)

                time.sleep(sample_interval)

            # 分析内存使用趋势
            memory_trend = self._analyze_memory_trend(memory_samples, sample_times)

            metrics = []

            # 内存增长率
            memory_growth_rate = memory_trend['growth_rate_per_minute']
            growth_level = self._evaluate_performance_level('memory_leak_rate', abs(memory_growth_rate))

            metrics.append(PerformanceMetric(
                metric_name="内存增长率",
                value=memory_growth_rate,
                unit="MB/分钟",
                threshold_excellent=self.performance_thresholds['memory_leak_rate']['excellent'],
                threshold_good=self.performance_thresholds['memory_leak_rate']['good'],
                threshold_fair=self.performance_thresholds['memory_leak_rate']['fair'],
                level=growth_level,
                details=memory_trend
            ))

            # 最大内存使用
            max_memory = max(memory_samples)
            max_memory_level = self._evaluate_performance_level('memory_usage', max_memory)

            metrics.append(PerformanceMetric(
                metric_name="最大内存使用",
                value=max_memory,
                unit="MB",
                threshold_excellent=self.performance_thresholds['memory_usage']['excellent'],
                threshold_good=self.performance_thresholds['memory_usage']['good'],
                threshold_fair=self.performance_thresholds['memory_usage']['fair'],
                level=max_memory_level,
                details={"initial": initial_memory, "peak": max_memory, "samples": len(memory_samples)}
            ))

            # 内存稳定性
            memory_variance = statistics.variance(memory_samples) if len(memory_samples) > 1 else 0
            stability_level = self._evaluate_memory_stability(memory_variance)

            metrics.append(PerformanceMetric(
                metric_name="内存稳定性",
                value=memory_variance,
                unit="MB²",
                threshold_excellent=100,
                threshold_good=500,
                threshold_fair=1000,
                level=stability_level,
                details={"variance": memory_variance, "std_dev": memory_variance ** 0.5}
            ))

            execution_time = time.time() - start_time

            # 判断是否有内存泄漏
            has_memory_leak = memory_growth_rate > 2.0  # 超过2MB/分钟认为有泄漏
            overall_level = min([m.level for m in metrics], key=lambda x: list(PerformanceLevel).index(x))
            status = "failed" if has_memory_leak else "passed"

            message = "检测到内存泄漏" if has_memory_leak else f"内存使用正常，总体等级: {overall_level.value}"

            return PerformanceTestResult(
                test_name="内存泄漏检测",
                test_category="stability",
                status=status,
                execution_time=execution_time,
                metrics=metrics,
                message=message,
                details={
                    "duration_minutes": duration_minutes,
                    "memory_samples": memory_samples,
                    "sample_times": sample_times,
                    "has_memory_leak": has_memory_leak
                }
            )

        except Exception as e:
            execution_time = time.time() - start_time
            return PerformanceTestResult(
                test_name="内存泄漏检测",
                test_category="stability",
                status="error",
                execution_time=execution_time,
                metrics=[],
                message=f"内存泄漏检测失败: {str(e)}",
                details={"error": str(e)}
            )

    def test_concurrent_user_simulation(self, user_count: int = 10) -> PerformanceTestResult:
        """并发用户模拟测试"""
        start_time = time.time()

        try:
            print(f"🔍 模拟 {user_count} 个并发用户...")

            # 创建多个WebDriver实例模拟并发用户
            concurrent_results = []

            def simulate_single_user(user_id: int):
                """模拟单个用户行为"""
                try:
                    # 创建独立的Chrome实例
                    chrome_options = Options()
                    chrome_options.add_argument("--headless")
                    chrome_options.add_argument("--no-sandbox")
                    chrome_options.add_argument("--disable-dev-shm-usage")

                    user_driver = webdriver.Chrome(options=chrome_options)
                    user_start_time = time.time()

                    # 用户访问Dashboard
                    user_driver.get(f"{self.base_url}/dashboard")

                    # 等待页面加载
                    user_wait = WebDriverWait(user_driver, 30)
                    user_wait.until(EC.presence_of_element_located((By.TAG_NAME, "body")))

                    # 模拟用户交互
                    time.sleep(2)  # 浏览时间

                    # 点击一些元素
                    clickable_elements = user_driver.find_elements(By.CSS_SELECTOR, "button, .el-button")
                    if clickable_elements:
                        clickable_elements[0].click()
                        time.sleep(1)

                    user_load_time = time.time() - user_start_time

                    user_driver.quit()

                    return {
                        'user_id': user_id,
                        'load_time': user_load_time,
                        'status': 'success'
                    }

                except Exception as e:
                    return {
                        'user_id': user_id,
                        'load_time': 30.0,
                        'status': 'error',
                        'error': str(e)
                    }

            # 并发执行用户模拟
            with ThreadPoolExecutor(max_workers=min(user_count, 5)) as executor:  # 限制最大并发数
                futures = [executor.submit(simulate_single_user, i) for i in range(user_count)]

                for future in as_completed(futures):
                    result = future.result()
                    concurrent_results.append(result)

            # 分析并发性能
            successful_users = [r for r in concurrent_results if r['status'] == 'success']
            failed_users = [r for r in concurrent_results if r['status'] == 'error']

            metrics = []

            if successful_users:
                load_times = [r['load_time'] for r in successful_users]
                avg_load_time = statistics.mean(load_times)
                max_load_time = max(load_times)

                # 平均负载时间
                avg_level = self._evaluate_performance_level('page_load_time', avg_load_time)
                metrics.append(PerformanceMetric(
                    metric_name="并发平均加载时间",
                    value=avg_load_time,
                    unit="秒",
                    threshold_excellent=self.performance_thresholds['page_load_time']['excellent'] * 1.5,  # 并发时放宽阈值
                    threshold_good=self.performance_thresholds['page_load_time']['good'] * 1.5,
                    threshold_fair=self.performance_thresholds['page_load_time']['fair'] * 1.5,
                    level=avg_level,
                    details={"max": max_load_time, "min": min(load_times)}
                ))

            # 并发成功率
            success_rate = (len(successful_users) / user_count) * 100
            success_level = self._evaluate_performance_level_by_range(success_rate, 95, 90, 80)

            metrics.append(PerformanceMetric(
                metric_name="并发成功率",
                value=success_rate,
                unit="%",
                threshold_excellent=95,
                threshold_good=90,
                threshold_fair=80,
                level=success_level,
                details={"successful_users": len(successful_users), "failed_users": len(failed_users)}
            ))

            # 系统资源使用
            system_cpu = psutil.cpu_percent(interval=1)
            system_memory = psutil.virtual_memory().percent

            cpu_level = self._evaluate_performance_level('cpu_usage', system_cpu)
            memory_level = self._evaluate_performance_level_by_range(system_memory, 70, 85, 95, reverse=True)

            metrics.extend([
                PerformanceMetric(
                    metric_name="系统CPU使用率",
                    value=system_cpu,
                    unit="%",
                    threshold_excellent=self.performance_thresholds['cpu_usage']['excellent'],
                    threshold_good=self.performance_thresholds['cpu_usage']['good'],
                    threshold_fair=self.performance_thresholds['cpu_usage']['fair'],
                    level=cpu_level,
                    details={"during_concurrent_test": True}
                ),
                PerformanceMetric(
                    metric_name="系统内存使用率",
                    value=system_memory,
                    unit="%",
                    threshold_excellent=70,
                    threshold_good=85,
                    threshold_fair=95,
                    level=memory_level,
                    details={"during_concurrent_test": True}
                )
            ])

            execution_time = time.time() - start_time

            # 判断总体结果
            overall_level = min([m.level for m in metrics], key=lambda x: list(PerformanceLevel).index(x)) if metrics else PerformanceLevel.CRITICAL
            status = "passed" if overall_level in [PerformanceLevel.EXCELLENT, PerformanceLevel.GOOD] and success_rate >= 90 else "failed"

            return PerformanceTestResult(
                test_name="并发用户模拟",
                test_category="stability",
                status=status,
                execution_time=execution_time,
                metrics=metrics,
                message=f"并发测试完成，{user_count}用户，成功率{success_rate:.1f}%",
                details={"concurrent_results": concurrent_results, "user_count": user_count}
            )

        except Exception as e:
            execution_time = time.time() - start_time
            return PerformanceTestResult(
                test_name="并发用户模拟",
                test_category="stability",
                status="error",
                execution_time=execution_time,
                metrics=[],
                message=f"并发用户模拟测试失败: {str(e)}",
                details={"error": str(e), "user_count": user_count}
            )

    def run_all_performance_tests(self) -> List[PerformanceTestResult]:
        """运行所有性能测试"""
        print("🚀 开始运行Dashboard性能和稳定性验证测试")
        print("严格验证真实环境下的性能表现和生产级质量标准")

        all_results = []

        # 设置浏览器
        self.setup_driver()

        try:
            # 页面加载性能测试
            print("\n📊 测试页面加载性能...")
            result = self.test_page_load_performance()
            all_results.append(result)
            self._print_test_result(result)

            # Widget渲染性能测试
            print("\n🖼️ 测试Widget渲染性能...")
            result = self.test_widget_render_performance()
            all_results.append(result)
            self._print_test_result(result)

            # API性能测试
            print("\n🌐 测试API性能...")
            result = self.test_api_performance()
            all_results.append(result)
            self._print_test_result(result)

            # 内存泄漏检测（较短时间用于演示）
            print("\n🧠 测试内存稳定性...")
            result = self.test_memory_leak_detection(duration_minutes=2)
            all_results.append(result)
            self._print_test_result(result)

            # 并发用户模拟
            print("\n👥 测试并发用户负载...")
            result = self.test_concurrent_user_simulation(user_count=5)
            all_results.append(result)
            self._print_test_result(result)

        finally:
            self.teardown_driver()

        return all_results

    def _test_single_api(self, endpoint: str) -> Dict:
        """测试单个API端点"""
        start_time = time.time()
        try:
            response = self.session.get(f"{self.api_base_url}{endpoint}", timeout=10)
            response_time = time.time() - start_time

            return {
                'response_time': response_time,
                'status_code': response.status_code,
                'response_size': len(response.content)
            }
        except Exception as e:
            response_time = time.time() - start_time
            return {
                'response_time': response_time,
                'status_code': 0,
                'error': str(e)
            }

    def _get_browser_memory_usage(self) -> float:
        """获取浏览器内存使用（MB）"""
        try:
            memory_info = self.driver.execute_script("""
                return {
                    usedJSHeapSize: performance.memory ? performance.memory.usedJSHeapSize : 0,
                    totalJSHeapSize: performance.memory ? performance.memory.totalJSHeapSize : 0
                };
            """)
            return memory_info['usedJSHeapSize'] / 1024 / 1024  # 转换为MB
        except:
            return 0

    def _simulate_user_interaction(self):
        """模拟用户交互"""
        try:
            # 滚动页面
            self.driver.execute_script("window.scrollBy(0, 200);")
            time.sleep(0.5)

            # 查找并点击按钮
            buttons = self.driver.find_elements(By.CSS_SELECTOR, "button, .el-button")
            if buttons and len(buttons) > 0:
                button_index = len(buttons) % 3  # 循环点击不同按钮
                if button_index < len(buttons):
                    try:
                        buttons[button_index].click()
                        time.sleep(0.5)
                    except:
                        pass
        except:
            pass

    def _analyze_memory_trend(self, memory_samples: List[float], sample_times: List[int]) -> Dict:
        """分析内存使用趋势"""
        if len(memory_samples) < 2:
            return {"growth_rate_per_minute": 0, "trend": "insufficient_data"}

        # 使用线性回归分析趋势
        n = len(memory_samples)
        sum_x = sum(sample_times)
        sum_y = sum(memory_samples)
        sum_xy = sum(x * y for x, y in zip(sample_times, memory_samples))
        sum_x2 = sum(x * x for x in sample_times)

        # 计算斜率 (增长率 per second)
        slope = (n * sum_xy - sum_x * sum_y) / (n * sum_x2 - sum_x * sum_x) if (n * sum_x2 - sum_x * sum_x) != 0 else 0

        # 转换为每分钟的增长率
        growth_rate_per_minute = slope * 60

        # 计算相关系数
        mean_x = sum_x / n
        mean_y = sum_y / n
        correlation = sum((x - mean_x) * (y - mean_y) for x, y in zip(sample_times, memory_samples))
        correlation = correlation / (sum((x - mean_x) ** 2 for x in sample_times) * sum((y - mean_y) ** 2 for y in memory_samples)) ** 0.5 if correlation != 0 else 0

        return {
            "growth_rate_per_minute": growth_rate_per_minute,
            "correlation": correlation,
            "trend": "increasing" if growth_rate_per_minute > 1 else "stable" if growth_rate_per_minute > -1 else "decreasing",
            "initial_memory": memory_samples[0],
            "final_memory": memory_samples[-1],
            "peak_memory": max(memory_samples),
            "min_memory": min(memory_samples)
        }

    def _evaluate_performance_level(self, metric_type: str, value: float) -> PerformanceLevel:
        """评估性能等级"""
        thresholds = self.performance_thresholds.get(metric_type, {})
        excellent = thresholds.get('excellent', 0)
        good = thresholds.get('good', 0)
        fair = thresholds.get('fair', 0)

        if value <= excellent:
            return PerformanceLevel.EXCELLENT
        elif value <= good:
            return PerformanceLevel.GOOD
        elif value <= fair:
            return PerformanceLevel.FAIR
        else:
            return PerformanceLevel.POOR

    def _evaluate_performance_level_by_range(self, value: float, excellent: float, good: float, fair: float, reverse: bool = False) -> PerformanceLevel:
        """按范围评估性能等级"""
        if reverse:
            if value >= excellent:
                return PerformanceLevel.EXCELLENT
            elif value >= good:
                return PerformanceLevel.GOOD
            elif value >= fair:
                return PerformanceLevel.FAIR
            else:
                return PerformanceLevel.POOR
        else:
            if value <= excellent:
                return PerformanceLevel.EXCELLENT
            elif value <= good:
                return PerformanceLevel.GOOD
            elif value <= fair:
                return PerformanceLevel.FAIR
            else:
                return PerformanceLevel.POOR

    def _evaluate_memory_stability(self, variance: float) -> PerformanceLevel:
        """评估内存稳定性"""
        if variance <= 100:
            return PerformanceLevel.EXCELLENT
        elif variance <= 500:
            return PerformanceLevel.GOOD
        elif variance <= 1000:
            return PerformanceLevel.FAIR
        else:
            return PerformanceLevel.POOR

    def _print_test_result(self, result: PerformanceTestResult):
        """打印测试结果"""
        status_icon = {"passed": "✅", "failed": "❌", "error": "💥"}
        print(f"  {status_icon.get(result.status, '❓')} {result.test_name}: {result.message}")

        for metric in result.metrics:
            level_icon = {
                PerformanceLevel.EXCELLENT: "🟢",
                PerformanceLevel.GOOD: "🔵",
                PerformanceLevel.FAIR: "🟡",
                PerformanceLevel.POOR: "🟠",
                PerformanceLevel.CRITICAL: "🔴"
            }
            print(f"    {level_icon.get(metric.level, '❓')} {metric.metric_name}: {metric.value:.2f} {metric.unit} ({metric.level.value})")

    def generate_performance_report(self, results: List[PerformanceTestResult]) -> Dict:
        """生成性能测试报告"""
        report_time = time.strftime("%Y-%m-%d %H:%M:%S")

        # 统计数据
        total_tests = len(results)
        passed_tests = len([r for r in results if r.status == "passed"])
        failed_tests = len([r for r in results if r.status == "failed"])
        error_tests = len([r for r in results if r.status == "error"])

        # 收集所有性能指标
        all_metrics = []
        for result in results:
            all_metrics.extend(result.metrics)

        # 按性能等级分组
        metrics_by_level = {}
        for level in PerformanceLevel:
            metrics_by_level[level.value] = [m for m in all_metrics if m.level == level]

        # 计算总执行时间
        total_execution_time = sum(r.execution_time for r in results)

        # 关键性能指标汇总
        key_metrics = {
            'page_load_time': [m for m in all_metrics if '加载时间' in m.metric_name],
            'memory_usage': [m for m in all_metrics if '内存' in m.metric_name],
            'api_performance': [m for m in all_metrics if 'API' in m.metric_name],
            'stability': [m for m in all_metrics if '稳定' in m.metric_name or '泄漏' in m.metric_name]
        }

        report = {
            "metadata": {
                "report_time": report_time,
                "total_execution_time": round(total_execution_time, 2),
                "test_framework": "Dashboard Performance Test Suite v1.0.0",
                "environment": {
                    "base_url": self.base_url,
                    "api_base_url": self.api_base_url
                }
            },
            "summary": {
                "total_tests": total_tests,
                "passed": passed_tests,
                "failed": failed_tests,
                "errors": error_tests,
                "success_rate": round((passed_tests / total_tests * 100) if total_tests > 0 else 0, 1),
                "overall_performance_level": self._get_overall_performance_level(all_metrics),
                "total_metrics_evaluated": len(all_metrics)
            },
            "performance_breakdown": {
                "by_level": {
                    level: {
                        "count": len(metrics),
                        "percentage": round((len(metrics) / len(all_metrics) * 100) if all_metrics else 0, 1)
                    }
                    for level, metrics in metrics_by_level.items()
                },
                "by_category": {
                    category: {
                        "metric_count": len(metrics),
                        "avg_performance": self._get_average_performance_level(metrics) if metrics else "N/A"
                    }
                    for category, metrics in key_metrics.items()
                }
            },
            "detailed_results": [
                {
                    "test_name": r.test_name,
                    "test_category": r.test_category,
                    "status": r.status,
                    "execution_time": r.execution_time,
                    "message": r.message,
                    "metrics": [
                        {
                            "metric_name": m.metric_name,
                            "value": m.value,
                            "unit": m.unit,
                            "level": m.level.value,
                            "thresholds": {
                                "excellent": m.threshold_excellent,
                                "good": m.threshold_good,
                                "fair": m.threshold_fair
                            },
                            "details": m.details
                        }
                        for m in r.metrics
                    ],
                    "details": r.details
                }
                for r in results
            ],
            "critical_issues": [
                {
                    "test_name": r.test_name,
                    "message": r.message,
                    "critical_metrics": [
                        {
                            "metric_name": m.metric_name,
                            "value": m.value,
                            "unit": m.unit,
                            "level": m.level.value
                        }
                        for m in r.metrics
                        if m.level in [PerformanceLevel.POOR, PerformanceLevel.CRITICAL]
                    ]
                }
                for r in results
                if any(m.level in [PerformanceLevel.POOR, PerformanceLevel.CRITICAL] for m in r.metrics)
            ],
            "performance_thresholds": self.performance_thresholds,
            "recommendations": self._generate_performance_recommendations(results)
        }

        return report

    def save_performance_report(self, report: Dict, output_path: str = None):
        """保存性能测试报告"""
        if output_path is None:
            timestamp = time.strftime("%Y%m%d_%H%M%S")
            output_path = f"/home/await/project/docker-auto/frontend/tests/dashboard/performance_report_{timestamp}.json"

        try:
            Path(output_path).parent.mkdir(parents=True, exist_ok=True)
            with open(output_path, 'w', encoding='utf-8') as f:
                json.dump(report, f, ensure_ascii=False, indent=2)
            print(f"📊 性能测试报告已保存: {output_path}")
        except Exception as e:
            print(f"❌ 保存性能测试报告失败: {e}")

    def print_performance_summary(self, report: Dict):
        """打印性能测试摘要"""
        print("\n" + "="*80)
        print("🔍 Dashboard 性能和稳定性验证测试报告")
        print("="*80)

        summary = report["summary"]
        print(f"📊 测试摘要:")
        print(f"   • 总测试数: {summary['total_tests']}")
        print(f"   • 通过: {summary['passed']} ✅")
        print(f"   • 失败: {summary['failed']} ❌")
        print(f"   • 错误: {summary['errors']} 💥")
        print(f"   • 成功率: {summary['success_rate']}%")
        print(f"   • 总体性能等级: {summary['overall_performance_level']}")
        print(f"   • 评估指标数: {summary['total_metrics_evaluated']}")
        print(f"   • 执行时间: {report['metadata']['total_execution_time']}秒")

        # 性能分布
        print(f"\n⚡ 性能分布:")
        breakdown = report["performance_breakdown"]["by_level"]
        for level, data in breakdown.items():
            if data["count"] > 0:
                level_icon = {
                    "excellent": "🟢",
                    "good": "🔵",
                    "fair": "🟡",
                    "poor": "🟠",
                    "critical": "🔴"
                }
                print(f"   {level_icon.get(level, '❓')} {level.upper()}: {data['count']} 项 ({data['percentage']}%)")

        # 关键性能指标
        print(f"\n📈 关键性能类别:")
        categories = report["performance_breakdown"]["by_category"]
        for category, data in categories.items():
            if data["metric_count"] > 0:
                print(f"   • {category}: {data['metric_count']} 个指标, 平均性能: {data['avg_performance']}")

        # 临界问题
        critical_issues = report["critical_issues"]
        print(f"\n🚨 性能问题:")
        if critical_issues:
            for issue in critical_issues[:3]:  # 只显示前3个
                print(f"   • {issue['test_name']}: {issue['message']}")
                for metric in issue['critical_metrics'][:2]:  # 每个问题只显示前2个指标
                    print(f"     - {metric['metric_name']}: {metric['value']:.2f} {metric['unit']} ({metric['level']})")
        else:
            print("   • 无性能问题 ✅")

        print(f"\n📋 优化建议:")
        recommendations = report["recommendations"]
        for i, rec in enumerate(recommendations[:5], 1):
            print(f"   {i}. {rec}")

        print("\n" + "="*80)

    def _get_overall_performance_level(self, metrics: List[PerformanceMetric]) -> str:
        """获取总体性能等级"""
        if not metrics:
            return "未知"

        level_counts = {}
        for metric in metrics:
            level_counts[metric.level] = level_counts.get(metric.level, 0) + 1

        # 按严重程度排序
        for level in [PerformanceLevel.CRITICAL, PerformanceLevel.POOR, PerformanceLevel.FAIR, PerformanceLevel.GOOD, PerformanceLevel.EXCELLENT]:
            if level_counts.get(level, 0) > 0:
                percentage = (level_counts[level] / len(metrics)) * 100
                return f"{level.value} ({percentage:.1f}%)"

        return "未知"

    def _get_average_performance_level(self, metrics: List[PerformanceMetric]) -> str:
        """获取平均性能等级"""
        if not metrics:
            return "N/A"

        level_scores = {
            PerformanceLevel.EXCELLENT: 5,
            PerformanceLevel.GOOD: 4,
            PerformanceLevel.FAIR: 3,
            PerformanceLevel.POOR: 2,
            PerformanceLevel.CRITICAL: 1
        }

        avg_score = sum(level_scores[m.level] for m in metrics) / len(metrics)

        if avg_score >= 4.5:
            return "excellent"
        elif avg_score >= 3.5:
            return "good"
        elif avg_score >= 2.5:
            return "fair"
        elif avg_score >= 1.5:
            return "poor"
        else:
            return "critical"

    def _generate_performance_recommendations(self, results: List[PerformanceTestResult]) -> List[str]:
        """生成性能优化建议"""
        recommendations = []

        # 分析测试结果中的问题
        all_metrics = []
        for result in results:
            all_metrics.extend(result.metrics)

        # 页面加载问题
        slow_load_metrics = [m for m in all_metrics if '加载' in m.metric_name and m.level in [PerformanceLevel.POOR, PerformanceLevel.CRITICAL]]
        if slow_load_metrics:
            recommendations.append("优化页面加载性能：压缩静态资源，使用CDN，启用浏览器缓存")

        # 内存问题
        memory_issues = [m for m in all_metrics if '内存' in m.metric_name and m.level in [PerformanceLevel.POOR, PerformanceLevel.CRITICAL]]
        if memory_issues:
            recommendations.append("优化内存使用：清理不必要的事件监听器，优化组件生命周期，使用对象池")

        # API性能问题
        api_issues = [m for m in all_metrics if 'API' in m.metric_name and m.level in [PerformanceLevel.POOR, PerformanceLevel.CRITICAL]]
        if api_issues:
            recommendations.append("优化API性能：实施请求缓存，优化数据库查询，使用连接池")

        # Widget渲染问题
        render_issues = [m for m in all_metrics if 'Widget' in m.metric_name and m.level in [PerformanceLevel.POOR, PerformanceLevel.CRITICAL]]
        if render_issues:
            recommendations.append("优化Widget渲染：使用虚拟滚动，延迟加载，避免不必要的重渲染")

        # 并发性能问题
        concurrent_issues = [m for m in all_metrics if '并发' in m.metric_name and m.level in [PerformanceLevel.POOR, PerformanceLevel.CRITICAL]]
        if concurrent_issues:
            recommendations.append("提升并发处理能力：增加服务器资源，优化数据库连接池，实施负载均衡")

        # 通用建议
        recommendations.extend([
            "建立性能监控和告警系统",
            "定期进行性能基准测试",
            "实施前端资源优化（代码分割、懒加载）",
            "优化网络请求（HTTP/2、请求合并、预取）",
            "建立性能预算和持续集成检查"
        ])

        return recommendations

def main():
    """主函数"""
    print("🚀 启动Dashboard性能和稳定性验证测试套件")
    print("严格验证真实环境下的性能表现和生产级质量标准")

    # 初始化测试器
    tester = DashboardPerformanceTester()

    try:
        # 运行所有性能测试
        results = tester.run_all_performance_tests()

        # 生成报告
        report = tester.generate_performance_report(results)

        # 打印摘要
        tester.print_performance_summary(report)

        # 保存报告
        tester.save_performance_report(report)

        print("\n🎯 Dashboard性能和稳定性验证测试完成！")
        return report

    except Exception as e:
        print(f"❌ 性能测试套件执行失败: {e}")
        raise

if __name__ == "__main__":
    main()
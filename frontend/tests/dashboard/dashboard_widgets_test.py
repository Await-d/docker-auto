#!/usr/bin/env python3
"""
🔍 Dashboard Widget功能测试套件
专注于验证Widget组件真实数据集成的完整性和质量

严格遵循三大核心原则：
1. 🚫 绝不使用模拟数据！！！ - 验证所有组件完全消除模拟数据
2. 🚫 绝不使用简化方案！！！ - 验证实现了完整的功能和错误处理
3. 🚫 绝不使用临时方案！！！ - 确保所有实现达到生产级质量标准
"""

import pytest
import asyncio
import json
import time
import requests
import websockets
from pathlib import Path
from typing import Dict, List, Optional, Any
from dataclasses import dataclass, field
from enum import Enum
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.chrome.options import Options
from selenium.common.exceptions import TimeoutException, WebDriverException

class TestSeverity(Enum):
    """测试严重程度"""
    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"

@dataclass
class TestResult:
    """测试结果"""
    test_name: str
    component_name: str
    status: str  # passed, failed, error
    severity: TestSeverity
    message: str
    execution_time: float
    details: Dict = field(default_factory=dict)

class DashboardWidgetTester:
    """Dashboard Widget测试器"""

    def __init__(self, base_url: str = "http://localhost:3000"):
        self.base_url = base_url
        self.api_base_url = "http://localhost:8080/api"
        self.websocket_url = "ws://localhost:8080/ws"
        self.driver = None
        self.test_results = []
        self.timeout = 30

        # Widget组件映射
        self.widgets = {
            'update_activity': {
                'name': 'UpdateActivity Widget',
                'selector': '.update-activity-widget',
                'expected_api_endpoints': ['/api/updates', '/api/containers/updates'],
                'expected_data_fields': ['container_name', 'update_status', 'timestamp'],
                'real_time_updates': True
            },
            'realtime_monitor': {
                'name': 'RealtimeMonitor Widget',
                'selector': '.realtime-monitor-widget',
                'expected_api_endpoints': ['/api/monitoring/metrics', '/api/monitoring/events'],
                'expected_data_fields': ['cpu', 'memory', 'network', 'disk'],
                'real_time_updates': True
            },
            'security_dashboard': {
                'name': 'SecurityDashboard Widget',
                'selector': '.security-dashboard-widget',
                'expected_api_endpoints': ['/api/security/status', '/api/security/vulnerabilities'],
                'expected_data_fields': ['security_score', 'vulnerabilities', 'last_scan'],
                'real_time_updates': False
            },
            'health_monitor': {
                'name': 'HealthMonitor Widget',
                'selector': '.health-monitor-widget',
                'expected_api_endpoints': ['/api/containers/health', '/api/system/health'],
                'expected_data_fields': ['container_health', 'system_status'],
                'real_time_updates': True
            },
            'recent_activities': {
                'name': 'RecentActivities Widget',
                'selector': '.recent-activities-widget',
                'expected_api_endpoints': ['/api/activities', '/api/logs/recent'],
                'expected_data_fields': ['activity_type', 'description', 'timestamp'],
                'real_time_updates': True
            },
            'system_overview': {
                'name': 'SystemOverview Widget',
                'selector': '.system-overview-widget',
                'expected_api_endpoints': ['/api/system/overview', '/api/containers/stats'],
                'expected_data_fields': ['total_containers', 'running_containers', 'system_load'],
                'real_time_updates': False
            },
            'quick_actions': {
                'name': 'QuickActions Widget',
                'selector': '.quick-actions-widget',
                'expected_api_endpoints': ['/api/actions', '/api/containers/actions'],
                'expected_data_fields': ['available_actions', 'container_list'],
                'real_time_updates': False
            },
            'notification_center': {
                'name': 'NotificationCenter Widget',
                'selector': '.notification-center-widget',
                'expected_api_endpoints': ['/api/notifications', '/api/alerts'],
                'expected_data_fields': ['notifications', 'unread_count'],
                'real_time_updates': True
            },
            'resource_charts': {
                'name': 'ResourceCharts Widget',
                'selector': '.resource-charts-widget',
                'expected_api_endpoints': ['/api/monitoring/resources', '/api/metrics/history'],
                'expected_data_fields': ['cpu_history', 'memory_history', 'chart_data'],
                'real_time_updates': True
            },
            'container_stats': {
                'name': 'ContainerStats Widget',
                'selector': '.container-stats-widget',
                'expected_api_endpoints': ['/api/containers/stats', '/api/containers/list'],
                'expected_data_fields': ['container_stats', 'resource_usage'],
                'real_time_updates': True
            }
        }

    def setup_driver(self):
        """设置Selenium WebDriver"""
        chrome_options = Options()
        chrome_options.add_argument("--headless")
        chrome_options.add_argument("--no-sandbox")
        chrome_options.add_argument("--disable-dev-shm-usage")
        chrome_options.add_argument("--disable-gpu")
        chrome_options.add_argument("--window-size=1920,1080")

        try:
            self.driver = webdriver.Chrome(options=chrome_options)
            print("✅ Chrome WebDriver 初始化成功")
        except Exception as e:
            print(f"❌ WebDriver 初始化失败: {e}")
            raise

    def teardown_driver(self):
        """清理WebDriver"""
        if self.driver:
            self.driver.quit()
            self.driver = None

    def test_widget_renders_without_mock_data(self, widget_key: str) -> TestResult:
        """测试Widget组件渲染时不包含模拟数据"""
        start_time = time.time()
        widget = self.widgets[widget_key]

        try:
            # 导航到Dashboard页面
            self.driver.get(f"{self.base_url}/dashboard")

            # 等待Widget加载
            wait = WebDriverWait(self.driver, self.timeout)
            widget_element = wait.until(
                EC.presence_of_element_located((By.CSS_SELECTOR, widget['selector']))
            )

            # 检查是否包含模拟数据标识符
            widget_html = widget_element.get_attribute('innerHTML')
            mock_indicators = [
                'mock-data', 'mockData', 'placeholder', 'demo-data',
                'test-data', 'sample-data', 'hardcoded', 'lorem ipsum',
                'coming soon', 'not implemented', 'under development'
            ]

            found_mock_data = []
            for indicator in mock_indicators:
                if indicator.lower() in widget_html.lower():
                    found_mock_data.append(indicator)

            execution_time = time.time() - start_time

            if found_mock_data:
                return TestResult(
                    test_name="render_without_mock_data",
                    component_name=widget['name'],
                    status="failed",
                    severity=TestSeverity.CRITICAL,
                    message=f"发现模拟数据指标: {', '.join(found_mock_data)}",
                    execution_time=execution_time,
                    details={"mock_indicators_found": found_mock_data}
                )
            else:
                return TestResult(
                    test_name="render_without_mock_data",
                    component_name=widget['name'],
                    status="passed",
                    severity=TestSeverity.CRITICAL,
                    message="未发现模拟数据指标",
                    execution_time=execution_time
                )

        except Exception as e:
            execution_time = time.time() - start_time
            return TestResult(
                test_name="render_without_mock_data",
                component_name=widget['name'],
                status="error",
                severity=TestSeverity.CRITICAL,
                message=f"测试执行错误: {str(e)}",
                execution_time=execution_time,
                details={"error": str(e)}
            )

    def test_widget_api_integration(self, widget_key: str) -> TestResult:
        """测试Widget API集成"""
        start_time = time.time()
        widget = self.widgets[widget_key]

        try:
            # 检查API端点是否可访问
            api_results = {}
            for endpoint in widget['expected_api_endpoints']:
                try:
                    full_url = f"{self.api_base_url.rstrip('/')}{endpoint}"
                    response = requests.get(full_url, timeout=10)
                    api_results[endpoint] = {
                        'status_code': response.status_code,
                        'response_time': response.elapsed.total_seconds(),
                        'has_data': len(response.content) > 0
                    }
                except requests.RequestException as e:
                    api_results[endpoint] = {
                        'error': str(e),
                        'status_code': 0
                    }

            execution_time = time.time() - start_time

            # 检查是否有成功的API调用
            successful_apis = [
                endpoint for endpoint, result in api_results.items()
                if result.get('status_code', 0) in [200, 201, 202]
            ]

            if not successful_apis:
                return TestResult(
                    test_name="api_integration",
                    component_name=widget['name'],
                    status="failed",
                    severity=TestSeverity.HIGH,
                    message="所有API端点都无法访问",
                    execution_time=execution_time,
                    details={"api_results": api_results}
                )
            else:
                return TestResult(
                    test_name="api_integration",
                    component_name=widget['name'],
                    status="passed",
                    severity=TestSeverity.HIGH,
                    message=f"成功访问 {len(successful_apis)}/{len(widget['expected_api_endpoints'])} 个API端点",
                    execution_time=execution_time,
                    details={"api_results": api_results, "successful_apis": successful_apis}
                )

        except Exception as e:
            execution_time = time.time() - start_time
            return TestResult(
                test_name="api_integration",
                component_name=widget['name'],
                status="error",
                severity=TestSeverity.HIGH,
                message=f"API集成测试错误: {str(e)}",
                execution_time=execution_time,
                details={"error": str(e)}
            )

    def test_widget_real_time_updates(self, widget_key: str) -> TestResult:
        """测试Widget实时更新功能"""
        start_time = time.time()
        widget = self.widgets[widget_key]

        if not widget['real_time_updates']:
            return TestResult(
                test_name="real_time_updates",
                component_name=widget['name'],
                status="skipped",
                severity=TestSeverity.MEDIUM,
                message="组件不支持实时更新",
                execution_time=0
            )

        try:
            # 尝试建立WebSocket连接
            async def test_websocket():
                try:
                    async with websockets.connect(self.websocket_url) as websocket:
                        # 发送订阅消息
                        subscribe_message = {
                            "type": "subscribe",
                            "channel": widget_key,
                            "widget_id": f"test_{widget_key}"
                        }
                        await websocket.send(json.dumps(subscribe_message))

                        # 等待接收数据
                        try:
                            message = await asyncio.wait_for(websocket.recv(), timeout=5.0)
                            data = json.loads(message)
                            return {"success": True, "data": data}
                        except asyncio.TimeoutError:
                            return {"success": False, "error": "WebSocket响应超时"}

                except Exception as e:
                    return {"success": False, "error": str(e)}

            # 运行异步测试
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            result = loop.run_until_complete(test_websocket())
            loop.close()

            execution_time = time.time() - start_time

            if result["success"]:
                return TestResult(
                    test_name="real_time_updates",
                    component_name=widget['name'],
                    status="passed",
                    severity=TestSeverity.MEDIUM,
                    message="WebSocket实时更新正常工作",
                    execution_time=execution_time,
                    details={"websocket_data": result["data"]}
                )
            else:
                return TestResult(
                    test_name="real_time_updates",
                    component_name=widget['name'],
                    status="failed",
                    severity=TestSeverity.MEDIUM,
                    message=f"WebSocket连接失败: {result['error']}",
                    execution_time=execution_time,
                    details={"error": result["error"]}
                )

        except Exception as e:
            execution_time = time.time() - start_time
            return TestResult(
                test_name="real_time_updates",
                component_name=widget['name'],
                status="error",
                severity=TestSeverity.MEDIUM,
                message=f"实时更新测试错误: {str(e)}",
                execution_time=execution_time,
                details={"error": str(e)}
            )

    def test_widget_error_handling(self, widget_key: str) -> TestResult:
        """测试Widget错误处理"""
        start_time = time.time()
        widget = self.widgets[widget_key]

        try:
            # 导航到Dashboard页面
            self.driver.get(f"{self.base_url}/dashboard")

            # 等待Widget加载
            wait = WebDriverWait(self.driver, self.timeout)
            widget_element = wait.until(
                EC.presence_of_element_located((By.CSS_SELECTOR, widget['selector']))
            )

            # 模拟网络错误 - 通过修改请求URL或拦截网络请求
            self.driver.execute_script("""
                // 拦截fetch请求并模拟错误
                const originalFetch = window.fetch;
                window.fetch = function(...args) {
                    if (args[0].includes('/api/')) {
                        return Promise.reject(new Error('Network error simulated'));
                    }
                    return originalFetch.apply(this, args);
                };
            """)

            # 触发数据刷新
            self.driver.refresh()

            # 等待错误处理
            time.sleep(5)

            # 检查是否有错误处理UI
            error_indicators = self.driver.find_elements(By.CSS_SELECTOR,
                ".error, .error-message, .loading-error, .network-error, .retry-button")

            # 检查是否有优雅降级
            loading_indicators = self.driver.find_elements(By.CSS_SELECTOR,
                ".loading, .skeleton, .placeholder, .fallback")

            execution_time = time.time() - start_time

            has_error_handling = len(error_indicators) > 0 or len(loading_indicators) > 0

            if has_error_handling:
                return TestResult(
                    test_name="error_handling",
                    component_name=widget['name'],
                    status="passed",
                    severity=TestSeverity.HIGH,
                    message=f"发现错误处理机制: {len(error_indicators)} 个错误指示器, {len(loading_indicators)} 个加载指示器",
                    execution_time=execution_time,
                    details={
                        "error_indicators": len(error_indicators),
                        "loading_indicators": len(loading_indicators)
                    }
                )
            else:
                return TestResult(
                    test_name="error_handling",
                    component_name=widget['name'],
                    status="failed",
                    severity=TestSeverity.HIGH,
                    message="未发现错误处理机制",
                    execution_time=execution_time
                )

        except Exception as e:
            execution_time = time.time() - start_time
            return TestResult(
                test_name="error_handling",
                component_name=widget['name'],
                status="error",
                severity=TestSeverity.HIGH,
                message=f"错误处理测试失败: {str(e)}",
                execution_time=execution_time,
                details={"error": str(e)}
            )

    def test_widget_data_validation(self, widget_key: str) -> TestResult:
        """测试Widget数据验证"""
        start_time = time.time()
        widget = self.widgets[widget_key]

        try:
            # 导航到Dashboard页面
            self.driver.get(f"{self.base_url}/dashboard")

            # 等待Widget加载
            wait = WebDriverWait(self.driver, self.timeout)
            widget_element = wait.until(
                EC.presence_of_element_located((By.CSS_SELECTOR, widget['selector']))
            )

            # 检查Widget是否显示真实数据
            widget_text = widget_element.text.lower()

            # 检查是否包含预期的数据字段
            data_field_found = []
            for field in widget['expected_data_fields']:
                # 简化匹配，查找相关关键词
                field_keywords = field.replace('_', ' ').split()
                for keyword in field_keywords:
                    if keyword in widget_text:
                        data_field_found.append(field)
                        break

            # 检查数据有效性指标
            validity_indicators = []

            # 检查是否有数值数据
            import re
            numbers = re.findall(r'\d+(?:\.\d+)?%?', widget_text)
            if numbers:
                validity_indicators.append(f"数值数据: {len(numbers)} 个")

            # 检查是否有时间戳
            time_patterns = [r'\d{1,2}:\d{2}', r'\d+[smh]\s*ago', r'\d{4}-\d{2}-\d{2}']
            for pattern in time_patterns:
                if re.search(pattern, widget_text):
                    validity_indicators.append("时间数据")
                    break

            execution_time = time.time() - start_time

            data_completeness = len(data_field_found) / len(widget['expected_data_fields']) if widget['expected_data_fields'] else 0

            if data_completeness >= 0.5 or validity_indicators:
                return TestResult(
                    test_name="data_validation",
                    component_name=widget['name'],
                    status="passed",
                    severity=TestSeverity.HIGH,
                    message=f"数据验证通过: {data_completeness:.1%} 字段完整性, {len(validity_indicators)} 个有效性指标",
                    execution_time=execution_time,
                    details={
                        "data_fields_found": data_field_found,
                        "validity_indicators": validity_indicators,
                        "completeness": data_completeness
                    }
                )
            else:
                return TestResult(
                    test_name="data_validation",
                    component_name=widget['name'],
                    status="failed",
                    severity=TestSeverity.HIGH,
                    message=f"数据验证失败: {data_completeness:.1%} 字段完整性, 缺少有效性指标",
                    execution_time=execution_time,
                    details={
                        "data_fields_found": data_field_found,
                        "validity_indicators": validity_indicators,
                        "completeness": data_completeness
                    }
                )

        except Exception as e:
            execution_time = time.time() - start_time
            return TestResult(
                test_name="data_validation",
                component_name=widget['name'],
                status="error",
                severity=TestSeverity.HIGH,
                message=f"数据验证测试错误: {str(e)}",
                execution_time=execution_time,
                details={"error": str(e)}
            )

    def run_all_widget_tests(self) -> List[TestResult]:
        """运行所有Widget测试"""
        print("🚀 开始运行Dashboard Widget功能测试套件")
        print("严格验证真实数据集成和生产级质量标准")

        all_results = []

        # 设置WebDriver
        self.setup_driver()

        try:
            for widget_key, widget_info in self.widgets.items():
                print(f"\n📝 测试组件: {widget_info['name']}")

                # 运行所有测试
                test_methods = [
                    self.test_widget_renders_without_mock_data,
                    self.test_widget_api_integration,
                    self.test_widget_real_time_updates,
                    self.test_widget_error_handling,
                    self.test_widget_data_validation
                ]

                for test_method in test_methods:
                    try:
                        result = test_method(widget_key)
                        all_results.append(result)

                        # 打印测试结果
                        status_icon = {"passed": "✅", "failed": "❌", "error": "💥", "skipped": "⏭️"}
                        print(f"  {status_icon.get(result.status, '❓')} {result.test_name}: {result.message}")

                    except Exception as e:
                        error_result = TestResult(
                            test_name=test_method.__name__,
                            component_name=widget_info['name'],
                            status="error",
                            severity=TestSeverity.HIGH,
                            message=f"测试执行异常: {str(e)}",
                            execution_time=0,
                            details={"error": str(e)}
                        )
                        all_results.append(error_result)
                        print(f"  💥 {test_method.__name__}: 测试执行异常: {str(e)}")

        finally:
            self.teardown_driver()

        self.test_results = all_results
        return all_results

    def generate_test_report(self, results: List[TestResult]) -> Dict:
        """生成测试报告"""
        report_time = time.strftime("%Y-%m-%d %H:%M:%S")

        # 统计数据
        total_tests = len(results)
        passed_tests = len([r for r in results if r.status == "passed"])
        failed_tests = len([r for r in results if r.status == "failed"])
        error_tests = len([r for r in results if r.status == "error"])
        skipped_tests = len([r for r in results if r.status == "skipped"])

        # 按组件分组
        by_component = {}
        for result in results:
            if result.component_name not in by_component:
                by_component[result.component_name] = []
            by_component[result.component_name].append(result)

        # 按严重程度分组
        by_severity = {severity: [] for severity in TestSeverity}
        for result in results:
            by_severity[result.severity].append(result)

        # 计算总执行时间
        total_execution_time = sum(result.execution_time for result in results)

        report = {
            "metadata": {
                "report_time": report_time,
                "total_execution_time": round(total_execution_time, 2),
                "test_framework": "Dashboard Widget Test Suite v1.0.0"
            },
            "summary": {
                "total_tests": total_tests,
                "passed": passed_tests,
                "failed": failed_tests,
                "errors": error_tests,
                "skipped": skipped_tests,
                "success_rate": round((passed_tests / total_tests * 100) if total_tests > 0 else 0, 1),
                "overall_status": self._get_overall_test_status(passed_tests, failed_tests, error_tests, total_tests)
            },
            "by_component": {
                component: {
                    "total_tests": len(component_results),
                    "passed": len([r for r in component_results if r.status == "passed"]),
                    "failed": len([r for r in component_results if r.status == "failed"]),
                    "errors": len([r for r in component_results if r.status == "error"]),
                    "component_status": self._get_component_test_status(component_results),
                    "tests": [
                        {
                            "test_name": r.test_name,
                            "status": r.status,
                            "severity": r.severity.value,
                            "message": r.message,
                            "execution_time": r.execution_time,
                            "details": r.details
                        }
                        for r in component_results
                    ]
                }
                for component, component_results in by_component.items()
            },
            "by_severity": {
                severity.value: {
                    "total": len(severity_results),
                    "passed": len([r for r in severity_results if r.status == "passed"]),
                    "failed": len([r for r in severity_results if r.status == "failed"]),
                    "errors": len([r for r in severity_results if r.status == "error"])
                }
                for severity, severity_results in by_severity.items()
            },
            "critical_issues": [
                {
                    "component": r.component_name,
                    "test": r.test_name,
                    "message": r.message,
                    "details": r.details
                }
                for r in results
                if r.severity == TestSeverity.CRITICAL and r.status != "passed"
            ],
            "recommendations": self._generate_test_recommendations(results)
        }

        return report

    def save_test_report(self, report: Dict, output_path: str = None):
        """保存测试报告"""
        if output_path is None:
            timestamp = time.strftime("%Y%m%d_%H%M%S")
            output_path = f"/home/await/project/docker-auto/frontend/tests/dashboard/widget_test_report_{timestamp}.json"

        try:
            Path(output_path).parent.mkdir(parents=True, exist_ok=True)
            with open(output_path, 'w', encoding='utf-8') as f:
                json.dump(report, f, ensure_ascii=False, indent=2)
            print(f"📊 测试报告已保存: {output_path}")
        except Exception as e:
            print(f"❌ 保存测试报告失败: {e}")

    def print_test_summary(self, report: Dict):
        """打印测试摘要"""
        print("\n" + "="*80)
        print("🔍 Dashboard Widget 功能测试报告")
        print("="*80)

        summary = report["summary"]
        print(f"📊 测试摘要:")
        print(f"   • 总测试数: {summary['total_tests']}")
        print(f"   • 通过: {summary['passed']} ✅")
        print(f"   • 失败: {summary['failed']} ❌")
        print(f"   • 错误: {summary['errors']} 💥")
        print(f"   • 跳过: {summary['skipped']} ⏭️")
        print(f"   • 成功率: {summary['success_rate']}%")
        print(f"   • 总体状态: {summary['overall_status']}")
        print(f"   • 执行时间: {report['metadata']['total_execution_time']}秒")

        print(f"\n🚨 临界问题:")
        critical_issues = report["critical_issues"]
        if critical_issues:
            for issue in critical_issues[:5]:  # 只显示前5个
                print(f"   • {issue['component']} - {issue['test']}: {issue['message']}")
        else:
            print("   • 无临界问题 ✅")

        print(f"\n📋 建议:")
        recommendations = report["recommendations"]
        for i, rec in enumerate(recommendations[:5], 1):
            print(f"   {i}. {rec}")

        print("\n" + "="*80)

    def _get_overall_test_status(self, passed: int, failed: int, errors: int, total: int) -> str:
        """获取总体测试状态"""
        if total == 0:
            return "❓ 无测试数据"

        success_rate = (passed / total) * 100

        if errors > 0:
            return f"💥 系统错误 - {errors} 个测试执行错误"
        elif failed > total * 0.5:
            return f"🚨 严重失败 - 成功率 {success_rate:.1f}%"
        elif failed > total * 0.2:
            return f"⚠️ 部分失败 - 成功率 {success_rate:.1f}%"
        elif success_rate >= 90:
            return f"✅ 优秀 - 成功率 {success_rate:.1f}%"
        else:
            return f"🔶 良好 - 成功率 {success_rate:.1f}%"

    def _get_component_test_status(self, results: List[TestResult]) -> str:
        """获取组件测试状态"""
        passed = len([r for r in results if r.status == "passed"])
        total = len(results)

        if total == 0:
            return "❓ 无测试"

        success_rate = (passed / total) * 100

        if success_rate >= 90:
            return "✅ 优秀"
        elif success_rate >= 70:
            return "🔶 良好"
        elif success_rate >= 50:
            return "⚠️ 一般"
        else:
            return "🚨 差"

    def _generate_test_recommendations(self, results: List[TestResult]) -> List[str]:
        """生成测试建议"""
        recommendations = []

        # 分析失败的测试
        failed_tests = [r for r in results if r.status == "failed"]
        error_tests = [r for r in results if r.status == "error"]

        # 模拟数据问题
        mock_data_failures = [r for r in failed_tests if r.test_name == "render_without_mock_data"]
        if mock_data_failures:
            recommendations.append(f"立即清理 {len(mock_data_failures)} 个组件中的模拟数据")

        # API集成问题
        api_failures = [r for r in failed_tests if r.test_name == "api_integration"]
        if api_failures:
            recommendations.append(f"修复 {len(api_failures)} 个组件的API集成问题")

        # 错误处理问题
        error_handling_failures = [r for r in failed_tests if r.test_name == "error_handling"]
        if error_handling_failures:
            recommendations.append(f"为 {len(error_handling_failures)} 个组件添加错误处理机制")

        # 数据验证问题
        data_validation_failures = [r for r in failed_tests if r.test_name == "data_validation"]
        if data_validation_failures:
            recommendations.append(f"改善 {len(data_validation_failures)} 个组件的数据验证")

        # 通用建议
        recommendations.extend([
            "实施持续集成自动化测试",
            "建立Widget组件质量检查清单",
            "定期进行端到端测试验证",
            "监控生产环境中的组件性能",
            "建立组件错误报警机制"
        ])

        return recommendations

def main():
    """主函数"""
    print("🚀 启动Dashboard Widget功能测试套件")
    print("严格验证真实数据集成和生产级质量标准")

    # 初始化测试器
    tester = DashboardWidgetTester()

    try:
        # 运行所有测试
        results = tester.run_all_widget_tests()

        # 生成报告
        report = tester.generate_test_report(results)

        # 打印摘要
        tester.print_test_summary(report)

        # 保存报告
        tester.save_test_report(report)

        print("\n🎯 Dashboard Widget功能测试完成！")
        return report

    except Exception as e:
        print(f"❌ 测试套件执行失败: {e}")
        raise

if __name__ == "__main__":
    main()
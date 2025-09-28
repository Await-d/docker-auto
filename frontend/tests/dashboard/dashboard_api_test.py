#!/usr/bin/env python3
"""
🔍 Dashboard API端点验证测试套件
专注于验证所有Dashboard API端点返回真实Docker数据，无任何模拟内容

严格遵循三大核心原则：
1. 🚫 绝不使用模拟数据！！！ - 验证API返回真实Docker数据
2. 🚫 绝不使用简化方案！！！ - 验证完整的API功能和错误处理
3. 🚫 绝不使用临时方案！！！ - 确保API达到生产级质量标准
"""

import asyncio
import json
import time
import requests
import websockets
from pathlib import Path
from typing import Dict, List, Optional, Any, Union
from dataclasses import dataclass, field
from enum import Enum
import concurrent.futures
from urllib.parse import urljoin
import ssl

class ApiTestSeverity(Enum):
    """API测试严重程度"""
    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"

@dataclass
class ApiEndpoint:
    """API端点定义"""
    path: str
    method: str
    description: str
    expected_status: int = 200
    requires_auth: bool = True
    expected_fields: List[str] = field(default_factory=list)
    expected_data_types: Dict[str, type] = field(default_factory=dict)
    mock_data_indicators: List[str] = field(default_factory=list)

@dataclass
class ApiTestResult:
    """API测试结果"""
    endpoint: str
    method: str
    status_code: int
    response_time: float
    response_size: int
    test_status: str  # passed, failed, error
    severity: ApiTestSeverity
    message: str
    details: Dict = field(default_factory=dict)
    response_data: Any = None

class DashboardApiTester:
    """Dashboard API测试器"""

    def __init__(self, api_base_url: str = "http://localhost:8080", websocket_url: str = "ws://localhost:8080/ws"):
        self.api_base_url = api_base_url.rstrip('/')
        self.websocket_url = websocket_url
        self.session = requests.Session()
        self.auth_token = None
        self.timeout = 30

        # 设置请求头
        self.session.headers.update({
            'Content-Type': 'application/json',
            'Accept': 'application/json',
            'User-Agent': 'Dashboard-API-Tester/1.0.0'
        })

        # Dashboard相关的API端点定义
        self.endpoints = self._define_dashboard_endpoints()

    def _define_dashboard_endpoints(self) -> Dict[str, List[ApiEndpoint]]:
        """定义Dashboard相关的API端点"""
        return {
            'monitoring': [
                ApiEndpoint(
                    path='/api/monitoring/metrics',
                    method='GET',
                    description='获取当前系统指标',
                    expected_fields=['cpu', 'memory', 'network', 'disk'],
                    expected_data_types={'cpu': (int, float), 'memory': dict},
                    mock_data_indicators=['mock', 'fake', 'test', 'sample', '45.2', '3221225472']
                ),
                ApiEndpoint(
                    path='/api/monitoring/events',
                    method='GET',
                    description='获取监控事件',
                    expected_fields=['events', 'timestamp'],
                    mock_data_indicators=['Container web-server started', 'High network traffic detected']
                ),
                ApiEndpoint(
                    path='/api/monitoring/resources',
                    method='GET',
                    description='获取资源使用历史',
                    expected_fields=['cpu_history', 'memory_history'],
                    mock_data_indicators=['mock', 'generated', 'sample']
                ),
                ApiEndpoint(
                    path='/api/monitoring/alerts',
                    method='GET',
                    description='获取活跃警报',
                    expected_fields=['alerts'],
                    mock_data_indicators=['High CPU usage on container web-server', 'Container database unhealthy']
                )
            ],
            'containers': [
                ApiEndpoint(
                    path='/api/containers',
                    method='GET',
                    description='获取容器列表',
                    expected_fields=['containers'],
                    expected_data_types={'containers': list}
                ),
                ApiEndpoint(
                    path='/api/containers/stats',
                    method='GET',
                    description='获取容器统计信息',
                    expected_fields=['stats'],
                    mock_data_indicators=['web-server-1', 'database', 'cache']
                ),
                ApiEndpoint(
                    path='/api/containers/health',
                    method='GET',
                    description='获取容器健康状态',
                    expected_fields=['health_status'],
                    mock_data_indicators=['mock', 'fake']
                ),
                ApiEndpoint(
                    path='/api/containers/updates',
                    method='GET',
                    description='获取容器更新信息',
                    expected_fields=['updates'],
                    mock_data_indicators=['nginx:latest', 'postgres:14']
                )
            ],
            'updates': [
                ApiEndpoint(
                    path='/api/updates',
                    method='GET',
                    description='获取可用更新',
                    expected_fields=['updates', 'lastChecked'],
                    mock_data_indicators=['mock', 'fake', 'test']
                ),
                ApiEndpoint(
                    path='/api/updates/running',
                    method='GET',
                    description='获取正在进行的更新',
                    expected_fields=['running_updates'],
                    mock_data_indicators=['mock', 'test']
                ),
                ApiEndpoint(
                    path='/api/updates/history',
                    method='GET',
                    description='获取更新历史',
                    expected_fields=['items', 'total'],
                    mock_data_indicators=['sample', 'test']
                )
            ],
            'security': [
                ApiEndpoint(
                    path='/api/security/status',
                    method='GET',
                    description='获取安全状态',
                    expected_fields=['security_score', 'vulnerabilities'],
                    mock_data_indicators=['85%', '3个中等', '2小时前']
                ),
                ApiEndpoint(
                    path='/api/security/vulnerabilities',
                    method='GET',
                    description='获取漏洞信息',
                    expected_fields=['vulnerabilities'],
                    mock_data_indicators=['mock', 'sample']
                )
            ],
            'system': [
                ApiEndpoint(
                    path='/api/system/overview',
                    method='GET',
                    description='获取系统概览',
                    expected_fields=['total_containers', 'running_containers'],
                    expected_data_types={'total_containers': int, 'running_containers': int},
                    mock_data_indicators=['mock', '12', '156']
                ),
                ApiEndpoint(
                    path='/api/system/health',
                    method='GET',
                    description='获取系统健康状态',
                    expected_fields=['status'],
                    mock_data_indicators=['mock', 'fake']
                )
            ],
            'notifications': [
                ApiEndpoint(
                    path='/api/notifications',
                    method='GET',
                    description='获取通知',
                    expected_fields=['notifications', 'unread_count'],
                    expected_data_types={'unread_count': int}
                ),
                ApiEndpoint(
                    path='/api/alerts',
                    method='GET',
                    description='获取警报',
                    expected_fields=['alerts'],
                    mock_data_indicators=['mock', 'test']
                )
            ],
            'activities': [
                ApiEndpoint(
                    path='/api/activities',
                    method='GET',
                    description='获取活动记录',
                    expected_fields=['activities'],
                    mock_data_indicators=['Container web-server started', 'System backup completed']
                ),
                ApiEndpoint(
                    path='/api/logs/recent',
                    method='GET',
                    description='获取最近日志',
                    expected_fields=['logs'],
                    mock_data_indicators=['mock', 'sample']
                )
            ],
            'actions': [
                ApiEndpoint(
                    path='/api/actions',
                    method='GET',
                    description='获取可用操作',
                    expected_fields=['available_actions'],
                    mock_data_indicators=['mock', 'test']
                ),
                ApiEndpoint(
                    path='/api/containers/actions',
                    method='GET',
                    description='获取容器操作',
                    expected_fields=['container_list', 'actions'],
                    mock_data_indicators=['mock', 'sample']
                )
            ]
        }

    def authenticate(self, username: str = "admin", password: str = "password") -> bool:
        """API认证"""
        try:
            auth_url = f"{self.api_base_url}/api/auth/login"
            response = self.session.post(auth_url, json={
                "username": username,
                "password": password
            }, timeout=self.timeout)

            if response.status_code == 200:
                data = response.json()
                self.auth_token = data.get('token') or data.get('access_token')
                if self.auth_token:
                    self.session.headers['Authorization'] = f'Bearer {self.auth_token}'
                    print("✅ API认证成功")
                    return True

            print(f"❌ API认证失败: {response.status_code}")
            return False

        except Exception as e:
            print(f"❌ API认证错误: {e}")
            return False

    def test_api_endpoint(self, endpoint: ApiEndpoint, category: str) -> ApiTestResult:
        """测试单个API端点"""
        start_time = time.time()
        full_url = f"{self.api_base_url}{endpoint.path}"

        try:
            # 发送请求
            if endpoint.method == 'GET':
                response = self.session.get(full_url, timeout=self.timeout)
            elif endpoint.method == 'POST':
                response = self.session.post(full_url, timeout=self.timeout)
            elif endpoint.method == 'PUT':
                response = self.session.put(full_url, timeout=self.timeout)
            elif endpoint.method == 'DELETE':
                response = self.session.delete(full_url, timeout=self.timeout)
            else:
                raise ValueError(f"不支持的HTTP方法: {endpoint.method}")

            response_time = time.time() - start_time
            response_size = len(response.content)

            # 解析响应数据
            response_data = None
            try:
                if response.content:
                    response_data = response.json()
            except json.JSONDecodeError:
                response_data = {"raw_content": response.text}

            # 验证响应
            return self._validate_api_response(
                endpoint, category, response, response_data, response_time, response_size
            )

        except requests.exceptions.Timeout:
            response_time = time.time() - start_time
            return ApiTestResult(
                endpoint=endpoint.path,
                method=endpoint.method,
                status_code=0,
                response_time=response_time,
                response_size=0,
                test_status="failed",
                severity=ApiTestSeverity.HIGH,
                message="请求超时",
                details={"error": "timeout", "timeout": self.timeout}
            )

        except requests.exceptions.ConnectionError:
            response_time = time.time() - start_time
            return ApiTestResult(
                endpoint=endpoint.path,
                method=endpoint.method,
                status_code=0,
                response_time=response_time,
                response_size=0,
                test_status="failed",
                severity=ApiTestSeverity.CRITICAL,
                message="连接失败 - API服务器不可达",
                details={"error": "connection_error", "url": full_url}
            )

        except Exception as e:
            response_time = time.time() - start_time
            return ApiTestResult(
                endpoint=endpoint.path,
                method=endpoint.method,
                status_code=0,
                response_time=response_time,
                response_size=0,
                test_status="error",
                severity=ApiTestSeverity.HIGH,
                message=f"测试执行错误: {str(e)}",
                details={"error": str(e), "error_type": type(e).__name__}
            )

    def _validate_api_response(self, endpoint: ApiEndpoint, category: str,
                             response: requests.Response, response_data: Any,
                             response_time: float, response_size: int) -> ApiTestResult:
        """验证API响应"""
        details = {
            "category": category,
            "url": response.url,
            "headers": dict(response.headers),
            "response_data_sample": self._get_data_sample(response_data) if response_data else None
        }

        # 检查状态码
        if response.status_code != endpoint.expected_status:
            return ApiTestResult(
                endpoint=endpoint.path,
                method=endpoint.method,
                status_code=response.status_code,
                response_time=response_time,
                response_size=response_size,
                test_status="failed",
                severity=ApiTestSeverity.HIGH,
                message=f"状态码错误: 期望 {endpoint.expected_status}, 实际 {response.status_code}",
                details=details,
                response_data=response_data
            )

        # 检查响应是否为空
        if not response_data and endpoint.expected_fields:
            return ApiTestResult(
                endpoint=endpoint.path,
                method=endpoint.method,
                status_code=response.status_code,
                response_time=response_time,
                response_size=response_size,
                test_status="failed",
                severity=ApiTestSeverity.HIGH,
                message="响应数据为空",
                details=details
            )

        # 检查模拟数据指标
        mock_data_found = []
        if response_data and endpoint.mock_data_indicators:
            response_str = json.dumps(response_data, default=str).lower()
            for indicator in endpoint.mock_data_indicators:
                if indicator.lower() in response_str:
                    mock_data_found.append(indicator)

        if mock_data_found:
            return ApiTestResult(
                endpoint=endpoint.path,
                method=endpoint.method,
                status_code=response.status_code,
                response_time=response_time,
                response_size=response_size,
                test_status="failed",
                severity=ApiTestSeverity.CRITICAL,
                message=f"发现模拟数据指标: {', '.join(mock_data_found)}",
                details={**details, "mock_indicators_found": mock_data_found},
                response_data=response_data
            )

        # 检查必需字段
        missing_fields = []
        if response_data and isinstance(response_data, dict) and endpoint.expected_fields:
            for field in endpoint.expected_fields:
                if field not in response_data:
                    missing_fields.append(field)

        if missing_fields:
            return ApiTestResult(
                endpoint=endpoint.path,
                method=endpoint.method,
                status_code=response.status_code,
                response_time=response_time,
                response_size=response_size,
                test_status="failed",
                severity=ApiTestSeverity.MEDIUM,
                message=f"缺少必需字段: {', '.join(missing_fields)}",
                details={**details, "missing_fields": missing_fields},
                response_data=response_data
            )

        # 检查数据类型
        type_errors = []
        if response_data and isinstance(response_data, dict) and endpoint.expected_data_types:
            for field, expected_type in endpoint.expected_data_types.items():
                if field in response_data:
                    value = response_data[field]
                    if isinstance(expected_type, tuple):
                        if not isinstance(value, expected_type):
                            type_errors.append(f"{field}: 期望 {expected_type}, 实际 {type(value)}")
                    else:
                        if not isinstance(value, expected_type):
                            type_errors.append(f"{field}: 期望 {expected_type.__name__}, 实际 {type(value).__name__}")

        if type_errors:
            return ApiTestResult(
                endpoint=endpoint.path,
                method=endpoint.method,
                status_code=response.status_code,
                response_time=response_time,
                response_size=response_size,
                test_status="failed",
                severity=ApiTestSeverity.MEDIUM,
                message=f"数据类型错误: {'; '.join(type_errors)}",
                details={**details, "type_errors": type_errors},
                response_data=response_data
            )

        # 检查响应时间
        performance_warning = ""
        if response_time > 5.0:
            performance_warning = f" (响应时间过长: {response_time:.2f}s)"

        # 通过所有验证
        return ApiTestResult(
            endpoint=endpoint.path,
            method=endpoint.method,
            status_code=response.status_code,
            response_time=response_time,
            response_size=response_size,
            test_status="passed",
            severity=ApiTestSeverity.LOW if response_time < 2.0 else ApiTestSeverity.MEDIUM,
            message=f"API测试通过{performance_warning}",
            details=details,
            response_data=response_data
        )

    async def test_websocket_endpoint(self) -> ApiTestResult:
        """测试WebSocket端点"""
        start_time = time.time()

        try:
            # 配置WebSocket连接
            ssl_context = None
            if self.websocket_url.startswith('wss://'):
                ssl_context = ssl.create_default_context()
                ssl_context.check_hostname = False
                ssl_context.verify_mode = ssl.CERT_NONE

            async with websockets.connect(
                self.websocket_url,
                ssl=ssl_context,
                timeout=self.timeout
            ) as websocket:

                # 发送测试消息
                test_message = {
                    "type": "subscribe",
                    "channel": "dashboard",
                    "widget_id": "test_widget"
                }
                await websocket.send(json.dumps(test_message))

                # 等待响应
                try:
                    response = await asyncio.wait_for(websocket.recv(), timeout=5.0)
                    response_data = json.loads(response)
                    response_time = time.time() - start_time

                    # 检查是否包含模拟数据
                    response_str = json.dumps(response_data, default=str).lower()
                    mock_indicators = ['mock', 'fake', 'test', 'sample', 'hardcoded']
                    found_indicators = [ind for ind in mock_indicators if ind in response_str]

                    if found_indicators:
                        return ApiTestResult(
                            endpoint="WebSocket",
                            method="WS",
                            status_code=200,
                            response_time=response_time,
                            response_size=len(response),
                            test_status="failed",
                            severity=ApiTestSeverity.CRITICAL,
                            message=f"WebSocket返回模拟数据: {', '.join(found_indicators)}",
                            details={"mock_indicators_found": found_indicators},
                            response_data=response_data
                        )

                    return ApiTestResult(
                        endpoint="WebSocket",
                        method="WS",
                        status_code=200,
                        response_time=response_time,
                        response_size=len(response),
                        test_status="passed",
                        severity=ApiTestSeverity.LOW,
                        message="WebSocket连接和数据传输正常",
                        details={"connected": True, "data_received": True},
                        response_data=response_data
                    )

                except asyncio.TimeoutError:
                    response_time = time.time() - start_time
                    return ApiTestResult(
                        endpoint="WebSocket",
                        method="WS",
                        status_code=200,
                        response_time=response_time,
                        response_size=0,
                        test_status="failed",
                        severity=ApiTestSeverity.MEDIUM,
                        message="WebSocket连接成功但未收到数据",
                        details={"connected": True, "data_received": False}
                    )

        except websockets.exceptions.ConnectionClosed:
            response_time = time.time() - start_time
            return ApiTestResult(
                endpoint="WebSocket",
                method="WS",
                status_code=0,
                response_time=response_time,
                response_size=0,
                test_status="failed",
                severity=ApiTestSeverity.HIGH,
                message="WebSocket连接被关闭",
                details={"error": "connection_closed"}
            )

        except Exception as e:
            response_time = time.time() - start_time
            return ApiTestResult(
                endpoint="WebSocket",
                method="WS",
                status_code=0,
                response_time=response_time,
                response_size=0,
                test_status="error",
                severity=ApiTestSeverity.HIGH,
                message=f"WebSocket测试错误: {str(e)}",
                details={"error": str(e), "error_type": type(e).__name__}
            )

    def run_parallel_api_tests(self) -> List[ApiTestResult]:
        """并行运行所有API测试"""
        print("🚀 开始并行API端点测试")
        print(f"🔗 API基础URL: {self.api_base_url}")
        print(f"🔌 WebSocket URL: {self.websocket_url}")

        all_results = []

        # 尝试认证
        auth_success = self.authenticate()
        if not auth_success:
            print("⚠️ 认证失败，继续进行无认证测试")

        # 收集所有API端点
        all_endpoints = []
        for category, endpoints in self.endpoints.items():
            for endpoint in endpoints:
                all_endpoints.append((category, endpoint))

        # 使用线程池并行测试
        with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
            future_to_endpoint = {
                executor.submit(self.test_api_endpoint, endpoint, category): (category, endpoint)
                for category, endpoint in all_endpoints
            }

            for future in concurrent.futures.as_completed(future_to_endpoint):
                category, endpoint = future_to_endpoint[future]
                try:
                    result = future.result()
                    all_results.append(result)

                    # 打印测试结果
                    status_icon = {"passed": "✅", "failed": "❌", "error": "💥"}
                    print(f"  {status_icon.get(result.test_status, '❓')} [{category}] {result.method} {result.endpoint}: {result.message}")

                except Exception as e:
                    error_result = ApiTestResult(
                        endpoint=endpoint.path,
                        method=endpoint.method,
                        status_code=0,
                        response_time=0,
                        response_size=0,
                        test_status="error",
                        severity=ApiTestSeverity.HIGH,
                        message=f"测试执行异常: {str(e)}",
                        details={"error": str(e), "category": category}
                    )
                    all_results.append(error_result)
                    print(f"  💥 [{category}] {endpoint.method} {endpoint.path}: 测试执行异常: {str(e)}")

        # 测试WebSocket连接
        print(f"\n🔌 测试WebSocket连接...")
        try:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            ws_result = loop.run_until_complete(self.test_websocket_endpoint())
            all_results.append(ws_result)

            status_icon = {"passed": "✅", "failed": "❌", "error": "💥"}
            print(f"  {status_icon.get(ws_result.test_status, '❓')} WebSocket: {ws_result.message}")

            loop.close()
        except Exception as e:
            print(f"  💥 WebSocket测试异常: {str(e)}")

        return all_results

    def analyze_api_performance(self, results: List[ApiTestResult]) -> Dict:
        """分析API性能"""
        if not results:
            return {}

        response_times = [r.response_time for r in results if r.response_time > 0]
        response_sizes = [r.response_size for r in results if r.response_size > 0]

        if not response_times:
            return {"error": "没有有效的响应时间数据"}

        return {
            "avg_response_time": sum(response_times) / len(response_times),
            "max_response_time": max(response_times),
            "min_response_time": min(response_times),
            "avg_response_size": sum(response_sizes) / len(response_sizes) if response_sizes else 0,
            "slow_endpoints": [
                {"endpoint": r.endpoint, "response_time": r.response_time}
                for r in results if r.response_time > 3.0
            ],
            "large_responses": [
                {"endpoint": r.endpoint, "size": r.response_size}
                for r in results if r.response_size > 100000  # > 100KB
            ]
        }

    def generate_api_test_report(self, results: List[ApiTestResult]) -> Dict:
        """生成API测试报告"""
        report_time = time.strftime("%Y-%m-%d %H:%M:%S")

        # 统计数据
        total_tests = len(results)
        passed_tests = len([r for r in results if r.test_status == "passed"])
        failed_tests = len([r for r in results if r.test_status == "failed"])
        error_tests = len([r for r in results if r.test_status == "error"])

        # 按类别分组
        by_category = {}
        for result in results:
            category = result.details.get('category', 'unknown')
            if category not in by_category:
                by_category[category] = []
            by_category[category].append(result)

        # 按严重程度分组
        by_severity = {severity: [] for severity in ApiTestSeverity}
        for result in results:
            by_severity[result.severity].append(result)

        # 分析性能
        performance_analysis = self.analyze_api_performance(results)

        # 模拟数据检测
        mock_data_issues = [
            r for r in results
            if r.test_status == "failed" and "模拟数据" in r.message
        ]

        report = {
            "metadata": {
                "report_time": report_time,
                "api_base_url": self.api_base_url,
                "websocket_url": self.websocket_url,
                "test_framework": "Dashboard API Test Suite v1.0.0",
                "authentication_used": self.auth_token is not None
            },
            "summary": {
                "total_tests": total_tests,
                "passed": passed_tests,
                "failed": failed_tests,
                "errors": error_tests,
                "success_rate": round((passed_tests / total_tests * 100) if total_tests > 0 else 0, 1),
                "overall_status": self._get_overall_api_status(passed_tests, failed_tests, error_tests, total_tests)
            },
            "by_category": {
                category: {
                    "total_tests": len(category_results),
                    "passed": len([r for r in category_results if r.test_status == "passed"]),
                    "failed": len([r for r in category_results if r.test_status == "failed"]),
                    "errors": len([r for r in category_results if r.test_status == "error"]),
                    "avg_response_time": sum(r.response_time for r in category_results) / len(category_results) if category_results else 0,
                    "tests": [
                        {
                            "endpoint": r.endpoint,
                            "method": r.method,
                            "status": r.test_status,
                            "status_code": r.status_code,
                            "response_time": r.response_time,
                            "message": r.message,
                            "severity": r.severity.value,
                            "details": r.details
                        }
                        for r in category_results
                    ]
                }
                for category, category_results in by_category.items()
            },
            "mock_data_analysis": {
                "total_mock_data_issues": len(mock_data_issues),
                "affected_endpoints": [
                    {
                        "endpoint": r.endpoint,
                        "method": r.method,
                        "mock_indicators": r.details.get("mock_indicators_found", []),
                        "message": r.message
                    }
                    for r in mock_data_issues
                ],
                "mock_data_status": "🚨 发现模拟数据" if mock_data_issues else "✅ 无模拟数据检测到"
            },
            "performance_analysis": performance_analysis,
            "critical_issues": [
                {
                    "endpoint": r.endpoint,
                    "method": r.method,
                    "status": r.test_status,
                    "message": r.message,
                    "severity": r.severity.value,
                    "details": r.details
                }
                for r in results
                if r.severity == ApiTestSeverity.CRITICAL and r.test_status != "passed"
            ],
            "recommendations": self._generate_api_recommendations(results, mock_data_issues)
        }

        return report

    def save_api_test_report(self, report: Dict, output_path: str = None):
        """保存API测试报告"""
        if output_path is None:
            timestamp = time.strftime("%Y%m%d_%H%M%S")
            output_path = f"/home/await/project/docker-auto/frontend/tests/dashboard/api_test_report_{timestamp}.json"

        try:
            Path(output_path).parent.mkdir(parents=True, exist_ok=True)
            with open(output_path, 'w', encoding='utf-8') as f:
                json.dump(report, f, ensure_ascii=False, indent=2)
            print(f"📊 API测试报告已保存: {output_path}")
        except Exception as e:
            print(f"❌ 保存API测试报告失败: {e}")

    def print_api_test_summary(self, report: Dict):
        """打印API测试摘要"""
        print("\n" + "="*80)
        print("🔍 Dashboard API端点验证测试报告")
        print("="*80)

        summary = report["summary"]
        print(f"📊 测试摘要:")
        print(f"   • 总测试数: {summary['total_tests']}")
        print(f"   • 通过: {summary['passed']} ✅")
        print(f"   • 失败: {summary['failed']} ❌")
        print(f"   • 错误: {summary['errors']} 💥")
        print(f"   • 成功率: {summary['success_rate']}%")
        print(f"   • 总体状态: {summary['overall_status']}")

        # 性能分析
        perf = report.get("performance_analysis", {})
        if perf and "avg_response_time" in perf:
            print(f"\n⚡ 性能分析:")
            print(f"   • 平均响应时间: {perf['avg_response_time']:.3f}s")
            print(f"   • 最慢响应: {perf['max_response_time']:.3f}s")
            print(f"   • 慢端点数量: {len(perf.get('slow_endpoints', []))}")

        # 模拟数据分析
        mock_analysis = report["mock_data_analysis"]
        print(f"\n🚨 模拟数据检测:")
        print(f"   • 状态: {mock_analysis['mock_data_status']}")
        print(f"   • 发现问题: {mock_analysis['total_mock_data_issues']} 个端点")

        if mock_analysis["affected_endpoints"]:
            print(f"   • 受影响端点:")
            for endpoint in mock_analysis["affected_endpoints"][:3]:  # 只显示前3个
                print(f"     - {endpoint['method']} {endpoint['endpoint']}: {', '.join(endpoint['mock_indicators'])}")

        # 临界问题
        critical_issues = report["critical_issues"]
        print(f"\n🔥 临界问题:")
        if critical_issues:
            for issue in critical_issues[:3]:  # 只显示前3个
                print(f"   • {issue['method']} {issue['endpoint']}: {issue['message']}")
        else:
            print("   • 无临界问题 ✅")

        print(f"\n📋 建议:")
        recommendations = report["recommendations"]
        for i, rec in enumerate(recommendations[:5], 1):
            print(f"   {i}. {rec}")

        print("\n" + "="*80)

    def _get_data_sample(self, data: Any, max_length: int = 500) -> Any:
        """获取数据样本"""
        if isinstance(data, dict):
            sample = {}
            for k, v in list(data.items())[:3]:  # 只取前3个键
                if isinstance(v, str) and len(v) > 100:
                    sample[k] = v[:100] + "..."
                elif isinstance(v, (list, dict)) and len(str(v)) > 200:
                    sample[k] = str(v)[:200] + "..."
                else:
                    sample[k] = v
            return sample
        elif isinstance(data, list) and data:
            return data[:3] if len(data) > 3 else data
        elif isinstance(data, str) and len(data) > max_length:
            return data[:max_length] + "..."
        else:
            return data

    def _get_overall_api_status(self, passed: int, failed: int, errors: int, total: int) -> str:
        """获取总体API状态"""
        if total == 0:
            return "❓ 无测试数据"

        success_rate = (passed / total) * 100

        if errors > total * 0.3:
            return f"💥 系统故障 - {errors} 个API错误"
        elif failed > total * 0.5:
            return f"🚨 严重问题 - 成功率 {success_rate:.1f}%"
        elif failed > total * 0.2:
            return f"⚠️ 部分问题 - 成功率 {success_rate:.1f}%"
        elif success_rate >= 95:
            return f"✅ 优秀 - 成功率 {success_rate:.1f}%"
        else:
            return f"🔶 良好 - 成功率 {success_rate:.1f}%"

    def _generate_api_recommendations(self, results: List[ApiTestResult], mock_data_issues: List[ApiTestResult]) -> List[str]:
        """生成API改进建议"""
        recommendations = []

        # 模拟数据问题
        if mock_data_issues:
            recommendations.append(f"立即清理 {len(mock_data_issues)} 个API端点中的模拟数据")

        # 连接问题
        connection_errors = [r for r in results if "连接失败" in r.message or "不可达" in r.message]
        if connection_errors:
            recommendations.append(f"修复 {len(connection_errors)} 个API端点的连接问题")

        # 性能问题
        slow_endpoints = [r for r in results if r.response_time > 3.0]
        if slow_endpoints:
            recommendations.append(f"优化 {len(slow_endpoints)} 个响应过慢的API端点")

        # 错误处理
        error_responses = [r for r in results if r.status_code >= 500]
        if error_responses:
            recommendations.append(f"修复 {len(error_responses)} 个服务器错误响应")

        # 数据完整性
        data_issues = [r for r in results if "缺少必需字段" in r.message]
        if data_issues:
            recommendations.append(f"完善 {len(data_issues)} 个API端点的数据结构")

        # 通用建议
        recommendations.extend([
            "实施API监控和性能基准测试",
            "建立API文档和版本控制",
            "添加API速率限制和安全检查",
            "实施自动化API测试CI/CD流水线",
            "建立API错误日志和监控告警"
        ])

        return recommendations

def main():
    """主函数"""
    print("🚀 启动Dashboard API端点验证测试套件")
    print("严格验证API返回真实Docker数据，无任何模拟内容")

    # 初始化测试器
    tester = DashboardApiTester()

    try:
        # 运行所有API测试
        results = tester.run_parallel_api_tests()

        # 生成报告
        report = tester.generate_api_test_report(results)

        # 打印摘要
        tester.print_api_test_summary(report)

        # 保存报告
        tester.save_api_test_report(report)

        print("\n🎯 Dashboard API端点验证测试完成！")
        return report

    except Exception as e:
        print(f"❌ API测试套件执行失败: {e}")
        raise

if __name__ == "__main__":
    main()
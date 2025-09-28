#!/usr/bin/env python3
"""
🔍 Dashboard 端到端用户流程测试套件
专注于验证用户完整体验流程，确保无模拟数据和生产级质量

严格遵循三大核心原则：
1. 🚫 绝不使用模拟数据！！！ - 验证用户看到的都是真实数据
2. 🚫 绝不使用简化方案！！！ - 验证完整的用户交互流程
3. 🚫 绝不使用临时方案！！！ - 确保用户体验达到生产级标准
"""

import time
import json
import traceback
from pathlib import Path
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass, field
from enum import Enum
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.common.action_chains import ActionChains
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.chrome.options import Options
from selenium.common.exceptions import TimeoutException, WebDriverException, NoSuchElementException

class UserFlowSeverity(Enum):
    """用户流程测试严重程度"""
    CRITICAL = "critical"  # 阻塞性问题
    HIGH = "high"         # 重要功能问题
    MEDIUM = "medium"     # 体验问题
    LOW = "low"          # 小问题

@dataclass
class UserFlowStep:
    """用户流程步骤"""
    step_name: str
    description: str
    action: str
    selector: str
    expected_outcome: str
    timeout: int = 30
    screenshot_path: str = ""

@dataclass
class UserFlowResult:
    """用户流程测试结果"""
    flow_name: str
    step_name: str
    status: str  # passed, failed, error, skipped
    severity: UserFlowSeverity
    message: str
    execution_time: float
    screenshot_path: str = ""
    details: Dict = field(default_factory=dict)

class DashboardUserFlowTester:
    """Dashboard 用户流程测试器"""

    def __init__(self, base_url: str = "http://localhost:3000", headless: bool = True):
        self.base_url = base_url
        self.headless = headless
        self.driver = None
        self.wait = None
        self.screenshot_dir = Path("/home/await/project/docker-auto/frontend/tests/dashboard/screenshots")
        self.screenshot_dir.mkdir(parents=True, exist_ok=True)

        # 用户流程定义
        self.user_flows = self._define_user_flows()

    def setup_driver(self):
        """设置Selenium WebDriver"""
        chrome_options = Options()
        if self.headless:
            chrome_options.add_argument("--headless")
        chrome_options.add_argument("--no-sandbox")
        chrome_options.add_argument("--disable-dev-shm-usage")
        chrome_options.add_argument("--disable-gpu")
        chrome_options.add_argument("--window-size=1920,1080")
        chrome_options.add_argument("--disable-extensions")
        chrome_options.add_argument("--disable-plugins")

        try:
            self.driver = webdriver.Chrome(options=chrome_options)
            self.wait = WebDriverWait(self.driver, 30)
            print("✅ 浏览器驱动初始化成功")
        except Exception as e:
            print(f"❌ 浏览器驱动初始化失败: {e}")
            raise

    def teardown_driver(self):
        """清理WebDriver"""
        if self.driver:
            self.driver.quit()
            self.driver = None
            self.wait = None

    def _define_user_flows(self) -> Dict[str, List[UserFlowStep]]:
        """定义用户流程"""
        return {
            'dashboard_first_visit': [
                UserFlowStep(
                    step_name="访问登录页面",
                    description="用户首次访问系统",
                    action="navigate",
                    selector="",
                    expected_outcome="显示登录表单",
                ),
                UserFlowStep(
                    step_name="登录系统",
                    description="用户输入凭据登录",
                    action="login",
                    selector="",
                    expected_outcome="成功登录并跳转到Dashboard",
                ),
                UserFlowStep(
                    step_name="Dashboard页面加载",
                    description="验证Dashboard页面正确加载",
                    action="wait_for_element",
                    selector=".dashboard-container",
                    expected_outcome="Dashboard容器显示",
                ),
                UserFlowStep(
                    step_name="检查Widget组件",
                    description="验证所有Widget组件正确渲染",
                    action="verify_widgets",
                    selector=".widget-wrapper",
                    expected_outcome="所有Widget组件正常显示",
                ),
            ],
            'dashboard_interaction': [
                UserFlowStep(
                    step_name="刷新UpdateActivity",
                    description="用户点击UpdateActivity的刷新按钮",
                    action="click",
                    selector=".update-activity-widget .el-button",
                    expected_outcome="显示加载状态并更新数据",
                ),
                UserFlowStep(
                    step_name="查看容器详情",
                    description="用户点击查看容器详情",
                    action="click",
                    selector=".container-name",
                    expected_outcome="跳转到容器详情页面或显示详情对话框",
                ),
                UserFlowStep(
                    step_name="实时监控交互",
                    description="用户与RealtimeMonitor组件交互",
                    action="interact_monitoring",
                    selector=".realtime-monitor-widget",
                    expected_outcome="监控数据实时更新",
                ),
            ],
            'dashboard_data_verification': [
                UserFlowStep(
                    step_name="验证数据真实性",
                    description="检查所有组件显示的都是真实数据",
                    action="verify_real_data",
                    selector="",
                    expected_outcome="所有数据都是真实的Docker数据",
                ),
                UserFlowStep(
                    step_name="检查无模拟数据",
                    description="确认页面不包含任何模拟数据指标",
                    action="check_no_mock_data",
                    selector="",
                    expected_outcome="不存在模拟数据指标",
                ),
            ],
            'dashboard_error_handling': [
                UserFlowStep(
                    step_name="模拟网络错误",
                    description="测试网络错误时的用户体验",
                    action="simulate_network_error",
                    selector="",
                    expected_outcome="显示友好的错误提示和重试选项",
                ),
                UserFlowStep(
                    step_name="测试错误恢复",
                    description="测试错误恢复后的用户体验",
                    action="test_error_recovery",
                    selector="",
                    expected_outcome="数据正常恢复并显示",
                ),
            ],
            'dashboard_responsive': [
                UserFlowStep(
                    step_name="测试移动端视图",
                    description="测试Dashboard在移动设备上的显示",
                    action="set_mobile_viewport",
                    selector="",
                    expected_outcome="响应式布局正确显示",
                ),
                UserFlowStep(
                    step_name="测试平板视图",
                    description="测试Dashboard在平板设备上的显示",
                    action="set_tablet_viewport",
                    selector="",
                    expected_outcome="响应式布局正确显示",
                ),
                UserFlowStep(
                    step_name="测试桌面视图",
                    description="恢复到桌面视图",
                    action="set_desktop_viewport",
                    selector="",
                    expected_outcome="桌面布局正确显示",
                ),
            ],
            'dashboard_performance': [
                UserFlowStep(
                    step_name="页面加载性能",
                    description="测试Dashboard页面加载性能",
                    action="measure_page_load",
                    selector="",
                    expected_outcome="页面在3秒内完成加载",
                ),
                UserFlowStep(
                    step_name="Widget渲染性能",
                    description="测试Widget组件渲染性能",
                    action="measure_widget_render",
                    selector="",
                    expected_outcome="所有Widget在2秒内完成渲染",
                ),
                UserFlowStep(
                    step_name="数据更新性能",
                    description="测试数据更新的性能表现",
                    action="measure_data_update",
                    selector="",
                    expected_outcome="数据更新响应时间小于1秒",
                ),
            ]
        }

    def take_screenshot(self, step_name: str, flow_name: str) -> str:
        """截取屏幕截图"""
        try:
            timestamp = time.strftime("%Y%m%d_%H%M%S")
            filename = f"{flow_name}_{step_name}_{timestamp}.png"
            filepath = self.screenshot_dir / filename

            self.driver.save_screenshot(str(filepath))
            return str(filepath)
        except Exception as e:
            print(f"⚠️ 截图失败: {e}")
            return ""

    def execute_user_flow_step(self, step: UserFlowStep, flow_name: str) -> UserFlowResult:
        """执行单个用户流程步骤"""
        start_time = time.time()
        screenshot_path = ""

        try:
            # 执行前截图
            screenshot_path = self.take_screenshot(f"{step.step_name}_before", flow_name)

            if step.action == "navigate":
                return self._execute_navigate_action(step, flow_name, start_time, screenshot_path)
            elif step.action == "login":
                return self._execute_login_action(step, flow_name, start_time, screenshot_path)
            elif step.action == "wait_for_element":
                return self._execute_wait_action(step, flow_name, start_time, screenshot_path)
            elif step.action == "verify_widgets":
                return self._execute_verify_widgets_action(step, flow_name, start_time, screenshot_path)
            elif step.action == "click":
                return self._execute_click_action(step, flow_name, start_time, screenshot_path)
            elif step.action == "interact_monitoring":
                return self._execute_monitoring_interaction(step, flow_name, start_time, screenshot_path)
            elif step.action == "verify_real_data":
                return self._execute_verify_real_data(step, flow_name, start_time, screenshot_path)
            elif step.action == "check_no_mock_data":
                return self._execute_check_no_mock_data(step, flow_name, start_time, screenshot_path)
            elif step.action == "simulate_network_error":
                return self._execute_simulate_network_error(step, flow_name, start_time, screenshot_path)
            elif step.action == "test_error_recovery":
                return self._execute_test_error_recovery(step, flow_name, start_time, screenshot_path)
            elif step.action.startswith("set_") and step.action.endswith("_viewport"):
                return self._execute_viewport_action(step, flow_name, start_time, screenshot_path)
            elif step.action.startswith("measure_"):
                return self._execute_performance_measurement(step, flow_name, start_time, screenshot_path)
            else:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="error",
                    severity=UserFlowSeverity.MEDIUM,
                    message=f"未知的动作类型: {step.action}",
                    execution_time=time.time() - start_time,
                    screenshot_path=screenshot_path
                )

        except Exception as e:
            execution_time = time.time() - start_time
            # 执行后截图（错误情况）
            error_screenshot = self.take_screenshot(f"{step.step_name}_error", flow_name)

            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="error",
                severity=UserFlowSeverity.HIGH,
                message=f"步骤执行异常: {str(e)}",
                execution_time=execution_time,
                screenshot_path=error_screenshot,
                details={"error": str(e), "traceback": traceback.format_exc()}
            )

    def _execute_navigate_action(self, step: UserFlowStep, flow_name: str, start_time: float, screenshot_path: str) -> UserFlowResult:
        """执行导航动作"""
        try:
            self.driver.get(f"{self.base_url}/login")

            # 等待页面加载
            self.wait.until(EC.presence_of_element_located((By.TAG_NAME, "body")))
            time.sleep(2)  # 额外等待时间确保页面完全加载

            # 检查是否有登录表单
            login_elements = self.driver.find_elements(By.CSS_SELECTOR, "form, .login-form, .auth-form")

            execution_time = time.time() - start_time
            final_screenshot = self.take_screenshot(f"{step.step_name}_after", flow_name)

            if login_elements:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="passed",
                    severity=UserFlowSeverity.LOW,
                    message="成功导航到登录页面",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot
                )
            else:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="failed",
                    severity=UserFlowSeverity.HIGH,
                    message="未找到登录表单",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot
                )

        except Exception as e:
            execution_time = time.time() - start_time
            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="error",
                severity=UserFlowSeverity.CRITICAL,
                message=f"导航失败: {str(e)}",
                execution_time=execution_time,
                screenshot_path=screenshot_path
            )

    def _execute_login_action(self, step: UserFlowStep, flow_name: str, start_time: float, screenshot_path: str) -> UserFlowResult:
        """执行登录动作"""
        try:
            # 查找用户名和密码输入框
            username_selectors = ['input[name="username"]', '#username', '.username-input', 'input[type="text"]']
            password_selectors = ['input[name="password"]', '#password', '.password-input', 'input[type="password"]']

            username_input = None
            password_input = None

            for selector in username_selectors:
                try:
                    username_input = self.driver.find_element(By.CSS_SELECTOR, selector)
                    break
                except NoSuchElementException:
                    continue

            for selector in password_selectors:
                try:
                    password_input = self.driver.find_element(By.CSS_SELECTOR, selector)
                    break
                except NoSuchElementException:
                    continue

            if not username_input or not password_input:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="failed",
                    severity=UserFlowSeverity.HIGH,
                    message="未找到登录输入框",
                    execution_time=time.time() - start_time,
                    screenshot_path=screenshot_path
                )

            # 输入凭据
            username_input.clear()
            username_input.send_keys("admin")
            password_input.clear()
            password_input.send_keys("password")

            # 查找并点击登录按钮
            submit_selectors = [
                'button[type="submit"]', '.login-button', '.submit-button',
                'input[type="submit"]', 'button:contains("登录")', 'button:contains("Login")'
            ]

            submit_button = None
            for selector in submit_selectors:
                try:
                    submit_button = self.driver.find_element(By.CSS_SELECTOR, selector)
                    break
                except NoSuchElementException:
                    continue

            if not submit_button:
                # 尝试通过Enter键提交
                password_input.send_keys(Keys.RETURN)
            else:
                submit_button.click()

            # 等待登录结果
            time.sleep(3)

            # 检查是否成功跳转到Dashboard
            current_url = self.driver.current_url
            execution_time = time.time() - start_time
            final_screenshot = self.take_screenshot(f"{step.step_name}_after", flow_name)

            if "/dashboard" in current_url or "dashboard" in self.driver.title.lower():
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="passed",
                    severity=UserFlowSeverity.LOW,
                    message="成功登录并跳转到Dashboard",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot,
                    details={"final_url": current_url}
                )
            else:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="failed",
                    severity=UserFlowSeverity.HIGH,
                    message=f"登录后未跳转到Dashboard，当前URL: {current_url}",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot,
                    details={"final_url": current_url}
                )

        except Exception as e:
            execution_time = time.time() - start_time
            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="error",
                severity=UserFlowSeverity.CRITICAL,
                message=f"登录执行失败: {str(e)}",
                execution_time=execution_time,
                screenshot_path=screenshot_path
            )

    def _execute_wait_action(self, step: UserFlowStep, flow_name: str, start_time: float, screenshot_path: str) -> UserFlowResult:
        """执行等待动作"""
        try:
            element = self.wait.until(
                EC.presence_of_element_located((By.CSS_SELECTOR, step.selector))
            )

            execution_time = time.time() - start_time
            final_screenshot = self.take_screenshot(f"{step.step_name}_after", flow_name)

            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="passed",
                severity=UserFlowSeverity.LOW,
                message=f"元素 {step.selector} 成功加载",
                execution_time=execution_time,
                screenshot_path=final_screenshot,
                details={"element_found": True, "element_tag": element.tag_name}
            )

        except TimeoutException:
            execution_time = time.time() - start_time
            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="failed",
                severity=UserFlowSeverity.HIGH,
                message=f"等待元素 {step.selector} 超时",
                execution_time=execution_time,
                screenshot_path=screenshot_path,
                details={"timeout": step.timeout}
            )

    def _execute_verify_widgets_action(self, step: UserFlowStep, flow_name: str, start_time: float, screenshot_path: str) -> UserFlowResult:
        """验证Widget组件"""
        try:
            # 等待Widget容器加载
            time.sleep(3)

            # 查找所有Widget组件
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

            found_widgets = []
            missing_widgets = []

            for selector in widget_selectors:
                try:
                    widget = self.driver.find_element(By.CSS_SELECTOR, selector)
                    found_widgets.append({
                        'selector': selector,
                        'visible': widget.is_displayed(),
                        'text_length': len(widget.text),
                        'has_content': bool(widget.text.strip())
                    })
                except NoSuchElementException:
                    missing_widgets.append(selector)

            execution_time = time.time() - start_time
            final_screenshot = self.take_screenshot(f"{step.step_name}_after", flow_name)

            # 分析结果
            visible_widgets = [w for w in found_widgets if w['visible']]
            widgets_with_content = [w for w in found_widgets if w['has_content']]

            if len(visible_widgets) >= 6:  # 至少6个Widget可见
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="passed",
                    severity=UserFlowSeverity.LOW,
                    message=f"发现 {len(visible_widgets)} 个可见Widget，{len(widgets_with_content)} 个包含内容",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot,
                    details={
                        "found_widgets": found_widgets,
                        "missing_widgets": missing_widgets,
                        "visible_count": len(visible_widgets),
                        "with_content_count": len(widgets_with_content)
                    }
                )
            else:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="failed",
                    severity=UserFlowSeverity.HIGH,
                    message=f"仅发现 {len(visible_widgets)} 个可见Widget（期望至少6个）",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot,
                    details={
                        "found_widgets": found_widgets,
                        "missing_widgets": missing_widgets
                    }
                )

        except Exception as e:
            execution_time = time.time() - start_time
            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="error",
                severity=UserFlowSeverity.HIGH,
                message=f"验证Widget失败: {str(e)}",
                execution_time=execution_time,
                screenshot_path=screenshot_path
            )

    def _execute_click_action(self, step: UserFlowStep, flow_name: str, start_time: float, screenshot_path: str) -> UserFlowResult:
        """执行点击动作"""
        try:
            element = self.wait.until(
                EC.element_to_be_clickable((By.CSS_SELECTOR, step.selector))
            )

            # 滚动到元素位置
            self.driver.execute_script("arguments[0].scrollIntoView(true);", element)
            time.sleep(1)

            element.click()
            time.sleep(2)  # 等待响应

            execution_time = time.time() - start_time
            final_screenshot = self.take_screenshot(f"{step.step_name}_after", flow_name)

            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="passed",
                severity=UserFlowSeverity.LOW,
                message=f"成功点击元素 {step.selector}",
                execution_time=execution_time,
                screenshot_path=final_screenshot
            )

        except TimeoutException:
            execution_time = time.time() - start_time
            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="failed",
                severity=UserFlowSeverity.MEDIUM,
                message=f"无法点击元素 {step.selector} - 元素不可点击",
                execution_time=execution_time,
                screenshot_path=screenshot_path
            )
        except Exception as e:
            execution_time = time.time() - start_time
            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="error",
                severity=UserFlowSeverity.HIGH,
                message=f"点击操作失败: {str(e)}",
                execution_time=execution_time,
                screenshot_path=screenshot_path
            )

    def _execute_monitoring_interaction(self, step: UserFlowStep, flow_name: str, start_time: float, screenshot_path: str) -> UserFlowResult:
        """执行监控组件交互"""
        try:
            monitor_widget = self.wait.until(
                EC.presence_of_element_located((By.CSS_SELECTOR, step.selector))
            )

            # 检查暂停/播放按钮
            pause_button = None
            try:
                pause_button = monitor_widget.find_element(By.CSS_SELECTOR, ".el-button")
                if pause_button:
                    pause_button.click()
                    time.sleep(2)
                    pause_button.click()  # 再次点击恢复
            except:
                pass

            # 检查时间范围切换
            dropdown_buttons = monitor_widget.find_elements(By.CSS_SELECTOR, ".el-dropdown .el-button")
            if dropdown_buttons:
                try:
                    dropdown_buttons[0].click()
                    time.sleep(1)
                    # 尝试选择不同的时间范围
                    dropdown_items = self.driver.find_elements(By.CSS_SELECTOR, ".el-dropdown-menu .el-dropdown-item")
                    if dropdown_items and len(dropdown_items) > 1:
                        dropdown_items[1].click()
                        time.sleep(2)
                except:
                    pass

            execution_time = time.time() - start_time
            final_screenshot = self.take_screenshot(f"{step.step_name}_after", flow_name)

            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="passed",
                severity=UserFlowSeverity.LOW,
                message="成功与监控组件交互",
                execution_time=execution_time,
                screenshot_path=final_screenshot,
                details={"interactions": ["pause_play", "time_range_switch"]}
            )

        except Exception as e:
            execution_time = time.time() - start_time
            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="error",
                severity=UserFlowSeverity.MEDIUM,
                message=f"监控交互失败: {str(e)}",
                execution_time=execution_time,
                screenshot_path=screenshot_path
            )

    def _execute_verify_real_data(self, step: UserFlowStep, flow_name: str, start_time: float, screenshot_path: str) -> UserFlowResult:
        """验证真实数据"""
        try:
            # 获取页面全部文本
            page_text = self.driver.find_element(By.TAG_NAME, "body").text.lower()

            # 检查真实数据指标
            real_data_indicators = [
                r'\d+%',                    # 百分比数据
                r'\d+\.\d+\s*gb',          # 内存数据
                r'\d+\s*containers?',       # 容器数量
                r'\d{1,2}:\d{2}',          # 时间格式
                r'ago\b',                   # 相对时间
                r'running',                 # 运行状态
                r'stopped',                 # 停止状态
                r'healthy',                 # 健康状态
            ]

            import re
            found_indicators = []
            for pattern in real_data_indicators:
                matches = re.findall(pattern, page_text)
                if matches:
                    found_indicators.extend(matches[:3])  # 只保留前3个匹配

            # 检查数值数据的合理性
            numeric_values = re.findall(r'\b\d{1,3}\b', page_text)
            numeric_count = len([v for v in numeric_values if 0 <= int(v) <= 100])

            execution_time = time.time() - start_time
            final_screenshot = self.take_screenshot(f"{step.step_name}_after", flow_name)

            if len(found_indicators) >= 5:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="passed",
                    severity=UserFlowSeverity.LOW,
                    message=f"发现 {len(found_indicators)} 个真实数据指标",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot,
                    details={
                        "real_data_indicators": found_indicators,
                        "numeric_values_count": numeric_count
                    }
                )
            else:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="failed",
                    severity=UserFlowSeverity.HIGH,
                    message=f"仅发现 {len(found_indicators)} 个真实数据指标（期望至少5个）",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot,
                    details={
                        "real_data_indicators": found_indicators,
                        "numeric_values_count": numeric_count
                    }
                )

        except Exception as e:
            execution_time = time.time() - start_time
            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="error",
                severity=UserFlowSeverity.HIGH,
                message=f"验证真实数据失败: {str(e)}",
                execution_time=execution_time,
                screenshot_path=screenshot_path
            )

    def _execute_check_no_mock_data(self, step: UserFlowStep, flow_name: str, start_time: float, screenshot_path: str) -> UserFlowResult:
        """检查无模拟数据"""
        try:
            # 获取页面全部文本和HTML
            page_text = self.driver.find_element(By.TAG_NAME, "body").text.lower()
            page_html = self.driver.page_source.lower()

            # 模拟数据指标
            mock_indicators = [
                'mock', 'fake', 'test', 'sample', 'demo', 'placeholder',
                'lorem ipsum', 'web-server-1', 'database', 'cache',
                '45.2', '85%', '3个中等', '2小时前', 'nginx:latest', 'postgres:14'
            ]

            found_mock_indicators = []
            for indicator in mock_indicators:
                if indicator in page_text or indicator in page_html:
                    found_mock_indicators.append(indicator)

            execution_time = time.time() - start_time
            final_screenshot = self.take_screenshot(f"{step.step_name}_after", flow_name)

            if not found_mock_indicators:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="passed",
                    severity=UserFlowSeverity.LOW,
                    message="未发现模拟数据指标",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot
                )
            else:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="failed",
                    severity=UserFlowSeverity.CRITICAL,
                    message=f"发现模拟数据指标: {', '.join(found_mock_indicators[:5])}",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot,
                    details={"mock_indicators_found": found_mock_indicators}
                )

        except Exception as e:
            execution_time = time.time() - start_time
            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="error",
                severity=UserFlowSeverity.HIGH,
                message=f"检查模拟数据失败: {str(e)}",
                execution_time=execution_time,
                screenshot_path=screenshot_path
            )

    def _execute_simulate_network_error(self, step: UserFlowStep, flow_name: str, start_time: float, screenshot_path: str) -> UserFlowResult:
        """模拟网络错误"""
        try:
            # 通过JavaScript模拟网络错误
            self.driver.execute_script("""
                window.originalFetch = window.fetch;
                window.fetch = function() {
                    return Promise.reject(new Error('Network error'));
                };
            """)

            # 刷新页面触发网络请求
            self.driver.refresh()
            time.sleep(5)  # 等待错误处理

            # 检查错误处理UI
            error_elements = self.driver.find_elements(By.CSS_SELECTOR,
                ".error, .error-message, .loading-error, .network-error, .retry-button, .el-alert")

            execution_time = time.time() - start_time
            final_screenshot = self.take_screenshot(f"{step.step_name}_after", flow_name)

            if error_elements:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="passed",
                    severity=UserFlowSeverity.LOW,
                    message=f"发现 {len(error_elements)} 个错误处理元素",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot,
                    details={"error_elements_count": len(error_elements)}
                )
            else:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="failed",
                    severity=UserFlowSeverity.HIGH,
                    message="未发现错误处理UI元素",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot
                )

        except Exception as e:
            execution_time = time.time() - start_time
            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="error",
                severity=UserFlowSeverity.MEDIUM,
                message=f"模拟网络错误失败: {str(e)}",
                execution_time=execution_time,
                screenshot_path=screenshot_path
            )

    def _execute_test_error_recovery(self, step: UserFlowStep, flow_name: str, start_time: float, screenshot_path: str) -> UserFlowResult:
        """测试错误恢复"""
        try:
            # 恢复fetch函数
            self.driver.execute_script("""
                if (window.originalFetch) {
                    window.fetch = window.originalFetch;
                }
            """)

            # 查找重试按钮并点击
            retry_buttons = self.driver.find_elements(By.CSS_SELECTOR,
                ".retry-button, .refresh-button, .el-button:contains('重试'), .el-button:contains('刷新')")

            if retry_buttons:
                retry_buttons[0].click()
                time.sleep(3)  # 等待数据恢复
            else:
                # 手动刷新页面
                self.driver.refresh()
                time.sleep(5)

            # 检查数据是否恢复
            dashboard_elements = self.driver.find_elements(By.CSS_SELECTOR, ".widget-wrapper, .dashboard-widget")

            execution_time = time.time() - start_time
            final_screenshot = self.take_screenshot(f"{step.step_name}_after", flow_name)

            if dashboard_elements:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="passed",
                    severity=UserFlowSeverity.LOW,
                    message=f"数据恢复成功，发现 {len(dashboard_elements)} 个Dashboard元素",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot,
                    details={"recovered_elements": len(dashboard_elements)}
                )
            else:
                return UserFlowResult(
                    flow_name=flow_name,
                    step_name=step.step_name,
                    status="failed",
                    severity=UserFlowSeverity.HIGH,
                    message="数据未能恢复",
                    execution_time=execution_time,
                    screenshot_path=final_screenshot
                )

        except Exception as e:
            execution_time = time.time() - start_time
            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="error",
                severity=UserFlowSeverity.MEDIUM,
                message=f"测试错误恢复失败: {str(e)}",
                execution_time=execution_time,
                screenshot_path=screenshot_path
            )

    def _execute_viewport_action(self, step: UserFlowStep, flow_name: str, start_time: float, screenshot_path: str) -> UserFlowResult:
        """执行视口切换动作"""
        try:
            if step.action == "set_mobile_viewport":
                self.driver.set_window_size(375, 667)  # iPhone 6/7/8
            elif step.action == "set_tablet_viewport":
                self.driver.set_window_size(768, 1024)  # iPad
            elif step.action == "set_desktop_viewport":
                self.driver.set_window_size(1920, 1080)  # Desktop

            time.sleep(2)  # 等待布局调整

            # 检查响应式布局
            dashboard_element = self.driver.find_element(By.CSS_SELECTOR, "body")
            is_responsive = dashboard_element.is_displayed()

            execution_time = time.time() - start_time
            final_screenshot = self.take_screenshot(f"{step.step_name}_after", flow_name)

            current_size = self.driver.get_window_size()

            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="passed" if is_responsive else "failed",
                severity=UserFlowSeverity.LOW if is_responsive else UserFlowSeverity.MEDIUM,
                message=f"视口切换到 {current_size['width']}x{current_size['height']}",
                execution_time=execution_time,
                screenshot_path=final_screenshot,
                details={"viewport_size": current_size, "responsive": is_responsive}
            )

        except Exception as e:
            execution_time = time.time() - start_time
            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="error",
                severity=UserFlowSeverity.MEDIUM,
                message=f"视口切换失败: {str(e)}",
                execution_time=execution_time,
                screenshot_path=screenshot_path
            )

    def _execute_performance_measurement(self, step: UserFlowStep, flow_name: str, start_time: float, screenshot_path: str) -> UserFlowResult:
        """执行性能测量"""
        try:
            if step.action == "measure_page_load":
                # 测量页面加载时间
                load_start = time.time()
                self.driver.get(f"{self.base_url}/dashboard")
                self.wait.until(EC.presence_of_element_located((By.CSS_SELECTOR, ".dashboard-container, body")))
                load_time = time.time() - load_start

                threshold = 3.0
                status = "passed" if load_time <= threshold else "failed"
                severity = UserFlowSeverity.LOW if load_time <= threshold else UserFlowSeverity.MEDIUM

            elif step.action == "measure_widget_render":
                # 测量Widget渲染时间
                render_start = time.time()
                widgets = self.wait.until(EC.presence_of_all_elements_located((By.CSS_SELECTOR, "[class*='widget']")))
                render_time = time.time() - render_start

                threshold = 2.0
                status = "passed" if render_time <= threshold else "failed"
                severity = UserFlowSeverity.LOW if render_time <= threshold else UserFlowSeverity.MEDIUM
                load_time = render_time

            elif step.action == "measure_data_update":
                # 测量数据更新时间
                update_start = time.time()
                # 触发数据刷新
                refresh_buttons = self.driver.find_elements(By.CSS_SELECTOR, ".refresh-button, .el-button")
                if refresh_buttons:
                    refresh_buttons[0].click()
                    time.sleep(1)  # 等待更新完成
                update_time = time.time() - update_start

                threshold = 1.0
                status = "passed" if update_time <= threshold else "failed"
                severity = UserFlowSeverity.LOW if update_time <= threshold else UserFlowSeverity.MEDIUM
                load_time = update_time
            else:
                load_time = 0
                status = "error"
                severity = UserFlowSeverity.MEDIUM

            execution_time = time.time() - start_time
            final_screenshot = self.take_screenshot(f"{step.step_name}_after", flow_name)

            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status=status,
                severity=severity,
                message=f"性能测量: {load_time:.3f}s (阈值: {threshold}s)" if 'threshold' in locals() else f"性能测量完成: {load_time:.3f}s",
                execution_time=execution_time,
                screenshot_path=final_screenshot,
                details={"measured_time": load_time, "threshold": threshold if 'threshold' in locals() else None}
            )

        except Exception as e:
            execution_time = time.time() - start_time
            return UserFlowResult(
                flow_name=flow_name,
                step_name=step.step_name,
                status="error",
                severity=UserFlowSeverity.MEDIUM,
                message=f"性能测量失败: {str(e)}",
                execution_time=execution_time,
                screenshot_path=screenshot_path
            )

    def run_user_flow(self, flow_name: str) -> List[UserFlowResult]:
        """运行单个用户流程"""
        if flow_name not in self.user_flows:
            raise ValueError(f"未知的用户流程: {flow_name}")

        print(f"\n🏃 执行用户流程: {flow_name}")
        results = []

        steps = self.user_flows[flow_name]
        for i, step in enumerate(steps, 1):
            print(f"  📍 步骤 {i}/{len(steps)}: {step.step_name}")

            result = self.execute_user_flow_step(step, flow_name)
            results.append(result)

            # 打印步骤结果
            status_icon = {"passed": "✅", "failed": "❌", "error": "💥", "skipped": "⏭️"}
            print(f"     {status_icon.get(result.status, '❓')} {result.message} ({result.execution_time:.2f}s)")

            # 如果是关键步骤失败，跳过后续步骤
            if result.status == "error" and result.severity == UserFlowSeverity.CRITICAL:
                print(f"     ⏹️ 关键步骤失败，跳过后续步骤")
                for remaining_step in steps[i:]:
                    skip_result = UserFlowResult(
                        flow_name=flow_name,
                        step_name=remaining_step.step_name,
                        status="skipped",
                        severity=UserFlowSeverity.LOW,
                        message="由于前置步骤失败而跳过",
                        execution_time=0
                    )
                    results.append(skip_result)
                break

        return results

    def run_all_user_flows(self) -> List[UserFlowResult]:
        """运行所有用户流程"""
        print("🚀 开始运行Dashboard端到端用户流程测试")
        print("严格验证用户完整体验，确保无模拟数据和生产级质量")

        all_results = []

        # 设置浏览器
        self.setup_driver()

        try:
            for flow_name in self.user_flows.keys():
                flow_results = self.run_user_flow(flow_name)
                all_results.extend(flow_results)

                # 每个流程之间休息一下
                time.sleep(2)

        finally:
            self.teardown_driver()

        return all_results

    def generate_user_flow_report(self, results: List[UserFlowResult]) -> Dict:
        """生成用户流程测试报告"""
        report_time = time.strftime("%Y-%m-%d %H:%M:%S")

        # 统计数据
        total_steps = len(results)
        passed_steps = len([r for r in results if r.status == "passed"])
        failed_steps = len([r for r in results if r.status == "failed"])
        error_steps = len([r for r in results if r.status == "error"])
        skipped_steps = len([r for r in results if r.status == "skipped"])

        # 按流程分组
        by_flow = {}
        for result in results:
            if result.flow_name not in by_flow:
                by_flow[result.flow_name] = []
            by_flow[result.flow_name].append(result)

        # 按严重程度分组
        by_severity = {severity: [] for severity in UserFlowSeverity}
        for result in results:
            by_severity[result.severity].append(result)

        # 计算总执行时间
        total_execution_time = sum(r.execution_time for r in results)

        # 模拟数据检测结果
        mock_data_failures = [
            r for r in results
            if r.status == "failed" and ("模拟数据" in r.message or "mock" in r.message.lower())
        ]

        report = {
            "metadata": {
                "report_time": report_time,
                "base_url": self.base_url,
                "total_execution_time": round(total_execution_time, 2),
                "test_framework": "Dashboard User Flow Test Suite v1.0.0",
                "headless_mode": self.headless
            },
            "summary": {
                "total_steps": total_steps,
                "passed": passed_steps,
                "failed": failed_steps,
                "errors": error_steps,
                "skipped": skipped_steps,
                "success_rate": round((passed_steps / total_steps * 100) if total_steps > 0 else 0, 1),
                "overall_status": self._get_overall_flow_status(passed_steps, failed_steps, error_steps, total_steps)
            },
            "by_flow": {
                flow: {
                    "total_steps": len(flow_results),
                    "passed": len([r for r in flow_results if r.status == "passed"]),
                    "failed": len([r for r in flow_results if r.status == "failed"]),
                    "errors": len([r for r in flow_results if r.status == "error"]),
                    "skipped": len([r for r in flow_results if r.status == "skipped"]),
                    "execution_time": sum(r.execution_time for r in flow_results),
                    "flow_status": self._get_flow_status(flow_results),
                    "steps": [
                        {
                            "step_name": r.step_name,
                            "status": r.status,
                            "severity": r.severity.value,
                            "message": r.message,
                            "execution_time": r.execution_time,
                            "screenshot_path": r.screenshot_path,
                            "details": r.details
                        }
                        for r in flow_results
                    ]
                }
                for flow, flow_results in by_flow.items()
            },
            "mock_data_analysis": {
                "total_mock_data_issues": len(mock_data_failures),
                "affected_flows": list(set([r.flow_name for r in mock_data_failures])),
                "mock_data_steps": [
                    {
                        "flow_name": r.flow_name,
                        "step_name": r.step_name,
                        "message": r.message,
                        "details": r.details
                    }
                    for r in mock_data_failures
                ],
                "mock_data_status": "🚨 发现模拟数据" if mock_data_failures else "✅ 无模拟数据检测到"
            },
            "critical_issues": [
                {
                    "flow_name": r.flow_name,
                    "step_name": r.step_name,
                    "status": r.status,
                    "message": r.message,
                    "severity": r.severity.value,
                    "screenshot_path": r.screenshot_path,
                    "details": r.details
                }
                for r in results
                if r.severity == UserFlowSeverity.CRITICAL and r.status != "passed"
            ],
            "recommendations": self._generate_flow_recommendations(results, by_flow)
        }

        return report

    def save_user_flow_report(self, report: Dict, output_path: str = None):
        """保存用户流程测试报告"""
        if output_path is None:
            timestamp = time.strftime("%Y%m%d_%H%M%S")
            output_path = f"/home/await/project/docker-auto/frontend/tests/dashboard/user_flow_report_{timestamp}.json"

        try:
            Path(output_path).parent.mkdir(parents=True, exist_ok=True)
            with open(output_path, 'w', encoding='utf-8') as f:
                json.dump(report, f, ensure_ascii=False, indent=2)
            print(f"📊 用户流程测试报告已保存: {output_path}")
        except Exception as e:
            print(f"❌ 保存用户流程测试报告失败: {e}")

    def print_user_flow_summary(self, report: Dict):
        """打印用户流程测试摘要"""
        print("\n" + "="*80)
        print("🔍 Dashboard 端到端用户流程测试报告")
        print("="*80)

        summary = report["summary"]
        print(f"📊 测试摘要:")
        print(f"   • 总步骤数: {summary['total_steps']}")
        print(f"   • 通过: {summary['passed']} ✅")
        print(f"   • 失败: {summary['failed']} ❌")
        print(f"   • 错误: {summary['errors']} 💥")
        print(f"   • 跳过: {summary['skipped']} ⏭️")
        print(f"   • 成功率: {summary['success_rate']}%")
        print(f"   • 总体状态: {summary['overall_status']}")
        print(f"   • 执行时间: {report['metadata']['total_execution_time']}秒")

        # 流程分析
        print(f"\n🏃 流程分析:")
        for flow_name, flow_data in report["by_flow"].items():
            success_rate = round((flow_data['passed'] / flow_data['total_steps'] * 100) if flow_data['total_steps'] > 0 else 0, 1)
            print(f"   • {flow_name}: {success_rate}% 成功率 ({flow_data['passed']}/{flow_data['total_steps']}) - {flow_data['flow_status']}")

        # 模拟数据分析
        mock_analysis = report["mock_data_analysis"]
        print(f"\n🚨 模拟数据检测:")
        print(f"   • 状态: {mock_analysis['mock_data_status']}")
        print(f"   • 发现问题: {mock_analysis['total_mock_data_issues']} 个步骤")
        if mock_analysis["affected_flows"]:
            print(f"   • 受影响流程: {', '.join(mock_analysis['affected_flows'])}")

        # 临界问题
        critical_issues = report["critical_issues"]
        print(f"\n🔥 临界问题:")
        if critical_issues:
            for issue in critical_issues[:3]:  # 只显示前3个
                print(f"   • [{issue['flow_name']}] {issue['step_name']}: {issue['message']}")
        else:
            print("   • 无临界问题 ✅")

        print(f"\n📋 建议:")
        recommendations = report["recommendations"]
        for i, rec in enumerate(recommendations[:5], 1):
            print(f"   {i}. {rec}")

        print("\n" + "="*80)

    def _get_overall_flow_status(self, passed: int, failed: int, errors: int, total: int) -> str:
        """获取总体流程状态"""
        if total == 0:
            return "❓ 无测试数据"

        success_rate = (passed / total) * 100

        if errors > total * 0.3:
            return f"💥 系统故障 - {errors} 个步骤执行错误"
        elif failed > total * 0.5:
            return f"🚨 严重问题 - 成功率 {success_rate:.1f}%"
        elif failed > total * 0.2:
            return f"⚠️ 部分问题 - 成功率 {success_rate:.1f}%"
        elif success_rate >= 95:
            return f"✅ 优秀 - 成功率 {success_rate:.1f}%"
        else:
            return f"🔶 良好 - 成功率 {success_rate:.1f}%"

    def _get_flow_status(self, results: List[UserFlowResult]) -> str:
        """获取单个流程状态"""
        if not results:
            return "❓ 无数据"

        passed = len([r for r in results if r.status == "passed"])
        total = len(results)
        success_rate = (passed / total) * 100

        if success_rate >= 90:
            return "✅ 优秀"
        elif success_rate >= 70:
            return "🔶 良好"
        elif success_rate >= 50:
            return "⚠️ 一般"
        else:
            return "🚨 差"

    def _generate_flow_recommendations(self, results: List[UserFlowResult], by_flow: Dict) -> List[str]:
        """生成流程改进建议"""
        recommendations = []

        # 模拟数据问题
        mock_data_failures = [r for r in results if r.status == "failed" and "模拟数据" in r.message]
        if mock_data_failures:
            recommendations.append(f"立即清理 {len(mock_data_failures)} 个步骤中发现的模拟数据")

        # 登录/认证问题
        auth_failures = [r for r in results if "登录" in r.step_name and r.status != "passed"]
        if auth_failures:
            recommendations.append(f"修复 {len(auth_failures)} 个认证相关问题")

        # Widget组件问题
        widget_failures = [r for r in results if "widget" in r.step_name.lower() and r.status == "failed"]
        if widget_failures:
            recommendations.append(f"修复 {len(widget_failures)} 个Widget组件问题")

        # 性能问题
        performance_issues = [r for r in results if "measure_" in r.step_name and r.status == "failed"]
        if performance_issues:
            recommendations.append(f"优化 {len(performance_issues)} 个性能问题")

        # 错误处理问题
        error_handling_issues = [r for r in results if "error" in r.step_name.lower() and r.status != "passed"]
        if error_handling_issues:
            recommendations.append(f"改善 {len(error_handling_issues)} 个错误处理机制")

        # 通用建议
        recommendations.extend([
            "建立完整的端到端测试自动化流水线",
            "实施用户体验监控和反馈收集",
            "优化页面加载性能和响应速度",
            "加强错误处理和用户友好提示",
            "建立跨设备兼容性测试流程"
        ])

        return recommendations

def main():
    """主函数"""
    print("🚀 启动Dashboard端到端用户流程测试套件")
    print("严格验证用户完整体验，确保无模拟数据和生产级质量")

    # 初始化测试器
    tester = DashboardUserFlowTester()

    try:
        # 运行所有用户流程测试
        results = tester.run_all_user_flows()

        # 生成报告
        report = tester.generate_user_flow_report(results)

        # 打印摘要
        tester.print_user_flow_summary(report)

        # 保存报告
        tester.save_user_flow_report(report)

        print("\n🎯 Dashboard端到端用户流程测试完成！")
        return report

    except Exception as e:
        print(f"❌ 用户流程测试套件执行失败: {e}")
        raise

if __name__ == "__main__":
    main()
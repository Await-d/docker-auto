#!/usr/bin/env python3
"""
🔍 Dashboard Widget 模拟数据检测和验证工具
专注于检测和验证所有Dashboard组件完全消除模拟数据

严格遵循三大核心原则：
1. 🚫 绝不使用模拟数据！！！ - 检测和验证所有组件完全消除模拟数据
2. 🚫 绝不使用简化方案！！！ - 验证实现了完整的功能和错误处理
3. 🚫 绝不使用临时方案！！！ - 确保所有实现达到生产级质量标准
"""

import os
import re
import json
import time
from pathlib import Path
from typing import List, Dict, Set, Tuple, Optional
from dataclasses import dataclass, field
from enum import Enum

class MockDataSeverity(Enum):
    """模拟数据严重程度"""
    CRITICAL = "critical"  # 硬编码数据
    HIGH = "high"         # 模拟数据生成
    MEDIUM = "medium"     # 临时占位符
    LOW = "low"          # 开发注释

@dataclass
class MockDataDetection:
    """模拟数据检测结果"""
    file_path: str
    line_number: int
    content: str
    severity: MockDataSeverity
    pattern_matched: str
    description: str
    context_lines: List[str] = field(default_factory=list)

@dataclass
class ComponentAnalysis:
    """组件分析结果"""
    component_name: str
    file_path: str
    total_lines: int
    mock_detections: List[MockDataDetection] = field(default_factory=list)
    api_calls: List[str] = field(default_factory=list)
    has_real_data_integration: bool = False
    completion_score: float = 0.0

class MockDataDetector:
    """Dashboard Widget 模拟数据检测器"""

    def __init__(self, project_root: str):
        self.project_root = Path(project_root)
        self.frontend_src = self.project_root / "frontend" / "src"
        self.widget_dir = self.frontend_src / "components" / "dashboard" / "widgets"
        self.api_dir = self.frontend_src / "api"

        # 模拟数据检测模式
        self.mock_patterns = {
            # 临界严重 - 硬编码数据
            MockDataSeverity.CRITICAL: [
                r'mock[-_]?data',  # mock-data, mockData, mock_data
                r'\.mock\s*=',     # .mock =
                r'mockData\s*[:=]', # mockData: 或 mockData =
                r'generateMock.*\(',  # generateMockXxx(
                r'const\s+\w*[Mm]ock\w*\s*=',  # const mockXxx =
                r'let\s+\w*[Mm]ock\w*\s*=',    # let mockXxx =
                r'hardcoded',      # hardcoded 注释或变量名
                r'placeholder.*data',  # placeholder data
                r'test[-_]?data',  # test-data, testData
                r'fake[-_]?data',  # fake-data, fakeData
                r'dummy[-_]?data', # dummy-data, dummyData
                r'sample[-_]?data', # sample-data, sampleData
            ],

            # 高严重 - 模拟数据生成
            MockDataSeverity.HIGH: [
                r'Math\.random\(\)',           # 随机数生成
                r'Date\.now\(\)\s*[-+%]',     # 时间戳计算
                r'new\s+Date\([^)]*\d{4}',    # 硬编码日期
                r'lorem\s*ipsum',             # Lorem ipsum 文本
                r'demo.*data',                # demo data
                r'example.*data',             # example data
                r'static.*data',              # static data
                r'fixture.*data',             # fixture data
                r'stub.*data',                # stub data
            ],

            # 中等严重 - 临时占位符
            MockDataSeverity.MEDIUM: [
                r'TODO:.*data',               # TODO: data related
                r'FIXME:.*data',              # FIXME: data related
                r'placeholder',               # placeholder
                r'coming\s*soon',             # coming soon
                r'under\s*development',       # under development
                r'not\s*implemented',         # not implemented
                r'temporary',                 # temporary
                r'temp\w*data',              # tempData, temporary_data
            ],

            # 低严重 - 开发注释
            MockDataSeverity.LOW: [
                r'\/\/.*mock',                # // mock comment
                r'\/\*.*mock.*\*\/',          # /* mock comment */
                r'console\.log.*mock',        # console.log mock
                r'DEBUG.*mock',               # DEBUG mock
            ]
        }

        # API 调用检测模式
        self.api_patterns = [
            r'\w+API\.\w+\(',              # xxxAPI.method(
            r'api\.\w+\(',                 # api.method(
            r'await\s+\w*api\w*\.',        # await apiCall.
            r'fetch\s*\(',                 # fetch(
            r'axios\.\w+\(',               # axios.method(
            r'http\w*\.\w+\(',             # httpClient.method(
            r'request\s*\(',               # request(
            r'get\w*Data\(',               # getData, getUserData etc.
            r'post\w*Data\(',              # postData etc.
            r'update\w*Data\(',            # updateData etc.
            r'delete\w*Data\(',            # deleteData etc.
        ]

        # Widget组件名称映射
        self.widget_names = {
            'UpdateActivity.vue': 'UpdateActivity Widget',
            'RealtimeMonitor.vue': 'RealtimeMonitor Widget',
            'SecurityDashboard.vue': 'SecurityDashboard Widget',
            'HealthMonitor.vue': 'HealthMonitor Widget',
            'RecentActivities.vue': 'RecentActivities Widget',
            'SystemOverview.vue': 'SystemOverview Widget',
            'QuickActions.vue': 'QuickActions Widget',
            'NotificationCenter.vue': 'NotificationCenter Widget',
            'ResourceCharts.vue': 'ResourceCharts Widget',
            'ContainerStats.vue': 'ContainerStats Widget'
        }

    def detect_mock_data_in_file(self, file_path: Path) -> List[MockDataDetection]:
        """检测文件中的模拟数据"""
        detections = []

        if not file_path.exists():
            return detections

        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                lines = f.readlines()
        except Exception as e:
            print(f"❌ 无法读取文件 {file_path}: {e}")
            return detections

        for line_num, line in enumerate(lines, 1):
            line_content = line.strip()

            # 检查每个严重程度的模式
            for severity, patterns in self.mock_patterns.items():
                for pattern in patterns:
                    matches = re.finditer(pattern, line_content, re.IGNORECASE)
                    for match in matches:
                        # 获取上下文行
                        context_start = max(0, line_num - 3)
                        context_end = min(len(lines), line_num + 2)
                        context_lines = [
                            f"{i+1:3d}: {lines[i].rstrip()}"
                            for i in range(context_start, context_end)
                        ]

                        detection = MockDataDetection(
                            file_path=str(file_path),
                            line_number=line_num,
                            content=line_content,
                            severity=severity,
                            pattern_matched=pattern,
                            description=self._get_description(pattern, severity),
                            context_lines=context_lines
                        )
                        detections.append(detection)

        return detections

    def detect_api_calls_in_file(self, file_path: Path) -> List[str]:
        """检测文件中的API调用"""
        api_calls = []

        if not file_path.exists():
            return api_calls

        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
        except Exception as e:
            print(f"❌ 无法读取文件 {file_path}: {e}")
            return api_calls

        for pattern in self.api_patterns:
            matches = re.finditer(pattern, content, re.IGNORECASE | re.MULTILINE)
            for match in matches:
                api_calls.append(match.group())

        return list(set(api_calls))  # 去重

    def analyze_component(self, file_path: Path) -> ComponentAnalysis:
        """分析单个组件"""
        component_name = self.widget_names.get(file_path.name, file_path.name)

        # 读取文件获取总行数
        total_lines = 0
        if file_path.exists():
            try:
                with open(file_path, 'r', encoding='utf-8') as f:
                    total_lines = len(f.readlines())
            except Exception:
                total_lines = 0

        # 检测模拟数据
        mock_detections = self.detect_mock_data_in_file(file_path)

        # 检测API调用
        api_calls = self.detect_api_calls_in_file(file_path)

        # 判断是否有真实数据集成
        has_real_data_integration = len(api_calls) > 0

        # 计算完成度分数
        completion_score = self._calculate_completion_score(
            mock_detections, api_calls, total_lines
        )

        return ComponentAnalysis(
            component_name=component_name,
            file_path=str(file_path),
            total_lines=total_lines,
            mock_detections=mock_detections,
            api_calls=api_calls,
            has_real_data_integration=has_real_data_integration,
            completion_score=completion_score
        )

    def analyze_all_widgets(self) -> List[ComponentAnalysis]:
        """分析所有Widget组件"""
        analyses = []

        print("🔍 开始分析所有Dashboard Widget组件...")
        print(f"📁 Widget目录: {self.widget_dir}")

        if not self.widget_dir.exists():
            print(f"❌ Widget目录不存在: {self.widget_dir}")
            return analyses

        for widget_file in self.widget_dir.glob("*.vue"):
            print(f"📝 分析组件: {widget_file.name}")
            analysis = self.analyze_component(widget_file)
            analyses.append(analysis)

        return analyses

    def generate_detection_report(self, analyses: List[ComponentAnalysis]) -> Dict:
        """生成模拟数据检测报告"""
        report_time = time.strftime("%Y-%m-%d %H:%M:%S")

        # 统计数据
        total_components = len(analyses)
        total_detections = sum(len(analysis.mock_detections) for analysis in analyses)
        critical_detections = sum(
            len([d for d in analysis.mock_detections if d.severity == MockDataSeverity.CRITICAL])
            for analysis in analyses
        )
        high_detections = sum(
            len([d for d in analysis.mock_detections if d.severity == MockDataSeverity.HIGH])
            for analysis in analyses
        )

        components_with_real_data = sum(
            1 for analysis in analyses if analysis.has_real_data_integration
        )

        avg_completion_score = (
            sum(analysis.completion_score for analysis in analyses) / total_components
            if total_components > 0 else 0
        )

        # 风险组件
        critical_components = [
            analysis for analysis in analyses
            if any(d.severity == MockDataSeverity.CRITICAL for d in analysis.mock_detections)
        ]

        high_risk_components = [
            analysis for analysis in analyses
            if len([d for d in analysis.mock_detections
                   if d.severity in [MockDataSeverity.CRITICAL, MockDataSeverity.HIGH]]) > 5
        ]

        report = {
            "metadata": {
                "report_time": report_time,
                "tool_version": "1.0.0",
                "project_root": str(self.project_root),
                "analysis_scope": "Dashboard Widgets"
            },
            "summary": {
                "total_components": total_components,
                "total_detections": total_detections,
                "critical_detections": critical_detections,
                "high_detections": high_detections,
                "components_with_real_data": components_with_real_data,
                "components_without_real_data": total_components - components_with_real_data,
                "avg_completion_score": round(avg_completion_score, 2),
                "overall_status": self._get_overall_status(critical_detections, avg_completion_score)
            },
            "component_analyses": [
                {
                    "component_name": analysis.component_name,
                    "file_path": analysis.file_path,
                    "total_lines": analysis.total_lines,
                    "mock_detections_count": len(analysis.mock_detections),
                    "api_calls_count": len(analysis.api_calls),
                    "has_real_data_integration": analysis.has_real_data_integration,
                    "completion_score": analysis.completion_score,
                    "status": self._get_component_status(analysis),
                    "mock_detections": [
                        {
                            "line_number": d.line_number,
                            "content": d.content,
                            "severity": d.severity.value,
                            "pattern_matched": d.pattern_matched,
                            "description": d.description,
                            "context_lines": d.context_lines
                        }
                        for d in analysis.mock_detections
                    ],
                    "api_calls": analysis.api_calls
                }
                for analysis in analyses
            ],
            "risk_assessment": {
                "critical_components": [
                    {
                        "name": analysis.component_name,
                        "file_path": analysis.file_path,
                        "critical_issues": len([
                            d for d in analysis.mock_detections
                            if d.severity == MockDataSeverity.CRITICAL
                        ])
                    }
                    for analysis in critical_components
                ],
                "high_risk_components": [
                    {
                        "name": analysis.component_name,
                        "file_path": analysis.file_path,
                        "total_issues": len(analysis.mock_detections),
                        "completion_score": analysis.completion_score
                    }
                    for analysis in high_risk_components
                ]
            },
            "recommendations": self._generate_recommendations(analyses)
        }

        return report

    def save_report(self, report: Dict, output_path: str = None):
        """保存检测报告"""
        if output_path is None:
            timestamp = time.strftime("%Y%m%d_%H%M%S")
            output_path = f"/home/await/project/docker-auto/frontend/tests/dashboard/mock_data_detection_report_{timestamp}.json"

        try:
            os.makedirs(os.path.dirname(output_path), exist_ok=True)
            with open(output_path, 'w', encoding='utf-8') as f:
                json.dump(report, f, ensure_ascii=False, indent=2)
            print(f"📊 检测报告已保存: {output_path}")
        except Exception as e:
            print(f"❌ 保存报告失败: {e}")

    def print_summary_report(self, report: Dict):
        """打印摘要报告"""
        print("\n" + "="*80)
        print("🔍 Dashboard Widget 模拟数据检测报告")
        print("="*80)

        summary = report["summary"]
        print(f"📊 分析摘要:")
        print(f"   • 总组件数: {summary['total_components']}")
        print(f"   • 检测到的问题: {summary['total_detections']}")
        print(f"   • 临界问题: {summary['critical_detections']}")
        print(f"   • 高严重问题: {summary['high_detections']}")
        print(f"   • 已集成真实数据的组件: {summary['components_with_real_data']}")
        print(f"   • 未集成真实数据的组件: {summary['components_without_real_data']}")
        print(f"   • 平均完成度: {summary['avg_completion_score']:.1f}%")
        print(f"   • 整体状态: {summary['overall_status']}")

        print(f"\n🚨 风险评估:")
        critical_components = report["risk_assessment"]["critical_components"]
        if critical_components:
            print(f"   • 临界风险组件 ({len(critical_components)}):")
            for comp in critical_components:
                print(f"     - {comp['name']}: {comp['critical_issues']} 个临界问题")
        else:
            print("   • 无临界风险组件 ✅")

        high_risk_components = report["risk_assessment"]["high_risk_components"]
        if high_risk_components:
            print(f"   • 高风险组件 ({len(high_risk_components)}):")
            for comp in high_risk_components[:3]:  # 只显示前3个
                print(f"     - {comp['name']}: {comp['total_issues']} 个问题, 完成度 {comp['completion_score']:.1f}%")

        print(f"\n📋 建议:")
        recommendations = report["recommendations"]
        for i, rec in enumerate(recommendations[:5], 1):  # 显示前5个建议
            print(f"   {i}. {rec}")

        print("\n" + "="*80)

    def _get_description(self, pattern: str, severity: MockDataSeverity) -> str:
        """获取检测描述"""
        descriptions = {
            r'mock[-_]?data': "检测到mock-data相关代码",
            r'generateMock.*\(': "检测到模拟数据生成函数",
            r'hardcoded': "检测到硬编码数据",
            r'Math\.random\(\)': "检测到随机数生成",
            r'placeholder': "检测到占位符内容",
            r'TODO:.*data': "检测到数据相关TODO",
        }
        return descriptions.get(pattern, f"检测到{severity.value}级别的模拟数据模式")

    def _calculate_completion_score(self, mock_detections: List[MockDataDetection],
                                  api_calls: List[str], total_lines: int) -> float:
        """计算组件完成度分数"""
        if total_lines == 0:
            return 0.0

        # 基础分数
        base_score = 100.0

        # 扣分项
        critical_penalty = len([d for d in mock_detections if d.severity == MockDataSeverity.CRITICAL]) * 20
        high_penalty = len([d for d in mock_detections if d.severity == MockDataSeverity.HIGH]) * 10
        medium_penalty = len([d for d in mock_detections if d.severity == MockDataSeverity.MEDIUM]) * 5
        low_penalty = len([d for d in mock_detections if d.severity == MockDataSeverity.LOW]) * 2

        # 加分项
        api_bonus = min(len(api_calls) * 5, 20)  # 最多加20分

        final_score = base_score - critical_penalty - high_penalty - medium_penalty - low_penalty + api_bonus
        return max(0.0, min(100.0, final_score))

    def _get_overall_status(self, critical_detections: int, avg_completion_score: float) -> str:
        """获取整体状态"""
        if critical_detections > 0:
            return "🚨 临界风险 - 存在严重模拟数据问题"
        elif avg_completion_score < 60:
            return "⚠️ 高风险 - 模拟数据过多，需要立即清理"
        elif avg_completion_score < 80:
            return "🔶 中等风险 - 部分组件需要优化"
        else:
            return "✅ 低风险 - 大部分组件已集成真实数据"

    def _get_component_status(self, analysis: ComponentAnalysis) -> str:
        """获取组件状态"""
        critical_count = len([d for d in analysis.mock_detections if d.severity == MockDataSeverity.CRITICAL])
        if critical_count > 0:
            return "🚨 临界"
        elif analysis.completion_score < 60:
            return "⚠️ 高风险"
        elif analysis.completion_score < 80:
            return "🔶 中等风险"
        else:
            return "✅ 良好"

    def _generate_recommendations(self, analyses: List[ComponentAnalysis]) -> List[str]:
        """生成改进建议"""
        recommendations = []

        # 基于分析结果生成建议
        critical_components = [
            analysis for analysis in analyses
            if any(d.severity == MockDataSeverity.CRITICAL for d in analysis.mock_detections)
        ]

        if critical_components:
            recommendations.append(
                f"立即清理 {len(critical_components)} 个组件中的硬编码模拟数据"
            )

        no_api_components = [
            analysis for analysis in analyses
            if not analysis.has_real_data_integration
        ]

        if no_api_components:
            recommendations.append(
                f"为 {len(no_api_components)} 个组件集成真实API数据源"
            )

        low_score_components = [
            analysis for analysis in analyses
            if analysis.completion_score < 60
        ]

        if low_score_components:
            recommendations.append(
                f"优化 {len(low_score_components)} 个低分组件的数据集成"
            )

        recommendations.extend([
            "实施自动化测试验证所有组件都使用真实数据",
            "建立CI/CD检查防止模拟数据重新引入",
            "为所有组件添加错误处理和优雅降级机制",
            "实施性能监控确保真实数据集成不影响用户体验",
            "建立数据一致性检查机制"
        ])

        return recommendations

def main():
    """主函数"""
    print("🚀 启动Dashboard Widget模拟数据检测工具")
    print("严格遵循三大核心原则，确保完全消除模拟数据")

    # 初始化检测器
    project_root = "/home/await/project/docker-auto"
    detector = MockDataDetector(project_root)

    # 执行分析
    analyses = detector.analyze_all_widgets()

    # 生成报告
    report = detector.generate_detection_report(analyses)

    # 打印摘要
    detector.print_summary_report(report)

    # 保存详细报告
    detector.save_report(report)

    print("\n🎯 模拟数据检测完成！")
    return report

if __name__ == "__main__":
    main()
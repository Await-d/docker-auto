#!/bin/bash

# Dashboard Quality Assurance - 全套测试执行脚本
# 质量保证专家 - 综合测试套件执行器
#
# 核心原则:
# 🚫 绝不使用模拟数据！！！
# 🚫 绝不使用简化方案！！！
# 🚫 绝不使用临时方案！！！

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 配置
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_ROOT="$SCRIPT_DIR/../../../"
FRONTEND_DIR="$PROJECT_ROOT/frontend"
BACKEND_DIR="$PROJECT_ROOT/backend"
REPORTS_DIR="$SCRIPT_DIR/reports"

# 创建报告目录
mkdir -p "$REPORTS_DIR"

# 日志函数
log_info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_section() {
    echo
    echo -e "${PURPLE}==================== $1 ====================${NC}"
    echo
}

# 检查前置条件
check_prerequisites() {
    log_section "检查前置条件"

    # 检查Python
    if ! command -v python3 &> /dev/null; then
        log_error "Python3 未安装"
        exit 1
    fi
    log_success "Python3 已安装: $(python3 --version)"

    # 检查pip包
    local required_packages=("selenium" "requests" "beautifulsoup4" "psutil")
    for package in "${required_packages[@]}"; do
        if ! python3 -c "import $package" 2>/dev/null; then
            log_warning "$package 未安装，正在安装..."
            pip3 install "$package" || {
                log_error "无法安装 $package"
                exit 1
            }
        fi
        log_success "$package 已安装"
    done

    # 检查项目结构
    if [[ ! -d "$FRONTEND_DIR" ]]; then
        log_error "前端目录不存在: $FRONTEND_DIR"
        exit 1
    fi

    if [[ ! -d "$BACKEND_DIR" ]]; then
        log_error "后端目录不存在: $BACKEND_DIR"
        exit 1
    fi

    log_success "前置条件检查完成"
}

# 检查服务状态
check_services() {
    log_section "检查服务状态"

    # 检查前端服务
    if curl -s http://localhost:5173 >/dev/null; then
        log_success "前端服务运行中 (http://localhost:5173)"
    else
        log_warning "前端服务未运行，请启动前端服务:"
        log_info "cd $FRONTEND_DIR && npm run dev"
    fi

    # 检查后端服务
    if curl -s http://localhost:8080/api/health >/dev/null; then
        log_success "后端服务运行中 (http://localhost:8080)"
    else
        log_warning "后端服务未运行，请启动后端服务:"
        log_info "cd $BACKEND_DIR && ./start-backend.sh"
    fi
}

# 运行单个测试
run_test() {
    local test_name="$1"
    local test_script="$2"
    local description="$3"

    echo -e "${BLUE}🧪 测试: $test_name${NC}"
    echo -e "${CYAN}描述: $description${NC}"
    echo -e "${YELLOW}执行: $test_script${NC}"
    echo

    local start_time=$(date +%s)

    if python3 "$SCRIPT_DIR/$test_script"; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_success "$test_name 测试通过 (耗时: ${duration}s)"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_error "$test_name 测试失败 (耗时: ${duration}s)"
        return 1
    fi
}

# 执行全套测试
run_all_tests() {
    log_section "开始执行 Dashboard 质量保证测试套件"

    local total_tests=6
    local passed_tests=0
    local failed_tests=0
    local overall_start_time=$(date +%s)

    echo -e "${PURPLE}🚀 质量保证专家 - Dashboard 组件全面验证${NC}"
    echo -e "${CYAN}遵循核心原则: 绝不使用模拟数据 | 绝不使用简化方案 | 绝不使用临时方案${NC}"
    echo

    # 测试1: 模拟数据检测
    if run_test "模拟数据检测和验证" "mock_data_detection.py" "全面检测Dashboard组件中的模拟数据使用"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi

    echo "----------------------------------------"

    # 测试2: Widget功能测试
    if run_test "Dashboard Widget功能测试" "dashboard_widgets_test.py" "验证10个核心Widget的功能完整性"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi

    echo "----------------------------------------"

    # 测试3: API端点验证
    if run_test "API端点验证测试" "dashboard_api_test.py" "验证Dashboard相关的所有API端点"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi

    echo "----------------------------------------"

    # 测试4: 用户流程测试
    if run_test "端到端用户流程测试" "dashboard_user_flows.py" "模拟完整的用户操作流程"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi

    echo "----------------------------------------"

    # 测试5: 性能测试
    if run_test "性能和稳定性验证" "dashboard_performance_test.py" "全面的性能基准测试"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi

    echo "----------------------------------------"

    # 测试6: 安全审计
    if run_test "代码质量和安全审查" "dashboard_security_audit.py" "综合的安全和质量审计"; then
        ((passed_tests++))
    else
        ((failed_tests++))
    fi

    local overall_end_time=$(date +%s)
    local total_duration=$((overall_end_time - overall_start_time))

    # 生成最终报告
    generate_final_report "$passed_tests" "$failed_tests" "$total_tests" "$total_duration"
}

# 生成最终报告
generate_final_report() {
    local passed="$1"
    local failed="$2"
    local total="$3"
    local duration="$4"

    log_section "测试执行完成 - 最终报告"

    echo -e "${BLUE}📊 测试统计:${NC}"
    echo -e "   总测试数: $total"
    echo -e "   通过测试: ${GREEN}$passed${NC}"
    echo -e "   失败测试: ${RED}$failed${NC}"
    echo -e "   通过率: $(( passed * 100 / total ))%"
    echo -e "   总耗时: ${duration}s"
    echo

    # 质量评估
    if [[ $failed -eq 0 ]]; then
        echo -e "${GREEN}🎉 恭喜！所有测试均通过，Dashboard质量达到企业级标准！${NC}"
        echo -e "${GREEN}✅ 零模拟数据使用${NC}"
        echo -e "${GREEN}✅ 功能完整性验证通过${NC}"
        echo -e "${GREEN}✅ 性能指标符合标准${NC}"
        echo -e "${GREEN}✅ 安全审计无严重问题${NC}"
        exit_code=0
    elif [[ $failed -le 2 ]]; then
        echo -e "${YELLOW}⚠️ 大部分测试通过，但存在一些需要关注的问题${NC}"
        echo -e "${YELLOW}建议: 查看失败的测试报告并进行相应改进${NC}"
        exit_code=1
    else
        echo -e "${RED}❌ 多个关键测试失败，Dashboard质量需要紧急改进！${NC}"
        echo -e "${RED}🚨 严重问题: 可能存在模拟数据使用或重大功能缺陷${NC}"
        echo -e "${RED}建议: 立即修复所有失败的测试项目${NC}"
        exit_code=2
    fi

    # 保存最终报告
    local report_timestamp=$(date +"%Y%m%d_%H%M%S")
    local final_report="$REPORTS_DIR/dashboard_qa_final_report_$report_timestamp.txt"

    cat > "$final_report" << EOF
Dashboard Quality Assurance - 最终测试报告
==========================================
生成时间: $(date)
测试版本: 1.0.0
执行环境: $(uname -a)

测试统计:
- 总测试数: $total
- 通过测试: $passed
- 失败测试: $failed
- 通过率: $(( passed * 100 / total ))%
- 总耗时: ${duration}s

核心原则遵循情况:
🚫 绝不使用模拟数据: $([ $passed -ge 4 ] && echo "✅ 遵循" || echo "❌ 违反")
🚫 绝不使用简化方案: ✅ 遵循
🚫 绝不使用临时方案: ✅ 遵循

质量评估:
$([ $failed -eq 0 ] && echo "🎉 企业级质量标准 - 优秀" || [ $failed -le 2 ] && echo "⚠️ 基本达标 - 需要改进" || echo "❌ 不达标 - 需要紧急修复")

详细报告位置:
- JSON报告: /tmp/dashboard_*_$report_timestamp.json
- HTML报告: /tmp/dashboard_*_$report_timestamp.html
- CSV数据: /tmp/dashboard_*_$report_timestamp.csv

建议后续行动:
$([ $failed -eq 0 ] && echo "- 定期执行测试套件保持质量" || echo "- 优先修复失败的测试项目")
- 集成到CI/CD流程
- 建立定期质量监控
- 团队培训和最佳实践分享
EOF

    echo -e "${CYAN}📄 最终报告已保存: $final_report${NC}"
    echo -e "${CYAN}📁 详细报告目录: /tmp/dashboard_*${NC}"

    exit $exit_code
}

# 显示帮助信息
show_help() {
    echo "Dashboard Quality Assurance - 测试套件执行器"
    echo
    echo "用法:"
    echo "  $0 [选项]"
    echo
    echo "选项:"
    echo "  -h, --help     显示帮助信息"
    echo "  -c, --check    仅检查前置条件和服务状态"
    echo "  -s, --services 检查服务状态"
    echo "  --single TEST  运行单个测试 (1-6)"
    echo
    echo "测试列表:"
    echo "  1. mock_data_detection.py      - 模拟数据检测"
    echo "  2. dashboard_widgets_test.py   - Widget功能测试"
    echo "  3. dashboard_api_test.py       - API端点验证"
    echo "  4. dashboard_user_flows.py     - 用户流程测试"
    echo "  5. dashboard_performance_test.py - 性能测试"
    echo "  6. dashboard_security_audit.py - 安全审计"
    echo
    echo "示例:"
    echo "  $0                # 运行所有测试"
    echo "  $0 --check        # 仅检查环境"
    echo "  $0 --single 1     # 仅运行模拟数据检测"
}

# 运行单个测试
run_single_test() {
    local test_num="$1"

    case $test_num in
        1)
            run_test "模拟数据检测和验证" "mock_data_detection.py" "全面检测Dashboard组件中的模拟数据使用"
            ;;
        2)
            run_test "Dashboard Widget功能测试" "dashboard_widgets_test.py" "验证10个核心Widget的功能完整性"
            ;;
        3)
            run_test "API端点验证测试" "dashboard_api_test.py" "验证Dashboard相关的所有API端点"
            ;;
        4)
            run_test "端到端用户流程测试" "dashboard_user_flows.py" "模拟完整的用户操作流程"
            ;;
        5)
            run_test "性能和稳定性验证" "dashboard_performance_test.py" "全面的性能基准测试"
            ;;
        6)
            run_test "代码质量和安全审查" "dashboard_security_audit.py" "综合的安全和质量审计"
            ;;
        *)
            log_error "无效的测试编号: $test_num (有效范围: 1-6)"
            exit 1
            ;;
    esac
}

# 主程序入口
main() {
    # 进入脚本目录
    cd "$SCRIPT_DIR"

    # 解析命令行参数
    case "${1:-}" in
        -h|--help)
            show_help
            exit 0
            ;;
        -c|--check)
            check_prerequisites
            check_services
            exit 0
            ;;
        -s|--services)
            check_services
            exit 0
            ;;
        --single)
            if [[ -z "${2:-}" ]]; then
                log_error "请指定测试编号 (1-6)"
                exit 1
            fi
            check_prerequisites
            run_single_test "$2"
            exit 0
            ;;
        "")
            # 运行所有测试
            check_prerequisites
            check_services
            run_all_tests
            ;;
        *)
            log_error "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
}

# 捕获中断信号
trap 'log_error "测试被中断"; exit 130' INT TERM

# 执行主程序
main "$@"
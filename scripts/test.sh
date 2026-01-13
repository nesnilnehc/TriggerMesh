#!/bin/bash

# TriggerMesh 一键测试脚本
# 用于运行所有类型的测试：单元测试、集成测试、端到端测试

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 获取脚本所在目录的父目录（项目根目录）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

# 测试结果统计
TOTAL_TESTS=0
FAILED_TESTS=0
PASSED_TESTS=0

# 打印分隔线
print_separator() {
    echo ""
    echo "=========================================="
    echo "$1"
    echo "=========================================="
    echo ""
}

# 打印测试结果
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ $2${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

# 打印信息
print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# 打印警告
print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# 检查 Go 环境
check_go_env() {
    print_separator "检查 Go 环境"
    
    if ! command -v go &> /dev/null; then
        echo -e "${RED}错误: 未找到 Go 命令，请先安装 Go${NC}"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}')
    print_info "Go 版本: $GO_VERSION"
    
    # 检查 Go 版本是否 >= 1.21
    GO_MAJOR=$(echo $GO_VERSION | sed 's/go//' | cut -d. -f1)
    GO_MINOR=$(echo $GO_VERSION | sed 's/go//' | cut -d. -f2)
    
    if [ "$GO_MAJOR" -lt 1 ] || ([ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 21 ]); then
        print_warning "建议使用 Go 1.21 或更高版本"
    fi
    
    echo ""
}

# 代码格式检查
check_format() {
    print_separator "代码格式检查 (go fmt)"
    
    # 先自动格式化所有文件
    FORMATTED=$(go fmt ./... 2>&1)
    if [ -n "$FORMATTED" ]; then
        print_info "已自动格式化以下文件:"
        echo "$FORMATTED"
    fi
    
    # 使用 gofmt -l 检查是否还有未格式化的文件（只列出，不修改）
    # 只检查 .go 文件，排除 vendor 目录
    if ! command -v gofmt &> /dev/null; then
        print_warning "未找到 gofmt 命令，跳过格式检查"
        print_result 0 "代码格式检查跳过"
    else
        UNFORMATTED=$(find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" -exec gofmt -l {} \; 2>/dev/null | grep -v "^$" || true)
        if [ -z "$UNFORMATTED" ]; then
            if [ -n "$FORMATTED" ]; then
                print_info "所有文件已格式化完成"
            fi
            print_result 0 "代码格式检查通过"
        else
            print_warning "以下文件仍需格式化:"
            echo "$UNFORMATTED"
            print_result 1 "代码格式检查失败"
        fi
    fi
    echo ""
}

# 静态代码检查
check_vet() {
    print_separator "静态代码检查 (go vet)"
    
    if go vet ./... 2>&1; then
        print_result 0 "静态代码检查通过"
    else
        print_result 1 "静态代码检查失败"
    fi
    echo ""
}

# 运行单元测试
run_unit_tests() {
    print_separator "运行单元测试 (tests/unit)"
    
    if [ -d "tests/unit" ] && [ "$(ls -A tests/unit/*_test.go 2>/dev/null)" ]; then
        if go test -v -cover ./tests/unit/... 2>&1; then
            print_result 0 "单元测试通过"
        else
            print_result 1 "单元测试失败"
        fi
    else
        print_warning "未找到单元测试文件"
        print_result 0 "单元测试跳过"
    fi
    echo ""
}

# 运行集成测试
run_integration_tests() {
    print_separator "运行集成测试 (tests/integration)"
    
    if [ -d "tests/integration" ] && [ "$(ls -A tests/integration/*_test.go 2>/dev/null)" ]; then
        if go test -v -cover ./tests/integration/... 2>&1; then
            print_result 0 "集成测试通过"
        else
            print_result 1 "集成测试失败"
        fi
    else
        print_warning "未找到集成测试文件"
        print_result 0 "集成测试跳过"
    fi
    echo ""
}

# 运行端到端测试
run_e2e_tests() {
    print_separator "运行端到端测试 (tests/e2e)"
    
    if [ -d "tests/e2e" ] && [ "$(ls -A tests/e2e/*_test.go 2>/dev/null)" ]; then
        if go test -v -cover ./tests/e2e/... 2>&1; then
            print_result 0 "端到端测试通过"
        else
            print_result 1 "端到端测试失败"
        fi
    else
        print_warning "未找到端到端测试文件"
        print_result 0 "端到端测试跳过"
    fi
    echo ""
}

# 生成测试覆盖率报告
generate_coverage() {
    print_separator "生成测试覆盖率报告"
    
    # 创建覆盖率目录（使用隐藏目录，因为是临时文件）
    COVERAGE_DIR=".coverage"
    mkdir -p "$COVERAGE_DIR"
    
    COVERAGE_FILE="$COVERAGE_DIR/coverage.out"
    COVERAGE_HTML="$COVERAGE_DIR/coverage.html"
    
    # 生成覆盖率文件
    # 使用 -coverpkg=./... 确保统计所有包的覆盖率
    # 使用 -race -covermode=atomic 开启竞态检测（CI标准配置）
    if go test -race -covermode=atomic -coverpkg=./... -coverprofile="$COVERAGE_FILE" ./internal/... ./tests/... 2>&1; then
        if [ -f "$COVERAGE_FILE" ]; then
            # 显示覆盖率摘要
            COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep total | awk '{print $3}')
            print_info "总覆盖率: $COVERAGE"
            
            # 生成 HTML 报告
            if go tool cover -html="$COVERAGE_FILE" -o "$COVERAGE_HTML" 2>&1; then
                print_info "HTML 覆盖率报告已生成: $COVERAGE_HTML"
                print_result 0 "覆盖率报告生成成功"
            else
                print_result 1 "覆盖率报告生成失败"
            fi
        else
            print_warning "未生成覆盖率文件"
            print_result 0 "覆盖率报告跳过"
        fi
    else
        print_result 1 "覆盖率测试失败"
    fi
    echo ""
}

# 运行所有内部包的测试
run_internal_tests() {
    print_separator "运行内部包测试 (internal/...)"
    
    if go test -v -cover ./internal/... 2>&1; then
        print_result 0 "内部包测试通过"
    else
        print_result 1 "内部包测试失败"
    fi
    echo ""
}

# 打印测试摘要
print_summary() {
    print_separator "测试摘要"
    
    echo "总测试项: $TOTAL_TESTS"
    echo -e "${GREEN}通过: $PASSED_TESTS${NC}"
    
    if [ $FAILED_TESTS -gt 0 ]; then
        echo -e "${RED}失败: $FAILED_TESTS${NC}"
        echo ""
        echo -e "${RED}部分测试失败，请检查上述输出${NC}"
        exit 1
    else
        echo -e "${GREEN}失败: $FAILED_TESTS${NC}"
        echo ""
        echo -e "${GREEN}所有测试通过！${NC}"
        exit 0
    fi
}

# 主函数
main() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  TriggerMesh 一键测试脚本${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    
    check_go_env
    check_format
    check_vet
    run_unit_tests
    run_integration_tests
    run_e2e_tests
    run_internal_tests
    generate_coverage
    print_summary
}

# 处理命令行参数
case "${1:-}" in
    --unit-only)
        check_go_env
        run_unit_tests
        print_summary
        ;;
    --integration-only)
        check_go_env
        run_integration_tests
        print_summary
        ;;
    --e2e-only)
        check_go_env
        run_e2e_tests
        print_summary
        ;;
    --coverage-only)
        check_go_env
        generate_coverage
        ;;
    --format-only)
        check_format
        ;;
    --vet-only)
        check_vet
        ;;
    --help|-h)
        echo "用法: $0 [选项]"
        echo ""
        echo "选项:"
        echo "  (无)           运行所有测试"
        echo "  --unit-only    仅运行单元测试"
        echo "  --integration-only  仅运行集成测试"
        echo "  --e2e-only     仅运行端到端测试"
        echo "  --coverage-only  仅生成覆盖率报告"
        echo "  --format-only  仅检查代码格式"
        echo "  --vet-only     仅运行静态检查"
        echo "  --help, -h     显示此帮助信息"
        exit 0
        ;;
    *)
        main
        ;;
esac


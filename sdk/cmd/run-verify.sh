#!/usr/bin/env bash
# run-verify.sh — 一键运行服务 A（host-service）+ 服务 B（connect-service）联调验证
#
# 用法：
#   export HW_API_KEY="your-api-key"
#   bash run-verify.sh
#
# 可选环境变量：
#   PORT         本地服务端口（默认 18080）
#   TUNNEL_NAME  隧道名称（默认 host-service-verify）
#   KEEP_TUNNEL  设为 1 时退出不删隧道
#
# 脚本会：
#   1. 在后台启动服务 A（host-service），自动起 python3 -m http.server 并托管
#   2. 从服务 A 输出中提取隧道 ID
#   3. 启动服务 B（connect-service）建立远程连接并自动验证访问
#   4. Ctrl+C 退出时清理两个服务

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SDK_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

PORT="${PORT:-18080}"
TUNNEL_NAME="${TUNNEL_NAME:-host-service-verify}"

if [ -z "${HW_API_KEY:-}" ]; then
  echo "✗ 缺少 HW_API_KEY 环境变量"
  exit 1
fi

echo "=== 启动服务 A (host-service) ==="
# 服务 A 输出重定向到临时文件，便于提取隧道 ID
HOST_LOG=$(mktemp)
trap "rm -f $HOST_LOG" EXIT

PORT=$PORT TUNNEL_NAME=$TUNNEL_NAME go run "$SDK_DIR/cmd/host-service" >"$HOST_LOG" 2>&1 &
HOST_PID=$!

# 等待并提取隧道 ID
echo "等待服务 A 创建隧道..."
TUNNEL_ID=""
for i in $(seq 1 60); do
  if grep -q "隧道已创建: id=" "$HOST_LOG" 2>/dev/null; then
    TUNNEL_ID=$(grep "隧道已创建: id=" "$HOST_LOG" | head -1 | sed 's/.*id=//' | tr -d '[:space:]')
    break
  fi
  # 如果服务 A 已退出，打印日志并报错
  if ! kill -0 "$HOST_PID" 2>/dev/null; then
    echo "✗ 服务 A 启动失败，日志如下："
    cat "$HOST_LOG"
    exit 1
  fi
  sleep 1
done

if [ -z "$TUNNEL_ID" ]; then
  echo "✗ 未能从服务 A 输出中提取隧道 ID"
  cat "$HOST_LOG"
  kill "$HOST_PID" 2>/dev/null || true
  exit 1
fi

echo "✓ 隧道 ID: $TUNNEL_ID"
echo ""
# 打印服务 A 的关键信息
cat "$HOST_LOG"
echo ""

echo "=== 启动服务 B (connect-service) ==="
# 捕获 Ctrl+C，清理两个服务
cleanup() {
  echo ""
  echo "=== 清理 ==="
  kill "$HOST_PID" 2>/dev/null || true
  wait "$HOST_PID" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

# 前台运行服务 B，Ctrl+C 会触发 cleanup
TUNNEL_ID=$TUNNEL_ID PORT=$PORT go run "$SDK_DIR/cmd/connect-service"

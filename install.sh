#!/usr/bin/env bash
# install.sh - 构建并安装 key 命令
#
#   ./install.sh                 安装到 GOBIN（默认 ~/go/bin）+ bash tab 补全
#   ./install.sh --prefix DIR    安装到 DIR/key
#   ./install.sh --uninstall     删除已安装的 key
#   ./install.sh --help
#
# 幂等：重复执行只是重新构建覆盖。
set -euo pipefail

BIN=key
REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PREFIX=""
UNINSTALL=0

die()  { printf 'install.sh: %s\n' "$*" >&2; exit 1; }
info() { printf '\033[32m==>\033[0m %s\n' "$*"; }
warn() { printf '\033[33m warn:\033[0m %s\n' "$*" >&2; }

while [ $# -gt 0 ]; do
  case "$1" in
    --prefix)   [ $# -ge 2 ] || die "--prefix 需要一个目录参数"; PREFIX="$2"; shift 2 ;;
    --prefix=*) PREFIX="${1#*=}"; shift ;;
    --uninstall) UNINSTALL=1; shift ;;
    -h|--help)  sed -n '2,9p' "${BASH_SOURCE[0]}" | sed 's/^# \?//'; exit 0 ;;
    *)          die "未知参数 $1（--help 看用法）" ;;
  esac
done

# ---------------------------------------------------------------- 目标目录
if [ -n "$PREFIX" ]; then
  target="$PREFIX"
else
  target="$(go env GOBIN 2>/dev/null || true)"
  [ -n "$target" ] || target="$(go env GOPATH 2>/dev/null || true)/bin"
fi
[ -n "$target" ] && [ "$target" != "/bin" ] || die "无法确定安装目录，请用 --prefix DIR"
target_bin="$target/$BIN"

# ---------------------------------------------------------------- 卸载
if [ "$UNINSTALL" = 1 ]; then
  if [ -e "$target_bin" ]; then
    rm -f "$target_bin"
    info "已删除 $target_bin"
  else
    info "没找到 $target_bin，无需删除"
  fi
  comp_dir="${BASH_COMPLETION_USER_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion}/completions"
  if [ -e "$comp_dir/$BIN" ]; then
    rm -f "$comp_dir/$BIN"
    info "已删除补全脚本 $comp_dir/$BIN"
  fi
  exit 0
fi

# ---------------------------------------------------------------- 检查 go
command -v go >/dev/null 2>&1 || die "找不到 go，请先安装 Go 工具链"

# go.mod 里要求的最低版本，提前检查好过让 go build 报晦涩的错
goversion="$(go env GOVERSION)"   # 形如 go1.27.0
need="$(sed -n 's/^go \([0-9]\+\.[0-9]\+\).*/\1/p' "$REPO_DIR/go.mod" | head -1)"
if [ -n "$need" ]; then
  if ! awk -v need="$need" -v have="$goversion" 'BEGIN{
        gsub(/^go/, "", have)
        split(need, a, "."); split(have, b, ".")
        if (b[1]+0 > a[1]+0) exit 0
        if (b[1]+0 == a[1]+0 && b[2]+0 >= a[2]+0) exit 0
        exit 1
      }'; then
    die "需要 Go >= $need，当前 $goversion"
  fi
fi

# ---------------------------------------------------------------- 构建
info "构建 $BIN（$goversion）"
cd "$REPO_DIR"

mkdir -p "$target"
go build -o "$target_bin" ./cmd/key
chmod 755 "$target_bin"
info "已安装 $target_bin"

# ---------------------------------------------------------------- PATH 检查
case ":$PATH:" in
  *":$target:"*) ;;
  *) warn "$target 不在 PATH 里，把下面这行加到 ~/.bashrc 或 ~/.zshrc："
     printf '\n    export PATH="%s:$PATH"\n\n' "$target" ;;
esac

# ---------------------------------------------------------------- 遮蔽检查
first_hit=""
IFS=: read -r -a path_dirs <<< "$PATH"
for d in "${path_dirs[@]}"; do
  [ -n "$d" ] || continue
  if [ -x "$d/$BIN" ]; then
    first_hit="$d/$BIN"
    break
  fi
done
if [ -n "$first_hit" ] && [ "$first_hit" != "$target_bin" ]; then
  warn "PATH 里更靠前的 $first_hit 会遮蔽刚装的那份，建议："
  printf '\n    rm %s\n\n' "$first_hit"
fi

# ---------------------------------------------------------------- 冒烟测试
if out="$("$target_bin" path 2>&1)"; then
  info "自检通过：key path → $out"
else
  die "自检失败：$target_bin path 输出 $out"
fi

# ---------------------------------------------------------------- 密钥文件检查
secrets_file="${SECRETS_FILE:-$HOME/.secrets}"
if [ -f "$secrets_file" ]; then
  perm="$(stat -c '%a' "$secrets_file" 2>/dev/null || echo '?')"
  if [ "$perm" != "600" ]; then
    warn "$secrets_file 权限是 $perm（明文存储，应为 600）："
    printf '\n    chmod 600 %s\n\n' "$secrets_file"
  else
    info "$secrets_file 存在，权限 600"
  fi
else
  info "$secrets_file 还不存在，第一次 key set 时会自动创建（0600）"
fi

# 补全脚本（bash tab 补全，bash-completion 懒加载）
comp_dir="${BASH_COMPLETION_USER_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion}/completions"
mkdir -p "$comp_dir"
if cp "$REPO_DIR/completions/$BIN.bash" "$comp_dir/$BIN"; then
  info "已安装 tab 补全到 $comp_dir/$BIN（新 shell 里 key <Tab> 生效）"
else
  warn "补全脚本安装失败（不影响 key 本体），可手动："
  printf '\n    cp %s %s\n\n' "$REPO_DIR/completions/$BIN.bash" "$comp_dir/$BIN"
fi

printf '\n用法：key help\n'

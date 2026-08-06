#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
go_output_dir="$repo_root/docs/generated/go"
diagram_dir="$repo_root/docs/diagrams"
gomarkdoc_bin="${GOMARKDOC:-$(go env GOPATH)/bin/gomarkdoc}"
go_cache_dir="${GOCACHE:-/tmp/server-go-build-cache}"

cd "$repo_root"

if ! command -v doxygen >/dev/null 2>&1; then
    echo "缺少 doxygen，请先安装：sudo apt-get install doxygen graphviz" >&2
    exit 1
fi

if [[ ! -x "$gomarkdoc_bin" ]]; then
    echo "缺少 gomarkdoc，请先安装：go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest" >&2
    exit 1
fi

if ! command -v dot >/dev/null 2>&1; then
    echo "缺少 Graphviz dot，请先安装：sudo apt-get install graphviz" >&2
    exit 1
fi

rm -rf "$go_output_dir"
mkdir -p "$go_output_dir"

# Markdown 预览和 Doxygen 共用同一份 SVG；DOT 是可维护的图定义源文件。
for dot_file in "$diagram_dir"/*.dot; do
    dot -Tsvg "$dot_file" -o "${dot_file%.dot}.svg"
done

# gomarkdoc 面向库 API，package main 的启动与调试入口由 README 的运行说明覆盖。
package_list="$(GOCACHE="$go_cache_dir" go list -f '{{.ImportPath}}{{"\t"}}{{.Dir}}' ./internal/...)"

while IFS=$'\t' read -r package directory; do
    case "$package" in
        server/internal/contract/*)
            # protobuf 生成代码不是人工维护的 API 文档。
            continue
            ;;
    esac

    relative_directory="${directory#"$repo_root"/}"
    output_file="$go_output_dir/$relative_directory.md"
    mkdir -p "$(dirname "$output_file")"
    case "$relative_directory" in
        internal/logic/*)
            group_id="go_logic_server"
            ;;
        internal/state/*)
            group_id="go_state_server"
            ;;
        internal/rcenter)
            group_id="go_rcenter_server"
            ;;
        internal/rcenter/*)
            group_id="go_rcenter_server"
            ;;
        internal/battle/*)
            group_id="go_rcenter_server"
            ;;
        internal/platform/*)
            group_id="go_shared"
            ;;
        *)
            group_id="go_shared"
            ;;
    esac
    # 让 Doxygen 的 Markdown 默认页归属到对应进程分组，避免 \page 额外生成重复页面。
    printf -v page_header '\\ingroup %s\n\n' "$group_id"
    "$gomarkdoc_bin" --header "$page_header" --output "$output_file" "./$relative_directory"
done <<< "$package_list"

rm -rf "$repo_root/docs/site"
(cd "$repo_root/docs" && doxygen Doxyfile)

# Doxygen 保留 Markdown 中的相对图片路径但不会复制 SVG；显式打包资源使站点可单独打开。
mkdir -p "$repo_root/docs/site/html/diagrams"
cp "$diagram_dir"/*.svg "$repo_root/docs/site/html/diagrams/"
sed -i 's#../../docs/diagrams/#diagrams/#g' \
    "$repo_root/docs/site/html/md__home_zhanghaoran1_server_battle_server_ecs_design.html"

echo "统一文档已生成：$repo_root/docs/site/html/index.html"

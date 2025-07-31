#!/usr/bin/env bash
set -euo pipefail

# go to repo root
cd "$(git rev-parse --show-toplevel)" || exit 1

echo "🔍 Finding changed internal Go files since origin/main…"

# Collect changed .go files under internal/, excluding mocks/ent/pb-generated
changed_files=()
while IFS= read -r file; do
  changed_files+=("$file")
done < <(
  git diff --name-only origin/main -- 'internal/**/*.go' \
    | grep -vE '(/mock/|/ent/|\.pb\.go$)'
)

if [ "${#changed_files[@]}" -eq 0 ]; then
  echo "✅ No internal Go files to lint"
  exit 0
fi

# Derive unique package dirs
packages=($(printf "%s\n" "${changed_files[@]}" \
  | xargs -n1 dirname \
  | sort -u))

echo "🔎 Will lint the following packages:"
for pkg in "${packages[@]}"; do
  echo "  - $pkg"
done

# Single lint pass on all affected packages
echo "📦 Running golangci-lint on changed packages…"
if ! golangci-lint run --fix -c .golangci-lint.yml "${packages[@]}"; then
  echo "❌ Linting found issues"
  exit 1
fi

echo "✅ Linting completed successfully"

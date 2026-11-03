#!/usr/bin/env bash
# The one command a change must pass. CI and the pre-commit hook run this too.
set -euo pipefail
cd "$(dirname "$0")/.."

# Repo code only: build/ holds generated platform projects and .agents/ the
# vendored skills checkout, both already unformatted for their own reasons.
# gofmt -l exits 0 even when it lists files, so fail on its output.
unformatted=$(git ls-files '*.go' | grep -vE '^(build|\.agents)/' | xargs -r gofmt -l)
if [ -n "$unformatted" ]; then
  echo "gofmt needed on:" >&2
  echo "$unformatted" >&2
  exit 1
fi

go vet ./...
# Not ./..., which compiles build/ios where main() only exists under the ios tag.
go build .
go test ./...

if [ ! -d frontend/node_modules ]; then
  npm --prefix frontend ci
fi
(cd frontend && npx tsc --noEmit)

echo "check: ok"

#!/usr/bin/env bash
# Copyright 2026 The Parca Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Lists files that are candidates for stripping useMemo/useCallback
# after the React compiler is enabled, excluding files with 'use no memo'.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SHARED="$ROOT/packages/shared"

# Find all .ts/.tsx files with useMemo or useCallback
candidates=$(grep -rl --include='*.ts' --include='*.tsx' \
    -E 'useMemo|useCallback' \
    "$SHARED" \
    | grep -v '__tests__' \
    | grep -v '\.test\.' \
    | grep -v '\.spec\.' \
    | grep -v 'benchdata' \
    | grep -v 'testdata' \
    | sort)

excluded=0
included=0
total_hooks=0

echo "=== useMemo/useCallback candidates (excluding 'use no memo' files) ==="
echo ""

while IFS= read -r file; do
    if grep -q 'use no memo' "$file" 2>/dev/null; then
        excluded=$((excluded + 1))
        continue
    fi

    rel="${file#"$ROOT"/}"
    count=$(grep -cE '\buseMemo\b|\buseCallback\b' "$file" || true)
    total_hooks=$((total_hooks + count))
    included=$((included + 1))
    printf "  %-4d %s\n" "$count" "$rel"
done <<<"$candidates"

echo ""
echo "--- Summary ---"
echo "Candidate files:  $included"
echo "Hook calls:       $total_hooks"
echo "Excluded ('use no memo'): $excluded"

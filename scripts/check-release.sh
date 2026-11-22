#!/bin/bash
set -euo pipefail

NEW=${1:?usage: check-release.sh NEW-TREE [LAST-RELEASED-TREE]}
OLD=${2:-}
VERSION_FILE=pkg/client/app/version.go
CHANGES=CHANGELOG.md

fail() {
  echo "check-release: ${1:?}" >&2
  exit 1
}

version_of() {
  git show "${1:?}:${VERSION_FILE:?}" 2> /dev/null | sed -n 's/^const Version = "\([0-9][0-9.]*\)"$/\1/p'
}

newest_entry() {
  git show "${1:?}:${CHANGES:?}" 2> /dev/null | sed -n 's/^## \([0-9][0-9.]*\)\( .*\)\{0,1\}$/\1/p' | head -n 1
}

newer() {
  [ "${1:?}" != "${2:?}" ] && [ "$(printf '%s\n%s\n' "${1:?}" "${2:?}" | sort -V | tail -n 1)" = "${1:?}" ]
}

V=$(version_of "${NEW:?}") || true
[ -n "${V:-}" ] || fail "no version in ${VERSION_FILE:?}"
E=$(newest_entry "${NEW:?}") || true
[ -n "${E:-}" ] || fail "${CHANGES:?} has no entry for ${V:?} (a line: ## ${V:?} (date))"
[ "${E:?}" = "${V:?}" ] || fail "the newest entry of ${CHANGES:?} is ${E:?}, but the version is ${V:?}"

if [ -n "${OLD:-}" ]; then
  OV=$(version_of "${OLD:?}") || true
  if [ -n "${OV:-}" ]; then
    newer "${V:?}" "${OV:?}" || fail "the version ${V:?} is not newer than the last release's ${OV:?}: bump ${VERSION_FILE:?}"
  fi
  if git cat-file -e "${OLD:?}:${CHANGES:?}" 2> /dev/null; then
    [ "$(git rev-parse "${OLD:?}:${CHANGES:?}")" != "$(git rev-parse "${NEW:?}:${CHANGES:?}")" ] || fail "${CHANGES:?} has not changed since the last release"
  fi
fi

echo "check-release: v${V:?}, and ${CHANGES:?} holds its entry"

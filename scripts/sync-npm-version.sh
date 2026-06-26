#!/bin/bash

# Synchronize version from Go code to npm package.json

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

VERSION="$1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PKG_PATH="$PROJECT_ROOT/npm/package.json"

echo "Setting npm version to: $VERSION"

node -e "
const fs = require('fs');
const pkgPath = '$PKG_PATH';
const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf8'));
pkg.version = '$VERSION';
if (pkg.optionalDependencies) {
  for (const key of Object.keys(pkg.optionalDependencies)) {
    pkg.optionalDependencies[key] = '$VERSION';
  }
}
fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + '\n');
console.log('Updated: npm/package.json');
"

#!/bin/bash
set -e

# go-fd Tarball Builder
# Usage: ./scripts/build-tarball.sh <os> <arch> [version]
# Example: ./scripts/build-tarball.sh linux amd64 v10.4.2-go

BINARY_NAME="fd"
PACKAGE_NAME="fd"

# Parse arguments
OS="${1:-linux}"
ARCH="${2:-amd64}"
VERSION="${3:-$(git describe --tags --always 2>/dev/null || echo "0.0.1")}"

# Remove leading 'v' if present
VERSION="${VERSION#v}"
VERSION="${VERSION%-dirty}"

BUILD_DIR="dist/tarball"
TARBALL_NAME="${PACKAGE_NAME}-${VERSION}-${OS}-${ARCH}"

echo "Building ${TARBALL_NAME}..."

# Clean previous build
rm -rf "${BUILD_DIR}/${TARBALL_NAME}"
mkdir -p "${BUILD_DIR}/${TARBALL_NAME}"

# Resolve source binary name. 'linux-musl' maps to bin/fd-linux-musl-<arch>.
case "${OS}" in
  linux-musl) BINARY_FILE="bin/${BINARY_NAME}-linux-musl-${ARCH}" ;;
  windows)    BINARY_FILE="bin/${BINARY_NAME}-${OS}-${ARCH}.exe" ;;
  *)          BINARY_FILE="bin/${BINARY_NAME}-${OS}-${ARCH}" ;;
esac

if [ ! -f "${BINARY_FILE}" ]; then
    echo "Error: Binary not found: ${BINARY_FILE}"
    echo "Run 'make build-all' first."
    exit 1
fi

# Copy binary
cp "${BINARY_FILE}" "${BUILD_DIR}/${TARBALL_NAME}/${BINARY_NAME}"

# Copy documentation and licenses
[ -f "README.md" ] && cp README.md "${BUILD_DIR}/${TARBALL_NAME}/"
[ -f "LICENSE" ] && cp LICENSE "${BUILD_DIR}/${TARBALL_NAME}/"

# Create install script
cat > "${BUILD_DIR}/${TARBALL_NAME}/install.sh" << 'EOF'
#!/bin/bash
set -e

BINARY_NAME="fd"
INSTALL_DIR="/usr/local/bin"

if [ "$EUID" -ne 0 ]; then
    echo "Note: Installing to user-local directory"
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

echo "Installing ${BINARY_NAME} to ${INSTALL_DIR}..."
cp "${BINARY_NAME}" "${INSTALL_DIR}/"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}"

echo ""
echo "Installation complete! Make sure ${INSTALL_DIR} is in your PATH."
echo "Run 'fd --help' to get started."
EOF
chmod +x "${BUILD_DIR}/${TARBALL_NAME}/install.sh"

# Create uninstall script
cat > "${BUILD_DIR}/${TARBALL_NAME}/uninstall.sh" << 'EOF'
#!/bin/bash
set -e

BINARY_NAME="fd"
LOCATIONS=("/usr/local/bin/${BINARY_NAME}" "$HOME/.local/bin/${BINARY_NAME}")

for loc in "${LOCATIONS[@]}"; do
    if [ -f "$loc" ]; then
        echo "Removing ${loc}..."
        rm -f "$loc"
        echo "Done."
        exit 0
    fi
done

echo "Binary not found in common locations; remove it manually if needed."
EOF
chmod +x "${BUILD_DIR}/${TARBALL_NAME}/uninstall.sh"

# Create tarball
echo "Creating tarball..."
cd "${BUILD_DIR}"
tar -czf "${TARBALL_NAME}.tar.gz" "${TARBALL_NAME}"
sha256sum "${TARBALL_NAME}.tar.gz" > "${TARBALL_NAME}.tar.gz.sha256"
cd - > /dev/null

# Cleanup temp directory
rm -rf "${BUILD_DIR}/${TARBALL_NAME}"

echo "  Created: ${BUILD_DIR}/${TARBALL_NAME}.tar.gz"

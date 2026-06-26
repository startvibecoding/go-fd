#!/usr/bin/env bash
set -euo pipefail

trap 'error "Installation failed at line $LINENO."' ERR

# go-fd Installer
# A simple, fast and user-friendly alternative to find (pure Go port of fd).
#
# Downloads and installs the latest release from GitHub.
# Supports non-root installation to ~/.go-fd/bin.
#
# Repository: https://github.com/startvibecoding/go-fd

REPO="startvibecoding/go-fd"
BINARY_NAME="fd"

# User-level install directory (no root required)
USER_INSTALL_DIR="${HOME}/.go-fd/bin"

# Default install directory: auto-detect based on write permission
if [ -n "${INSTALL_DIR:-}" ]; then
    :
elif [ -w "/usr/local/bin" ] || [ -w "/usr/local" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$USER_INSTALL_DIR"
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
error()   { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

show_help() {
    echo ""
    echo "go-fd Installer — https://github.com/startvibecoding/go-fd"
    echo ""
    echo "Usage: install.sh [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help        Show this help message"
    echo "  -u, --uninstall   Uninstall go-fd"
    echo "  -d, --dir DIR     Install to DIR (default: auto-detect)"
    echo ""
    echo "Environment variables:"
    echo "  INSTALL_DIR       Install directory (same as -d)"
    echo ""
    echo "Examples:"
    echo "  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | bash"
    echo "  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | bash -s -- -d ~/.local/bin"
    echo "  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | bash -s -- --uninstall"
    echo ""
}

detect_shell_config() {
    local shell_name
    shell_name="$(basename "${SHELL:-bash}")"
    case "$shell_name" in
        zsh)
            if [ -f "${HOME}/.zshenv" ]; then echo "${HOME}/.zshenv";
            elif [ -f "${HOME}/.zshrc" ]; then echo "${HOME}/.zshrc";
            else echo "${HOME}/.zshenv"; fi ;;
        bash)
            if [ "$(uname -s)" = "Darwin" ]; then
                if [ -f "${HOME}/.bash_profile" ]; then echo "${HOME}/.bash_profile";
                elif [ -f "${HOME}/.bashrc" ]; then echo "${HOME}/.bashrc";
                else echo "${HOME}/.bash_profile"; fi
            else
                if [ -f "${HOME}/.bashrc" ]; then echo "${HOME}/.bashrc";
                elif [ -f "${HOME}/.bash_profile" ]; then echo "${HOME}/.bash_profile";
                else echo "${HOME}/.bashrc"; fi
            fi ;;
        fish) echo "${HOME}/.config/fish/config.fish" ;;
        *)    echo "${HOME}/.profile" ;;
    esac
}

uninstall() {
    echo ""
    echo "go-fd Uninstaller"
    echo ""
    local found_paths=()
    for dir in "/usr/local/bin" "$USER_INSTALL_DIR" "$HOME/.local/bin"; do
        [ -f "$dir/$BINARY_NAME" ] && found_paths+=("$dir/$BINARY_NAME")
    done
    if command -v "$BINARY_NAME" &> /dev/null; then
        local which_path
        which_path=$(command -v "$BINARY_NAME" 2>/dev/null || true)
        if [ -n "$which_path" ] && [[ ! " ${found_paths[*]:-} " =~ " ${which_path} " ]]; then
            found_paths+=("$which_path")
        fi
    fi

    if [ ${#found_paths[@]} -eq 0 ]; then
        warn "go-fd not found in common locations"
        return 0
    fi

    info "Found go-fd installations:"
    for p in "${found_paths[@]}"; do echo "  - $p"; done
    echo ""
    local answer
    read -rp "Remove all installations? [y/N] " answer
    answer="${answer:-N}"
    [[ ! "$answer" =~ ^[Yy]$ ]] && { info "Uninstall cancelled"; return 0; }

    for p in "${found_paths[@]}"; do
        if [ -f "$p" ]; then
            if [ -w "$(dirname "$p")" ]; then
                rm -f "$p" && success "Removed: $p"
            else
                sudo rm -f "$p" && success "Removed: $p" || warn "Failed to remove: $p"
            fi
        fi
    done

    # Uninstall npm package if present
    if command -v npm &> /dev/null; then
        local npm_global
        npm_global=$(npm root -g 2>/dev/null || true)
        if [ -n "$npm_global" ] && { [ -d "$npm_global/go-fd" ] || [ -d "$npm_global/go-fd-installer" ]; }; then
            local pkg_to_uninstall="go-fd"
            if [ -d "$npm_global/go-fd-installer" ]; then
                pkg_to_uninstall="go-fd-installer"
            fi
            read -rp "Uninstall npm package ($pkg_to_uninstall)? [y/N] " answer
            answer="${answer:-N}"
            if [[ "$answer" =~ ^[Yy]$ ]]; then
                npm uninstall -g "$pkg_to_uninstall" && success "Uninstalled npm package" || warn "npm uninstall failed"
            fi
        fi
    fi

    echo ""
    success "Uninstall complete!"
}

detect_platform() {
    local os arch
    case "$(uname -s)" in
        Linux*)   os="linux" ;;
        Darwin*)  os="darwin" ;;
        FreeBSD*) os="freebsd" ;;
        CYGWIN*|MINGW*|MSYS*) os="windows" ;;
        *) error "Unsupported OS: $(uname -s)" ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64)    arch="amd64" ;;
        aarch64|arm64)   arch="arm64" ;;
        armv7l|armv6l|armhf|arm) arch="arm" ;;
        i386|i686|x86)   arch="386" ;;
        loongarch64)     arch="loong64" ;;
        riscv64)         arch="riscv64" ;;
        ppc64le)         arch="ppc64le" ;;
        s390x)           arch="s390x" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac
    echo "${os}/${arch}"
}

get_latest_version() {
    local version
    version=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    [ -z "$version" ] && error "Failed to fetch latest version from GitHub"
    echo "$version"
}

download() {
    local url="$1" dest="$2"
    info "Downloading: ${url}"
    if command -v curl &> /dev/null; then
        curl -sL -o "$dest" "$url"
    elif command -v wget &> /dev/null; then
        wget -qO "$dest" "$url"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

verify_checksum() {
    local file="$1" checksum_file="$2"
    [ ! -f "$checksum_file" ] && { warn "Checksum file not found, skipping verification"; return 0; }
    local expected
    expected=$(grep "$(basename "$file")" "$checksum_file" | awk '{print $1}' || true)
    [ -z "$expected" ] && { warn "Checksum not found for $(basename "$file")"; return 0; }
    local actual
    if command -v sha256sum &> /dev/null; then
        actual=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &> /dev/null; then
        actual=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        warn "No sha256sum/shasum found, skipping verification"; return 0
    fi
    [ "$actual" != "$expected" ] && error "Checksum mismatch: expected ${expected}, got ${actual}"
    success "Checksum verified"
}

add_to_path() {
    local config_file="$1"
    local config_dir
    config_dir="$(dirname "$config_file")"
    [ ! -d "$config_dir" ] && mkdir -p "$config_dir"
    [ ! -f "$config_file" ] && touch "$config_file"
    grep -q "\.go-fd/bin" "$config_file" 2>/dev/null && { info "PATH already configured in ${config_file}"; return 0; }

    local shell_name path_line
    shell_name="$(basename "${SHELL:-bash}")"
    case "$shell_name" in
        fish) path_line="set -gx PATH ${INSTALL_DIR} "'$PATH' ;;
        *)    path_line="export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
    esac
    {
        echo ""
        echo "# go-fd"
        echo "$path_line"
    } >> "$config_file"
    success "Added ${INSTALL_DIR} to PATH in ${config_file}"
}

check_path() {
    local config_file
    config_file=$(detect_shell_config)
    if echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
        return 0
    fi
    if [ "$INSTALL_DIR" = "$USER_INSTALL_DIR" ]; then
        local answer
        read -rp "Add ${INSTALL_DIR} to PATH automatically? [Y/n] " answer
        answer="${answer:-Y}"
        if [[ "$answer" =~ ^[Yy]$ ]]; then
            add_to_path "$config_file"
            warn "Run this to apply immediately: source ${config_file}"
        else
            warn "${INSTALL_DIR} is not in your PATH. Add it manually:"
            echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
        fi
    else
        warn "${INSTALL_DIR} is not in your PATH. Add it manually:"
        echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    fi
}

install_binary() {
    local binary_path="$1"
    if [ ! -d "$INSTALL_DIR" ]; then
        info "Creating install directory: ${INSTALL_DIR}"
        mkdir -p "$INSTALL_DIR" || error "Failed to create ${INSTALL_DIR}."
    fi
    mv "$binary_path" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    success "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
}

main_install() {
    echo ""
    echo "go-fd Installer — https://github.com/startvibecoding/go-fd"
    echo ""
    if [ "$INSTALL_DIR" = "$USER_INSTALL_DIR" ]; then
        info "Install mode: user-level (no root required)"
    else
        info "Install mode: system-level"
    fi
    info "Install directory: ${INSTALL_DIR}"

    local platform os arch
    platform=$(detect_platform)
    os="${platform%/*}"
    arch="${platform#*/}"
    info "Detected platform: ${os}/${arch}"

    local version version_num
    version=$(get_latest_version)
    info "Latest version: ${version}"
    version_num="${version#v}"

    local archive_name
    if [ "$os" = "windows" ]; then
        archive_name="${BINARY_NAME}-${version_num}-${os}-${arch}.zip"
    else
        archive_name="${BINARY_NAME}-${version_num}-${os}-${arch}.tar.gz"
    fi

    local download_url="https://github.com/${REPO}/releases/download/${version}/${archive_name}"
    local checksum_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"

    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf ${tmp_dir}" EXIT

    local archive_path="${tmp_dir}/${archive_name}"
    download "$download_url" "$archive_path"

    local checksum_path="${tmp_dir}/checksums.txt"
    download "$checksum_url" "$checksum_path" 2>/dev/null || true
    verify_checksum "$archive_path" "$checksum_path"

    info "Extracting archive..."
    local extract_dir="${tmp_dir}/extract"
    mkdir -p "$extract_dir"
    local binary_path
    if [ "$os" = "windows" ]; then
        command -v unzip &> /dev/null || error "unzip not found. Please install unzip."
        unzip -q "$archive_path" -d "$extract_dir"
        binary_path=$(find "$extract_dir" -name "${BINARY_NAME}.exe" | head -1)
    else
        tar -xzf "$archive_path" -C "$extract_dir"
        binary_path=$(find "$extract_dir" -name "${BINARY_NAME}" -type f | head -1)
    fi
    [ -z "$binary_path" ] || [ ! -f "$binary_path" ] && error "Binary not found in archive"

    install_binary "$binary_path"
    check_path

    echo ""
    success "Installation complete!"
    if command -v "$BINARY_NAME" &> /dev/null; then
        echo "  Version: $("$BINARY_NAME" --version 2>/dev/null || echo unknown)"
        echo "  Get started: ${BINARY_NAME} --help"
    fi
    echo ""
}

# Parse arguments
ACTION="install"
while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help) show_help; exit 0 ;;
        -u|--uninstall) ACTION="uninstall"; shift ;;
        -d|--dir)
            [ -z "${2:-}" ] && error "Option $1 requires an argument"
            INSTALL_DIR="$2"; shift 2 ;;
        *) error "Unknown option: $1 (use --help)" ;;
    esac
done

case "$ACTION" in
    install) main_install ;;
    uninstall) uninstall ;;
esac

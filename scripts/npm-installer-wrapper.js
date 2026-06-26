#!/usr/bin/env node

// Wrapper script that resolves and executes the platform-specific binary.
// When installed via `npm i -g go-fd`, this script finds the
// correct binary from the platform-specific optional dependency package.

const { execFileSync } = require('child_process');
const path = require('path');
const fs = require('fs');

// Map npm os/cpu (+libc on linux) to package name
const PLATFORM_MAP = {
  'linux-x64-glibc': 'go-fd-linux-x64',
  'linux-x64-musl': 'go-fd-linux-musl-x64',
  'linux-arm64-glibc': 'go-fd-linux-arm64',
  'linux-arm64-musl': 'go-fd-linux-musl-arm64',
  'linux-arm-glibc': 'go-fd-linux-arm',
  'linux-arm-musl': 'go-fd-linux-arm',
  'linux-ia32-glibc': 'go-fd-linux-ia32',
  'linux-loong64-glibc': 'go-fd-linux-loong64',
  'linux-riscv64-glibc': 'go-fd-linux-riscv64',
  'linux-ppc64-glibc': 'go-fd-linux-ppc64',
  'linux-s390x-glibc': 'go-fd-linux-s390x',
  'darwin-x64': 'go-fd-darwin-x64',
  'darwin-arm64': 'go-fd-darwin-arm64',
  'win32-x64': 'go-fd-win32-x64',
  'win32-arm64': 'go-fd-win32-arm64',
  'win32-ia32': 'go-fd-win32-ia32',
  'freebsd-x64': 'go-fd-freebsd-x64',
  'freebsd-arm64': 'go-fd-freebsd-arm64',
};

function detectPlatform() {
  const os = process.platform;   // 'linux', 'darwin', 'win32'
  const arch = process.arch;     // 'x64', 'arm64'

  if (os === 'linux') {
    // Detect libc: musl or glibc
    const isMusl = (() => {
      try {
        // Check for Alpine's musl
        if (fs.existsSync('/etc/alpine-release')) return true;
        // Check ldd output for musl
        const { execSync } = require('child_process');
        const output = execSync('ldd --version 2>&1 || true', { encoding: 'utf8' });
        return output.includes('musl');
      } catch {
        return false;
      }
    })();

    return `${os}-${arch}-${isMusl ? 'musl' : 'glibc'}`;
  }

  return `${os}-${arch}`;
}

function findBinary() {
  const platform = detectPlatform();
  const packageName = PLATFORM_MAP[platform];

  if (!packageName) {
    console.error(`Unsupported platform: ${platform}`);
    console.error(`Supported platforms: ${Object.keys(PLATFORM_MAP).join(', ')}`);
    process.exit(1);
  }

  const searchDirs = [];
  const addSearchDir = (dir) => {
    if (dir && !searchDirs.includes(dir)) {
      searchDirs.push(dir);
    }
  };

  try {
    addSearchDir(path.dirname(require.resolve(`${packageName}/package.json`)));
  } catch {
    // Keep explicit fallbacks below for unusual npm layouts.
  }

  // npm usually installs dependencies under this package. Some global installs
  // or package managers may hoist them as siblings, so check both layouts.
  addSearchDir(path.join(__dirname, '..', 'node_modules', packageName));
  addSearchDir(path.join(__dirname, '..', '..', packageName));

  for (const pkgDir of searchDirs) {
    const binName = process.platform === 'win32' ? 'fd.exe' : 'fd';
    const binPath = path.join(pkgDir, 'bin', binName);

    if (fs.existsSync(binPath)) {
      return binPath;
    }
  }

  // Fallback: check if there's a binary directly in the main package's bin/
  const fallbackBinName = (() => {
    const suffix = process.platform === 'win32' ? '.exe' : '';
    const osMap = { linux: 'linux', darwin: 'darwin', win32: 'windows', freebsd: 'freebsd' };
    const archMap = {
      x64: 'amd64', arm64: 'arm64', arm: 'arm', ia32: '386',
      loong64: 'loong64', riscv64: 'riscv64', ppc64: 'ppc64le', s390x: 's390x',
    };
    return `fd-${osMap[process.platform]}-${archMap[process.arch]}${suffix}`;
  })();

  const fallbackPath = path.join(__dirname, fallbackBinName);
  if (fs.existsSync(fallbackPath)) {
    return fallbackPath;
  }

  console.error(`Could not find go-fd binary for platform: ${detectPlatform()}`);
  console.error(`Searched for package: ${packageName}`);
  console.error(`Searched in: ${searchDirs.join(', ')}`);
  console.error('');
  console.error('If you installed globally, try reinstalling:');
  console.error('  npm install -g go-fd');
  console.error('');
  console.error('If the problem persists, install via the one-line script instead:');
  console.error('  curl -fsSL https://raw.githubusercontent.com/startvibecoding/go-fd/main/install.sh | bash');
  process.exit(1);
}

// Main
const binaryPath = findBinary();
const args = process.argv.slice(2);

try {
  execFileSync(binaryPath, args, { stdio: 'inherit' });
} catch (err) {
  // Forward the exit code
  if (err.status !== undefined) {
    process.exit(err.status);
  }
  if (err.code) {
    process.exit(1);
  }
  process.exit(1);
}

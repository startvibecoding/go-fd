#!/usr/bin/env node

// Skip postinstall output in CI or when suppressed
if (process.env.CI || process.env.npm_config_yes || process.env.GO_FD_SKIP_POSTINSTALL) {
  process.exit(0);
}

const RESET  = '\x1b[0m';
const BOLD   = '\x1b[1m';
const DIM    = '\x1b[2m';
const CYAN   = '\x1b[36m';
const BRIGHT_CYAN = '\x1b[96m';
const WHITE  = '\x1b[97m';

const logo = [
  '   __        ______',
  '  / /  ___  / __/ /',
  ' / _ \\/ _ \\/ _// _ \\',
  '/_.__/\\___/_/ /_//_/',
].join('\n');

function pkgVersion() {
  try {
    return require('../package.json').version;
  } catch {
    return '';
  }
}

const ver = pkgVersion();
const verStr = ver ? ` ${DIM}v${ver}${RESET}` : '';

console.log();
console.log(`${BRIGHT_CYAN}${BOLD}${logo}${RESET}${verStr}`);
console.log();
console.log(`  ${BOLD}${WHITE}A simple, fast and user-friendly alternative to find.${RESET}`);
console.log();
console.log(`  ${DIM}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}`);
console.log();
console.log(`  ${BOLD}Quick Start${RESET}`);
console.log();
console.log(`    fd "pattern"                   ${DIM}Search files recursively${RESET}`);
console.log(`    fd -e go "main"               ${DIM}Filter by extension${RESET}`);
console.log(`    fd -t d "docs"                ${DIM}Search directories only${RESET}`);
console.log(`    fd -H -I "config"             ${DIM}Include hidden + ignored${RESET}`);
console.log(`    fd -e jpg -x convert {} {.}.png  ${DIM}Run a command per result${RESET}`);
console.log();
console.log(`  ${BOLD}Docs${RESET}   ${CYAN}https://github.com/startvibecoding/go-fd${RESET}`);
console.log(`  ${BOLD}Code${RESET}   ${CYAN}https://github.com/startvibecoding/go-fd${RESET}`);
console.log();
console.log(`  ${DIM}Run 'fd --help' for the full option list.${RESET}`);
console.log();

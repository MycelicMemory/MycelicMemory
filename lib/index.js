#!/usr/bin/env node

/**
 * Ultrathink - AI-powered persistent memory system
 *
 * This module provides the main entry point for the ultrathink CLI.
 * It locates and executes the platform-specific binary, downloading it if necessary.
 */

const { spawn } = require('child_process');
const path = require('path');
const os = require('os');
const fs = require('fs');

const { getBinaryName, getBinDir, install } = require('./install');

/**
 * Get the full path to the binary for this platform
 */
function getBinaryPath() {
  const binaryName = getBinaryName();
  const binDir = getBinDir();
  return path.join(binDir, binaryName);
}

/**
 * Check if the binary exists
 */
function binaryExists() {
  const binaryPath = getBinaryPath();
  return fs.existsSync(binaryPath);
}

/**
 * Run ultrathink with the given arguments
 */
async function run(args = process.argv.slice(2)) {
  // Ensure binary exists (download if needed)
  if (!binaryExists()) {
    console.log('Binary not found, downloading...');
    await install();
  }

  const binaryPath = getBinaryPath();

  return new Promise((resolve, reject) => {
    const child = spawn(binaryPath, args, {
      stdio: 'inherit',
      cwd: process.cwd(),
      env: process.env
    });

    child.on('exit', (code, signal) => {
      if (signal) {
        process.kill(process.pid, signal);
      } else {
        resolve(code || 0);
      }
    });

    child.on('error', (error) => {
      reject(error);
    });

    // Forward signals to child
    ['SIGINT', 'SIGTERM', 'SIGHUP'].forEach((signal) => {
      process.on(signal, () => {
        if (child.pid) {
          child.kill(signal);
        }
      });
    });
  });
}

/**
 * Main entry point
 */
async function main() {
  try {
    const code = await run();
    process.exit(code);
  } catch (error) {
    console.error(`Error: ${error.message}`);
    console.error('');
    console.error('Troubleshooting:');
    console.error('1. Run: npm uninstall -g ultrathink && npm install -g ultrathink');
    console.error('2. On macOS: Allow binary in System Preferences > Security');
    console.error('3. Report issues: https://github.com/MycelicMemory/ultrathink/issues');
    process.exit(1);
  }
}

// Export for programmatic use
module.exports = {
  run,
  getBinaryPath,
  binaryExists
};

// Run if executed directly
if (require.main === module) {
  main();
}

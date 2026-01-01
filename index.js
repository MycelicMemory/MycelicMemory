#!/usr/bin/env node

const { spawn } = require('child_process');
const os = require('os');
const path = require('path');
const fs = require('fs');

/**
 * Get the binary name for the current platform
 * @returns {string} The binary filename
 */
function getBinaryName() {
  const platform = os.platform();
  const arch = os.arch();

  switch (platform) {
    case 'darwin':
      return arch === 'arm64' ? 'ultrathink-macos-arm64' : 'ultrathink-macos-x64';
    case 'linux':
      return arch === 'arm64' ? 'ultrathink-linux-arm64' : 'ultrathink-linux-x64';
    case 'win32':
      return arch === 'arm64' ? 'ultrathink-windows-arm64.exe' : 'ultrathink-windows-x64.exe';
    default:
      throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }
}

/**
 * Get the full path to the binary
 * @returns {string} The full binary path
 */
function getBinaryPath() {
  const binaryName = getBinaryName();
  const binDir = path.join(__dirname, 'bin');
  const binaryPath = path.join(binDir, binaryName);

  // Also check for generic name
  const genericPath = path.join(binDir, 'ultrathink');

  if (fs.existsSync(binaryPath)) {
    return binaryPath;
  }

  if (fs.existsSync(genericPath)) {
    return genericPath;
  }

  throw new Error(`Binary not found. Expected at: ${binaryPath}`);
}

/**
 * Main entry point
 */
function main() {
  try {
    const binaryPath = getBinaryPath();
    const args = process.argv.slice(2);

    const child = spawn(binaryPath, args, {
      stdio: 'inherit',
      cwd: process.cwd(),
      env: process.env
    });

    child.on('exit', (code, signal) => {
      if (signal) {
        process.kill(process.pid, signal);
      } else {
        process.exit(code || 0);
      }
    });

    child.on('error', (error) => {
      console.error(`Error executing ultrathink: ${error.message}`);
      console.error('');
      console.error('Troubleshooting steps:');
      console.error('1. Try reinstalling: npm uninstall -g ultrathink && npm install -g github:MycelicMemory/ultrathink');
      console.error('2. Check that the binary was downloaded correctly');
      console.error('3. On macOS, you may need to allow the binary in System Preferences > Security');
      process.exit(1);
    });

    // Handle parent process signals
    ['SIGINT', 'SIGTERM', 'SIGHUP'].forEach((signal) => {
      process.on(signal, () => {
        if (child.pid) {
          child.kill(signal);
        }
      });
    });

  } catch (error) {
    console.error(`Error: ${error.message}`);
    console.error('');
    console.error('The ultrathink binary was not found. This usually means the');
    console.error('post-install script failed to download it.');
    console.error('');
    console.error('Try reinstalling: npm uninstall -g ultrathink && npm install -g github:MycelicMemory/ultrathink');
    process.exit(1);
  }
}

// Export for programmatic use
module.exports = {
  getBinaryPath,
  getBinaryName
};

// Run if executed directly
if (require.main === module) {
  main();
}

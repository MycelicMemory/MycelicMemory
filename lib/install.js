#!/usr/bin/env node

/**
 * Ultrathink postinstall script
 * Downloads the pre-built binary for the current platform from GitHub releases
 */

const https = require('https');
const http = require('http');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');
const { URL } = require('url');

const GITHUB_OWNER = 'MycelicMemory';
const GITHUB_REPO = 'ultrathink';
const VERSION = require('../package.json').version;

/**
 * Get the binary filename for the current platform
 */
function getBinaryName() {
  const platform = os.platform();
  const arch = os.arch();

  const names = {
    'darwin-arm64': 'ultrathink-macos-arm64',
    'darwin-x64': 'ultrathink-macos-x64',
    'linux-arm64': 'ultrathink-linux-arm64',
    'linux-x64': 'ultrathink-linux-x64',
    'win32-x64': 'ultrathink-windows-x64.exe',
    'win32-arm64': 'ultrathink-windows-x64.exe', // Use x64 binary for Windows ARM (emulation)
  };

  const key = `${platform}-${arch}`;
  const name = names[key];

  if (!name) {
    console.error(`Unsupported platform: ${key}`);
    console.error('Supported: darwin-arm64, darwin-x64, linux-arm64, linux-x64, win32-x64');
    process.exit(1);
  }

  return name;
}

/**
 * Get the bin directory where binaries should be stored
 */
function getBinDir() {
  return path.join(__dirname, '..', 'bin');
}

/**
 * Download a file with redirect support
 */
function downloadFile(url, dest, maxRedirects = 10) {
  return new Promise((resolve, reject) => {
    if (maxRedirects <= 0) {
      reject(new Error('Too many redirects'));
      return;
    }

    const parsedUrl = new URL(url);
    const protocol = parsedUrl.protocol === 'https:' ? https : http;

    const request = protocol.get(url, {
      headers: {
        'User-Agent': `ultrathink-installer/${VERSION}`,
        'Accept': 'application/octet-stream'
      }
    }, (response) => {
      // Handle redirects (GitHub releases use 302 redirects)
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        const redirectUrl = new URL(response.headers.location, url).toString();
        downloadFile(redirectUrl, dest, maxRedirects - 1)
          .then(resolve)
          .catch(reject);
        return;
      }

      if (response.statusCode !== 200) {
        reject(new Error(`HTTP ${response.statusCode}: ${response.statusMessage}`));
        return;
      }

      const file = fs.createWriteStream(dest);
      let downloadedBytes = 0;
      const totalBytes = parseInt(response.headers['content-length'], 10) || 0;

      response.on('data', (chunk) => {
        downloadedBytes += chunk.length;
        if (totalBytes > 0) {
          const percent = Math.round((downloadedBytes / totalBytes) * 100);
          process.stdout.write(`\rDownloading... ${percent}%`);
        }
      });

      response.pipe(file);

      file.on('finish', () => {
        file.close();
        process.stdout.write('\n');
        resolve();
      });

      file.on('error', (err) => {
        fs.unlink(dest, () => {});
        reject(err);
      });
    });

    request.on('error', (err) => {
      fs.unlink(dest, () => {});
      reject(err);
    });

    request.setTimeout(120000, () => {
      request.destroy();
      reject(new Error('Download timeout (120s)'));
    });
  });
}

/**
 * Verify the downloaded binary works
 */
function verifyBinary(binaryPath) {
  const stats = fs.statSync(binaryPath);
  if (stats.size < 1000000) { // Binary should be at least 1MB
    throw new Error(`Binary too small (${stats.size} bytes), download may have failed`);
  }

  // Set executable permissions on Unix
  if (os.platform() !== 'win32') {
    fs.chmodSync(binaryPath, 0o755);
  }

  // Try to run --version to verify it works
  try {
    const result = execSync(`"${binaryPath}" --version`, {
      encoding: 'utf8',
      timeout: 10000,
      stdio: ['pipe', 'pipe', 'pipe']
    });
    return result.trim();
  } catch (err) {
    // On macOS, unsigned binaries may be blocked
    if (os.platform() === 'darwin' && err.message.includes('killed')) {
      console.log('\nNote: On macOS, you may need to allow the binary in:');
      console.log('System Preferences > Security & Privacy > General');
    }
    throw new Error(`Binary verification failed: ${err.message}`);
  }
}

/**
 * Main installation function
 */
async function install() {
  console.log('Ultrathink v' + VERSION);
  console.log(`Platform: ${os.platform()}-${os.arch()}`);
  console.log('');

  const binaryName = getBinaryName();
  const binDir = getBinDir();
  const binaryPath = path.join(binDir, binaryName);

  // Create bin directory if needed
  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  // Check if binary already exists and is valid
  if (fs.existsSync(binaryPath)) {
    try {
      const version = verifyBinary(binaryPath);
      console.log(`Binary already installed (${version})`);
      return;
    } catch (err) {
      console.log('Existing binary invalid, re-downloading...');
      try { fs.unlinkSync(binaryPath); } catch {}
    }
  }

  // Download URLs to try (version-specific first, then latest)
  const urls = [
    `https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/v${VERSION}/${binaryName}`,
    `https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest/download/${binaryName}`
  ];

  let lastError = null;
  for (const url of urls) {
    console.log(`Downloading from GitHub releases...`);
    try {
      await downloadFile(url, binaryPath);
      const version = verifyBinary(binaryPath);
      console.log(`Successfully installed ultrathink ${version}`);
      return;
    } catch (err) {
      lastError = err;
      console.log(`Failed: ${err.message}`);
      try { fs.unlinkSync(binaryPath); } catch {}
    }
  }

  // All downloads failed
  console.error('');
  console.error('Failed to download ultrathink binary.');
  console.error('');
  console.error('Manual installation:');
  console.error(`1. Download ${binaryName} from:`);
  console.error(`   https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest`);
  console.error(`2. Place it in: ${binDir}`);
  console.error('');
  console.error('Or build from source:');
  console.error(`   git clone https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}`);
  console.error('   cd ultrathink && make deps && make build && make install');
  console.error('');

  if (lastError) {
    console.error(`Last error: ${lastError.message}`);
  }

  process.exit(1);
}

// Run if executed directly
if (require.main === module) {
  install().catch((err) => {
    console.error(`Installation failed: ${err.message}`);
    process.exit(1);
  });
}

module.exports = { install, getBinaryName, getBinDir };

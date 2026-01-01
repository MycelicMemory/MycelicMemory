#!/usr/bin/env node

const https = require('https');
const http = require('http');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');
const { URL } = require('url');

// Configuration
const GITHUB_OWNER = 'MycelicMemory';
const GITHUB_REPO = 'ultrathink';
const VERSION = require('../package.json').version;

// Download sources (tried in order)
const DOWNLOAD_SOURCES = [
  `https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/v${VERSION}`,
  `https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/latest/download`
];

/**
 * Get the binary name for the current platform
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
 * Generate all possible download URLs
 */
function generateDownloadUrls(binaryName) {
  const urls = [];
  for (const base of DOWNLOAD_SOURCES) {
    urls.push(`${base}/${binaryName}`);
  }
  return urls;
}

/**
 * Download a file with redirect support
 */
function downloadFile(url, dest, maxRedirects = 5) {
  return new Promise((resolve, reject) => {
    if (maxRedirects <= 0) {
      reject(new Error('Too many redirects'));
      return;
    }

    const parsedUrl = new URL(url);
    const protocol = parsedUrl.protocol === 'https:' ? https : http;

    const request = protocol.get(url, {
      headers: {
        'User-Agent': `mycelic-memory-installer/${VERSION}`
      }
    }, (response) => {
      // Handle redirects
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

      response.pipe(file);

      file.on('finish', () => {
        file.close();
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

    request.setTimeout(60000, () => {
      request.destroy();
      reject(new Error('Download timeout'));
    });
  });
}

/**
 * Verify the downloaded binary
 */
function verifyBinary(binaryPath) {
  try {
    // Check file exists and has content
    const stats = fs.statSync(binaryPath);
    if (stats.size < 1000) {
      throw new Error('Binary file too small');
    }

    // Set executable permissions on Unix
    if (os.platform() !== 'win32') {
      fs.chmodSync(binaryPath, 0o755);
    }

    // Try to run --version
    try {
      const result = execSync(`"${binaryPath}" --version`, {
        encoding: 'utf8',
        timeout: 5000
      });
      return result.trim();
    } catch (err) {
      // Version check failed, but binary might still work
      console.warn('Version check failed, but binary exists');
      return 'unknown';
    }
  } catch (err) {
    throw new Error(`Binary verification failed: ${err.message}`);
  }
}

/**
 * Main installation function
 */
async function install() {
  console.log('Ultrathink post-install script');
  console.log(`Platform: ${os.platform()}-${os.arch()}`);
  console.log(`Version: ${VERSION}`);
  console.log('');

  const binaryName = getBinaryName();
  const binDir = path.join(__dirname, '..', 'bin');
  const binaryPath = path.join(binDir, binaryName);

  // Create bin directory
  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  // Check if binary already exists and is valid
  if (fs.existsSync(binaryPath)) {
    try {
      const version = verifyBinary(binaryPath);
      console.log(`Binary already exists and is valid (version: ${version})`);
      return;
    } catch (err) {
      console.log('Existing binary is invalid, re-downloading...');
      fs.unlinkSync(binaryPath);
    }
  }

  // Generate download URLs
  const urls = generateDownloadUrls(binaryName);

  // Try each URL
  for (const url of urls) {
    console.log(`Downloading from: ${url}`);
    try {
      await downloadFile(url, binaryPath);

      // Verify the download
      const version = verifyBinary(binaryPath);
      console.log(`Successfully installed ultrathink v${version}`);
      return;
    } catch (err) {
      console.log(`Failed: ${err.message}`);
      // Clean up partial download
      if (fs.existsSync(binaryPath)) {
        fs.unlinkSync(binaryPath);
      }
    }
  }

  // All URLs failed
  console.error('');
  console.error('Failed to download ultrathink binary.');
  console.error('');
  console.error('Manual installation options:');
  console.error(`1. Download from: https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases`);
  console.error(`2. Place the binary in: ${binDir}`);
  console.error(`3. Name it: ${binaryName}`);
  console.error('');
  console.error('If you continue to have issues, please report at:');
  console.error(`https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/issues`);

  // Exit with error
  process.exit(1);
}

// Run installation
install().catch((err) => {
  console.error(`Installation failed: ${err.message}`);
  process.exit(1);
});

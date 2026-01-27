const { execSync, exec } = require('child_process');
const os = require('os');
const https = require('https');

/**
 * Check if a command exists and get its version
 */
function checkCommand(command, versionFlag = '--version') {
  try {
    const result = execSync(`${command} ${versionFlag}`, {
      encoding: 'utf8',
      timeout: 10000,
      stdio: ['pipe', 'pipe', 'pipe']
    });
    return { installed: true, version: result.trim().split('\n')[0] };
  } catch {
    return { installed: false, version: null };
  }
}

/**
 * Check if a service is running on a port
 */
function checkPort(port) {
  return new Promise((resolve) => {
    const http = require('http');
    const req = http.get(`http://localhost:${port}`, (res) => {
      resolve(true);
    });
    req.on('error', () => resolve(false));
    req.setTimeout(2000, () => {
      req.destroy();
      resolve(false);
    });
  });
}

/**
 * Check all dependencies
 */
async function checkDependencies() {
  const results = {
    node: { installed: false, version: null, required: true },
    ultrathink: { installed: false, version: null, required: true },
    ollama: { installed: false, version: null, running: false, required: false },
    qdrant: { installed: false, running: false, required: false },
    docker: { installed: false, version: null, required: false }
  };

  // Check Node.js
  results.node = {
    ...checkCommand('node', '--version'),
    required: true
  };

  // Check ultrathink
  results.ultrathink = {
    ...checkCommand('ultrathink', '--version'),
    required: true
  };

  // Check Ollama
  const ollamaCheck = checkCommand('ollama', '--version');
  results.ollama = {
    ...ollamaCheck,
    running: ollamaCheck.installed ? await checkPort(11434) : false,
    required: false
  };

  // Check Docker
  results.docker = {
    ...checkCommand('docker', '--version'),
    required: false
  };

  // Check Qdrant (via Docker or direct)
  results.qdrant = {
    installed: results.docker.installed,
    running: await checkPort(6333),
    required: false
  };

  return results;
}

/**
 * Install ultrathink via npm
 */
function installUltrathink(onProgress) {
  return new Promise((resolve, reject) => {
    onProgress({ status: 'starting', message: 'Starting installation...' });

    const child = exec('npm install -g ultrathink', { timeout: 300000 });
    let output = '';

    child.stdout.on('data', (data) => {
      output += data;
      onProgress({ status: 'installing', message: data.toString() });
    });

    child.stderr.on('data', (data) => {
      output += data;
      onProgress({ status: 'installing', message: data.toString() });
    });

    child.on('close', (code) => {
      if (code === 0) {
        onProgress({ status: 'complete', message: 'Installation complete!' });
        resolve({ success: true, output });
      } else {
        onProgress({ status: 'error', message: 'Installation failed' });
        reject(new Error(`Installation failed with code ${code}`));
      }
    });

    child.on('error', (err) => {
      onProgress({ status: 'error', message: err.message });
      reject(err);
    });
  });
}

/**
 * Get download URLs for dependencies
 */
function getDependencyUrls() {
  const platform = os.platform();

  return {
    ollama: {
      name: 'Ollama',
      description: 'Required for AI-powered semantic search and analysis',
      url: platform === 'darwin'
        ? 'https://ollama.ai/download/mac'
        : 'https://ollama.ai/download/windows',
      instructions: [
        'Download and install Ollama',
        'Run: ollama serve',
        'Run: ollama pull nomic-embed-text',
        'Run: ollama pull qwen2.5:3b'
      ]
    },
    qdrant: {
      name: 'Qdrant (via Docker)',
      description: 'Optional: High-performance vector search for large collections',
      url: 'https://www.docker.com/products/docker-desktop/',
      instructions: [
        'Install Docker Desktop',
        'Run: docker run -d -p 6333:6333 qdrant/qdrant'
      ]
    },
    node: {
      name: 'Node.js',
      description: 'Required runtime for ultrathink',
      url: 'https://nodejs.org/en/download/',
      instructions: [
        'Download and install Node.js LTS',
        'Restart this installer after installation'
      ]
    }
  };
}

module.exports = {
  checkDependencies,
  installUltrathink,
  getDependencyUrls
};

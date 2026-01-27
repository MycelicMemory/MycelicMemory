// State
let dependencies = null;
let installComplete = false;

// Screen navigation
function goToScreen(screenId) {
  document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
  document.getElementById(`screen-${screenId}`).classList.add('active');

  if (screenId === 'check') {
    checkDependencies();
  }
}

// Check dependencies
async function checkDependencies() {
  const deps = await window.mycelicmemory.checkDependencies();
  dependencies = deps;
  updateDependencyUI(deps);
}

function updateDependencyUI(deps) {
  // Node.js
  updateDepItem('node', deps.node.installed, deps.node.version || 'Not installed');

  // MyclicMemory
  updateDepItem('mycelicmemory', deps.mycelicmemory.installed, deps.mycelicmemory.version || 'Not installed');

  // Ollama
  const ollamaStatus = deps.ollama.installed
    ? (deps.ollama.running ? `Running (${deps.ollama.version})` : 'Installed (not running)')
    : 'Not installed';
  updateDepItem('ollama', deps.ollama.installed && deps.ollama.running, ollamaStatus, true);

  // Qdrant
  const qdrantStatus = deps.qdrant.running ? 'Running' : 'Not running';
  updateDepItem('qdrant', deps.qdrant.running, qdrantStatus, true);

  // Enable continue button if Node.js is installed
  const canContinue = deps.node.installed;
  document.getElementById('btn-continue').disabled = !canContinue;
}

function updateDepItem(dep, success, status, isOptional = false) {
  const item = document.querySelector(`.dep-item[data-dep="${dep}"]`);
  if (!item) return;

  item.classList.remove('success', 'error', 'warning');

  if (success) {
    item.classList.add('success');
    item.querySelector('.dep-icon').textContent = '✅';
  } else if (isOptional) {
    item.classList.add('warning');
    item.querySelector('.dep-icon').textContent = '⚠️';
  } else {
    item.classList.add('error');
    item.querySelector('.dep-icon').textContent = '❌';
  }

  item.querySelector('.dep-status').textContent = status;
}

// Install mycelicmemory
async function startInstall() {
  // Check if already installed
  if (dependencies && dependencies.mycelicmemory.installed) {
    goToScreen('optional');
    return;
  }

  const btnInstall = document.getElementById('btn-install');
  const btnBack = document.getElementById('btn-back-install');
  const spinner = document.querySelector('.spinner');
  const message = document.getElementById('install-message');
  const output = document.getElementById('install-output');
  const progressBar = document.getElementById('progress-bar');

  btnInstall.disabled = true;
  btnBack.disabled = true;
  spinner.classList.add('active');
  output.classList.add('active');
  output.textContent = '';
  message.textContent = 'Installing mycelicmemory...';
  progressBar.style.width = '10%';

  // Listen for progress
  window.mycelicmemory.onInstallProgress((progress) => {
    if (progress.message) {
      output.textContent += progress.message;
      output.scrollTop = output.scrollHeight;
    }

    if (progress.status === 'installing') {
      progressBar.style.width = '50%';
    } else if (progress.status === 'complete') {
      progressBar.style.width = '100%';
      message.textContent = 'Installation complete!';
      spinner.classList.remove('active');
      installComplete = true;

      setTimeout(() => {
        goToScreen('optional');
      }, 1500);
    } else if (progress.status === 'error') {
      message.textContent = 'Installation failed. Please try again.';
      spinner.classList.remove('active');
      btnInstall.disabled = false;
      btnBack.disabled = false;
    }
  });

  try {
    await window.mycelicmemory.installMyclicMemory();
  } catch (error) {
    message.textContent = `Error: ${error.message}`;
    spinner.classList.remove('active');
    btnInstall.disabled = false;
    btnBack.disabled = false;
  }
}

// Open external links
function openLink(type) {
  const urls = {
    ollama: navigator.platform.includes('Mac')
      ? 'https://ollama.ai/download/mac'
      : 'https://ollama.ai/download/windows',
    docker: 'https://www.docker.com/products/docker-desktop/',
    docs: 'https://github.com/MycelicMemory/mycelicmemory'
  };

  window.mycelicmemory.openExternal(urls[type]);
}

function openDocs() {
  openLink('docs');
}

// Finish installation
function finishInstall() {
  window.close();
}

// Initialize
document.addEventListener('DOMContentLoaded', () => {
  // Check if mycelicmemory is already installed on load
  if (dependencies && dependencies.mycelicmemory.installed) {
    document.getElementById('btn-install').textContent = 'Already Installed';
  }
});

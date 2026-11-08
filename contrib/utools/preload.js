// uTools preload: locate lingbox and run the launcher query contract.
//
// The renderer only sees plain JSON, so the same contract is shared with
// the Alfred adapter and can be tested from Node (see test.js).

const { execFile } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');

function binaryCandidates() {
  const exe = process.platform === 'win32' ? 'lingbox.exe' : 'lingbox';
  const home = os.homedir();
  const list = [];

  if (process.env.LINGBOX_BIN) {
    list.push(process.env.LINGBOX_BIN);
  }
  if (process.platform === 'darwin') {
    list.push('/opt/homebrew/bin/' + exe);
    list.push('/usr/local/bin/' + exe);
    list.push('/opt/local/bin/' + exe);
  } else if (process.platform === 'win32') {
    list.push(path.join(home, 'scoop', 'shims', exe));
    list.push('C:\\ProgramData\\chocolatey\\bin\\' + exe);
  }
  list.push(path.join(home, '.local', 'bin', exe));
  list.push(path.join(home, 'go', 'bin', exe));
  list.push(path.join(home, 'bin', exe));
  list.push(exe); // let the OS resolve PATH

  return list;
}

function findLingbox() {
  const mode = process.platform === 'win32' ? fs.constants.F_OK : fs.constants.X_OK;
  for (const candidate of binaryCandidates()) {
    if (!candidate) {
      continue;
    }
    if (candidate.includes(path.sep) || candidate.includes('/') || candidate.includes('\\')) {
      try {
        fs.accessSync(candidate, mode);
        return candidate;
      } catch (_) {
        // try the next location
      }
    } else {
      return candidate;
    }
  }
  return null;
}

function queryOnce(query) {
  return new Promise((resolve) => {
    const lb = findLingbox();
    if (!lb) {
      resolve({ error: 'ling-box not found', items: [] });
      return;
    }
    execFile(lb, ['launcher', 'query', '--query=' + query], {
      timeout: 10000,
      windowsHide: true,
      maxBuffer: 4 * 1024 * 1024,
    }, (err, stdout, stderr) => {
      if (err) {
        resolve({ error: (stderr || err.message || '').trim(), items: [] });
        return;
      }
      try {
        resolve(JSON.parse(stdout));
      } catch (_) {
        resolve({ error: 'invalid output from lingbox', items: [] });
      }
    });
  });
}

const services = {
  queryOnce,
  findLingbox,
  copyText: (text) => {
    if (typeof utools !== 'undefined') {
      utools.copyText(text);
    }
  },
  outPlugin: () => {
    if (typeof utools !== 'undefined') {
      utools.outPlugin();
    }
  },
};

if (typeof window !== 'undefined') {
  window.services = services;
}

module.exports = services;

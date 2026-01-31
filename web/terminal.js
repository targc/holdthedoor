let term = null;
let ws = null;
let fitAddon = null;
let currentVM = null;
let countdownInterval = null;

// Auth
function getToken() {
    return localStorage.getItem('token');
}

function getExpires() {
    return parseInt(localStorage.getItem('expires') || '0', 10);
}

function setAuth(token, expires) {
    localStorage.setItem('token', token);
    localStorage.setItem('expires', expires.toString());
}

function clearAuth() {
    localStorage.removeItem('token');
    localStorage.removeItem('expires');
}

function isLoggedIn() {
    const token = getToken();
    const expires = getExpires();
    return token && expires > Date.now() / 1000;
}

// UI State
function showLogin() {
    document.getElementById('login-container').style.display = 'flex';
    document.getElementById('app').style.display = 'none';
    stopCountdown();
}

function showApp() {
    document.getElementById('login-container').style.display = 'none';
    document.getElementById('app').style.display = 'flex';
    startCountdown();
    loadVMs();
}

// Login
document.getElementById('login-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;
    const errorEl = document.getElementById('login-error');

    try {
        const res = await fetch('/api/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });

        const data = await res.json();

        if (!res.ok) {
            errorEl.textContent = data.error || 'Login failed';
            errorEl.style.display = 'block';
            return;
        }

        setAuth(data.token, data.expires);
        errorEl.style.display = 'none';
        showApp();
    } catch (err) {
        errorEl.textContent = 'Connection error';
        errorEl.style.display = 'block';
    }
});

// Logout
document.getElementById('logout-btn').addEventListener('click', () => {
    logout();
});

function logout() {
    clearAuth();
    if (ws) {
        ws.close();
        ws = null;
    }
    if (term) {
        term.dispose();
        term = null;
    }
    currentVM = null;
    showLogin();
}

// Countdown
function startCountdown() {
    updateCountdown();
    countdownInterval = setInterval(updateCountdown, 1000);
}

function stopCountdown() {
    if (countdownInterval) {
        clearInterval(countdownInterval);
        countdownInterval = null;
    }
}

function updateCountdown() {
    const expires = getExpires();
    const now = Math.floor(Date.now() / 1000);
    const remaining = expires - now;

    if (remaining <= 0) {
        logout();
        return;
    }

    const minutes = Math.floor(remaining / 60);
    const seconds = remaining % 60;
    document.getElementById('countdown').textContent =
        `Session: ${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
}

// API calls with auth
async function fetchWithAuth(url, options = {}) {
    const token = getToken();
    const res = await fetch(url, {
        ...options,
        headers: {
            ...options.headers,
            'Authorization': `Bearer ${token}`
        }
    });

    if (res.status === 401) {
        logout();
        throw new Error('Unauthorized');
    }

    return res;
}

// VMs
async function loadVMs() {
    if (!isLoggedIn()) return;

    try {
        const res = await fetchWithAuth('/api/vms');
        const data = await res.json();
        renderVMList(data.vms || []);
    } catch (err) {
        if (err.message !== 'Unauthorized') {
            console.error('Failed to load VMs:', err);
            document.getElementById('vm-list').innerHTML = '<li class="no-vms">Failed to load VMs</li>';
        }
    }
}

function renderVMList(vms) {
    const list = document.getElementById('vm-list');
    if (vms.length === 0) {
        list.innerHTML = '<li class="no-vms">No VMs connected</li>';
        return;
    }

    list.innerHTML = vms.map(vm => `
        <li class="vm-item" data-id="${vm.id}" onclick="connectVM('${vm.id}', '${vm.name}')">
            <div class="hostname">${vm.name}</div>
            <div class="info">${vm.ip} · ${vm.os}</div>
        </li>
    `).join('');
}

function connectVM(id, hostname) {
    if (currentVM === id) return;
    if (!isLoggedIn()) return;

    // Update UI
    document.querySelectorAll('.vm-item').forEach(el => el.classList.remove('active'));
    document.querySelector(`[data-id="${id}"]`)?.classList.add('active');
    document.getElementById('status').innerHTML = `<span class="connecting">Connecting to ${hostname}...</span>`;

    // Close existing connection
    if (ws) {
        ws.close();
        ws = null;
    }

    // Clear terminal
    if (term) {
        term.dispose();
    }

    // Create new terminal
    term = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: 'Menlo, Monaco, "Courier New", monospace',
        theme: {
            background: '#1a1a2e',
            foreground: '#eee',
            cursor: '#e94560',
        }
    });

    fitAddon = new FitAddon.FitAddon();
    term.loadAddon(fitAddon);
    term.open(document.getElementById('terminal'));
    fitAddon.fit();

    // Connect WebSocket with JWT token
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const token = getToken();
    ws = new WebSocket(`${protocol}//${location.host}/ws/terminal/${id}?token=${token}`);

    ws.onopen = () => {
        currentVM = id;
        document.getElementById('status').textContent = `Connected to ${hostname}`;

        // Send initial size
        ws.send(JSON.stringify({
            type: 'resize',
            cols: term.cols,
            rows: term.rows
        }));
    };

    ws.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        if (msg.type === 'output') {
            term.write(msg.data);
        } else if (msg.type === 'error') {
            term.write(`\r\n\x1b[31mError: ${msg.data}\x1b[0m\r\n`);
        }
    };

    ws.onclose = () => {
        document.getElementById('status').textContent = `Disconnected from ${hostname}`;
        currentVM = null;
    };

    ws.onerror = (err) => {
        console.error('WebSocket error:', err);
        document.getElementById('status').textContent = `Connection error`;
    };

    // Send input to server
    term.onData(data => {
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'input', data }));
        }
    });

    // Handle resize
    window.addEventListener('resize', () => {
        if (fitAddon && term && ws && ws.readyState === WebSocket.OPEN) {
            fitAddon.fit();
            ws.send(JSON.stringify({
                type: 'resize',
                cols: term.cols,
                rows: term.rows
            }));
        }
    });
}

// Init
if (isLoggedIn()) {
    showApp();
} else {
    showLogin();
}

// Refresh VM list periodically
setInterval(() => {
    if (isLoggedIn()) {
        loadVMs();
    }
}, 5000);

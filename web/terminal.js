let term = null;
let ws = null;
let fitAddon = null;
let currentVM = null;

async function loadVMs() {
    try {
        const res = await fetch('/api/vms');
        const data = await res.json();
        renderVMList(data.vms || []);
    } catch (err) {
        console.error('Failed to load VMs:', err);
        document.getElementById('vm-list').innerHTML = '<li class="no-vms">Failed to load VMs</li>';
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

    // Connect WebSocket
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${location.host}/ws/terminal/${id}`);

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

// Initial load
loadVMs();

// Refresh VM list periodically
setInterval(loadVMs, 5000);

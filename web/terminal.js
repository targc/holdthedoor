function app() {
    return {
        // Auth
        loggedIn: false,
        username: '',
        password: '',
        loginError: '',
        token: '',
        expires: 0,

        // App
        vms: [],
        currentVM: null,
        status: '',
        countdown: '',

        // Internal
        term: null,
        ws: null,
        fitAddon: null,
        countdownInterval: null,
        vmInterval: null,

        init() {
            this.token = localStorage.getItem('token') || '';
            this.expires = parseInt(localStorage.getItem('expires') || '0', 10);

            if (this.token && this.expires > Date.now() / 1000) {
                this.loggedIn = true;
                this.startCountdown();
                this.loadVMs();
                this.vmInterval = setInterval(() => this.loadVMs(), 5000);
            }
        },

        async login() {
            this.loginError = '';
            try {
                const res = await fetch('/api/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username: this.username, password: this.password })
                });
                const data = await res.json();

                if (!res.ok) {
                    this.loginError = data.error || 'login failed';
                    return;
                }

                this.token = data.token;
                this.expires = data.expires;
                localStorage.setItem('token', this.token);
                localStorage.setItem('expires', this.expires.toString());

                this.loggedIn = true;
                this.username = '';
                this.password = '';
                this.startCountdown();
                this.loadVMs();
                this.vmInterval = setInterval(() => this.loadVMs(), 5000);
            } catch (e) {
                this.loginError = 'connection error';
            }
        },

        logout() {
            this.loggedIn = false;
            this.token = '';
            this.expires = 0;
            this.vms = [];
            this.currentVM = null;
            this.status = '';
            localStorage.removeItem('token');
            localStorage.removeItem('expires');

            if (this.ws) { this.ws.close(); this.ws = null; }
            if (this.term) { this.term.dispose(); this.term = null; }
            if (this.countdownInterval) { clearInterval(this.countdownInterval); }
            if (this.vmInterval) { clearInterval(this.vmInterval); }
        },

        startCountdown() {
            this.updateCountdown();
            this.countdownInterval = setInterval(() => this.updateCountdown(), 1000);
        },

        updateCountdown() {
            const now = Math.floor(Date.now() / 1000);
            const remaining = this.expires - now;

            if (remaining <= 0) {
                this.logout();
                return;
            }

            const m = Math.floor(remaining / 60);
            const s = remaining % 60;
            this.countdown = `[${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}]`;
        },

        async loadVMs() {
            if (!this.loggedIn) return;

            try {
                const res = await fetch('/api/vms', {
                    headers: { 'Authorization': `Bearer ${this.token}` }
                });

                if (res.status === 401) {
                    this.logout();
                    return;
                }

                const data = await res.json();
                this.vms = data.vms || [];
            } catch (e) {
                console.error('Failed to load VMs:', e);
            }
        },

        connect(vm) {
            if (this.currentVM === vm.id) return;

            this.status = `--connecting ${vm.name}...`;

            if (this.ws) { this.ws.close(); this.ws = null; }
            if (this.term) { this.term.dispose(); }

            this.term = new Terminal({
                cursorBlink: true,
                fontSize: 13,
                fontFamily: 'Menlo, Monaco, monospace',
                theme: {
                    background: '#0a0a0a',
                    foreground: '#e5e5e5',
                    cursor: '#dc2626',
                    selectionBackground: '#dc262644'
                }
            });

            this.fitAddon = new FitAddon.FitAddon();
            this.term.loadAddon(this.fitAddon);
            this.term.open(document.getElementById('terminal'));
            this.fitAddon.fit();

            const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
            this.ws = new WebSocket(`${protocol}//${location.host}/ws/terminal/${vm.id}?token=${this.token}`);

            this.ws.onopen = () => {
                this.currentVM = vm.id;
                this.status = `--${vm.name}`;
                this.ws.send(JSON.stringify({ type: 'resize', cols: this.term.cols, rows: this.term.rows }));
            };

            this.ws.onmessage = (e) => {
                const msg = JSON.parse(e.data);
                if (msg.type === 'output') this.term.write(msg.data);
                else if (msg.type === 'error') this.term.write(`\r\n\x1b[31m${msg.data}\x1b[0m\r\n`);
            };

            this.ws.onclose = () => {
                this.status = `--disconnected`;
                this.currentVM = null;
            };

            this.ws.onerror = () => {
                this.status = `--error`;
            };

            this.term.onData(data => {
                if (this.ws?.readyState === WebSocket.OPEN) {
                    this.ws.send(JSON.stringify({ type: 'input', data }));
                }
            });

            window.onresize = () => {
                if (this.fitAddon && this.term && this.ws?.readyState === WebSocket.OPEN) {
                    this.fitAddon.fit();
                    this.ws.send(JSON.stringify({ type: 'resize', cols: this.term.cols, rows: this.term.rows }));
                }
            };
        }
    };
}

const WebSocket = require('ws');
const http = require('http');

const PORT = process.env.PORT || 8080;
const HEARTBEAT_INTERVAL = 30000;

const server = http.createServer((req, res) => {
    if (req.url === '/health') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
            status: 'healthy',
            clients: clients.size,
            uptime: process.uptime()
        }));
    } else {
        res.writeHead(404);
        res.end('Not Found');
    }
});

const wss = new WebSocket.Server({ server });
const clients = new Map();

wss.on('connection', (ws, req) => {
    const clientIp = req.socket.remoteAddress;
    let clientId = null;
    let clientType = null;

    console.log(`[${new Date().toISOString()}] New connection from ${clientIp}`);

    ws.isAlive = true;
    ws.on('pong', () => { ws.isAlive = true; });

    ws.on('message', (data) => {
        try {
            const message = JSON.parse(data.toString());
            handleMessage(ws, message);
        } catch (error) {
            console.error('Failed to parse message:', error);
            ws.send(JSON.stringify({ type: 'error', message: 'Invalid JSON' }));
        }
    });

    ws.on('close', () => {
        if (clientId) {
            clients.delete(clientId);
            console.log(`[${new Date().toISOString()}] Client disconnected: ${clientId} (${clients.size} remaining)`);
            if (clientType === 'host') {
                broadcastHostDisconnected(clientId);
            }
        }
    });

    ws.on('error', (error) => {
        console.error(`WebSocket error for ${clientId}:`, error);
    });

    function handleMessage(ws, message) {
        switch (message.type) {
            case 'register':
                handleRegister(ws, message);
                break;
            case 'list-hosts':
                handleListHosts(ws);
                break;
            case 'offer':
            case 'answer':
            case 'ice-candidate':
                handleSignaling(ws, message);
                break;
            case 'ping':
                ws.send(JSON.stringify({ type: 'pong' }));
                break;
            default:
                console.warn('Unknown message type:', message.type);
        }
    }

    function handleRegister(ws, message) {
        clientId = message.id;
        clientType = message.clientType || 'controller';
        clients.set(clientId, ws);
        console.log(`[${new Date().toISOString()}] Registered: ${clientId} as ${clientType}`);
        ws.send(JSON.stringify({ type: 'registered', id: clientId, timestamp: Date.now() }));
        if (clientType === 'host') {
            broadcastHostList();
        }
    }

    function handleListHosts(ws) {
        const hosts = Array.from(clients.entries())
            .filter(([id]) => id.startsWith('host-'))
            .map(([id, client]) => ({ id, online: client.readyState === WebSocket.OPEN }));
        ws.send(JSON.stringify({ type: 'hosts', list: hosts }));
    }

    function handleSignaling(ws, message) {
        const targetId = message.target;
        const targetWs = clients.get(targetId);

        if (!targetWs || targetWs.readyState !== WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: 'error',
                message: `Target ${targetId} not found or not connected`
            }));
            return;
        }

        targetWs.send(JSON.stringify({
            type: message.type,
            from: clientId,
            payload: message.payload,
            timestamp: Date.now()
        }));
        console.log(`[${new Date().toISOString()}] Relayed ${message.type} from ${clientId} to ${targetId}`);
    }

    function broadcastHostList() {
        const hosts = Array.from(clients.entries())
            .filter(([id]) => id.startsWith('host-'))
            .map(([id, client]) => ({ id, online: client.readyState === WebSocket.OPEN }));
        const msg = JSON.stringify({ type: 'hosts-updated', list: hosts });
        clients.forEach((client, id) => {
            if (!id.startsWith('host-') && client.readyState === WebSocket.OPEN) {
                client.send(msg);
            }
        });
    }

    function broadcastHostDisconnected(hostId) {
        const msg = JSON.stringify({ type: 'host-disconnected', hostId });
        clients.forEach((client, id) => {
            if (!id.startsWith('host-') && client.readyState === WebSocket.OPEN) {
                client.send(msg);
            }
        });
    }
});

const heartbeat = setInterval(() => {
    wss.clients.forEach((ws) => {
        if (!ws.isAlive) return ws.terminate();
        ws.isAlive = false;
        ws.ping();
    });
}, HEARTBEAT_INTERVAL);

wss.on('close', () => { clearInterval(heartbeat); });

server.listen(PORT, () => {
    console.log(`[${new Date().toISOString()}] Signaling server running on port ${PORT}`);
    console.log(`Health check available at http://localhost:${PORT}/health`);
});

process.on('SIGTERM', () => {
    console.log('SIGTERM received, closing server...');
    wss.clients.forEach((ws) => ws.close());
    server.close(() => { process.exit(0); });
});

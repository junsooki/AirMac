# AirMac

Remote desktop control for macOS. View and control your Mac from another Mac over your local network.

## How It Works

AirMac has three parts: a **host** (the Mac being controlled), a **controller** (the Mac viewing/controlling it), and a **signaling server** that brokers the initial connection. Once connected, all screen data and input flows directly peer-to-peer via WebRTC.

```
┌──────────┐         WebSocket          ┌─────────────────┐
│          │◄──────────────────────────►│                  │
│   Host   │   register, SDP, ICE       │ Signaling Server │
│  (macOS) │                            │   (Node.js)      │
│          │                            └────────┬─────────┘
└────┬─────┘                                     │
     │                                           │ WebSocket
     │  WebRTC DataChannels (peer-to-peer)       │
     │                                           │
     │  ═══ "frames" ═══► raw JPEG bytes         │
     │  ◄══ "input"  ═══  JSON events            │
     │                                           │
     │                                    ┌──────┴──────┐
     └────────────────────────────────────┤  Controller  │
                                          │   (macOS)    │
                                          └─────────────┘
```

After the WebRTC peer connection is established, the signaling server is no longer involved. All data flows directly between host and controller.

## Connection Flow

1. Host and controller connect to the signaling server via WebSocket
2. Both register with unique IDs (`host-xxxx`, `controller-xxxx`) and their client type
3. Controller requests the host list, picks a host
4. **Controller creates a WebRTC offer** (SDP) and sends it through the signaling server
5. Host receives the offer, creates data channels, creates an answer, sends it back
6. Both sides exchange ICE candidates through signaling for NAT traversal
7. WebRTC peer connection establishes directly between host and controller
8. Host streams JPEG frames on the `"frames"` data channel
9. Controller sends input events as JSON on the `"input"` data channel

## Architecture

### Host (macOS)

The host captures the screen, encodes frames, and injects received input events.

```
Screen Capture (CGWindowListCreateImage via dlsym)
        │
        ▼
JPEG Encoder (image/jpeg, configurable quality 1-100)
        │
        ▼
WebRTC DataChannel "frames" ──────────► Controller
                                          │
WebRTC DataChannel "input" ◄──────────── │
        │
        ▼
Input Injector (CGEvent API via cgo)
        │
        ├── Mouse: CGEventCreateMouseEvent (move, click, scroll)
        └── Keyboard: CGEventCreateKeyboardEvent (key down/up + modifier flags)
```

**Screen capture** uses `CGWindowListCreateImage`. This function was removed from macOS 15 SDK headers but the symbol still exists in the CoreGraphics dylib, so it's loaded dynamically via `dlsym`. Captures run on a ticker at the configured FPS (default 30). Each frame is an RGBA bitmap rendered via `CGBitmapContextCreate` and copied into a Go `image.RGBA` buffer.

**JPEG encoding** compresses each RGBA frame using Go's standard `image/jpeg` encoder. The buffer is pre-allocated at 256KB to reduce GC pressure. Quality is configurable (default 70).

**Input injection** uses CoreGraphics CGEvent APIs via cgo:
- `CGEventCreateMouseEvent` for move, click (left/right/middle), with proper `CGEventType` dispatch
- `CGEventCreateScrollWheelEvent` for scroll (pixel-based, 2-axis)
- `CGEventCreateKeyboardEvent` for key down/up with macOS virtual key codes
- `CGEventSetFlags` for modifier keys (Shift, Ctrl, Alt, Cmd mapped to `CGEventFlags` bitmask)

All events are posted via `CGEventPost(kCGHIDEventTap, ...)` which injects at the HID level.

**Permissions required:**
- **Screen Recording** — for `CGWindowListCreateImage` to capture screen content
- **Accessibility** — for `CGEventPost` to inject input events

The host checks both permissions on startup and exits with instructions if either is missing.

### Controller (macOS)

The macOS controller uses [Ebitengine](https://ebitengine.org/) for display and input capture:

- Decodes incoming JPEG frames to `image.RGBA` via Go's `image/jpeg.Decode`
- Renders frames in an Ebitengine window, scaled to fit using aspect-fit (letterboxing)
- Captures mouse position, button clicks (left/right/middle), and scroll wheel events
- Captures all keyboard keys including function keys, with modifier state tracking
- Maps window coordinates to remote screen coordinates (accounting for scale and offset)
- Sends input events as JSON over the `"input"` data channel

### Signaling Server (Node.js)

A lightweight WebSocket relay server (~150 lines):

- Maintains a `Map<clientId, WebSocket>` of connected clients
- Routes `offer`, `answer`, and `ice-candidate` messages between peers by `target` ID
- Adds `from` field and `timestamp` when relaying messages
- Broadcasts `hosts-updated` to all controllers when a host registers
- Broadcasts `host-disconnected` when a host's WebSocket closes
- Responds to application-level `ping` with `pong` (separate from WebSocket ping/pong)
- Runs WebSocket-level heartbeat every 30 seconds to detect dead connections
- Health check endpoint at `GET /health` returns `{status, clients, uptime}`

The server is stateless — no persistent storage, no auth, no sessions. It only holds live WebSocket connections in memory.

## WebRTC Details

### Why DataChannels Instead of Media Tracks

AirMac sends screen frames as JPEG images over DataChannels rather than using WebRTC's built-in video codec pipeline (VP8/H.264 over RTP). Reasons:

1. **Simplicity** — no codec negotiation, no encoder/decoder factory, no RTP packetization
2. **Control** — full control over quality, frame rate, and encoding strategy
3. **Compatibility** — JPEG is universally supported; pion/webrtc handles DataChannels cleanly
4. **Local network** — bandwidth is not a concern on LAN; a 1080p JPEG at quality 70 is ~100-200KB per frame

The tradeoff is higher bandwidth than H.264/VP8 would use. For a WAN deployment, switching to a video track with hardware codec would be worthwhile.

### ICE / STUN

Both peers use Google's public STUN servers for NAT traversal:
- `stun:stun.l.google.com:19302`
- `stun:stun1.l.google.com:19302`

On a local network, ICE typically resolves to **host candidates** (direct LAN IP addresses) so data flows directly without any relay. TURN is not configured but could be added for WAN connectivity.

### Data Channels

The **host creates both data channels** before generating its SDP answer. The controller accepts them via `OnDataChannel`.

| Channel | Direction | Format | Config |
|---------|-----------|--------|--------|
| `frames` | Host → Controller | Raw JPEG bytes (binary) | `ordered: false`, `maxRetransmits: 0` |
| `input` | Controller → Host | JSON text | `ordered: true`, reliable (default) |

`frames` is configured as unreliable and unordered — if a frame packet is lost, it's better to skip it than delay the next frame. `input` uses reliable ordered delivery so no clicks or keystrokes are dropped or arrive out of order.

### SDP Wire Format

Offer and answer use the same JSON format as pion/webrtc's `SessionDescription` serialization:

```json
{"type": "offer", "sdp": "v=0\r\no=- 0 0 IN IP4 ..."}
{"type": "answer", "sdp": "v=0\r\no=- 0 0 IN IP4 ..."}
```

### ICE Candidate Wire Format

Matches pion/webrtc's `ICECandidateInit` JSON serialization:

```json
{"candidate": "candidate:1 1 udp ...", "sdpMLineIndex": 0, "sdpMid": "0"}
```

## Signaling Protocol

All messages are JSON over WebSocket. The envelope:

```json
{
  "type": "register",
  "id": "host-a1b2c3d4",
  "clientType": "host",
  "from": "",
  "target": "",
  "payload": {},
  "list": [],
  "hostId": "",
  "message": "",
  "timestamp": 0
}
```

Fields use `omitempty` — only relevant fields are present for each message type.

| Message | Direction | Key Fields | Purpose |
|---|---|---|---|
| `register` | Client → Server | `id`, `clientType` | Register with signaling server |
| `registered` | Server → Client | `id`, `timestamp` | Confirm registration |
| `list-hosts` | Client → Server | — | Request available hosts |
| `hosts` | Server → Client | `list` | Host list response |
| `hosts-updated` | Server → Controllers | `list` | Broadcast on host connect |
| `host-disconnected` | Server → Controllers | `hostId` | Broadcast on host drop |
| `offer` | Client → Server → Client | `target`, `payload` | SDP offer relay |
| `answer` | Client → Server → Client | `target`, `payload` | SDP answer relay |
| `ice-candidate` | Client → Server → Client | `target`, `payload` | ICE candidate relay |
| `ping` | Client → Server | — | Heartbeat |
| `pong` | Server → Client | — | Heartbeat response |
| `error` | Server → Client | `message` | Error notification |

The server identifies hosts by ID prefix: any client whose ID starts with `host-` is treated as a host. The `broadcastHostList` and `broadcastHostDisconnected` functions only send to clients whose IDs do **not** start with `host-`.

## Input Event Protocol

Input events are JSON sent on the `"input"` data channel from controller to host:

```json
{"type": "mouse_move", "x": 512.0, "y": 384.0}
{"type": "mouse_down", "x": 512.0, "y": 384.0}
{"type": "mouse_down", "x": 512.0, "y": 384.0, "button": 1}
{"type": "mouse_up", "x": 512.0, "y": 384.0}
{"type": "mouse_scroll", "scrollDY": -3.0}
{"type": "key_down", "keyCode": 0, "modifiers": 8}
{"type": "key_up", "keyCode": 0}
```

| Field | Type | Description |
|---|---|---|
| `type` | string | `mouse_move`, `mouse_down`, `mouse_up`, `mouse_scroll`, `key_down`, `key_up` |
| `x`, `y` | float64 | Remote screen coordinates in pixels |
| `button` | int | `0` = left (default), `1` = right, `2` = middle |
| `keyCode` | uint16 | macOS virtual key code |
| `modifiers` | uint8 | Bitfield: `1` = Shift, `2` = Ctrl, `4` = Alt/Option, `8` = Cmd |
| `scrollDX`, `scrollDY` | float64 | Scroll delta in pixels |

All fields use `omitempty` — zero-valued fields are omitted. This means `button: 0` (left click) is omitted and defaults to 0 on the receiving end, which is correct.

### Coordinate Mapping

Both controllers convert view/touch coordinates to remote screen coordinates using the same formula (from `internal/display/ebiten.go:120-132`):

```
scale = min(viewWidth / frameWidth, viewHeight / frameHeight)
offsetX = (viewWidth - frameWidth * scale) / 2
offsetY = (viewHeight - frameHeight * scale) / 2

remoteX = (viewX - offsetX) / scale
remoteY = (viewY - offsetY) / scale
```

This accounts for aspect-fit letterboxing — the frame is scaled to fit the view while maintaining aspect ratio, with black bars on the sides or top/bottom.

### macOS Virtual Key Codes

The keyboard uses macOS virtual key codes (not USB HID or ASCII). These are the CGEvent key codes:

| Key | Code | Key | Code | Key | Code | Key | Code |
|-----|------|-----|------|-----|------|-----|------|
| A | 0x00 | S | 0x01 | D | 0x02 | F | 0x03 |
| H | 0x04 | G | 0x05 | Z | 0x06 | X | 0x07 |
| C | 0x08 | V | 0x09 | B | 0x0B | Q | 0x0C |
| W | 0x0D | E | 0x0E | R | 0x0F | Y | 0x10 |
| T | 0x11 | 1 | 0x12 | 2 | 0x13 | 3 | 0x14 |
| 4 | 0x15 | 6 | 0x16 | 5 | 0x17 | 9 | 0x19 |
| 7 | 0x1A | 8 | 0x1C | 0 | 0x1D | O | 0x1F |
| U | 0x20 | I | 0x22 | P | 0x23 | L | 0x25 |
| J | 0x26 | K | 0x28 | N | 0x2D | M | 0x2E |
| Return | 0x24 | Tab | 0x30 | Space | 0x31 | Delete | 0x33 |
| Escape | 0x35 | Left | 0x7B | Right | 0x7C | Down | 0x7D |
| Up | 0x7E | F1 | 0x7A | F2 | 0x78 | F3 | 0x63 |
| F4 | 0x76 | F5 | 0x60 | F6 | 0x61 | F7 | 0x62 |
| F8 | 0x64 | F9 | 0x65 | F10 | 0x6D | F11 | 0x67 |
| F12 | 0x6F | Fwd Del | 0x75 | Home | 0x73 | End | 0x77 |
| PgUp | 0x74 | PgDn | 0x79 | | | | |

Note: macOS key codes are **not sequential by keyboard position** — they follow the original Mac 128K scan code layout.

### Modifier Flags

The `modifiers` field is a bitfield sent with key events:

| Bit | Value | Modifier | CGEventFlags constant |
|-----|-------|----------|-----------------------|
| 0 | 1 | Shift | `kCGEventFlagMaskShift` (0x00020000) |
| 1 | 2 | Control | `kCGEventFlagMaskControl` (0x00040000) |
| 2 | 4 | Alt/Option | `kCGEventFlagMaskAlternate` (0x00080000) |
| 3 | 8 | Command | `kCGEventFlagMaskCommand` (0x00100000) |

Example: Cmd+A = `{"type":"key_down","keyCode":0,"modifiers":8}` (keyCode 0x00 = A, modifiers 8 = Cmd)

## Project Structure

```
AirMac/
├── cmd/
│   ├── host/main.go                  # Host entry point
│   └── controller/main.go            # macOS controller entry point
├── internal/
│   ├── capture/
│   │   ├── capture.go                # Capturer interface + Frame type
│   │   └── coregraphics.go           # CGWindowListCreateImage via cgo/dlsym
│   ├── encoder/
│   │   └── jpeg.go                   # JPEG encoding with configurable quality
│   ├── decoder/
│   │   └── jpeg.go                   # JPEG decoding to image.RGBA
│   ├── input/
│   │   ├── events.go                 # InputEvent struct + event types
│   │   ├── injector.go               # Injector interface
│   │   └── cgevent.go                # CGEvent injection via cgo
│   ├── display/
│   │   ├── display.go                # Display interface + InputCallback type
│   │   └── ebiten.go                 # Ebitengine rendering + input capture
│   ├── peer/
│   │   ├── peer.go                   # Shared PeerConnection factory + ICE config
│   │   ├── host.go                   # Host peer (creates data channels, answers)
│   │   └── controller.go             # Controller peer (creates offer, accepts channels)
│   ├── transport/
│   │   ├── transport.go              # FrameSender/Receiver + InputSender/Receiver interfaces
│   │   └── datachannel.go            # DataChannel-based transport implementation
│   ├── signaling/
│   │   ├── messages.go               # Message types + wire format structs
│   │   └── client.go                 # WebSocket client with ping loop
│   ├── permissions/
│   │   ├── screen.go                 # Screen Recording permission check
│   │   └── accessibility.go          # Accessibility permission check
│   └── config/
│       └── config.go                 # CLI flag parsing for host + controller
├── signaling-server/
│   ├── server.js                     # WebSocket signaling relay
│   ├── package.json
│   └── package-lock.json
├── Makefile
├── go.mod
└── go.sum
```

## Quick Start

### Prerequisites

- macOS host machine
- Go 1.21+
- Node.js 18+
### 1. Start the signaling server

```bash
make run-signaling
```

Runs on `ws://localhost:8080`. Health check at `http://localhost:8080/health`.

### 2. Start the host

```bash
make run-host
```

On first run, macOS will prompt for **Screen Recording** and **Accessibility** permissions. Grant both in System Settings and restart.

Options:

| Flag | Default | Description |
|------|---------|-------------|
| `-signaling` | `ws://localhost:8080` | Signaling server URL |
| `-id` | auto-generated | Custom host ID |
| `-display` | `0` | Display index (0 = primary) |
| `-fps` | `30` | Target frame rate |
| `-quality` | `70` | JPEG quality (1-100) |

### 3a. Connect from macOS

```bash
make run-controller
```

Requires `-host` flag with the host ID:

```bash
bin/airmac-controller -signaling ws://localhost:8080 -host host-a1b2c3d4
```

### Build all

```bash
make all          # Builds host + controller to bin/
make clean        # Removes bin/
make test         # Runs Go tests
```

## Dependencies

### Go (Host + macOS Controller)

| Package | Version | Purpose |
|---------|---------|---------|
| [pion/webrtc/v4](https://github.com/pion/webrtc) | v4.x | WebRTC peer connection, data channels, ICE |
| [gorilla/websocket](https://github.com/gorilla/websocket) | v1.5.x | WebSocket client for signaling |
| [hajimehoshi/ebiten/v2](https://github.com/hajimehoshi/ebiten) | v2.x | Window rendering + input capture (controller) |
| CoreGraphics (cgo) | system | Screen capture + input injection (host) |

### Node.js (Signaling Server)

| Package | Version | Purpose |
|---------|---------|---------|
| [ws](https://github.com/websockets/ws) | ^8.x | WebSocket server |

# Quichat – zero‑infra-P2P chat

Quichat is a small Go 1.24 program that lets you chat over the Internet without a server in the middle. It speaks both TCP and QUIC‑v1, finds other peers through the libp2p Kademlia DHT, and slips past most home routers with AutoRelay. Messages travel on libp2p’s GossipSub mesh and carry micro‑second time‑stamps, so you can see real latency in the log.

---

## Why bother with another chat tool?

* **No central server** – each laptop calls the other directly. If that fails, they borrow a relay for the handshake and keep going.
* **Modern transport** – QUIC‑v1 is the same tech under HTTP/3. Faster first handshake, built‑in multiplexing, and better luck with NATs.
* **Plain terminal UI** – readline gives you history, Ctrl‑R search, and colours without a heavy GUI.
* **One static binary** – `go build` produces a \~7 MB file that runs on any modern Linux, macOS, or Windows box. No config files, no database.

---

## Quick start

1. **Build**

   ```bash
   git clone https://github.com/you/quichat.git
   # Linux (Ubuntu/Debian)
   sudo apt update && sudo apt install golang-go 
   # macOS (Homebrew)
   brew install go 
   cd quichat
   go mod tidy
   ```
2. **Start a listener**

   ```bash
   go run . run --listen 4001 --nick alice
   ```
   ***Example Output***

   ```bash
      Your multiaddr: /ip4/10.61.34.169/tcp/4001/p2p/12D3KooWKWzVjFaizQNEYf3cJCDeZbAbXacHVB4haqwakSmQNZHo

        ___ ___ ___    ___  _   _ ___ ___ _  _   _ _____ 
      | _ |_  | _ \  / _ \| | | |_ _/ __| || | /_|_   _|
      |  _// /|  _/ | (_) | |_| || | (__| __ |/ _ \| |  
      |_| /___|_|    \__\_\\___/|___\___|_||_/_/ \_|_|  
                                                                                                                                                                                                                                                        

      Welcome to P2P Quichat! 🚀
      *** alice joined the chat ***
    ```

  Copy the multi‑address the program prints.

3. **Start a dialer** (new terminal or second machine)

   ```bash
   ./go run --listen 4003 \
                --bootstrap "<multi‑addr‑from‑Alice>" \
                --nick bob
   ```

   Say hello – you should see the message in both windows.


---

#### Built‑in slash commands

| Command | Description                      |
| ------- | -------------------------------- |
| `/list` | List peers currently in the room |
| `/ping` | Round‑trip latency to each peer  |
| `/help` | Show in‑terminal cheat‑sheet     |
| `/quit` | Graceful leave                   |

---

## How it works (short version)

* **Kad‑DHT** – every node stores its address in a shared hash‑table. You don’t need to run a tracker.
* **AutoRelay** – if direct UDP fails, the peers fall back to TCP; if that fails too, they talk through a public relay. No port‑forwarding needed.
* **GossipSub** – a self‑healing broadcast layer; each peer helps spread messages, so the chat stays alive even if some users drop out.
* **QUIC‑v1** – one 1‑RTT handshake sets up TLS 1.3, and all chat lines travel in a multiplexed stream. If the remote side doesn’t speak QUIC, we fall back to TCP without fuss.

---

### Typical flow

1. Alice starts Quichat → announces herself in the DHT.
2. Bob gets Alice’s address and tries QUIC.
3. If QUIC fails, Bob tries TCP; if that fails, they relay.
4. Once a path is up, every line is encrypted end‑to‑end and forwarded by the mesh.

---
## 🛠  Tech stack

* **Go 1.24** – zero external dependencies.
* **go‑libp2p** subsystems
  – TCP & QUIC‑v1 transports
  – AutoRelay + NAT service
  – Kademlia DHT for global routing
  – GossipSub v1.2 for pub‑sub
* **`x/sync/errgroup`** – structured goroutine lifecycles
* **`chzyer/readline`** – ANSI T‑UI

---


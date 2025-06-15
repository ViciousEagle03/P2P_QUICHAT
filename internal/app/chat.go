package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
	"golang.org/x/sync/errgroup"
)

type Message struct {
	Nick string    `json:"nick"`
	Text string    `json:"text"`
	Ts   time.Time `json:"ts"`
}

const helpText = `Available commands:
/help           Show this help
/quit           Leave the chat
/list           Show peers currently in the room
/ping           Measure round-trip latency to all peers`

var pingOutstanding = make(map[string]time.Time) // id → timestamp

func makeID() string { // tiny UUID
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ChatLoop runs two goroutines: one to receive messages and one to send.
func ChatLoop(ctx context.Context, n *Node, nick string) error {
	// single shared readline instance
	rl, err := readline.New("> ")
	if err != nil {
		return fmt.Errorf("init readline: %w", err)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer rl.Close()

	g, ctx := errgroup.WithContext(ctx)

	announceJoinWhenReady(ctx, n, nick) // ← new
	fmt.Printf("\r\033[1;32m*** %s joined the chat ***\033[0m\n> ", nick)

	// ─── Receiver ───────────────────────────────────────────────────────────────
	g.Go(func() error {
		for {
			msg, err := n.Sub.Next(ctx)
			if err != nil {
				return err
			}
			var m Message
			if err := json.Unmarshal(msg.Data, &m); err != nil {
				continue
			}

			if strings.HasPrefix(m.Text, "__PING__") {
				if m.Nick == nick { // ← ignore your own ping
					continue
				}
				// reply with PONG
				resp := Message{
					Nick: nick,
					Text: "__PONG__" + m.Text[8:], // copy the ID
					Ts:   time.Now().UTC(),
				}
				b, _ := json.Marshal(resp)
				_ = n.Topic.Publish(ctx, b)
				continue // swallow; don’t print as chat
			}

			// 2. PONG  ──────────────────────────────────────────────────────────────
			if strings.HasPrefix(m.Text, "__PONG__") {
				if m.Nick == nick {
					continue
				} // ignore your own PONG

				id := m.Text[8:]
				if t0, ok := pingOutstanding[id]; ok {
					dur := time.Since(t0).Milliseconds()
					delete(pingOutstanding, id)

					rl.Write([]byte(fmt.Sprintf(
						"\r\033[36mPong from %s: %d ms\033[0m\n> ", m.Nick, dur)))
				}
				continue // swallow even if no match
			}

			if m.Text == "__JOIN__" {
				if m.Nick != nick { // skip your own copy
					rl.Write([]byte(
						fmt.Sprintf("\r\033[1;32m*** %s joined the chat ***\033[0m\n> ", m.Nick)))
				}
				continue
			}

			// Replace newlines with \n
			m.Text = strings.ReplaceAll(m.Text, "\n", "\n» ")

			// Erase the current input line ("> <whatever>")
			rl.Write([]byte("\x1b[2K\r"))

			// Print chip-stack message with leading "> "
			delivered := time.Now()
			outStr := fmt.Sprintf(
				"> [%s] [%s]\n» %s\n\n",
				delivered.Format("2006-01-02 15:04:05"),
				"\033[32m"+m.Nick+"\033[0m",
				m.Text,
			)
			rl.Write([]byte(outStr))

			// Re-draw the prompt
			rl.Write([]byte(rl.Config.Prompt))
		}
	})

	// ─── Sender ─────────────────────────────────────────────────────────────────
	g.Go(func() error {
		for {
			line, err := rl.Readline()
			if err == readline.ErrInterrupt {
				continue // re-prompt on Ctrl+C
			} else if err != nil {
				// Exit on EOF or other errors
				return err
			}

			if strings.HasPrefix(line, "/") {
				cmd := strings.ToLower(strings.TrimSpace(line[1:]))

				switch cmd {
				case "list":
					rl.Write([]byte("\x1b[2K\r"))
					peers := n.Topic.ListPeers()
					fmt.Fprintf(rl.Stdout(), "Peers (%d): %v\n", len(peers), peers)
					rl.Write([]byte(rl.Config.Prompt))
					continue

				case "ping":
					id := makeID()
					pingOutstanding[id] = time.Now()

					payload, _ := json.Marshal(Message{
						Nick: nick, Text: "__PING__" + id, Ts: time.Now().UTC(),
					})
					_ = n.Topic.Publish(ctx, payload)
					continue

				case "help", "h", "?":
					// Print help without killing the prompt
					rl.Write([]byte("\x1b[2K\r")) // clear current line
					fmt.Fprintln(rl.Stdout(), helpText)
					rl.Write([]byte(rl.Config.Prompt))
					continue

				case "quit", "exit", "q":
					fmt.Fprintln(rl.Stdout(), "👋  Bye!")
					cancel()   // cancel ctx → both goroutines exit
					return nil // close sender

				default:
					rl.Write([]byte("\x1b[2K\r"))
					fmt.Fprintf(rl.Stdout(), "Unknown command: %s\n", cmd)
					rl.Write([]byte(rl.Config.Prompt))
					continue
				}
			}

			rl.Write([]byte("\x1b[1A\x1b[2K\r"))

			payload, err := json.Marshal(
				Message{Nick: nick, Text: line, Ts: time.Now().UTC()},
			)
			if err != nil {
				return err
			}
			if err := n.Topic.Publish(ctx, payload); err != nil {
				return err
			}
		}
	})

	return g.Wait()
}

func announceJoinWhenReady(ctx context.Context, n *Node, nick string) {
	// fire exactly once
	var once sync.Once
	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Wait until we see at least one other peer in this topic
				if len(n.Topic.ListPeers()) > 0 {
					once.Do(func() {
						join := Message{Nick: nick, Text: "__JOIN__", Ts: time.Now().UTC()}
						if b, _ := json.Marshal(join); b != nil {
							_ = n.Topic.Publish(ctx, b)
						}
					})
					return
				}
			}
		}
	}()
}

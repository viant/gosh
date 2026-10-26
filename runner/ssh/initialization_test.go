package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/viant/gosh/runner"
	sshprotocol "golang.org/x/crypto/ssh"
)

func TestRunnerInitializationFailureIsCached(t *testing.T) {
	for _, phase := range []string{"dial", "session", "environment"} {
		t.Run(phase, func(t *testing.T) {
			server := newInitializationFailureServer(t, phase)
			r := New(server.listener.Addr().String(), &sshprotocol.ClientConfig{
				User:            "test",
				HostKeyCallback: sshprotocol.InsecureIgnoreHostKey(),
				Timeout:         time.Second,
			}, runner.WithEnvironment(map[string]string{"GOSH_TEST": "value"}))
			t.Cleanup(func() { _ = r.Close() })

			output, code, firstErr := r.Run(context.Background(), "must not execute")
			if firstErr == nil || output != "" || code != 0 {
				t.Fatalf("initial Run = (%q, %d, %v), want empty output and initialization error", output, code, firstErr)
			}
			switch phase {
			case "dial":
				if !strings.Contains(firstErr.Error(), "failed to dial:") {
					t.Fatalf("dial failure was not propagated: %v", firstErr)
				}
			case "session":
				var rejected *sshprotocol.OpenChannelError
				if !errors.As(firstErr, &rejected) || rejected.Reason != sshprotocol.Prohibited {
					t.Fatalf("session rejection was not propagated: %v", firstErr)
				}
			case "environment":
				if !strings.Contains(firstErr.Error(), "setenv failed") {
					t.Fatalf("environment rejection was not propagated: %v", firstErr)
				}
			}

			// Failed initialization must release the transport without requiring Close.
			select {
			case <-server.closed:
			case <-time.After(3 * time.Second):
				t.Fatal("failed initialization left its SSH connection open")
			}
			for i := 0; i < 3; i++ {
				output, code, err := r.Run(context.Background(), "must not retry")
				if err != firstErr || output != "" || code != 0 {
					t.Fatalf("repeat Run = (%q, %d, %v), want original initialization error %v", output, code, err, firstErr)
				}
			}
			if got := server.connections.Load(); got != 1 {
				t.Fatalf("retried SSH initialization: accepted %d connections", got)
			}
			wantSessions, wantRequests := int32(1), int32(0)
			if phase == "dial" {
				wantSessions = 0
			}
			if phase == "environment" {
				wantRequests = 1
			}
			if got := server.sessions.Load(); got != wantSessions {
				t.Fatalf("session attempts = %d, want %d", got, wantSessions)
			}
			if got := server.requests.Load(); got != wantRequests {
				t.Fatalf("session requests = %d, want %d; shell setup continued after rejection", got, wantRequests)
			}
			if phase == "environment" {
				select {
				case request := <-server.environment:
					if request.Name != "GOSH_TEST" || request.Value != "value" {
						t.Fatalf("unexpected rejected environment: %+v", request)
					}
				default:
					t.Fatal("environment rejection was not exercised")
				}
			}
		})
	}
}

type initializationEnvironment struct {
	Name  string
	Value string
}

type initializationFailureServer struct {
	listener    net.Listener
	connections atomic.Int32
	sessions    atomic.Int32
	requests    atomic.Int32
	closed      chan struct{}
	environment chan initializationEnvironment
}

// The server uses an ephemeral loopback port and a generated host key. It never
// opens a shell or reads local SSH credentials, configuration, or known_hosts.
func newInitializationFailureServer(t *testing.T, phase string) *initializationFailureServer {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := sshprotocol.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	config := &sshprotocol.ServerConfig{NoClientAuth: true}
	config.AddHostKey(signer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &initializationFailureServer{
		listener: listener, closed: make(chan struct{}, 8),
		environment: make(chan initializationEnvironment, 8),
	}
	var workers sync.WaitGroup
	var connectionsMu sync.Mutex
	connections := make(map[net.Conn]struct{})
	workers.Add(1)
	go func() {
		defer workers.Done()
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			server.connections.Add(1)
			connectionsMu.Lock()
			connections[connection] = struct{}{}
			connectionsMu.Unlock()
			workers.Add(1)
			go func() {
				defer workers.Done()
				defer func() {
					_ = connection.Close()
					connectionsMu.Lock()
					delete(connections, connection)
					connectionsMu.Unlock()
					server.closed <- struct{}{}
				}()
				_ = connection.SetDeadline(time.Now().Add(10 * time.Second))
				if phase == "dial" {
					return // Fail ssh.Dial during its handshake.
				}
				transport, channels, requests, err := sshprotocol.NewServerConn(connection, config)
				if err != nil {
					return
				}
				defer transport.Close()
				go sshprotocol.DiscardRequests(requests)
				for incoming := range channels {
					server.sessions.Add(1)
					if phase == "session" {
						_ = incoming.Reject(sshprotocol.Prohibited, "test rejects sessions")
						continue
					}
					channel, requests, err := incoming.Accept()
					if err != nil {
						return
					}
					for request := range requests {
						server.requests.Add(1)
						if request.Type == "env" {
							var environment initializationEnvironment
							if sshprotocol.Unmarshal(request.Payload, &environment) == nil {
								server.environment <- environment
							}
						}
						_ = request.Reply(false, nil)
					}
					_ = channel.Close()
				}
			}()
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		connectionsMu.Lock()
		for connection := range connections {
			_ = connection.Close()
		}
		connectionsMu.Unlock()
		done := make(chan struct{})
		go func() { workers.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("SSH test server did not stop")
		}
	})
	return server
}

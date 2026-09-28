package P2P

import "C" // Enables Cgo so external languages (C, C++, Swift) can call this Go code

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/pion/webrtc/v4"
)

// getEnv returns the env var value or a fallback if it is empty.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

//export StartP2PEngine
func StartP2PEngine() {
	fmt.Println("[P2P Engine] Initializing STUN/TURN configuration...")

	// Load .env (ignored if the file doesn't exist, e.g. in production)
	if err := godotenv.Load(); err != nil {
		fmt.Println("[P2P Engine] No .env file found, using system environment variables")
	}

	stunURL := getEnv("STUN_URL", "stun:stun.l.google.com:19302")
	turnURL := os.Getenv("TURN_URL")
	turnUser := os.Getenv("TURN_USERNAME")
	turnCred := os.Getenv("TURN_CREDENTIAL")

	startTime := time.Now()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// STUN is always included; TURN only if configured
	iceServers := []webrtc.ICEServer{
		{URLs: []string{stunURL}},
	}
	if turnURL != "" {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs:       []string{turnURL},
			Username:   turnUser,
			Credential: turnCred, // credential type defaults to password
		})
	} else {
		fmt.Println("[P2P Engine] TURN_URL not set, running with STUN only")
	}

	config := webrtc.Configuration{ICEServers: iceServers}

	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		fmt.Printf("[Error] %v\n", err)
		return
	}

	defer func() {
		fmt.Println("\n[P2P Engine] Closing PeerConnection safely...")
		if err := peerConnection.Close(); err != nil {
			fmt.Printf("[Error] Closing PeerConnection: %v\n", err)
		} else {
			fmt.Println("[P2P Engine] PeerConnection closed successfully.")
		}
	}()

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c != nil {
			elapsed := time.Since(startTime)
			fmt.Printf("[P2P Engine] [%s] New ICE candidate discovered: %s\n", elapsed.Round(time.Millisecond), c.String())
		} else {
			totalTime := time.Since(startTime)
			fmt.Printf("\n[P2P Engine] All ICE candidates gathered successfully. Total time: %s\n", totalTime.Round(time.Millisecond))
		}
	})

	if _, err = peerConnection.CreateDataChannel("chat", nil); err != nil {
		fmt.Printf("[Error] %v\n", err)
		return
	}

	fmt.Println("[P2P Engine] Starting ICE gathering process...")

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		fmt.Printf("[Error] %v\n", err)
		return
	}

	if err = peerConnection.SetLocalDescription(offer); err != nil {
		fmt.Printf("[Error] %v\n", err)
		return
	}

	fmt.Println("[P2P Engine] Engine is running in background. Waiting for OS signals...")
	<-ctx.Done()
	fmt.Println("\n[P2P Engine] Shutdown signal received!")
}
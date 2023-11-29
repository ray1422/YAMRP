package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

func signalCandidate(c *webrtc.ICECandidate) error {
	payload := []byte(c.ToJSON().Candidate)
	// base64 encoding payload and print it so we can paste it in our remote peer
	encoded := base64.URLEncoding.EncodeToString(payload)
	fmt.Printf("ICE Candidate (Base64): \n%s\n\n", encoded)
	return nil
}

// askIceCandidateFromStdin is /candidate
func askIceCandidateFromStdin(pc *webrtc.PeerConnection) error {
	// Read candidate from stdin
	fmt.Println("Paste remote ICE Candidate (Base64) and press enter:")
	cand := ""
	_, err := fmt.Scanln(&cand)
	// trim space
	if cand[:3] == "end" {
		return io.EOF
	}
	if err != nil {
		return err
	}
	// base64 urlsafe decoding it
	decoded, err := base64.URLEncoding.DecodeString(cand)
	if err != nil {
		return err
	}
	fmt.Println("adding candidate..")
	go addICECandidate(pc, webrtc.ICECandidateInit{Candidate: string(decoded)})
	return nil
}

func addICECandidate(pc *webrtc.PeerConnection, candidate webrtc.ICECandidateInit) error {
	return pc.AddICECandidate(candidate)
}

// setRemoteDescription is /sdp
func setRemoteDescription(candidatesMux *sync.Mutex, pendingCandidates []*webrtc.ICECandidate, peerConnection *webrtc.PeerConnection) {
	sdp := webrtc.SessionDescription{}
	fmt.Println("Paste remote SDP (Base64) and press enter:")
	line := ""
	_, err := fmt.Scanln(&line)
	if err != nil {
		panic(err)
	}
	// base64 urlsafe decoding it
	decoded, err := base64.URLEncoding.DecodeString(line)
	if sdpErr := json.NewDecoder(
		bytes.NewBuffer(decoded),
	).Decode(&sdp); sdpErr != nil {
		panic(sdpErr)
	}

	if sdpErr := peerConnection.SetRemoteDescription(sdp); sdpErr != nil {
		panic(sdpErr)
	}

	candidatesMux.Lock()
	defer candidatesMux.Unlock()

	for _, c := range pendingCandidates {
		if onICECandidateErr := signalCandidate(c); onICECandidateErr != nil {
			panic(onICECandidateErr)
		}
	}
}

func main() {
	var candidatesMux sync.Mutex
	pendingCandidates := make([]*webrtc.ICECandidate, 0)

	// Everything below is the Pion WebRTC API! Thanks for using it ❤️.

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	defer func() {
		if cErr := peerConnection.Close(); cErr != nil {
			fmt.Printf("cannot close peerConnection: %v\n", cErr)
		}
	}()

	// When an ICE candidate is available send to the other Pion instance
	// the other Pion instance will add this candidate by calling AddICECandidate
	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		candidatesMux.Lock()
		defer candidatesMux.Unlock()

		desc := peerConnection.RemoteDescription()
		if desc == nil {
			pendingCandidates = append(pendingCandidates, c)
		} else if onICECandidateErr := signalCandidate(c); onICECandidateErr != nil {
			panic(onICECandidateErr)
		}
	})

	// Create a datachannel with label 'data'
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}

	// Set the handler for Peer connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			fmt.Println("Peer Connection has gone to failed exiting")
			os.Exit(0)
		}

		if s == webrtc.PeerConnectionStateClosed {
			// PeerConnection was explicitly closed. This usually happens from a DTLS CloseNotify
			fmt.Println("Peer Connection has gone to closed exiting")
			os.Exit(0)
		}
	})

	// Register channel opening handling
	dataChannel.OnOpen(func() {
		fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", dataChannel.Label(), dataChannel.ID())

		for range time.NewTicker(5 * time.Second).C {
			message := fmt.Sprintf("hello %d", time.Now().UnixNano())
			fmt.Printf("Sending '%s'\n", message)

			// Send the message as text
			sendTextErr := dataChannel.SendText(message)
			if sendTextErr != nil {
				panic(sendTextErr)
			}
		}
	})

	// Register text message handling
	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
	})

	// Create an offer to send to the other process
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	// Note: this will start the gathering of ICE candidates
	if err = peerConnection.SetLocalDescription(offer); err != nil {
		panic(err)
	}

	// Send our offer to the HTTP server listening in the other process
	payload, err := json.Marshal(offer)
	if err != nil {
		panic(err)
	}
	fmt.Println("paste this offer to server (SDP):")
	// print offer candidate with base64URL encoding
	encoded := base64.URLEncoding.EncodeToString(payload)
	fmt.Printf("Offer (Base64): \n%s\n\n", encoded)
	setRemoteDescription(&candidatesMux, pendingCandidates, peerConnection)
	for askIceCandidateFromStdin(peerConnection) == nil {
		fmt.Println("ICE Candidate added, type 'end' to exit, or paste another ICE Candidate:")
	}
	// Block forever
	select {}
}

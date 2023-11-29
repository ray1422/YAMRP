package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

// setRemoteDescription is /sdp
func setRemoteDescription(candidatesMux *sync.Mutex, pendingCandidates []*webrtc.ICECandidate, peerConnection *webrtc.PeerConnection) {
	sdp := webrtc.SessionDescription{}
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
	// set sdp
	if err := peerConnection.SetRemoteDescription(sdp); err != nil {
		panic(err)
	}

	// Create an answer to send to the other process
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		panic(err)
	}

	// Send our answer to the HTTP server listening in the other process
	payload, err := json.Marshal(answer)
	if err != nil {
		panic(err)
	}
	// print payload so we can paste it in our remote peer
	fmt.Println("Paste remote SDP (Base64) and press enter:")
	fmt.Printf("SDP (Base64): \n%s\n\n", base64.URLEncoding.EncodeToString(payload))

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		panic(err)
	}

	candidatesMux.Lock()
	for _, c := range pendingCandidates {
		onICECandidateErr := signalCandidate(c)
		if onICECandidateErr != nil {
			panic(onICECandidateErr)
		}
	}
	candidatesMux.Unlock()

}

func addICECandidate(pc *webrtc.PeerConnection, candidate webrtc.ICECandidateInit) error {
	return pc.AddICECandidate(candidate)
}
func askIceCandidateFromStdin(pc *webrtc.PeerConnection) error {
	// Read candidate from stdin
	fmt.Println("Paste remote ICE Candidate (Base64) and press enter:")
	cand := ""
	_, err := fmt.Scanln(&cand)
	if err != nil {
		return err
	}
	// base64 urlsafe decoding it
	decoded, err := base64.URLEncoding.DecodeString(cand)
	if err != nil {
		return err
	}

	return addICECandidate(pc, webrtc.ICECandidateInit{Candidate: string(decoded)})
}

func signalCandidate(c *webrtc.ICECandidate) error {
	payload := []byte(c.ToJSON().Candidate)
	// base64 encoding payload and print it so we can paste it in our remote peer
	encoded := base64.URLEncoding.EncodeToString(payload)
	fmt.Printf("ICE Candidate (Base64): \n%s\n\n", encoded)
	return nil
}

func main() { // nolint:gocognit
	// offerAddr := flag.String("offer-address", "localhost:50000", "Address that the Offer HTTP server is hosted on.")
	// answerAddr := flag.String("answer-address", ":60000", "Address that the Answer HTTP server is hosted on.")
	flag.Parse()

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
		if err := peerConnection.Close(); err != nil {
			fmt.Printf("cannot close peerConnection: %v\n", err)
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

	// Register data channel creation handling
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", d.Label(), d.ID())

			for range time.NewTicker(5 * time.Second).C {
				message := fmt.Sprintf("reply %d", time.Now().UnixNano())
				fmt.Printf("Sending '%s'\n", message)

				// Send the message as text
				sendTextErr := d.SendText(message)
				if sendTextErr != nil {
					panic(sendTextErr)
				}
			}
		})

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
		})
	})

	// Start HTTP server that accepts requests from the offer process to exchange SDP and Candidates
	// nolint: gosec
	// panic(http.ListenAndServe(*answerAddr, nil))
	setRemoteDescription(&candidatesMux, pendingCandidates, peerConnection)
	select {}
}

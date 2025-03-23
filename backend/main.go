package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"

	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Configuration
	serverAddr = flag.String("addr", ":8080", "WebSocket server address")
	chunkSize  = flag.Int("chunk-size", 8*1024*1024, "Size of test data chunks in bytes")
)

type SpeedTestMessage struct {
	Type     string  `json:"type"`
	Speed    float64 `json:"speed,omitempty"`    // Speed in Mbps
	Average  float64 `json:"average,omitempty"`
	Duration int     `json:"duration,omitempty"`
}

type SpeedTest struct {
	mu        sync.Mutex
	active    bool
	speeds    []float64
	startTime time.Time
	ctx       context.Context
	cancel    context.CancelFunc
}

func (st *SpeedTest) start() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.active = true
	st.speeds = make([]float64, 0)
	st.startTime = time.Now()
	st.ctx, st.cancel = context.WithCancel(context.Background())
}

func (st *SpeedTest) stop() {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.cancel != nil {
		st.cancel()
	}
	st.active = false
}

func (st *SpeedTest) addSpeed(speed float64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.active {
		st.speeds = append(st.speeds, speed)
	}
}

func (st *SpeedTest) getAverage() float64 {
	st.mu.Lock()
	defer st.mu.Unlock()
	if len(st.speeds) == 0 {
		return 0
	}
	sum := 0.0
	for _, speed := range st.speeds {
		sum += speed
	}
	return sum / float64(len(st.speeds))
}

// generateTestData creates a buffer of random data for testing
func generateTestData() []byte {
	data := make([]byte, *chunkSize)
	if _, err := rand.Read(data); err != nil {
		log.Printf("Error generating test data: %v", err)
		return nil
	}
	return data
}

// measureSpeed calculates speed in Mbps
func measureSpeed(bytes int64, duration time.Duration) float64 {
	bits := float64(bytes * 8)
	seconds := duration.Seconds()
	if seconds == 0 {
		return 0
	}
	return (bits / 1000000) / seconds // Convert to Mbps
}

func runSpeedTest(conn *websocket.Conn, speedTest *SpeedTest, duration int) {
	// Run tests for the specified duration
	endTime := time.Now().Add(time.Duration(duration) * time.Second)
	for time.Now().Before(endTime) && speedTest.active {
		select {
		case <-speedTest.ctx.Done():
			return
		default:
			// Generate test data
			testData := generateTestData()
			if testData == nil {
				return
			}

			// Send test data
			start := time.Now()
			if err := conn.WriteMessage(websocket.BinaryMessage, testData); err != nil {
				log.Printf("Write error: %v", err)
				return
			}

			// Calculate speed
			speed := measureSpeed(int64(len(testData)), time.Since(start))
			speedTest.addSpeed(speed)

			// Send speed update
			msg := SpeedTestMessage{
				Type:  "speed",
				Speed: speed,
			}

			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("Write error: %v", err)
				return
			}

			time.Sleep(time.Second)
		}
	}

	// Send final average if test completed successfully
	if speedTest.active {
		speedTest.stop()
		finalMsg := SpeedTestMessage{
			Type:    "final",
			Average: speedTest.getAverage(),
		}
		if err := conn.WriteJSON(finalMsg); err != nil {
			log.Printf("Write error: %v", err)
		}
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	speedTest := &SpeedTest{}

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Read error: %v", err)
			break
		}

		if messageType == websocket.TextMessage {
			var msg SpeedTestMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("JSON unmarshal error: %v", err)
				continue
			}

			if msg.Type == "start" {
				speedTest.start()
				duration := msg.Duration
				if duration == 0 {
					duration = 10
				}
				go runSpeedTest(conn, speedTest, duration)
			} else if msg.Type == "stop" {
				speedTest.stop()
			}
		}
	}
}

func main() {
	flag.Parse()

	// Start the WebSocket server
	http.HandleFunc("/ws", handleWebSocket)
	log.Printf("Starting WebSocket server on %s", *serverAddr)
	if err := http.ListenAndServe(*serverAddr, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
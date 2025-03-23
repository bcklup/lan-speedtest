package main

import (
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type SpeedTestMessage struct {
	Type     string  `json:"type"`
	Speed    float64 `json:"speed,omitempty"`
	Average  float64 `json:"average,omitempty"`
	Duration int     `json:"duration,omitempty"`
}

type SpeedTest struct {
	mu        sync.Mutex
	active    bool
	speeds    []float64
	startTime time.Time
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (st *SpeedTest) start() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.active = true
	st.speeds = make([]float64, 0)
	st.startTime = time.Now()
}

func (st *SpeedTest) stop() {
	st.mu.Lock()
	defer st.mu.Unlock()
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
					duration = 10 // default to 10 seconds if not specified
				}
				go runSpeedTest(conn, speedTest, duration)
			} else if msg.Type == "stop" {
				speedTest.stop()
				// Send final average
				finalMsg := SpeedTestMessage{
					Type:    "final",
					Average: speedTest.getAverage(),
				}
				if err := conn.WriteJSON(finalMsg); err != nil {
					log.Printf("Write error: %v", err)
					break
				}
			}
		}
	}
}

func runSpeedTest(conn *websocket.Conn, speedTest *SpeedTest, duration int) {
	// Simulate speed test for the specified duration
	for i := 0; i < duration; i++ {
		if !speedTest.active {
			break
		}

		// Simulate random speed between 100-1000 Mbps (typical LAN speeds)
		speed := 100.0 + math.Floor(rand.Float64()*900.0)
		speedTest.addSpeed(speed)

		msg := SpeedTestMessage{
			Type:  "speed",
			Speed: speed,
		}

		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("Write error: %v", err)
			break
		}

		time.Sleep(time.Second)
	}

	// If test completed successfully, send final message
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

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	log.Println("Starting speed test server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
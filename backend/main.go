package main

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

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
}

const (
	chunkSize    = 1024 * 1024  // 1MB chunks for testing
	testPort     = ":3001"      // Port for speed test TCP server
	warmupPeriod = time.Second   // Initial warmup period
)

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

// generateTestData creates a buffer of random data for testing
func generateTestData() []byte {
	data := make([]byte, chunkSize)
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

// runDownloadTest measures download speed
func runDownloadTest() (float64, error) {
	// Create a new connection for each test
	conn, err := net.Dial("tcp", "localhost"+testPort)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	start := time.Now()
	buffer := make([]byte, chunkSize)
	totalBytes := int64(0)

	// Read data until we get EOF or an error
	for {
		n, err := conn.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		totalBytes += int64(n)
	}

	speed := measureSpeed(totalBytes, time.Since(start))
	return speed, nil
}

// startSpeedTestServer starts a TCP server for speed testing
func startSpeedTestServer() {
	listener, err := net.Listen("tcp", testPort)
	if err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleSpeedTestConn(conn)
	}
}

func handleSpeedTestConn(conn net.Conn) {
	defer conn.Close()

	// Generate and send test data
	testData := generateTestData()
	if testData == nil {
		return
	}

	// Send data for download test
	if _, err := conn.Write(testData); err != nil {
		log.Printf("Error sending test data: %v", err)
		return
	}
}

func runSpeedTest(conn *websocket.Conn, speedTest *SpeedTest, duration int) {
	// Run tests for the specified duration
	endTime := time.Now().Add(time.Duration(duration) * time.Second)
	for time.Now().Before(endTime) && speedTest.active {
		// Run download test
		speed, err := runDownloadTest()
		if err != nil {
			log.Printf("Download test error: %v", err)
			break
		}

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
	// Start the TCP speed test server in a goroutine
	go startSpeedTestServer()

	// Start the WebSocket server
	http.HandleFunc("/ws", handleWebSocket)
	log.Println("Starting speed test server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
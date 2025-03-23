import { useState, useEffect, useRef } from "react";

type SpeedUnit = "bits" | "bytes";
type TestDuration = 5 | 10 | 15 | 25;

// Add Timer type to fix NodeJS namespace error
type Timer = ReturnType<typeof setInterval>;

function App() {
  const [isRunning, setIsRunning] = useState(false);
  const [currentSpeed, setCurrentSpeed] = useState<number | null>(null);
  const [averageSpeed, setAverageSpeed] = useState<number | null>(null);
  const [unit, setUnit] = useState<SpeedUnit>("bits");
  const [duration, setDuration] = useState<TestDuration>(10);
  const [progress, setProgress] = useState(0);
  const wsRef = useRef<WebSocket | null>(null);
  const progressInterval = useRef<Timer | null>(null);
  const speeds = useRef<number[]>([]);
  const pendingStart = useRef(false);

  const connectWebSocket = () => {
    return new Promise<void>((resolve) => {
      // Close existing connection if any
      if (wsRef.current) {
        wsRef.current.close();
      }

      // Get the hostname from the current window location
      const hostname = window.location.hostname;
      wsRef.current = new WebSocket(`ws://${hostname}:8080/ws`);

      wsRef.current.onopen = () => {
        resolve();
      };

      wsRef.current.onmessage = (event) => {
        const data = JSON.parse(event.data);

        if (data.type === "speed") {
          setCurrentSpeed(data.speed);
          speeds.current.push(data.speed);
        } else if (data.type === "final") {
          setAverageSpeed(data.average);
          setIsRunning(false);
          if (progressInterval.current) {
            clearInterval(progressInterval.current);
          }
          setProgress(100);
          wsRef.current?.close();
        }
      };
    });
  };

  useEffect(() => {
    connectWebSocket();

    return () => {
      if (progressInterval.current) {
        clearInterval(progressInterval.current);
      }
      wsRef.current?.close();
    };
  }, [duration]);

  const startTest = async () => {
    if (pendingStart.current) return;
    pendingStart.current = true;

    setCurrentSpeed(null);
    setAverageSpeed(null);
    setProgress(0);
    speeds.current = [];
    setIsRunning(true);

    try {
      // Ensure we have a WebSocket connection
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
        await connectWebSocket();
      }

      // Send start message
      wsRef.current?.send(JSON.stringify({ type: "start", duration }));

      // Start progress tracking
      if (progressInterval.current) {
        clearInterval(progressInterval.current);
      }

      progressInterval.current = setInterval(() => {
        setProgress((prev) => {
          const newProgress = Math.min(prev + 100 / (duration * 20), 100);
          return newProgress;
        });
      }, 50);
    } catch (error) {
      console.error("Failed to start test:", error);
      setIsRunning(false);
    } finally {
      pendingStart.current = false;
    }
  };

  const stopTest = () => {
    if (progressInterval.current) {
      clearInterval(progressInterval.current);
    }
    wsRef.current?.send(JSON.stringify({ type: "stop" }));
    setIsRunning(false);
  };

  const restartTest = () => {
    startTest();
  };

  const toggleUnit = () => {
    setUnit((prev) => (prev === "bits" ? "bytes" : "bits"));
  };

  const toggleDuration = () => {
    setDuration((prev) => {
      switch (prev) {
        case 5:
          return 10;
        case 10:
          return 15;
        case 15:
          return 25;
        case 25:
          return 5;
        default:
          return 10;
      }
    });
  };

  const formatSpeed = (speed: number | null) => {
    if (speed === null) return "0";

    // Convert to appropriate unit (bits or bytes)
    let value = speed;
    if (unit === "bytes") {
      value = speed / 8; // Convert from bits to bytes
    }

    // Format with appropriate prefix (M for mega, G for giga)
    if (value >= 1000) {
      return `${(value / 1000).toFixed(2)}`;
    }
    return `${Math.floor(value)}`;
  };

  const getUnitText = (speed: number | null) => {
    if (speed === null) return unit === "bits" ? "Mbps" : "MB/s";

    if (unit === "bits") {
      return speed >= 1000 ? "Gbps" : "Mbps";
    } else {
      return speed >= 8000 ? "GB/s" : "MB/s";
    }
  };

  return (
    <div className="min-h-screen w-full bg-[#f5f2e8] flex flex-col items-center justify-center p-4">
      <div className="max-w-2xl w-full text-center flex flex-col items-center bg-[#f5f2e8]">
        <h1 className="text-3xl font-medium text-[#373b4d] mb-6">
          LAN SPEED TEST
        </h1>

        {!isRunning && !averageSpeed ? (
          // Initial state - START button
          <button
            onClick={startTest}
            className="text-[120px] font-bold text-[#373b4d] leading-none tracking-tight hover:text-[#7c9a92] transition-colors cursor-pointer md:text-[180px]"
          >
            START
          </button>
        ) : (
          // Testing or Results state
          <div className="flex flex-col items-center gap-12">
            <div className="flex items-baseline justify-center gap-6 min-w-[600px]">
              <div
                className="text-[120px] text-[#373b4d] font-bold leading-none tracking-tight text-center md:text-[180px]"
                style={{ opacity: isRunning ? 0.3 : 1 }}
              >
                {formatSpeed(isRunning ? currentSpeed : averageSpeed)}
              </div>

              <button
                onClick={toggleUnit}
                className="text-3xl font-medium text-[#7c9a92] hover:text-[#6b8a82] transition-colors cursor-pointer"
              >
                {getUnitText(isRunning ? currentSpeed : averageSpeed)}
              </button>
            </div>

            {isRunning && (
              <>
                <div className="w-full max-w-[360px] px-4 md:max-w-[600px]">
                  <div className="w-full bg-[#e8f0eb] rounded-full h-1 px-2">
                    <div
                      className="bg-[#7c9a92] h-1 rounded-full transition-all duration-300"
                      style={{ width: `${progress}%` }}
                    />
                  </div>
                </div>

                <button
                  onClick={stopTest}
                  className="w-12 h-12 rounded-full border-2 border-[#7c9a92] flex items-center justify-center hover:bg-[#7c9a92] hover:text-white transition-colors group cursor-pointer"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    className="h-6 w-6 text-[#7c9a92] group-hover:text-white transition-colors"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M6 18L18 6M6 6l12 12"
                    />
                  </svg>
                </button>
              </>
            )}

            {!isRunning && averageSpeed && (
              <button
                onClick={restartTest}
                className="w-12 h-12 rounded-full border-2 border-[#7c9a92] flex items-center justify-center hover:bg-[#7c9a92] hover:text-white transition-colors group cursor-pointer"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  className="h-6 w-6 text-[#7c9a92] group-hover:text-white transition-colors"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                  />
                </svg>
              </button>
            )}
          </div>
        )}

        {/* Duration Toggle */}
        <div className="fixed bottom-4 md:bottom-8 md:right-8">
          <button
            onClick={toggleDuration}
            className="text-lg font-medium text-[#7c9a92] hover:text-[#6b8a82] transition-colors cursor-pointer"
          >
            {duration} seconds
          </button>
        </div>
      </div>
    </div>
  );
}

export default App;

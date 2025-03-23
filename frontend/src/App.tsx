import { useState, useEffect, useRef } from "react";

type SpeedUnit = "bits" | "bytes";

function App() {
  const [isRunning, setIsRunning] = useState(false);
  const [currentSpeed, setCurrentSpeed] = useState<number | null>(null);
  const [averageSpeed, setAverageSpeed] = useState<number | null>(null);
  const [unit, setUnit] = useState<SpeedUnit>("bits");
  const [progress, setProgress] = useState(0);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    wsRef.current = new WebSocket("ws://localhost:8080/ws");

    wsRef.current.onmessage = (event) => {
      const data = JSON.parse(event.data);

      if (data.type === "speed") {
        setCurrentSpeed(data.speed);
        setProgress((prev) => Math.min(prev + 5, 100));
      } else if (data.type === "final") {
        setAverageSpeed(data.average);
        setIsRunning(false);
      }
    };

    return () => {
      wsRef.current?.close();
    };
  }, []);

  const startTest = () => {
    setCurrentSpeed(null);
    setAverageSpeed(null);
    setProgress(0);
    setIsRunning(true);
    wsRef.current?.send(JSON.stringify({ type: "start" }));
  };

  const stopTest = () => {
    setIsRunning(false);
    wsRef.current?.send(JSON.stringify({ type: "stop" }));
  };

  const toggleUnit = () => {
    setUnit((prev) => (prev === "bits" ? "bytes" : "bits"));
  };

  const formatSpeed = (speed: number | null) => {
    if (speed === null) return "0";

    // Convert to appropriate unit (bits or bytes)
    let value = speed;
    if (unit === "bytes") {
      value = speed / 8;
    }

    // Format with appropriate prefix (M for mega, G for giga)
    if (value >= 1000) {
      return `${(value / 1000).toFixed(1)}`;
    }
    return `${Math.floor(value)}`;
  };

  return (
    <div className="min-h-screen w-full bg-[#f5f2e8] flex flex-col items-center justify-center p-4">
      <div className="max-w-2xl w-full text-center space-y-8">
        <div className="space-y-2">
          <h1 className="text-3xl font-medium text-[#373b4d] mb-8">
            LAN SPEED TEST
          </h1>

          <div className="relative">
            {isRunning && currentSpeed ? (
              <>
                <div
                  className="text-[120px] font-bold text-[#373b4d] leading-none tracking-tight"
                  style={{ opacity: isRunning || currentSpeed ? 1 : 0.3 }}
                >
                  {formatSpeed(currentSpeed)}
                </div>
                <button
                  onClick={toggleUnit}
                  className="text-2xl font-medium text-[#7c9a92] hover:text-[#6b8a82] transition-colors mt-2"
                >
                  {unit === "bits" ? "MBPS" : "MB/s"}
                </button>
              </>
            ) : (
              <div className="absolute inset-0 flex items-center justify-center">
                <button
                  onClick={startTest}
                  className="text-xl font-medium text-[#7c9a92] hover:text-[#6b8a82] transition-colors"
                >
                  START
                </button>
              </div>
            )}
          </div>

          {isRunning && (
            <div className="w-full max-w-md mx-auto mt-8">
              <div className="w-full bg-[#e8f0eb] rounded-full h-1">
                <div
                  className="bg-[#7c9a92] h-1 rounded-full transition-all duration-300"
                  style={{ width: `${progress}%` }}
                />
              </div>
            </div>
          )}

          {averageSpeed !== null && (
            <div className="text-lg text-[#373b4d] mt-4">
              Average: {formatSpeed(averageSpeed)}{" "}
              {unit === "bits" ? "MBPS" : "MB/s"}
            </div>
          )}

          {isRunning && (
            <button
              onClick={stopTest}
              className="text-lg font-medium text-[#373b4d] hover:text-[#7c9a92] transition-colors mt-4"
            >
              STOP
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

export default App;

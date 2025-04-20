package main

import (
	"encoding/json"
	"log"
	"net/http"

	"tcping-node/tcping"
)

type TcpingRequest struct {
	EventID string `json:"eventId"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
}

type TcpingResponse struct {
	EventID  string  `json:"eventId"`
	Success  bool    `json:"success"`
	Duration float64 `json:"duration"`
	Error    string  `json:"error,omitempty"`
}

func main() {
	http.HandleFunc("/tcping", handleTcping)
	http.HandleFunc("/", handleDashboard)
	http.HandleFunc("/api/tcping", handleTcping) // 将原来的tcping移到api路径下

	log.Printf("服务器启动在 0.0.0.0:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}

func handleTcping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持POST请求", http.StatusMethodNotAllowed)
		return
	}

	var req TcpingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	success, duration, err := tcping.Test(req.Host, req.Port)

	resp := TcpingResponse{
		EventID:  req.EventID,
		Success:  success,
		Duration: duration,
	}

	if err != nil {
		resp.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "dashboard.html")
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"tcping-node/tcping"
)

func main() {
	// 添加命令行参数
	port := flag.Int("p", 8081, "设置监听端口")
	flag.Parse()

	http.HandleFunc("/api/tcping", handleTcping)
	http.HandleFunc("/health", handleHealth) // 添加健康检查端点
	address := fmt.Sprintf("0.0.0.0:%d", *port)
	log.Printf("Agent 启动在 %s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

func handleTcping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "仅支持POST请求", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效的请求数据", http.StatusBadRequest)
		return
	}

	success, duration, err := tcping.Test(req.Host, req.Port)

	resp := map[string]interface{}{
		"success":  success,
		"duration": duration,
	}
	if err != nil {
		resp["error"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// 新增健康检查处理函数
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"status": true})
}

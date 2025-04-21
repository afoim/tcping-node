package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"
)

// 硬编码节点列表
var nodes = []struct {
	Name string
	URL  string
}{
	{"安徽合肥移动PC-家宽", "http://192.168.124.15:8081"},
	{"安徽合肥移动NAS-家宽", "http://192.168.124.14:8081"},
	{"Akile 台湾HiNET-家宽", "http://tw.072103.xyz:20347"},
	{"雨云 台湾3区-商宽", "http://154.37.213.180:8081"},
}

func main() {
	http.HandleFunc("/", handleDashboard)
	http.HandleFunc("/api/test", handleTest)
	http.HandleFunc("/api/check-nodes", handleCheckNodes) // 添加节点状态检查路由

	log.Printf("Dashboard 启动在 0.0.0.0:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>TCPing Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .form-group { margin: 10px 0; }
        .result { margin-top: 20px; }
        .success { color: green; }
        .error { color: red; }
        .nodes { margin: 20px 0; }
        #nodeStatus { margin: 20px 0; }
        .node-item { margin: 5px 0; }
    </style>
</head>
<body>
    <h1>TCPing Dashboard</h1>
    <div>
        <button onclick="checkNodesStatus()">检查节点状态</button>
        <div id="nodeStatus"></div>
    </div>
    <hr>
    <h2>TCP测试</h2>
    <div class="form-group">
        <label>主机:</label>
        <input type="text" id="host" value="example.com">
    </div>
    <div class="form-group">
        <label>端口:</label>
        <input type="number" id="port" value="80">
    </div>
    <button onclick="runTest()">开始测试</button>
    <div id="result" class="result"></div>

    <script>
        async function runTest() {
            const host = document.getElementById('host').value;
            const port = parseInt(document.getElementById('port').value);
            const resultDiv = document.getElementById('result');
            resultDiv.innerHTML = '测试中...';

            try {
                const response = await fetch('/api/test', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({host, port})
                });

                const results = await response.json();
                resultDiv.innerHTML = Object.entries(results).map(([nodeName, result]) => {
                    if (result.success) {
                        return '<div class="success">' + nodeName + ': ✓ 连接成功! 耗时: ' + result.duration.toFixed(3) + '秒</div>';
                    } else {
                        return '<div class="error">' + nodeName + ': ✗ 连接失败: ' + result.error + '</div>';
                    }
                }).join('');
            } catch (err) {
                resultDiv.innerHTML = '<div class="error">✗ 请求错误: ' + err.message + '</div>';
            }
        }

        async function checkNodesStatus() {
            const statusDiv = document.getElementById('nodeStatus');
            statusDiv.innerHTML = '正在检查节点状态...';
            
            try {
                const response = await fetch('/api/check-nodes', {
                    method: 'POST',
                });
                
                const results = await response.json();
                statusDiv.innerHTML = Object.entries(results).map(([nodeName, status]) => {
                    const statusClass = status ? 'success' : 'error';
                    const statusText = status ? '在线' : '离线';
                    return '<div class="node-item"><span class="' + statusClass + 
                           '">● </span>' + nodeName + ': ' + statusText + '</div>';
                }).join('');
            } catch (err) {
                statusDiv.innerHTML = '<div class="error">检查节点状态时出错: ' + err.message + '</div>';
            }
        }
    </script>
</body>
</html>`
	t := template.Must(template.New("dashboard").Parse(tmpl))
	t.Execute(w, nodes)
}

func handleTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results := make(map[string]interface{})
	var wg sync.WaitGroup
	resultsMu := sync.Mutex{}

	for _, node := range nodes {
		wg.Add(1)
		go func(node struct{ Name, URL string }) {
			defer wg.Done()
			reqBody, err := json.Marshal(req)
			if err != nil {
				resultsMu.Lock()
				results[node.Name] = map[string]interface{}{
					"success": false,
					"error":   "请求序列化失败: " + err.Error(),
				}
				resultsMu.Unlock()
				return
			}

			resp, err := http.Post(
				node.URL+"/api/tcping",
				"application/json",
				bytes.NewBuffer(reqBody),
			)

			var result interface{}
			if err != nil {
				result = map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}
			} else {
				defer resp.Body.Close()
				json.NewDecoder(resp.Body).Decode(&result)
			}

			resultsMu.Lock()
			results[node.Name] = result
			resultsMu.Unlock()
		}(node)
	}

	wg.Wait()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// 添加节点状态检查处理函数
func handleCheckNodes(w http.ResponseWriter, r *http.Request) {
	results := make(map[string]bool)
	var wg sync.WaitGroup
	resultsMu := sync.Mutex{}

	for _, node := range nodes {
		wg.Add(1)
		go func(node struct{ Name, URL string }) {
			defer wg.Done()

			client := http.Client{
				Timeout: 5 * time.Second,
			}

			resp, err := client.Get(node.URL + "/health")
			isOnline := err == nil && resp.StatusCode == http.StatusOK

			if resp != nil {
				resp.Body.Close()
			}

			resultsMu.Lock()
			results[node.Name] = isOnline
			resultsMu.Unlock()
		}(node)
	}

	wg.Wait()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

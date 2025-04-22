package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// 定义Node结构体
type Node struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// 定义配置结构体
type Config struct {
	Nodes []Node `json:"nodes"`
}

var (
	nodes       []Node
	onlineNodes = make(map[string]bool) // 记录节点在线状态
	nodesMutex  sync.RWMutex
)

func init() {
	// 读取节点配置文件
	data, err := os.ReadFile("nodes.json")
	if err != nil {
		log.Fatalf("无法读取节点配置文件: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatalf("解析节点配置文件失败: %v", err)
	}

	nodes = config.Nodes
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
        .button-disabled { 
            background-color: #cccccc;
            cursor: not-allowed;
        }
        .warning {
            color: #ff6b6b;
            margin: 10px 0;
        }
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
    <div id="testWarning" class="warning" style="display:none;">
        请先检查节点状态，确保所有节点在线后再进行测试
    </div>
    <div class="form-group">
        <label>主机:</label>
        <input type="text" id="host" value="example.com">
    </div>
    <div class="form-group">
        <label>端口:</label>
        <input type="number" id="port" value="80">
    </div>
    <button id="testButton" onclick="runTest()" disabled class="button-disabled">开始测试</button>
    <div id="result" class="result"></div>

    <script>
        let allNodesOffline = true;  // 添加全部离线标记

        function sanitizeError(error) {
            // 移除可能包含的IP地址和端口信息
            return error.replace(/(\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b(:\d+)?)/g, "节点")
                        .replace(/dial tcp.*:/g, "连接失败: ")
                        .replace(/connect: connection refused/g, "目标拒绝连接")
                        .replace(/connect: network is unreachable/g, "网络不可达")
                        .replace(/no such host/g, "域名解析失败")
                        .replace(/i\/o timeout/g, "连接超时")
                        .replace(/hostname resolving error/g, "域名解析错误");
        }

        async function runTest() {
            if (allNodesOffline) {
                document.getElementById('testWarning').style.display = 'block';
                return;
            }
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
                        return '<div class="success">' + nodeName + ': ✓ 连接成功! 延迟: ' + (result.duration * 1000).toFixed(1) + 'ms</div>';
                    } else {
                        return '<div class="error">' + nodeName + ': ✗ 连接失败: ' + sanitizeError(result.error) + '</div>';
                    }
                }).join('');
            } catch (err) {
                resultDiv.innerHTML = '<div class="error">✗ 请求错误: ' + err.message + '</div>';
            }
        }

        async function checkNodesStatus() {
            const statusDiv = document.getElementById('nodeStatus');
            const testButton = document.getElementById('testButton');
            const testWarning = document.getElementById('testWarning');
            
            statusDiv.innerHTML = '正在检查节点状态...';
            testButton.disabled = true;
            testButton.classList.add('button-disabled');
            
            try {
                const response = await fetch('/api/check-nodes', {
                    method: 'POST',
                });
                
                const results = await response.json();
                const offlineNodes = [];
                const onlineCount = Object.values(results).filter(status => status).length;
                allNodesOffline = onlineCount === 0;
                
                statusDiv.innerHTML = Object.entries(results).map(([nodeName, status]) => {
                    const statusClass = status ? 'success' : 'error';
                    const statusText = status ? '在线' : '离线';
                    if (!status) offlineNodes.push(nodeName);
                    return '<div class="node-item"><span class="' + statusClass + 
                           '">● </span>' + nodeName + ': ' + statusText + '</div>';
                }).join('');

                if (allNodesOffline) {
                    testButton.disabled = true;
                    testButton.classList.add('button-disabled');
                    testWarning.style.display = 'block';
                    testWarning.innerHTML = '所有节点离线，无法进行测试';
                } else {
                    testButton.disabled = false;
                    testButton.classList.remove('button-disabled');
                    testWarning.style.display = offlineNodes.length > 0 ? 'block' : 'none';
                    if (offlineNodes.length > 0) {
                        testWarning.innerHTML = '以下节点离线，将从测试中排除：<br>' + offlineNodes.join('<br>');
                    }
                }
            } catch (err) {
                statusDiv.innerHTML = '<div class="error">检查节点状态时出错: ' + err.message + '</div>';
                testButton.disabled = true;
                testButton.classList.add('button-disabled');
            }
        }

        // 页面加载时自动检查节点状态
        window.onload = checkNodesStatus;
    </script>
</body>
</html>`
	t := template.Must(template.New("dashboard").Parse(tmpl))
	t.Execute(w, nodes)
}

// 添加错误信息处理函数
func sanitizeErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()

	// 检查是否是节点连接错误
	if strings.Contains(msg, "dial tcp") || strings.Contains(msg, "Post") ||
		strings.Contains(msg, "connect:") || strings.Contains(msg, "http:") {
		return "节点离线"
	}

	// 目标服务错误信息替换
	errorMappings := map[string]string{
		"connection refused":           "目标服务拒绝连接",
		"network is unreachable":       "目标网络不可达",
		"no such host":                 "域名解析失败",
		"i/o timeout":                  "连接超时",
		"certificate has expired":      "证书过期",
		"certificate is not yet valid": "证书未生效",
		"no route to host":             "无法路由到目标",
	}

	for pattern, replacement := range errorMappings {
		if strings.Contains(strings.ToLower(msg), pattern) {
			return replacement
		}
	}

	// 如果没有匹配的错误模式，返回通用错误
	return "测试失败"
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

	nodesMutex.RLock()
	hasOnlineNodes := false
	for _, isOnline := range onlineNodes {
		if isOnline {
			hasOnlineNodes = true
			break
		}
	}
	nodesMutex.RUnlock()

	if !hasOnlineNodes {
		http.Error(w, "所有节点离线", http.StatusServiceUnavailable)
		return
	}

	results := make(map[string]interface{})
	var wg sync.WaitGroup
	resultsMu := sync.Mutex{}

	for _, node := range nodes {
		nodesMutex.RLock()
		isOnline := onlineNodes[node.Name]
		nodesMutex.RUnlock()

		if !isOnline {
			continue // 跳过离线节点
		}

		wg.Add(1)
		go func(node Node) {
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
					"error":   sanitizeErrorMessage(err),
				}
			} else {
				defer resp.Body.Close()
				var respData map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					result = map[string]interface{}{
						"success": false,
						"error":   "数据解析失败",
					}
				} else {
					// 处理来自agent的错误信息
					if errMsg, ok := respData["error"].(string); ok && errMsg != "" {
						respData["error"] = sanitizeErrorMessage(fmt.Errorf(errMsg))
					}
					result = respData
				}
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
		go func(node Node) {
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
			// 更新在线节点状态
			nodesMutex.Lock()
			onlineNodes[node.Name] = isOnline
			nodesMutex.Unlock()
			resultsMu.Unlock()
		}(node)
	}

	wg.Wait()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

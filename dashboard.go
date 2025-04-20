package main

import (
	"html/template"
	"net/http"
)

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
    </style>
</head>
<body>
    <h1>TCPing Dashboard</h1>
    <div class="form-group">
        <label>主机:</label>
        <input type="text" id="host" value="example.com">
    </div>
    <div class="form-group">
        <label>端口:</label>
        <input type="number" id="port" value="80">
    </div>
    <button onclick="runTest()">测试</button>
    <div id="result" class="result"></div>

    <script>
        async function runTest() {
            const host = document.getElementById('host').value;
            const port = parseInt(document.getElementById('port').value);
            const resultDiv = document.getElementById('result');

            try {
                const response = await fetch('/api/tcping', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        eventId: new Date().getTime().toString(),
                        host: host,
                        port: port
                    })
                });

                const data = await response.json();
                if (data.success) {
                    resultDiv.innerHTML = '<div class="success">✓ 连接成功! 耗时: ' + data.duration.toFixed(3) + '秒</div>';
                } else {
                    resultDiv.innerHTML = '<div class="error">✗ 连接失败: ' + data.error + '</div>';
                }
            } catch (err) {
                resultDiv.innerHTML = '<div class="error">✗ 请求错误: ' + err.message + '</div>';
            }
        }
    </script>
</body>
</html>
`
	t := template.Must(template.New("dashboard").Parse(tmpl))
	t.Execute(w, nil)
}

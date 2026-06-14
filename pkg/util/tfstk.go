package util

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"
)

// GenerateTFstk 通过 Node.js 子进程生成 tfstk cookie 值。
//
// 该函数调用 assets/gen_tfstk.js 脚本，脚本内部加载 et_f.js
// 阿里追踪 SDK 并在沙箱环境中执行以生成 tracking cookie。
//
// 参数:
//   - scriptPath: gen_tfstk.js 脚本的绝对路径
//
// 返回值:
//   - string: tfstk 值
//   - error: 执行失败时的错误
func GenerateTFstk(scriptPath string) (string, error) {
	cmd := exec.Command("node", scriptPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// 设置 30 秒超时，防止 SDK 卡死
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("util: tfstk generation failed: %w (stderr: %s)", err, stderr.String())
		}
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		return "", fmt.Errorf("util: tfstk generation timed out after 30s")
	}

	return string(bytes.TrimSpace(stdout.Bytes())), nil
}

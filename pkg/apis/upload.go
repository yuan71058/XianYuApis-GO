package apis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
)

// UploadMediaResult 媒体上传结果。
type UploadMediaResult struct {
	URL    string // 图片访问 URL
	Width  int    // 图片宽度
	Height int    // 图片高度
}

// UploadMedia 上传媒体文件（图片）到闲鱼服务器。
//
// 支持 PNG/JPG/JPEG/GIF 格式。上传成功后返回图片的访问 URL 和尺寸信息。
//
// 参数:
//   - ctx:      请求上下文，可用于超时控制
//   - filePath: 本地文件路径
//
// 返回值:
//   - *UploadMediaResult: 上传结果，包含 URL 和尺寸
//   - error: 上传失败时的错误
func (api *XianyuAPI) UploadMedia(ctx context.Context, filePath string) (*UploadMediaResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("apis: open file %s: %w", filePath, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("apis: stat file: %w", err)
	}

	// 构建 multipart body
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", stat.Name())
	if err != nil {
		return nil, fmt.Errorf("apis: create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("apis: write file to form: %w", err)
	}
	writer.Close()

	params := url.Values{
		"floderId":       {"0"},
		"appkey":         {"xy_chat"},
		"_input_charset": {"utf-8"},
	}

	urlStr := api.uploadURL + "/api/upload.api?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, &buf)
	if err != nil {
		return nil, fmt.Errorf("apis: new request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Origin", "https://www.goofish.com")
	req.Header.Set("Referer", "https://www.goofish.com/")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apis: do upload request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Object struct {
			URL string `json:"url"`
			Pix string `json:"pix"` // 格式: "widthxheight"
		} `json:"object"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("apis: decode upload response: %w", err)
	}

	// 解析尺寸
	var w, h int
	fmt.Sscanf(result.Object.Pix, "%dx%d", &w, &h)

	return &UploadMediaResult{
		URL:    result.Object.URL,
		Width:  w,
		Height: h,
	}, nil
}

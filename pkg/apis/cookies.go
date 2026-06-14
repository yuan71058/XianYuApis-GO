package apis

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/cv-cat/xianyuapis/pkg/util"
)

// BuildInitialCookies 纯 HTTP 方式构建初始 Cookie（不含登录态）。
//
// 该函数通过访问闲鱼相关接口获取基础 Cookie，不包含用户登录信息，
// 仅作为扫码登录的前置步骤。
//
// 返回值:
//   - *http.Client: 带 CookieJar 的 HTTP 客户端（含初始 Cookie）
//   - map[string]string: Cookie 字典，包含 cna、_m_h5_tk、cookie2、tfstk 等
//   - error: 构建失败时的错误
func BuildInitialCookies() (*http.Client, map[string]string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("cookies: create jar: %w", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 15 * time.Second,
	}

	// 1. 访问 log.mmstat.com 获取 cna
	resp, err := client.Get("https://log.mmstat.com/eg.js")
	if err == nil {
		resp.Body.Close()
	}
	cnaMMStat := util.GetCookieFromJar(jar, ".mmstat.com", "cna")
	if cnaMMStat != "" {
		util.SetCookieToJar(jar, ".goofish.com", "cna", cnaMMStat, "/")
	}

	// 2. 调用 mtop 接口获取 _m_h5_tk 和 cookie2
	// 注意: 不能使用 PostForm，因为 data=%7B%7D 会被二次编码成 %257B%257D
	// 必须手动构造 body 为 "data=%7B%7D"
	for _, apiName := range []string{
		"mtop.taobao.idlehome.home.webpc.feed",
		"mtop.gaia.nodejs.gaia.idle.data.gw.v2.index.get",
	} {
		params := url.Values{
			"jsv":           {"2.7.2"},
			"appKey":        {"34839810"},
			"t":             {fmt.Sprintf("%d", time.Now().UnixMilli())},
			"sign":          {""},
			"v":             {"1.0"},
			"type":          {"originaljson"},
			"dataType":      {"json"},
			"timeout":       {"20000"},
			"api":           {apiName},
			"sessionOption": {"AutoLoginOnly"},
			"spm_cnt":       {"a21ybx.home.0.0"},
		}

		reqURL := fmt.Sprintf("https://h5api.m.goofish.com/h5/%s/1.0/", apiName)
		req, _ := http.NewRequest(http.MethodPost, reqURL+"?"+params.Encode(),
			bytes.NewReader([]byte("data=%7B%7D")))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", UA)
		req.Header.Set("Origin", "https://www.goofish.com")
		req.Header.Set("Referer", "https://www.goofish.com/")

		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}

	// 3. 生成 tfstk
	tfstk, _ := util.GenerateTFstk("assets/gen_tfstk.js")
	if tfstk != "" {
		util.SetCookieToJar(jar, ".goofish.com", "tfstk", tfstk, "/")
	}

	cookies := map[string]string{}

	// 收集 .goofish.com 域的 Cookie
	if v := util.GetCookieFromJar(jar, ".goofish.com", "cna"); v != "" {
		cookies["cna"] = v
	} else if cnaMMStat != "" {
		cookies["cna"] = cnaMMStat
	}
	cookies["_m_h5_tk"] = util.GetCookieFromJar(jar, ".goofish.com", "_m_h5_tk")
	cookies["_m_h5_tk_enc"] = util.GetCookieFromJar(jar, ".goofish.com", "_m_h5_tk_enc")
	cookies["cookie2"] = util.GetCookieFromJar(jar, ".goofish.com", "cookie2")
	cookies["tfstk"] = tfstk
	cookies["xlly_s"] = "1"

	return client, cookies, nil
}

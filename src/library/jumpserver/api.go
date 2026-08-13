package jumpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lnatpunblhna/gopub/src/library/config"
)

// 对接 JumpServer 1.5.3 的资产/节点接口：先用用户名密码换 token，
// 再用 token 拉节点列表和节点下的资产 IP。
//
// 这里的每个请求都必须自己检查错误：jumpserver 地址默认指向一个示例域名
// （app.conf 的 jumpserver = "http://jump.test.com"），未对接时请求必然失败，
// 若沿用「忽略 error 直接 defer resp.Body.Close()」的写法就会空指针 panic。

// httpTimeout 限制单次请求耗时，避免 jumpserver 不可达时请求一直挂住
const httpTimeout = 10 * time.Second

var httpClient = &http.Client{Timeout: httpTimeout}

type authinfo struct {
	Token string `json:"Token"`
}

type asset struct {
	Ip       string `json:"ip"`
	Hostname string `json:"hostname"`
}

type node struct {
	Id    string `json:"id"`
	Value string `json:"value"`
}

// auth 用配置里的用户名密码换取访问 token
func auth() (string, error) {
	authAPIURL := config.String("jumpserver") + config.String("jump_auth_api")
	param, err := json.Marshal(map[string]string{
		"username": config.String("jump_username"),
		"password": config.String("jump_password"),
	})
	if err != nil {
		return "", fmt.Errorf("jumpserver 认证参数序列化失败: %w", err)
	}

	resp, err := httpClient.Post(authAPIURL, "application/json", strings.NewReader(string(param)))
	if err != nil {
		return "", fmt.Errorf("jumpserver 认证请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := readOK(resp, "认证")
	if err != nil {
		return "", err
	}
	rs := authinfo{}
	if err := json.Unmarshal(body, &rs); err != nil {
		return "", fmt.Errorf("jumpserver 认证响应解析失败: %w", err)
	}
	if rs.Token == "" {
		return "", fmt.Errorf("jumpserver 认证未返回 token，请检查 jump_username / jump_password")
	}
	return rs.Token, nil
}

// GetGroups 返回 JumpServer 上以 up_ 开头的节点，映射为 节点ID -> 节点名
func GetGroups() (map[string]string, error) {
	body, err := getWithToken(config.String("jumpserver")+config.String("jump_grouplist_api"), "节点列表")
	if err != nil {
		return nil, err
	}

	var rs []node
	if err := json.Unmarshal(body, &rs); err != nil {
		return nil, fmt.Errorf("jumpserver 节点列表响应解析失败: %w", err)
	}

	id2group := make(map[string]string)
	for _, nodeinfo := range rs {
		if strings.HasPrefix(nodeinfo.Value, "up_") {
			id2group[nodeinfo.Id] = nodeinfo.Value
		}
	}
	return id2group, nil
}

// GetIpsByGroupid 返回指定节点下的资产，映射为 IP -> 主机名
func GetIpsByGroupid(groupID string) (map[string]string, error) {
	url := config.String("jumpserver") + strings.Replace(config.String("jump_groupid2ips_api"), "%id", groupID, -1)
	body, err := getWithToken(url, "节点资产")
	if err != nil {
		return nil, err
	}

	var rs []asset
	if err := json.Unmarshal(body, &rs); err != nil {
		return nil, fmt.Errorf("jumpserver 节点资产响应解析失败: %w", err)
	}

	ip2hostname := make(map[string]string)
	for _, v := range rs {
		ip2hostname[v.Ip] = v.Hostname
	}
	return ip2hostname, nil
}

// getWithToken 先取 token，再带着 token 发 GET 请求并读回响应体
func getWithToken(url string, what string) ([]byte, error) {
	token, err := auth()
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("jumpserver %s请求构造失败: %w", what, err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("jumpserver %s请求失败: %w", what, err)
	}
	defer resp.Body.Close()

	return readOK(resp, what)
}

// readOK 校验 HTTP 状态码并读出响应体
func readOK(resp *http.Response, what string) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("jumpserver %s响应读取失败: %w", what, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jumpserver %s返回 HTTP %d: %s", what, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

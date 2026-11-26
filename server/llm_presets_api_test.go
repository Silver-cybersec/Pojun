package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func strReader(s string) *strings.Reader { return strings.NewReader(s) }

// 厂商知识层三端点（v0.4.91 配置重构）：预设清单 / 元数据推断 / 连通分诊。

func TestLLMPresetsEndpoint(t *testing.T) {
	s := &Server{}
	w := httptest.NewRecorder()
	s.llmPresets(w, httptest.NewRequest(http.MethodGet, "/api/llm/presets", nil))
	if w.Code != 200 {
		t.Fatalf("status=%d", w.Code)
	}
	var out struct {
		Presets []struct {
			ID     string `json:"id"`
			Format string `json:"format"`
		} `json:"presets"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Presets) != 32 {
		t.Fatalf("presets=%d，期望 32", len(out.Presets))
	}
	ids := map[string]bool{}
	for _, p := range out.Presets {
		ids[p.ID] = true
	}
	for _, want := range []string{"opencode-zen", "opencode-go", "openai", "anthropic", "siliconflow", "custom-relay"} {
		if !ids[want] {
			t.Fatalf("缺预设 %s", want)
		}
	}
}

func TestLLMModelMetaEndpoint(t *testing.T) {
	s := &Server{}
	get := func(model string) map[string]any {
		w := httptest.NewRecorder()
		s.llmModelMeta(w, httptest.NewRequest(http.MethodGet, "/api/llm/model-meta?model="+model, nil))
		if w.Code != 200 {
			t.Fatalf("status=%d", w.Code)
		}
		var out map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	if r := get("deepseek-v4.1-flash"); r["source"] != "meta" || r["context_k"].(float64) != 1024 {
		t.Fatalf("meta 推断不符: %v", r)
	}
	if r := get("claude-opus-4-8-20261101"); r["source"] != "meta" || r["context_k"].(float64) != 200 {
		t.Fatalf("日期后缀前缀推断不符: %v", r)
	}
	if r := get("totally-unknown"); r["source"] != "none" || r["context_k"].(float64) != 0 {
		t.Fatalf("未知模型应 none/0: %v", r)
	}
}

func TestLLMCheckClassification(t *testing.T) {
	s := &Server{}
	check := func(url string) map[string]any {
		body := `{"provider":"openai","base_url":"` + url + `","api_key":"sk-test"}`
		w := httptest.NewRecorder()
		s.llmCheck(w, httptest.NewRequest(http.MethodPost, "/api/llm/check", strReader(body)))
		if w.Code != 200 {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}

	// 401 → key_invalid
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()
	if r := check(srv.URL + "/v1"); r["status_class"] != "key_invalid" {
		t.Fatalf("401 应 key_invalid: %v", r)
	}

	// 404 → path_or_format
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv2.Close()
	if r := check(srv2.URL + "/v1"); r["status_class"] != "path_or_format" {
		t.Fatalf("404 应 path_or_format: %v", r)
	}

	// 200 有模型 → ok + models
	srv3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"m-a"},{"id":"m-b"}]}`))
	}))
	defer srv3.Close()
	r := check(srv3.URL + "/v1")
	if r["status_class"] != "ok" {
		t.Fatalf("200 应 ok: %v", r)
	}
	models, _ := r["models"].([]any)
	if len(models) != 2 {
		t.Fatalf("models 应 2 项: %v", r)
	}

	// 200 空列表 → empty_list
	srv4 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv4.Close()
	if r := check(srv4.URL + "/v1"); r["status_class"] != "empty_list" {
		t.Fatalf("200 空应 empty_list: %v", r)
	}

	// 网络不通 → network（关掉的端口）
	if r := check("http://127.0.0.1:1/v1"); r["status_class"] != "network" {
		t.Fatalf("网络错误应 network: %v", r)
	}

	// 缺 key → missing_key
	w := httptest.NewRecorder()
	s.llmCheck(w, httptest.NewRequest(http.MethodPost, "/api/llm/check", strReader(`{"provider":"openai"}`)))
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["status_class"] != "missing_key" {
		t.Fatalf("缺 key 应 missing_key: %v", out)
	}
}

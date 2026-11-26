package mihomo

import (
	"os"
	"strings"
	"testing"
)

func TestNameserverPolicySanitized(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{dir: dir, configPath: dir + "/config.yaml", logPath: dir + "/m.log"}
	in := "mixed-port: 7890\nproxies:\n- name: a\n  type: socks5\n  server: 1.2.3.4\n  port: 1080\nproxy-groups:\n- name: g\n  type: select\n  proxies: [a]\nrules:\n- MATCH,g\ndns:\n  nameserver-policy:\n    rule-set:cn,private: [a]\n    example.com: [b]\n"
	out, err := m.writeConfig(in, 17890, StartOptions{Mode: "group", Preferred: "g"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(out)
	s := string(b)
	if strings.Contains(s, "rule-set:") {
		t.Fatalf("rule-set reference survived:\n%s", s)
	}
	if !strings.Contains(s, "example.com") {
		t.Fatalf("legit nameserver-policy entry was dropped")
	}
	t.Log("sanitizer OK")
}

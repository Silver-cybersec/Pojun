package server

// 测试基建:异步收尾竞争的宽容目录。
//
// NewManager/引擎持有异步 goroutine(activity 写入、transcript 落盘),它们在
// m.Close() 之后还会短暂存活并写文件;t.TempDir() 的框架级清理是严格 RemoveAll,
// 碰到这种"已 Close 但 goroutine 未死"的目录会直接把测试判 FAIL
// (unlinkat ... directory not empty)。CI 上已三次复现(不同测试名)。
//
// testDir 用 t.Cleanup 宽松清理替代:稍等异步写手退场再删、失败不报错。
// 只把出过问题的测试迁过来;新测试默认仍可用 t.TempDir()。

import (
	"os"
	"strings"
	"testing"
	"time"
)

func testDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "pojun-test-"+sanitizeName(t.Name()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// 30 秒退避重试：异步写手 + 上万个小文件的模板树，单次 RemoveAll 往往删不干净。
		var last error
		for i := 0; i < 120; i++ {
			if last = os.RemoveAll(dir); last == nil {
				return
			}
			time.Sleep(250 * time.Millisecond)
		}
		t.Logf("cleanup temp dir %s: %v", dir, last) // best-effort，但不再静默
	})
	return dir
}

func sanitizeName(name string) string {
	return strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(name)
}

package db

import (
	"context"
	"testing"
)

// #45/#50：附件限额可由 settings 覆盖（默认 300/500MB），删除按 task_id 收口。
func TestArtifactQuotaSettingsOverride(t *testing.T) {
	dsn, _, err := DSN()
	if err != nil || dsn == "" {
		t.Skip("no test database configured")
	}
	d, err := Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.EnsureTaskArtifactsTable(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = d.Exec(`DELETE FROM task_artifacts WHERE task_id IN ('quota-t1','quota-t2')`)
		_ = d.SetSetting(SettingArtifactMaxCount, "")
		_ = d.SetSetting(SettingArtifactMaxBytes, "")
	})

	// 1) 无配置 → 默认
	c, b := d.ArtifactQuota()
	if c != ArtifactMaxTaskCountDefault || b != ArtifactMaxTaskBytesDefault {
		t.Fatalf("默认配额不符: count=%d bytes=%d", c, b)
	}

	// 2) settings 覆盖生效
	_ = d.SetSetting(SettingArtifactMaxCount, "5000")
	_ = d.SetSetting(SettingArtifactMaxBytes, "1073741824")
	c, b = d.ArtifactQuota()
	if c != 5000 || b != 1073741824 {
		t.Fatalf("覆盖配额不符: count=%d bytes=%d", c, b)
	}

	// 3) 非法值回落默认（不炸、不取 0）
	_ = d.SetSetting(SettingArtifactMaxCount, "-3")
	_ = d.SetSetting(SettingArtifactMaxBytes, "abc")
	c, b = d.ArtifactQuota()
	if c != ArtifactMaxTaskCountDefault || b != ArtifactMaxTaskBytesDefault {
		t.Fatalf("非法值应回落默认: count=%d bytes=%d", c, b)
	}
	_ = d.SetSetting(SettingArtifactMaxCount, "")
	_ = d.SetSetting(SettingArtifactMaxBytes, "")
}

func TestDeleteTaskArtifactScopedToTask(t *testing.T) {
	dsn, _, err := DSN()
	if err != nil || dsn == "" {
		t.Skip("no test database configured")
	}
	d, err := Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.EnsureTaskArtifactsTable(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = d.Exec(`DELETE FROM task_artifacts WHERE task_id IN ('quota-t1','quota-t2')`)
	})
	a := &TaskArtifact{TaskID: "quota-t1", Name: "shot.png", Kind: "image", SizeBytes: 1234, Storage: "tasks/quota-t1/artifacts/shot.png"}
	if err := d.InsertTaskArtifact(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if a.ID == 0 {
		t.Fatal("插入后未回填 ID")
	}

	// 跨任务删除必须拒绝（#50：防 id 撞库误删）
	if _, _, err := d.DeleteTaskArtifact("quota-t2", a.ID); err == nil {
		t.Fatal("跨任务删除应失败")
	}
	// 正常删除返回路径与大小
	path, size, err := d.DeleteTaskArtifact("quota-t1", a.ID)
	if err != nil || path != a.Storage || size != 1234 {
		t.Fatalf("删除返回异常: path=%s size=%d err=%v", path, size, err)
	}
	// 二次删除 404 语义（no rows）
	if _, _, err := d.DeleteTaskArtifact("quota-t1", a.ID); err == nil {
		t.Fatal("重复删除应报错")
	}
}

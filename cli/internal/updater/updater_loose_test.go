package updater

import "testing"

func TestVersionCompareLoose(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"0.1.9", "0.2.0", true},
		{"0.1.13.333", "0.2.0", true},      // 四段 build 号
		{"0.1.3", "0.1.13", true},          // patch 3 < 13
		{"0.1.13", "0.1.3", false},         // patch 13 > 3
		{"0.1.3.release", "0.2.0", true},   // .release 后缀
		{"0.1.3-release", "0.2.0", true},   // -release 后缀
		{"v0.1.9", "0.2.0", true},          // v 前缀
		{"0.2.0", "0.2.0", false},          // 相等
		{"0.2.0", "0.1.9", false},          // latest 更小
		{"0.1.13.333", "0.1.13.334", true}, // build 号比较
		{"0.1.13.334", "0.1.13.333", false},
		{"dev", "0.2.0", true}, // 自身非版本号 + 最新可解析 → 视为不是最新，提示
		{"", "0.2.0", true},    // 空版本号 → 同样提示
		{"dev", "dev", false},  // 最新也无法解析 → 不提示
	}
	for _, c := range cases {
		if got := IsNewer(c.current, c.latest); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestVersionFromTag(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"0.2.0-release", "0.2.0", true},
		{"0.1.13.333", "0.1.13.333", true},
		{"0.1.3.release", "0.1.3", true},
		{"v0.1.9", "0.1.9", true},
		{"latest", "", false},
		{"test-tag-do-not-use", "", false},
		{"test-delete-me", "", false},
	}
	for _, c := range cases {
		got, ok := versionFromTag(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("versionFromTag(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

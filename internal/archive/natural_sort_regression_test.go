package archive

import (
	"testing"
)

// 回归测试：文件名包含超过 20 位连续数字时，naturalSortKey 绝不能 panic
// （曾因 strings.Repeat("0", 20-len(num)) 传负数导致容器崩溃）
func TestNaturalSortKeyLongNumber(t *testing.T) {
	cases := []string{
		"IMG_10000000000000000000001.jpg", // 23 位数字
		"123456789012345678901234.jpg",    // 24 位数字
		"99999999999999999999999999999.png",
		"a1b22c333.jpg",
		"普通文件名.jpg",
		"",
		"no numbers here",
	}
	for _, c := range cases {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("naturalSortKey(%q) panicked: %v", c, r)
				}
			}()
			_ = naturalSortKey(c)
		}()
	}
	// 排序稳定性：超长数字按字典序仍可比
	if !naturalLess("IMG_10000000000000000000001.jpg", "IMG_99999999999999999999999.jpg") {
		t.Fatalf("naturalLess order wrong for long numbers")
	}
}

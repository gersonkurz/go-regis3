package regis3

import (
	"strings"
	"testing"
)

func TestDeclaredMarksOnlyHeaderKeys(t *testing.T) {
	content := strings.ReplaceAll(`Windows Registry Editor Version 5.00

[HKEY_LOCAL_MACHINE\SOFTWARE\Vendor]

[HKEY_LOCAL_MACHINE\SOFTWARE\Vendor\Product\Settings]
"A"="1"

[-HKEY_LOCAL_MACHINE\SOFTWARE\Vendor\Old]

[HKEY_LOCAL_MACHINE\SOFTWARE\Vendor\Unclosed`, "\n", "\r\n")
	root, err := Parse(content, nil)
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]bool{
		`HKEY_LOCAL_MACHINE`:                                  false,
		`HKEY_LOCAL_MACHINE\SOFTWARE`:                         false,
		`HKEY_LOCAL_MACHINE\SOFTWARE\Vendor`:                  true,
		`HKEY_LOCAL_MACHINE\SOFTWARE\Vendor\Product`:          false,
		`HKEY_LOCAL_MACHINE\SOFTWARE\Vendor\Product\Settings`: true,
		`HKEY_LOCAL_MACHINE\SOFTWARE\Vendor\Old`:              true,
		`HKEY_LOCAL_MACHINE\SOFTWARE\Vendor\Unclosed`:         true,
	} {
		key := root.FindKey(path)
		if key == nil {
			t.Fatalf("%s: not in tree", path)
		}
		if key.Declared() != want {
			t.Errorf("%s: Declared() = %v, want %v", path, key.Declared(), want)
		}
		if clone := key.Clone(nil); clone.Declared() != want {
			t.Errorf("%s: Clone lost Declared", path)
		}
	}
}

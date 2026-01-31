package regis3

import (
	"testing"
)

func TestWhitespaceParity(t *testing.T) {
	// Tests handling of weird whitespace
	input := `Windows Registry Editor Version 5.00

[HKEY_CURRENT_USER\Software\Test]

"SpacedValue"  =  "hello"

"TabbedValue"	=	"world"





[HKEY_CURRENT_USER\Software\Test\Sub]
"Dword"=dword:00000001
`
	// Enable IgnoreWhitespaces
	root, err := Parse(input, &ParseOptions{IgnoreWhitespaces: true})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	key := root.FindOrCreateKey("Software\\Test")
	if key.Values()["spacedvalue"].GetString("") != "hello" {
		t.Errorf("SpacedValue mismatch")
	}
	if key.Values()["tabbedvalue"].GetString("") != "world" {
		t.Errorf("TabbedValue mismatch")
	}
}

func TestEscapedStringParity(t *testing.T) {
	// Tests handling of escaped characters in strings
	input := `Windows Registry Editor Version 5.00

[HKEY_CURRENT_USER\Software\Escapes]
"Path"="C:\\Windows\\System32"
"Quote"="Say \"Hello\""
"Backslash"="\\"
"Mixed"="a\"b\\c"
`
	root, err := Parse(input, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	key := root.FindOrCreateKey("Software\\Escapes")

	tests := []struct {
		name     string
		expected string
	}{
		{"path", "C:\\Windows\\System32"},
		{"quote", "Say \"Hello\""},
		{"backslash", "\\"},
		{"mixed", "a\"b\\c"},
	}

	for _, tt := range tests {
		val := key.Values()[tt.name]
		if val == nil {
			t.Errorf("%s value not found", tt.name)
			continue
		}
		if val.GetString("") != tt.expected {
			t.Errorf("%s: expected '%s', got '%s'", tt.name, tt.expected, val.GetString(""))
		}
	}
}

func TestMultilineHexParity(t *testing.T) {
	// Tests line continuation with backslash
	input := `Windows Registry Editor Version 5.00

[HKEY_CURRENT_USER\Software\Multiline]
"Binary"=hex:01,02,03,04,\ 
  05,06,07,08,\ 
  09,0a
`
	root, err := Parse(input, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	key := root.FindOrCreateKey("Software\\Multiline")
	val := key.Values()["binary"]
	if val == nil {
		t.Fatal("Binary value not found")
	}

	expected := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	data := val.Data()
	if len(data) != len(expected) {
		t.Fatalf("Data length mismatch: expected %d, got %d", len(expected), len(data))
	}
	for i := range expected {
		if data[i] != expected[i] {
			t.Errorf("Data at %d mismatch: expected %d, got %d", i, expected[i], data[i])
		}
	}
}

func TestDeleteMarkerParity(t *testing.T) {
	// Tests deletion markers
	input := `Windows Registry Editor Version 5.00

[-HKEY_CURRENT_USER\Software\DeletedKey]

[HKEY_CURRENT_USER\Software\Existing]
"DeletedValue"=-
@=-
`
	root, err := Parse(input, nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Root key check
	deletedKey := root.FindOrCreateKey("Software\\DeletedKey")
	if !deletedKey.RemoveFlag() {
		t.Error("Expected RemoveFlag on DeletedKey")
	}

	existing := root.FindOrCreateKey("Software\\Existing")
	if !existing.Values()["deletedvalue"].RemoveFlag() {
		t.Error("Expected RemoveFlag on DeletedValue")
	}
	if !existing.DefaultValue().RemoveFlag() {
		t.Error("Expected RemoveFlag on default value")
	}
}

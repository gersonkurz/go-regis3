package regis3

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Command-line flag for test data folder
var testDataFolder = flag.String("testdata", "", "Folder containing .reg files for production tests")

func TestBasicParse(t *testing.T) {
	input := `Windows Registry Editor Version 5.00

[HKEY_CURRENT_USER\Software\Test]
"StringValue"="Hello World"
"DwordValue"=dword:00000001
`
	root, err := Parse(input, ImportOptionsNone)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Navigate to key
	// Because Parse unwraps the root if single child, root is HKEY_CURRENT_USER
	if strings.ToLower(root.Name()) != "hkey_current_user" {
		t.Logf("Root keys: %v", root.SubKeys())
		t.Logf("Root values: %v", root.Values())
		t.Fatalf("Expected root to be HKEY_CURRENT_USER, got '%s'", root.Name())
	}

	software := root.FindOrCreateKey("Software")
	testKey := software.FindOrCreateKey("Test")

	// Check String Value
	val := testKey.Values()["stringvalue"] // lowercase key
	if val == nil {
		t.Fatal("StringValue not found")
	}
	if val.GetString("") != "Hello World" {
		t.Errorf("Expected 'Hello World', got '%s'", val.GetString(""))
	}

	// Check Dword Value
	valDword := testKey.Values()["dwordvalue"]
	if valDword == nil {
		t.Fatal("DwordValue not found")
	}
	if valDword.GetDword(0) != 1 {
		t.Errorf("Expected 1, got %d", valDword.GetDword(0))
	}
}

func TestRegedit4(t *testing.T) {
	input := `REGEDIT4

[HKEY_CURRENT_USER\Software\Test]
"StringValue"="Hello World"
`
	root, err := Parse(input, ImportOptionsNone)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if strings.ToLower(root.Name()) != "hkey_current_user" {
		t.Fatalf("Expected root to be HKEY_CURRENT_USER, got %s", root.Name())
	}

	// Check path
	testKey := root.FindOrCreateKey("Software\\Test")
	if testKey == nil {
		t.Fatal("Key not found")
	}
}

func TestHexValues(t *testing.T) {
	input := `Windows Registry Editor Version 5.00

[HKEY_LOCAL_MACHINE\Software\HexTest]
"Binary"=hex:01,02,03,04
"Multi"=hex(7):48,00,65,00,6c,00,6c,00,6f,00,00,00,00,00
`
// Multi is "Hello" in UTF-16LE + double null
// H=0048, e=0065, l=006c, l=006c, o=006f

	root, err := Parse(input, ImportOptionsNone)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if strings.ToLower(root.Name()) != "hkey_local_machine" {
		t.Fatalf("Expected root to be HKEY_LOCAL_MACHINE, got %s", root.Name())
	}

	key := root.FindOrCreateKey("Software\\HexTest")

	// Check Binary
	bin := key.Values()["binary"]
	if bin == nil {
		t.Fatal("Binary value not found")
	}
	if bin.Kind() != RegBinary {
		t.Errorf("Expected RegBinary, got %d", bin.Kind())
	}
	data := bin.Data()
	if len(data) != 4 || data[0] != 1 || data[3] != 4 {
		t.Errorf("Binary data mismatch: %v", data)
	}

	// Check MultiSz
	multi := key.Values()["multi"]
	if multi == nil {
		t.Fatal("Multi value not found")
	}
	if multi.Kind() != RegMultiSz {
		t.Errorf("Expected RegMultiSz, got %d", multi.Kind())
	}
	strs := multi.GetMultiString()
	if len(strs) != 1 || strs[0] != "Hello" {
		t.Errorf("MultiSz mismatch: %v", strs)
	}
}

func TestRoundTrip(t *testing.T) {
	input := `Windows Registry Editor Version 5.00

[HKEY_CURRENT_USER\Software\RoundTrip]
"String"="Value"
"Dword"=dword:0000000a
"Binary"=hex:de,ad,be,ef
`
	// 1. Parse
	root1, err := Parse(input, ImportOptionsNone)
	if err != nil {
		t.Fatalf("Initial parse failed: %v", err)
	}

	// 2. Write
	writer := NewRegWriter(HeaderWindows5, ExportOptionsNone, false) // Use UTF-8 for easy string comparison
	var sb strings.Builder
	if err := writer.Write(&sb, root1); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	output := sb.String()

	// 3. Parse again
	root2, err := Parse(output, ImportOptionsNone)
	if err != nil {
		t.Logf("Output was:\n%s", output)
		t.Fatalf("Second parse failed: %v", err)
	}

	// 4. Compare key structure
	key1 := root1.FindOrCreateKey("Software\\RoundTrip")
	key2 := root2.FindOrCreateKey("Software\\RoundTrip")

	if key1.Values()["string"].GetString("") != key2.Values()["string"].GetString("") {
		t.Errorf("String mismatch: %s vs %s", key1.Values()["string"].GetString(""), key2.Values()["string"].GetString(""))
	}
	if key1.Values()["dword"].GetDword(0) != key2.Values()["dword"].GetDword(0) {
		t.Errorf("Dword mismatch: %d vs %d", key1.Values()["dword"].GetDword(0), key2.Values()["dword"].GetDword(0))
	}
}

func TestEscapedValues(t *testing.T) {
	input := `Windows Registry Editor Version 5.00

[HKEY_CURRENT_USER\Software\Escaped]
"Dword"=dword:$$VAR1$$
"Qword"=hex(b):$$VAR2$$
`
	// Need to allow variables for non-string types
	opts := AllowVariableNamesForNonStringVariables
	root, err := Parse(input, opts)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	key := root.FindOrCreateKey("Software\\Escaped")

	dv := key.Values()["dword"]
	if dv == nil {
		t.Fatal("Dword value not found")
	}
	if dv.Kind() != RegEscapedDword {
		t.Errorf("Expected RegEscapedDword, got %d", dv.Kind())
	}
	if dv.GetString("") != "$$VAR1$$" {
		t.Errorf("Expected $$VAR1$$, got '%s'", dv.GetString(""))
	}

	qv := key.Values()["qword"]
	if qv == nil {
		t.Fatal("Qword value not found")
	}
	if qv.Kind() != RegEscapedQword {
		t.Errorf("Expected RegEscapedQword, got %d", qv.Kind())
	}
	if qv.GetString("") != "$$VAR2$$" {
		t.Errorf("Expected $$VAR2$$, got '%s'", qv.GetString(""))
	}

	// Test Round-trip
	writer := NewRegWriter(HeaderWindows5, ExportOptionsNone, false)
	var sb strings.Builder
	if err := writer.Write(&sb, root); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	output := sb.String()

	if !strings.Contains(output, "\"Dword\"=dword:$$VAR1$$") {
		t.Errorf("Output missing escaped dword: %s", output)
	}
	if !strings.Contains(output, "\"Qword\"=hex(b):$$VAR2$$") {
		t.Errorf("Output missing escaped qword: %s", output)
	}
}

func TestDeleteMarkers(t *testing.T) {
	input := `Windows Registry Editor Version 5.00

[-HKEY_CURRENT_USER\Software\ToDelete]

[HKEY_CURRENT_USER\Software\WithDeletions]
"DeleteMe"=-
@=-
`
	root, err := Parse(input, ImportOptionsNone)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// 1. Check Key Deletion
	keyToDelete := root.FindOrCreateKey("Software\\ToDelete")
	if !keyToDelete.RemoveFlag() {
		t.Error("Expected RemoveFlag on ToDelete key")
	}

	// 2. Check Value Deletion
	withDeletions := root.FindOrCreateKey("Software\\WithDeletions")
	val := withDeletions.Values()["deleteme"]
	if val == nil {
		t.Fatal("DeleteMe value entry not found")
	}
	if !val.RemoveFlag() {
		t.Error("Expected RemoveFlag on DeleteMe value")
	}

	// 3. Check Default Value Deletion
	if withDeletions.DefaultValue() == nil {
		t.Fatal("Default value entry not found")
	}
	if !withDeletions.DefaultValue().RemoveFlag() {
		t.Error("Expected RemoveFlag on default value")
	}

	// 4. Test Round-trip for deletions
	writer := NewRegWriter(HeaderWindows5, ExportOptionsNone, false)
	var sb strings.Builder
	if err := writer.Write(&sb, root); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	output := sb.String()

	if !strings.Contains(output, "[-HKEY_CURRENT_USER\\Software\\ToDelete]") {
		t.Errorf("Output missing key deletion marker: %s", output)
	}
	if !strings.Contains(output, "\"DeleteMe\"=-") {
		t.Errorf("Output missing value deletion marker: %s", output)
	}
	if !strings.Contains(output, "@=-") {
		t.Errorf("Output missing default value deletion marker: %s", output)
	}
}

func TestWixWriter(t *testing.T) {
	input := `Windows Registry Editor Version 5.00

[HKEY_LOCAL_MACHINE\Software\WixTest]
"String"="Value"
"Dword"=dword:00000001
"Escaped"=dword:$$VAR$$
"Binary"=hex:01,02
`
	root, err := Parse(input, AllowVariableNamesForNonStringVariables)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	wixWriter := NewWixWriter()
	var sb strings.Builder
	if err := wixWriter.Write(&sb, root); err != nil {
		t.Fatalf("WixWriter failed: %v", err)
	}
	output := sb.String()

	// Verify basic structure
	if !strings.Contains(output, "<RegistryKey Root=\"HKLM\" Key=\"Software\">") {
		t.Errorf("Missing root key entry: %s", output)
	}
	if !strings.Contains(output, "<RegistryKey Key=\"WixTest\">") {
		t.Errorf("Missing child key entry: %s", output)
	}
	// Verify child doesn't repeat Root (redundant)
	if strings.Contains(output, "<RegistryKey Root=\"HKLM\" Key=\"WixTest\">") {
		t.Errorf("Child key should NOT have Root attribute: %s", output)
	}

	if !strings.Contains(output, "Type=\"string\"") {
		t.Errorf("Missing string type: %s", output)
	}
	if !strings.Contains(output, "Value=\"1\" Type=\"integer\"") {
		t.Errorf("Missing dword value: %s", output)
	}
	if !strings.Contains(output, "Value=\"$$VAR$$\" Type=\"integer\"") {
		t.Errorf("Missing escaped dword value: %s", output)
	}
	if !strings.Contains(output, "Value=\"0102\" Type=\"binary\"") {
		t.Errorf("Missing binary value (or not uppercase): %s", output)
	}
}

func TestWixWriterDeletions(t *testing.T) {
	input := `Windows Registry Editor Version 5.00

[-HKEY_CURRENT_USER\Software\ToDelete]

[HKEY_CURRENT_USER\Software\WithDeletions]
"DeleteMe"=-
@=-
`
	root, err := Parse(input, ImportOptionsNone)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	wixWriter := NewWixWriter()
	var sb strings.Builder
	if err := wixWriter.Write(&sb, root); err != nil {
		t.Fatalf("WixWriter failed: %v", err)
	}
	output := sb.String()

	if !strings.Contains(output, "<RemoveRegistryKey Action=\"removeOnInstall\" Root=\"HKCU\" Key=\"Software\\ToDelete\"/>") {
		t.Errorf("Missing key deletion marker in WiX: %s", output)
	}
	if !strings.Contains(output, "<RemoveRegistryValue Root=\"HKCU\" Key=\"Software\\WithDeletions\" Name=\"DeleteMe\"/>") {
		t.Errorf("Missing value deletion marker in WiX: %s", output)
	}
	if !strings.Contains(output, "<RemoveRegistryValue Root=\"HKCU\" Key=\"Software\\WithDeletions\"/>") {
		t.Errorf("Missing default value deletion marker in WiX: %s", output)
	}
}

// TestProductionFiles tests parsing of real .REG files from a configurable folder.
// Run with: go test -testdata=/path/to/reg/files
func TestProductionFiles(t *testing.T) {
	folder := *testDataFolder
	if folder == "" {
		folder = os.Getenv("REGIS3_TESTDATA")
	}
	if folder == "" {
		t.Skip("Skipping production file tests: no -testdata flag or REGIS3_TESTDATA env var set")
	}

	// Find all .reg files in the folder (recursively)
	var regFiles []string
	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".reg") {
			regFiles = append(regFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk folder %s: %v", folder, err)
	}

	if len(regFiles) == 0 {
		t.Fatalf("No .reg files found in %s", folder)
	}

	t.Logf("Found %d .reg files in %s", len(regFiles), folder)

	for _, file := range regFiles {
		relPath, _ := filepath.Rel(folder, file)
		t.Run(relPath, func(t *testing.T) {
			root, err := ParseFile(file, ImportOptionsNone)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			// Perform round-trip check
			writer := NewRegWriter(HeaderWindows5, ExportOptionsNone, false)
			var sb strings.Builder
			if err := writer.Write(&sb, root); err != nil {
				t.Fatalf("Write failed: %v", err)
			}
			output := sb.String()

			// Parse again to ensure semantic identity
			root2, err := Parse(output, ImportOptionsNone)
			if err != nil {
				t.Fatalf("Second parse failed: %v", err)
			}

			if root.Name() != root2.Name() {
				t.Errorf("Root name mismatch: %s vs %s", root.Name(), root2.Name())
			}
		})
	}
}

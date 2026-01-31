package regis3

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"
)

// WixWriter handles exporting KeyEntry trees to WiX XML format.
type WixWriter struct {
}

// NewWixWriter creates a new WixWriter.
func NewWixWriter() *WixWriter {
	return &WixWriter{}
}

// Write exports the key entry tree to the provided writer as WiX XML fragments.
// It performs two passes:
// 1. Writes <RemoveRegistryKey> and <RemoveRegistryValue> elements (flattened).
// 2. Writes <RegistryKey> hierarchy for additions/modifications.
func (w *WixWriter) Write(out io.Writer, root *KeyEntry) error {
	// 1. Detect recognized roots (e.g. HKLM) or iterate top-level keys

	// Helper to get top-level keys to process
	getTopLevelKeys := func(k *KeyEntry) ([]*KeyEntry, string, bool) {
		// If k is a root like "HKEY_LOCAL_MACHINE", return its children with mapped root
		wixRoot, exists := mapRoot(k.Name())
		if exists {
			subKeys := k.SubKeys()
			keys := make([]string, 0, len(subKeys))
			for name := range subKeys {
				keys = append(keys, name)
			}
			sort.Strings(keys)

			children := make([]*KeyEntry, len(keys))
			for i, name := range keys {
				children[i] = subKeys[name]
			}
			return children, wixRoot, true
		}
		return nil, "", false
	}

	// Determine start points
	var topKeys []*KeyEntry
	var startRoot string

	// Check if the root passed in IS a registry root (unwrapped case)
	children, wixRoot, isRoot := getTopLevelKeys(root)
	if isRoot {
		topKeys = children
		startRoot = wixRoot
	} else {
		// The root passed in is a container of roots (e.g. standard parser output)
		subKeys := root.SubKeys()
		rootNames := make([]string, 0, len(subKeys))
		for name := range subKeys {
			rootNames = append(rootNames, name)
		}
		sort.Strings(rootNames)

		for _, rootName := range rootNames {
			subRootKey := subKeys[rootName]
			subChildren, subWixRoot, subIsRoot := getTopLevelKeys(subRootKey)
			if !subIsRoot {
				return fmt.Errorf("unrecognized registry root: %s", subRootKey.Name())
			}
			// Process this root's children immediately to preserve order or accumulate?
			// To keep it simple, we'll process each root bucket sequentially.
			if err := w.processRootBucket(out, subChildren, subWixRoot); err != nil {
				return err
			}
		}
		return nil
	}

	return w.processRootBucket(out, topKeys, startRoot)
}

func (w *WixWriter) processRootBucket(out io.Writer, topKeys []*KeyEntry, wixRoot string) error {
	// Pass 1: Removals (Flat)
	for _, key := range topKeys {
		if err := w.writeRemovals(out, key, wixRoot, key.Name()); err != nil {
			return err
		}
	}

	// Pass 2: Additions (Nested)
	for _, key := range topKeys {
		if err := w.writeAdditions(out, key, wixRoot, true); err != nil {
			return err
		}
	}
	return nil
}

func (w *WixWriter) writeRemovals(out io.Writer, key *KeyEntry, root string, currentPath string) error {
	if key.removeFlag {
		// [-HKEY...] -> RemoveRegistryKey
		// Action "removeOnInstall" is standard for this, or "removeOnUninstall" depending on intent.
		// .reg [-Key] usually means "ensure it's gone".
		// We'll use "removeOnInstall" to match the immediacy of .reg import.
		if _, err := fmt.Fprintf(out, "<RemoveRegistryKey Action=\"removeOnInstall\" Root=\"%s\" Key=\"%s\"/>\n", root, escapeWixXml(currentPath)); err != nil {
			return err
		}
		// If key is removed, sub-values/keys are implicitly removed, so don't recurse?
		// Actually, explicit value deletions inside a deleted key are redundant.
		return nil
	}

	// Check values for deletions
	valNames := make([]string, 0, len(key.values))
	for name := range key.values {
		valNames = append(valNames, name)
	}
	sort.Strings(valNames)

	for _, name := range valNames {
		val := key.values[name]
		if val.removeFlag {
			nameAttr := fmt.Sprintf(" Name=\"%s\"", escapeWixXml(val.Name()))
			if val.IsDefaultValue() {
				nameAttr = "" // Default value deletion
			}
			// Value deletion needs Root/Key/Name
			if _, err := fmt.Fprintf(out, "<RemoveRegistryValue Root=\"%s\" Key=\"%s\"%s/>\n", root, escapeWixXml(currentPath), nameAttr); err != nil {
				return err
			}
		}
	}

	// Default value check
	if key.defaultValue != nil && key.defaultValue.removeFlag {
		if _, err := fmt.Fprintf(out, "<RemoveRegistryValue Root=\"%s\" Key=\"%s\"/>\n", root, escapeWixXml(currentPath)); err != nil {
			return err
		}
	}

	// Recurse subkeys
	subKeyNames := make([]string, 0, len(key.subKeys))
	for name := range key.subKeys {
		subKeyNames = append(subKeyNames, name)
	}
	sort.Strings(subKeyNames)

	for _, name := range subKeyNames {
		subKey := key.subKeys[name]
		childPath := currentPath + "\\" + subKey.Name()
		if err := w.writeRemovals(out, subKey, root, childPath); err != nil {
			return err
		}
	}

	return nil
}

func (w *WixWriter) writeAdditions(out io.Writer, key *KeyEntry, root string, isTopLevel bool) error {
	if key.removeFlag {
		// Key is being removed, so don't write it as an addition/container
		return nil
	}

	// Only write <RegistryKey> if we have something to add (values or subkeys)
	// OR if we want to force create. .reg usually implies creation.

	rootAttr := ""
	if isTopLevel {
		rootAttr = fmt.Sprintf(" Root=\"%s\"", root)
	}

	if _, err := fmt.Fprintf(out, "<RegistryKey%s Key=\"%s\">\n", rootAttr, escapeWixXml(key.Name())); err != nil {
		return err
	}

	// Default Value
	if key.defaultValue != nil && !key.defaultValue.removeFlag {
		if err := w.writeValue(out, key.defaultValue); err != nil {
			return err
		}
	}

	// Values
	valNames := make([]string, 0, len(key.values))
	for name := range key.values {
		valNames = append(valNames, name)
	}
	sort.Strings(valNames)

	for _, name := range valNames {
		val := key.values[name]
		if !val.removeFlag {
			if err := w.writeValue(out, val); err != nil {
				return err
			}
		}
	}

	// Subkeys (recursive)
	subKeyNames := make([]string, 0, len(key.subKeys))
	for name := range key.subKeys {
		subKeyNames = append(subKeyNames, name)
	}
	sort.Strings(subKeyNames)

	for _, name := range subKeyNames {
		if err := w.writeAdditions(out, key.subKeys[name], root, false); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(out, "</RegistryKey>\n"); err != nil {
		return err
	}

	return nil
}

func (w *WixWriter) writeValue(out io.Writer, val *ValueEntry) error {
	// ... (rest of writeValue remains the same, but remove logic is handled in pass 1)
	nameAttr := ""
	if !val.IsDefaultValue() {
		nameAttr = fmt.Sprintf(" Name=\"%s\"", escapeWixXml(val.Name()))
	}

	wixType := "string"
	valueStr := ""

	switch val.kind {
	case RegSz:
		wixType = "string"
		valueStr = val.GetString("")
	case RegExpandSz:
		wixType = "expandable"
		valueStr = val.GetString("")
	case RegDword:
		wixType = "integer"
		valueStr = fmt.Sprintf("%d", val.GetDword(0))
	case RegEscapedDword:
		wixType = "integer"
		valueStr = val.GetString("")
	case RegEscapedQword:
		wixType = "integer"
		valueStr = val.GetString("")
	case RegMultiSz:
		wixType = "multiString"
		strs := val.GetMultiString()
		if _, err := fmt.Fprintf(out, "  <RegistryValue%s Type=\"%s\">\n", nameAttr, wixType); err != nil {
			return err
		}
		for _, s := range strs {
			if _, err := fmt.Fprintf(out, "    <MultiStringValue>%s</MultiStringValue>\n", escapeWixXml(s)); err != nil {
				return err
			}
		}
		_, err := fmt.Fprint(out, "  </RegistryValue>\n")
		return err
	case RegBinary:
		wixType = "binary"
		valueStr = strings.ToUpper(hex.EncodeToString(val.data))
	default:
		wixType = "binary"
		valueStr = strings.ToUpper(hex.EncodeToString(val.data))
	}

	_, err := fmt.Fprintf(out, "  <RegistryValue%s Value=\"%s\" Type=\"%s\"/>\n", nameAttr, escapeWixXml(valueStr), wixType)
	return err
}

func escapeWixXml(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func mapRoot(name string) (string, bool) {
	switch strings.ToUpper(name) {
	case "HKEY_LOCAL_MACHINE":
		return "HKLM", true
	case "HKEY_CURRENT_USER":
		return "HKCU", true
	case "HKEY_CLASSES_ROOT":
		return "HKCR", true
	case "HKEY_USERS":
		return "HKU", true
	}
	return "", false
}

package regis3

import (
	"strings"
)

// KeyEntry represents a registry key tree node.
// Keys and values are stored case-insensitively (lowercase keys).
type KeyEntry struct {
	parent       *KeyEntry
	name         string
	subKeys      map[string]*KeyEntry   // map key is lowercase
	values       map[string]*ValueEntry // map key is lowercase
	defaultValue *ValueEntry
	removeFlag   bool
}

// NewKeyEntry creates a new named key with a parent.
// Pass nil for parent to create a root key.
func NewKeyEntry(parent *KeyEntry, name string) *KeyEntry {
	return &KeyEntry{
		parent:  parent,
		name:    name,
		subKeys: make(map[string]*KeyEntry),
		values:  make(map[string]*ValueEntry),
	}
}

// Name returns the key name (not the full path).
func (k *KeyEntry) Name() string {
	return k.name
}

// Parent returns the parent key.
func (k *KeyEntry) Parent() *KeyEntry {
	return k.parent
}

// SubKeys returns the map of subkeys.
func (k *KeyEntry) SubKeys() map[string]*KeyEntry {
	return k.subKeys
}

// Values returns the map of values.
func (k *KeyEntry) Values() map[string]*ValueEntry {
	return k.values
}

// DefaultValue returns the default value (if any).
func (k *KeyEntry) DefaultValue() *ValueEntry {
	return k.defaultValue
}

// RemoveFlag checks if this key should be removed.
func (k *KeyEntry) RemoveFlag() bool {
	return k.removeFlag
}

// SetRemoveFlag sets the remove flag.
func (k *KeyEntry) SetRemoveFlag(flag bool) {
	k.removeFlag = flag
}

// GetPath returns the full registry path.
func (k *KeyEntry) GetPath() string {
	var pathComponents []string
	curr := k

	for curr != nil {
		if curr.parent == nil {
			// Root node - only add if it has a name
			if curr.name != "" {
				pathComponents = append(pathComponents, curr.name)
			}
			break
		}
		pathComponents = append(pathComponents, curr.name)
		curr = curr.parent
	}

	// Reverse
	for i := 0; i < len(pathComponents)/2; i++ {
		j := len(pathComponents) - 1 - i
		pathComponents[i], pathComponents[j] = pathComponents[j], pathComponents[i]
	}

	return strings.Join(pathComponents, "\\")
}

// FindOrCreateKey finds or creates a subkey by path.
// Handles "-PATH" syntax for remove flag.
func (k *KeyEntry) FindOrCreateKey(path string) *KeyEntry {
	if path == "" {
		return k
	}

	copyOfPath := path
	removeThis := false

	// Handle leading minus sign for remove flag
	if len(copyOfPath) > 0 && copyOfPath[0] == '-' {
		removeThis = true
		copyOfPath = copyOfPath[1:]
	}

	result := k
	tokens := strings.Split(copyOfPath, "\\")

	for i, token := range tokens {
		if token == "" {
			continue
		}
		
		// If the first token matches the current key's name, treat it as "this"
		if i == 0 && strings.EqualFold(token, result.name) {
			continue
		}

		key := strings.ToLower(token)
		subKey, exists := result.subKeys[key]

		if !exists {
			subKey = NewKeyEntry(result, token)
			result.subKeys[key] = subKey
		}
		result = subKey
	}

	if removeThis {
		result.removeFlag = true
	}

	return result
}

// FindOrCreateValue finds or creates a named value.
// Empty string name creates/returns the default value.
func (k *KeyEntry) FindOrCreateValue(name string) *ValueEntry {
	if name == "" {
		if k.defaultValue == nil {
			k.defaultValue = NewValueEntry("")
		}
		return k.defaultValue
	}

	nameKey := strings.ToLower(name)
	val, exists := k.values[nameKey]
	if exists {
		return val
	}

	val = NewValueEntry(name)
	k.values[nameKey] = val
	return val
}

// Clone creates a deep copy of this key entry.
func (k *KeyEntry) Clone(newParent *KeyEntry) *KeyEntry {
	result := NewKeyEntry(newParent, k.name)
	result.removeFlag = k.removeFlag

	// Clone subkeys
	for key, child := range k.subKeys {
		result.subKeys[key] = child.Clone(result)
	}

	// Clone values
	for key, val := range k.values {
		// ValueEntry copy
		newVal := *val // shallow copy of struct
		// Deep copy of data slice
		if val.data != nil {
			newVal.data = make([]byte, len(val.data))
			copy(newVal.data, val.data)
		}
		result.values[key] = &newVal
	}

	// Clone default value
	if k.defaultValue != nil {
		newDef := *k.defaultValue
		if k.defaultValue.data != nil {
			newDef.data = make([]byte, len(k.defaultValue.data))
			copy(newDef.data, k.defaultValue.data)
		}
		result.defaultValue = &newDef
	}

	return result
}

// AskToAddKey creates or finds a key for adding content.
func (k *KeyEntry) AskToAddKey(addThis *KeyEntry) *KeyEntry {
	// Note: In C++, get_path() is called on addThis, but addThis might be from another tree.
	// We want to replicate the path structure relative to k.
	// The C++ implementation:
	// key_entry* key = find_or_create_key(add_this->get_path());
	// This implies addThis->get_path() returns the FULL path.
	// If k is Root, and addThis is HKLM\Soft, then find_or_create_key("HKLM\Soft") works.
	
	key := k.FindOrCreateKey(addThis.GetPath())

	// Copy subkeys (shallow copy of structure, deep copy of nodes)
	for subKeyName, subKey := range addThis.subKeys {
		cloned := subKey.Clone(key)
		key.subKeys[subKeyName] = cloned
	}

	// Copy values
	for valName, val := range addThis.values {
		newVal := *val
		if val.data != nil {
			newVal.data = make([]byte, len(val.data))
			copy(newVal.data, val.data)
		}
		key.values[valName] = &newVal
	}

	// Copy default value
	if addThis.defaultValue != nil {
		newDef := *addThis.defaultValue
		if addThis.defaultValue.data != nil {
			newDef.data = make([]byte, len(addThis.defaultValue.data))
			copy(newDef.data, addThis.defaultValue.data)
		}
		key.defaultValue = &newDef
	}

	return key
}

// AskToRemoveKey creates a key entry marked for removal.
func (k *KeyEntry) AskToRemoveKey(removeThis *KeyEntry) *KeyEntry {
	key := k.AskToAddKey(removeThis)
	key.SetRemoveFlag(true)
	return key
}

// AskToAddValue adds a value to the key (for diff/merge).
func (k *KeyEntry) AskToAddValue(key *KeyEntry, val *ValueEntry) {
	// key is the source key (from another tree), we need to find/create it in k's tree
	destKey := k.AskToAddKey(key)

	if val.IsDefaultValue() {
		newVal := *val
		if val.data != nil {
			newVal.data = make([]byte, len(val.data))
			copy(newVal.data, val.data)
		}
		destKey.defaultValue = &newVal
	} else {
		valName := strings.ToLower(val.Name())
		newVal := *val
		if val.data != nil {
			newVal.data = make([]byte, len(val.data))
			copy(newVal.data, val.data)
		}
		destKey.values[valName] = &newVal
	}
}

// AskToRemoveValue adds a value marked for removal (for diff/merge).
func (k *KeyEntry) AskToRemoveValue(key *KeyEntry, val *ValueEntry) {
	destKey := k.AskToAddKey(key)

	if val.IsDefaultValue() {
		newVal := *val
		if val.data != nil {
			newVal.data = make([]byte, len(val.data))
			copy(newVal.data, val.data)
		}
		newVal.SetRemoveFlag(true)
		destKey.defaultValue = &newVal
	} else {
		valName := strings.ToLower(val.Name())
		newVal := *val
		if val.data != nil {
			newVal.data = make([]byte, len(val.data))
			copy(newVal.data, val.data)
		}
		newVal.SetRemoveFlag(true)
		destKey.values[valName] = &newVal
	}
}

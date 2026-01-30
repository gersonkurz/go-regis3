# Test Data: Production .REG Files

This document lists all `.reg` files available in `production-examples/` for parsing and testing.

**Note:** All files below are UTF-16LE (Windows Registry Editor Version 5.00). REGEDIT4 (ANSI) format is not represented in current production but must still be fully supported.

---

## Quick Reference

| File | Size | Features |
|------|------|----------|
| only_during_update.reg | Tiny | Minimal, single key/value |
| automation.reg | Small | DWORD values, multiple keys |
| wosaxfs.reg | Small | Variable refs `[INSTALLDIR]`, path with `/` |
| permanent.reg | Small | Standard structure |
| security.reg | Small | Standard structure |
| dn_logo.reg | Small | Standard structure |
| ng_logo.reg | Small | Standard structure |
| aktconfig.reg | Medium | Standard structure |
| proakt.reg | Large | Comprehensive, stress test |
| proakt_poste_italiane.reg | Medium | Standard structure |

---

## proakt-3.7.0-prepackaged

```
C:\NGBT\MSIS\production-examples\proakt-3.7.0-prepackaged\setup\Registry\automation.reg
C:\NGBT\MSIS\production-examples\proakt-3.7.0-prepackaged\setup\Registry\dn_logo.reg
C:\NGBT\MSIS\production-examples\proakt-3.7.0-prepackaged\setup\Registry\ng_logo.reg
C:\NGBT\MSIS\production-examples\proakt-3.7.0-prepackaged\setup\Registry\only_during_update.reg
C:\NGBT\MSIS\production-examples\proakt-3.7.0-prepackaged\setup\Registry\permanent.reg
C:\NGBT\MSIS\production-examples\proakt-3.7.0-prepackaged\setup\Registry\proakt.reg
C:\NGBT\MSIS\production-examples\proakt-3.7.0-prepackaged\setup\Registry\security.reg
C:\NGBT\MSIS\production-examples\proakt-3.7.0-prepackaged\setup\Registry\wosaxfs.reg
```

**Recommended for initial testing:**
- `only_during_update.reg` - Minimal file, good for first parse test
- `automation.reg` - Simple structure with DWORD values
- `wosaxfs.reg` - Contains `[INSTALLDIR]` variable references

---

## proakt-for-desjardins-eftpos

```
C:\NGBT\MSIS\production-examples\proakt-for-desjardins-eftpos\Registry\proakt.reg
C:\NGBT\MSIS\production-examples\proakt-for-desjardins-eftpos\Registry\wosaxfs.reg
```

---

## proakt-4.2.0.x (x64)

```
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x64\Registry\automation.reg
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x64\Registry\only_during_update.reg
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x64\Registry\permanent.reg
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x64\Registry\proakt.reg
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x64\Registry\proakt_poste_italiane.reg
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x64\Registry\security.reg
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x64\Files_NorthAmerica\aktconfig.reg
```

---

## proakt-4.2.0.x (x86)

```
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x86\Registry\automation.reg
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x86\Registry\only_during_update.reg
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x86\Registry\permanent.reg
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x86\Registry\proakt.reg
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x86\Registry\proakt_poste_italiane.reg
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x86\Registry\security.reg
C:\NGBT\MSIS\production-examples\proakt-4.2.0.x\Setup\x86\Files_NorthAmerica\aktconfig.reg
```

---

## ng1-bmo

No `.reg` files in this example.

---

## Sample Content

### only_during_update.reg (minimal)
```
Windows Registry Editor Version 5.00

[HKEY_LOCAL_MACHINE\SOFTWARE\Pergamon\AKT\Start]
"PostInstallType"=dword:00000000
```

### automation.reg (simple with DWORD)
```
Windows Registry Editor Version 5.00

[HKEY_LOCAL_MACHINE\SOFTWARE\ProAKT]

[HKEY_LOCAL_MACHINE\SOFTWARE\ProAKT\Automation]

[HKEY_LOCAL_MACHINE\SOFTWARE\ProAKT\Automation\UI]
"ConfigurationFilename"=""
"LastKnownFilename"=""

[HKEY_LOCAL_MACHINE\SOFTWARE\Pergamon\AKT\Shared]
"AutomationEnabled"=dword:00000001
```

### wosaxfs.reg (variable references)
```
Windows Registry Editor Version 5.00

[HKEY_CLASSES_ROOT\WOSA/XFS_ROOT\SERVICE_PROVIDERS\PASION_HWDETECT]
"HardwareIni"="[INSTALLDIR]CONFIG\\HARDWARE.INI"
"SRDFileName"="[INSTALLDIR]HWDETECT.SRD"
"dllname"="[INSTALLDIR]PHardwareDetect.dll"
```

---

## Notes

- Files are UTF-16LE encoded (BOM: `FF FE`)
- Variable syntax `[VARNAME]` must be preserved as literal strings
- Path separator in `WOSA/XFS_ROOT` uses forward slash (valid registry path)
- The large `proakt.reg` files are good stress tests but start with smaller files

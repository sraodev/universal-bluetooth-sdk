// Package linuxrfcomm is the BlueZ-backed RFCOMM transport driver.
//
// On Linux, Send opens a SOCK_STREAM/BTPROTO_RFCOMM socket directly via
// the kernel — no shell-out, no PyBluez. Discover lists known/paired
// peers via `bluetoothctl devices` (BlueZ ships it on every distro that
// has the stack at all). Live scanning is left to a future BlueZ-D-Bus
// integration; the typed RPC contract is the same either way.
//
// On non-Linux the driver registers with Capability().Discover = false
// and Send/Discover return CodeNotImplemented, so the daemon stays
// portable: ubtd boots, advertises a degraded capability matrix, and
// the AI planner / typed CLI can react instead of crashing.
package linuxrfcomm

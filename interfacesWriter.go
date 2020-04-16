package netif

import (
	"fmt"
	"os"

	"strings"

	"github.com/n-marshall/fn"
	cp "github.com/n-marshall/go-cp"
)

func BackupPath(path string) fn.Option {
	return fn.MakeOption("backupPath", path)
}

//[changed] mgmtName added to append "ip addr flush" to management port settings
func (is *InterfaceSet) Write(mgmtName string, opts ...fn.Option) error {
	fnConfig := fn.MakeConfig(
		fn.Defaults{"path": "/etc/network/interfaces"},
		opts,
	)
	path := fnConfig.GetString("path")
	backupPath := fnConfig.GetString("backupPath")

	if backupPath == "" {
		backupPath = path + ".bak"
	}

	// Backup interface file
	err := copyFileIfExists(path, backupPath)
	if err != nil {
		return err
	}

	// try to open the interface file for writing
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		// Restore interface file
		err := copyFileIfExists(backupPath, path)
		if err != nil {
			return err
		}

		return err
	}
	defer f.Close()

	// write interface file

	err = is.WriteToFile(mgmtName, f)
	if err != nil {
		// Restore interface file
		err := copyFileIfExists(backupPath, path)
		if err != nil {
			return err
		}
	}

	return nil
}

func copyFileIfExists(path, backupPath string) error {
	if _, err := os.Stat(path); err == nil {
		err2 := cp.CopyFile(path, backupPath)
		if err2 != nil {
			return err
		}
	}
	return nil
}

//[changed] mgmtName added to append "ip addr flush" to management port settings
func (is *InterfaceSet) WriteToFile(mgmtName string, f *os.File) error {
	for _, adapter := range is.Adapters {
		adapterString, err := adapter.writeString()
		if err != nil {
			return err
		}

		fmt.Fprintf(f, "%s", adapterString)

		//[changed] mgmtName added to append "ip addr flush" to management port settings
		if mgmtName != "" && adapter.Name == mgmtName {
			fmt.Fprintf(f, "\n    post-down ip addr flush dev %s", mgmtName)
		}
		fmt.Fprintf(f, "\n\n")
	}
	return nil
}

func (a *NetworkAdapter) writeString() (string, error) {
	var lines []string
	if a.Auto {
		lines = append(lines, fmt.Sprintf("auto %s", a.Name))
	}
	if a.Hotplug {
		lines = append(lines, fmt.Sprintf("allow-hotplug %s", a.Name))
	}

	lines = append(lines, a.writeAddressFamily())

	if a.AddrSource == STATIC || a.AddrSource == MANUAL {
		for _, line := range a.writeIPLines() {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n"), nil
}

func (a *NetworkAdapter) GetAddrFamilyString() string {
	switch a.AddrFamily {
	case INET:
		return "inet"
	case INET6:
		return "inet6"
	}
	return "inet"
}

func (a *NetworkAdapter) GetSourceFamilyString() string {
	switch a.AddrSource {
	case DHCP:
		return "dhcp"
	case STATIC:
		return "static"
	case LOOPBACK:
		return "loopback"
	case MANUAL:
		return "manual"
	}
	return "dhcp"
}

func (a *NetworkAdapter) writeAddressFamily() string {
	var familyStr = a.GetAddrFamilyString()
	var sourceStr = a.GetSourceFamilyString()
	return fmt.Sprintf("iface %s %s %s", a.Name, familyStr, sourceStr)
}

func (a *NetworkAdapter) writeIPLines() (lines []string) {
	if a.Address != nil {
		lines = append(lines, fmt.Sprintf("    address %s", a.Address))
	}
	if a.Netmask != nil {
		lines = append(lines, fmt.Sprintf("    netmask %s", a.Netmask))
	}
	if a.Network != nil {
		lines = append(lines, fmt.Sprintf("    network %s", a.Network))
	}
	if a.Broadcast != nil {
		lines = append(lines, fmt.Sprintf("    broadcast %s", a.Broadcast))
	}
	if a.Gateway != nil {
		lines = append(lines, fmt.Sprintf("    gateway %s", a.Gateway))
	}
	if len(a.DNSNS) != 0 {
		message := ""
		for _, dnssvr := range a.DNSNS {
			message = message + " " + dnssvr.String()
		}
		lines = append(lines, fmt.Sprintf("    dns-nameservers %s", message))
	}
	return
}

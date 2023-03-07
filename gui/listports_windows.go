package gui

import (
	"fmt"

	"github.com/roffe/gocan/adapter/passthru"
	"go.bug.st/serial/enumerator"
)

func (m *mainWindow) listPorts() []string {
	var portsList []string
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		m.output(err.Error())
		return []string{}
	}
	if len(ports) == 0 {
		m.output("No serial ports found!")
		//return []string{}
	}
	for _, port := range ports {
		m.output(fmt.Sprintf("Found port: %s", port.Name))
		if port.IsUSB {
			m.output(fmt.Sprintf("  USB ID     %s:%s", port.VID, port.PID))
			m.output(fmt.Sprintf("  USB serial %s", port.SerialNumber))
			portsList = append(portsList, port.Name)
		}
	}

	dlls := passthru.FindDLLs()
	for _, dll := range dlls {
		m.output(fmt.Sprintf("J2534 DLL: %s", dll.Name))
		//portsList = append(portsList, filepath.Base(dll.FunctionLibrary))
		portsList = append(portsList, dll.Name)
	}

	state.portList = portsList
	return portsList
}

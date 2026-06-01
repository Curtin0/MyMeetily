//go:build !windows

package audio

import "fmt"

func listWASAPICaptureDevices() ([]string, error) {
	return nil, fmt.Errorf("WASAPI 仅支持 Windows")
}

func listWASAPILoopbackDevices() ([]string, error) {
	return nil, fmt.Errorf("WASAPI 仅支持 Windows")
}
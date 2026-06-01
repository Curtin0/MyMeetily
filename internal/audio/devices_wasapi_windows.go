//go:build windows

package audio

import (
	"fmt"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

type wasapiDevice struct {
	ID    string
	Name  string
	Flow  uint32
	Label string
}

func listWASAPICaptureDevices() ([]string, error) {
	devices, err := listWASAPIDevices(wca.ECapture)
	if err != nil {
		return nil, fmt.Errorf("枚举 WASAPI 麦克风设备: %w", err)
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("未检测到 WASAPI 麦克风设备")
	}
	return devices, nil
}

func listWASAPILoopbackDevices() ([]string, error) {
	devices, err := listWASAPIDevices(wca.ERender)
	if err != nil {
		return nil, fmt.Errorf("枚举 WASAPI 系统音频设备: %w", err)
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("未检测到 WASAPI 系统音频设备")
	}
	return devices, nil
}

func listWASAPIDevices(flow uint32) ([]string, error) {
	descriptors, err := listWASAPIDeviceDescriptors(flow)
	if err != nil {
		return nil, err
	}

	devices := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		devices = append(devices, descriptor.Label)
	}

	return devices, nil
}

func listWASAPIDeviceDescriptors(flow uint32) ([]wasapiDevice, error) {
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return nil, err
	}
	defer ole.CoUninitialize()

	return enumerateWASAPIDeviceDescriptors(flow)
}

func enumerateWASAPIDeviceDescriptors(flow uint32) ([]wasapiDevice, error) {
	enumerator, err := newWASAPIDeviceEnumerator()
	if err != nil {
		return nil, err
	}
	defer enumerator.Release()

	return enumerateWASAPIDeviceDescriptorsWithEnumerator(enumerator, flow)
}

func newWASAPIDeviceEnumerator() (*wca.IMMDeviceEnumerator, error) {

	var enumerator *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(
		wca.CLSID_MMDeviceEnumerator,
		0,
		wca.CLSCTX_ALL,
		wca.IID_IMMDeviceEnumerator,
		&enumerator,
	); err != nil {
		return nil, err
	}

	return enumerator, nil
}

	func enumerateWASAPIDeviceDescriptorsWithEnumerator(enumerator *wca.IMMDeviceEnumerator, flow uint32) ([]wasapiDevice, error) {
	var collection *wca.IMMDeviceCollection
	if err := enumerator.EnumAudioEndpoints(flow, wca.DEVICE_STATE_ACTIVE, &collection); err != nil {
		return nil, err
	}
	if collection == nil {
		return nil, nil
	}
	defer collection.Release()

	var count uint32
	if err := collection.GetCount(&count); err != nil {
		return nil, err
	}

	devices := make([]wasapiDevice, 0, count)
	for index := uint32(0); index < count; index++ {
		var device *wca.IMMDevice
		if err := collection.Item(index, &device); err != nil {
			return nil, fmt.Errorf("读取音频设备 %d: %w", index, err)
		}

		descriptor, err := readWASAPIDeviceDescriptor(device, flow)
		device.Release()
		if err != nil {
			return nil, err
		}

		devices = append(devices, descriptor)
	}

	return devices, nil
}

func findWASAPIDeviceBySelection(enumerator *wca.IMMDeviceEnumerator, flow uint32, selected string) (*wca.IMMDevice, wasapiDevice, error) {
	var zero wasapiDevice

	var collection *wca.IMMDeviceCollection
	if err := enumerator.EnumAudioEndpoints(flow, wca.DEVICE_STATE_ACTIVE, &collection); err != nil {
		return nil, zero, err
	}
	if collection == nil {
		return nil, zero, fmt.Errorf("未检测到 WASAPI 设备")
	}
	defer collection.Release()

	var count uint32
	if err := collection.GetCount(&count); err != nil {
		return nil, zero, err
	}

	for index := uint32(0); index < count; index++ {
		var device *wca.IMMDevice
		if err := collection.Item(index, &device); err != nil {
			return nil, zero, fmt.Errorf("读取音频设备 %d: %w", index, err)
		}

		descriptor, err := readWASAPIDeviceDescriptor(device, flow)
		if err != nil {
			device.Release()
			return nil, zero, err
		}

		if selected == "" || selected == descriptor.Label || selected == descriptor.ID || selected == descriptor.Name {
			return device, descriptor, nil
		}

		device.Release()
	}

	return nil, zero, fmt.Errorf("未找到所选 WASAPI 设备: %s", selected)
}

func readWASAPIDeviceDescriptor(device *wca.IMMDevice, flow uint32) (wasapiDevice, error) {
	var deviceID string
	if err := device.GetId(&deviceID); err != nil {
		return wasapiDevice{}, fmt.Errorf("读取 WASAPI 设备 ID: %w", err)
	}

	var props *wca.IPropertyStore
	if err := device.OpenPropertyStore(wca.STGM_READ, &props); err != nil {
		return wasapiDevice{}, fmt.Errorf("打开 WASAPI 设备属性 %s: %w", deviceID, err)
	}
	defer props.Release()

	var value wca.PROPVARIANT
	if err := props.GetValue(&wca.PKEY_Device_FriendlyName, &value); err != nil {
		return wasapiDevice{}, fmt.Errorf("读取 WASAPI 设备名称 %s: %w", deviceID, err)
	}

	name := strings.TrimSpace(value.String())
	if name == "" {
		name = "Unknown Device"
	}

	return wasapiDevice{
		ID:    deviceID,
		Name:  name,
		Flow:  flow,
		Label: fmt.Sprintf("%s [%s]", name, deviceID),
	}, nil
}
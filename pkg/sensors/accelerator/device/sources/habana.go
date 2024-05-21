//go:build habana
// +build habana

/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package sources

import (
	"errors"
	"time"

	hlml "github.com/HabanaAI/gohlml"
	dev "github.com/sustainable-computing-io/kepler/pkg/sensors/accelerator/device"
	"k8s.io/klog/v2"

	"github.com/sustainable-computing-io/kepler/pkg/config"
)

const (
	habanaDevice = "habana"
	habanaHwType = "gpu"
)

var (
	habanaAccImpl = GPUHabana{}
)

type GPUHabana struct {
	collectionSupported bool
	devices             map[int]dev.Device
}

func init() {
	if err := habanaAccImpl.InitLib(); err != nil {
		klog.Infof("Error initializing %s: %v", habanaAccImpl.GetName(), err)
	}
	klog.Infof("Using %s to obtain processor power", habanaAccImpl.GetName())
	dev.AddDeviceInterface(habanaDevice, habanaHwType, habanaDeviceStartup)
}

func habanaDeviceStartup() (dev.AcceleratorInterface, error) {

	if dType != habanaDevice {
		return nil, errors.New("invalid device type")
	}

	a := habanaAccImpl

	if err := a.Init(); err != nil {
		klog.Errorf("failed to StartupDevice: %v", err)
		return nil, err
	}

	return &a, nil
}

func (g *GPUHabana) GetName() string {
	return habanaDevice
}

func (g *GPUHabana) GetType() string {
	return habanaDevice
}

func (g *GPUHabana) GetHwType() string {
	return habanaHwType
}

func (g *GPUHabana) InitLib() error {
	return nil
}

// todo: refactor logic at invoking side, if gpu is not set?
func (g *GPUHabana) Init() error {
	ret := hlml.Initialize()
	if ret != nil {
		klog.Error("ERROR initializing hlml")
		g.collectionSupported = false
	} else {
		klog.Info("Initialized hlml and enabling collection support")
		g.collectionSupported = true
	}
	return ret
}

func (g *GPUHabana) Shutdown() bool {
	if ret := hlml.Shutdown(); ret != nil {
		return false
	}
	return true
}

func (g *GPUHabana) GetAbsEnergyFromDevice() []uint32 {
	gpuEnergy := []uint32{}

	for _, device := range g.devices {
		power, ret := device.DeviceHandler.(hlml.Device).PowerUsage()
		if ret != nil {
			klog.V(2).Infof("failed to get power usage on device %v: %v\n", device, ret)
			continue
		}
		energy := uint32(uint64(power) * config.SamplePeriodSec)
		gpuEnergy = append(gpuEnergy, energy)

		dname, _ := device.DeviceHandler.(hlml.Device).Name()
		klog.V(2).Infof("GetAbsEnergyFromDevice power usage on device %v: %v\n", dname, gpuEnergy)
	}

	return gpuEnergy
}

func (g *GPUHabana) GetDevices() map[int]dev.Device {
	count, ret := hlml.DeviceCount()
	if ret != nil {
		return nil
	}
	g.devices = make(map[int]dev.Device, count)
	for i := 0; i < int(count); i++ {
		if h, ret := hlml.DeviceHandleByIndex(uint(i)); ret == nil {
			g.devices[i] = dev.Device{
				DeviceHandler: h,
			}
		}
	}
	return g.devices
}

func (g *GPUHabana) GetDeviceInstances() map[int]map[int]dev.Device {
	var devices map[int]map[int]dev.Device
	return devices
}

func (g *GPUHabana) GetProcessResourceUtilizationPerDevice(device dev.Device, since time.Duration) (map[uint32]dev.ProcessUtilizationSample, error) {
	processAcceleratorMetrics := map[uint32]dev.ProcessUtilizationSample{}
	return processAcceleratorMetrics, nil
}

func (g *GPUHabana) IsDeviceCollectionSupported() bool {
	return g.collectionSupported
}

func (g *GPUHabana) SetDeviceCollectionSupported(supported bool) {
	g.collectionSupported = supported
}

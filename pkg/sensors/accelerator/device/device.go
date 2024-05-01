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

package device

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/exp/maps"
	"k8s.io/klog/v2"
)

type ProcessUtilizationSample struct {
	Pid         uint32
	TimeStamp   uint64
	ComputeUtil uint32
	MemUtil     uint32
	EncUtil     uint32
	DecUtil     uint32
}

// Device can hold GPU Device or Multi Instance GPU slice handler
type Device struct {
	DeviceHandler interface{}
	ID            int // Entity ID or Parent ID if Subdevice
	IsSubdevice   bool
	ParentID      int     // GPU Entity ID  or Parent GPU ID if MIG slice
	MIGSMRatio    float64 // Ratio of MIG SMs / Total GPU SMs to be used to normalize the MIG metrics
}

type AcceleratorInterface interface {
	// GetName returns the name of the device
	GetName() string
	// GetType returns the type of the device (nvml, qat, dcgm ...)
	GetType() string
	// GetHwType returns the type of hw the device is (gpu, processor)
	GetHwType() string
	// Init the external library loading, if any.
	InitLib() error
	// Init initizalize and start the metric device
	Init() error
	// Shutdown stops the metric device
	Shutdown() bool
	// GetDevices returns a map with devices
	GetDevices() map[int]Device
	// GetDeviceInstances returns a map with instances of each Device
	GetDeviceInstances() map[int]map[int]Device
	// TODO UPDATE GetAbsEnergyFromDevice returns a map with mJ in each gpu device. Absolute energy is the sum of Idle + Dynamic energy.
	GetAbsEnergyFromDevice() []uint32
	// GetProcessResourceUtilizationPerDevice returns a map of ProcessUtilizationSample where the key is the process pid
	GetProcessResourceUtilizationPerDevice(device Device, since time.Duration) (map[uint32]ProcessUtilizationSample, error)
	// IsDeviceCollectionSupported returns if it is possible to use this device
	IsDeviceCollectionSupported() bool
	// SetDeviceCollectionSupported manually set if it is possible to use this device. This is for testing purpose only.
	SetDeviceCollectionSupported(bool)
}

// Function prototype to create a new deviceCollector.
type deviceStartupFunc func(accType string) (AcceleratorInterface, error)

// Static map of supported acceleratorInterface.
var acceleratorInterfaces = map[string]deviceStartupFunc{}

// Adds a supported device interface, prints a fatal error in the case of double registration.
func AddDeviceInterface(name string, deviceStartup deviceStartupFunc) {
	if acceleratorInterfaces[name] != nil {
		klog.Fatalf("Multiple AcceleratorInterfaces attempting to register with name %q", name)
	}

	klog.Infof("Registered %s", name)

	acceleratorInterfaces[name] = deviceStartup
}

func GetAcceleratorInterfaces() []string {
	// TODO check the len of acceleratorInterfaces
	return maps.Keys(acceleratorInterfaces)
}

// StartupDevice Returns a new AcceleratorInterface according the required accType[nvml|dcgm|dummy|habana].
func StartupDevice(accType string) (AcceleratorInterface, error) {
	// We'll panic if accType are nil, this is intentional
	deviceStartup, ok := acceleratorInterfaces[accType]
	if !ok {
		var deviceList strings.Builder

		for d := range acceleratorInterfaces {
			if deviceList.Len() > 0 {
				deviceList.WriteString(", ")
			}

			deviceList.WriteString(d)
		}

		return nil, fmt.Errorf("unsupported device Type %s; supported accTypes: %s", accType, deviceList.String())
	}

	klog.Infof("Starting up %s", accType)

	return deviceStartup(accType)
}

//go:build dummy
// +build dummy

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

	acc "github.com/sustainable-computing-io/kepler/pkg/sensors/accelerator/device"
)

const (
	deviceType = "dummy"
)

type Dummy struct {
	deviceType          string
	name                string
	collectionSupported bool
}

func init() {
	acc.AddDeviceInterface(deviceType, dummyDeviceStartup)
}

func dummyDeviceStartup(dType string) (acc.AcceleratorInterface, error) {

	if dType != deviceType {
		return nil, errors.New("invalid device type")
	}

	d := Dummy{
		deviceType:          deviceType,
		name:                "dummy",
		collectionSupported: false,
	}

	return &d, nil
}

func (d *Dummy) GetName() string {
	return d.name
}

func (d *Dummy) GetType() string {
	return d.deviceType
}

func (d *Dummy) GetHwType() string {
	return d.deviceType
}

func (d *Dummy) InitLib() error {
	return nil
}

func (d *Dummy) Init() error {
	return nil
}

func (d *Dummy) Shutdown() bool {
	return true
}

func (d *Dummy) GetAbsEnergyFromDevice() []uint32 {
	return nil
}

func (d *Dummy) GetDevices() map[int]acc.Device {
	return nil
}

func (d *Dummy) GetDeviceInstances() map[int]map[int]acc.Device {
	return nil
}

func (d *Dummy) GetProcessResourceUtilizationPerDevice(device acc.Device, since time.Duration) (map[uint32]acc.ProcessUtilizationSample, error) {
	processAcceleratorMetrics := map[uint32]acc.ProcessUtilizationSample{}
	processAcceleratorMetrics[0] = acc.ProcessUtilizationSample{
		Pid:         0,
		TimeStamp:   uint64(time.Now().UnixNano()),
		ComputeUtil: 10,
		MemUtil:     10,
		EncUtil:     10,
		DecUtil:     10,
	}
	return processAcceleratorMetrics, nil
}

func (d *Dummy) IsDeviceCollectionSupported() bool {
	return d.collectionSupported
}

func (d *Dummy) SetDeviceCollectionSupported(supported bool) {
	d.collectionSupported = supported
}

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

package accelerator

//nolint:gci // The supported gpu imports are kept separate.
import (
	"golang.org/x/exp/slices"

	"github.com/pkg/errors"
	dev "github.com/sustainable-computing-io/kepler/pkg/sensors/accelerator/device"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	// Add supported devices.
	_ "github.com/sustainable-computing-io/kepler/pkg/sensors/accelerator/device/sources"
)

var (
	accelerators map[string]Accelerator
)

// Accelerator represents an implementation of... equivalent CableEngine
type Accelerator interface {
	// StartupAccelerator ...
	StartupAccelerator() error
	// GetAccelerator ...
	GetAccelerator() dev.AcceleratorInterface
	// GetAcceleratorType ...
	GetAcceleratorType() string
	// IsRunning ...
	IsRunning() bool
	// StopAccelerator ...
	StopAccelerator() error
}

type accelerator struct {
	acc           dev.AcceleratorInterface // Device Accelerator Interface
	accType       string                   // NVML|DCGM|Dummy
	running       bool
	installedtime metav1.Time
}

func GetAccelerators() map[string]Accelerator {
	return accelerators
}

// NewAccelerator creates a new Accelerator instance [NVML|DCGM|DUMMY] for the local node.
func NewAccelerator(accType string) Accelerator {

	containsType := slices.Contains(dev.GetAcceleratorInterfaces(), accType)
	if !containsType {
		klog.Error("Invalid Device Type")
		return nil
	}

	_, ok := accelerators[accType] // e.g. accelerators[nvml|dcgm|habana|dummy]
	if ok {
		klog.Infof("Accelerator with type %s already exists", accType)
		return accelerators[accType]
	}

	accelerators = map[string]Accelerator{
		accType: &accelerator{
			acc:           nil,
			running:       false,
			accType:       accType,
			installedtime: metav1.Time{},
		},
	}

	return accelerators[accType]
}

// StartupAccelerator of a particular type
func (a *accelerator) StartupAccelerator() error {
	var err error
	if a.acc, err = dev.StartupDevice(a.accType); err != nil {
		return errors.Wrap(err, "error creating the acc")
	}

	a.running = true
	a.installedtime = metav1.Now()

	klog.Infof("Accelerator started with acc type %s", a.accType)

	return nil
}

func (a *accelerator) StopAccelerator() error {
	if a.acc.Shutdown() != true {
		return errors.New("error shutting down the accelerator acc")
	}

	delete(accelerators, a.accType)

	a.running = false

	klog.Info("Accelerator acc stopped")

	return nil
}

func (a *accelerator) GetAcceleratorType() string {
	return a.accType
}

func (a *accelerator) IsRunning() bool {
	return a.IsRunning()
}

func (a *accelerator) GetAccelerator() dev.AcceleratorInterface {
	return a.acc
}

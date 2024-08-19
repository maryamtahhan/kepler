package accelerator

import (
	"testing"

	"github.com/sustainable-computing-io/kepler/pkg/sensors/accelerator/devices"
)

func newMockDevice() devices.Device {
	return devices.Startup(devices.MOCK.String())
}

func cleanupMockDevice() {
	Shutdown()
}

func TestRegistry(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() *Registry
		expectedLen int
		expectError bool
		cleanup     func()
	}{
		{
			name: "Empty registry",
			setup: func() *Registry {
				registry := Registry{
					Registry: map[string]Accelerator{},
				}
				SetRegistry(&registry)

				return GetRegistry()
			},
			expectedLen: 0,
			expectError: false,
			cleanup:     func() { cleanupMockDevice() },
		},
		{
			name: "Non-empty registry",
			setup: func() *Registry {
				registry := &Registry{
					Registry: map[string]Accelerator{},
				}
				SetRegistry(registry)
				devices.RegisterMockDevice()
				a := &accelerator{
					dev:     newMockDevice(),
					running: true,
				}
				registry.MustRegister(a)

				return GetRegistry()
			},
			expectedLen: 1,
			expectError: false,
			cleanup:     func() { cleanupMockDevice() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var a Accelerator
			var err error
			registry := tt.setup()
			if a, err = New("MOCK", true); err == nil {
				registry.MustRegister(a) // Register the accelerator with the registry
			}
			accs := registry.Accelerators()
			if tt.expectError && err == nil {
				t.Errorf("expected an error but got nil")
			}
			if tt.expectError && err != nil {
				t.Errorf("did not expect an error but got %v", err)
			}
			if len(accs) != tt.expectedLen {
				t.Errorf("expected %d accelerators, but got %d", tt.expectedLen, len(accs))
			}
			tt.cleanup()
		})
	}
}

func TestActiveAcceleratorByType(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() *Registry
		expectError bool
		cleanup     func()
	}{
		{
			name: "No accelerators of given type",
			setup: func() *Registry {
				return &Registry{
					Registry: map[string]Accelerator{},
				}
			},
			expectError: true,
			cleanup:     func() { cleanupMockDevice() },
		},
		{
			name: "One active accelerator of given type",
			setup: func() *Registry {
				registry := &Registry{
					Registry: map[string]Accelerator{},
				}
				SetRegistry(registry)
				devices.RegisterMockDevice()
				a := &accelerator{
					dev:     newMockDevice(),
					running: true,
				}
				registry.MustRegister(a)

				return GetRegistry()
			},
			expectError: false,
			cleanup:     func() { cleanupMockDevice() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := tt.setup()
			accs := registry.ActiveAcceleratorByType("MOCK")
			if tt.expectError && accs != nil {
				t.Errorf("expected an error")
			}
			if !tt.expectError && accs == nil {
				t.Errorf("did not expect an error")
			}
			tt.cleanup()
		})
	}
}

func TestCreateAndRegister(t *testing.T) {
	tests := []struct {
		name        string
		accType     string
		setup       func() *Registry
		sleep       bool
		expectError bool
		cleanup     func()
	}{
		{
			name:    "Unsupported accelerator",
			accType: "UNSUPPORTED", // invalid accelerator type
			setup: func() *Registry {
				return &Registry{
					Registry: map[string]Accelerator{},
				}
			},
			sleep:       false,
			expectError: true,
			cleanup:     func() { cleanupMockDevice() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := tt.setup()
			SetRegistry(registry)
			_, err := New(tt.accType, tt.sleep)
			if tt.expectError && err == nil {
				t.Errorf("expected an error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("did not expect an error but got %v", err)
			}
			tt.cleanup()
		})
	}
}

func TestShutdown(t *testing.T) {
	tests := []struct {
		name  string
		setup func() *Registry
	}{
		{
			name: "Shutdown active accelerators",
			setup: func() *Registry {
				registry := &Registry{
					Registry: map[string]Accelerator{},
				}

				SetRegistry(registry)
				devices.RegisterMockDevice()
				a := &accelerator{
					dev:     newMockDevice(),
					running: true,
				}
				registry.MustRegister(a)
				return GetRegistry()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			Shutdown()

			accs := GetRegistry().Accelerators()
			for _, a := range accs {
				if a.IsRunning() {
					t.Errorf("expected accelerator to be stopped but it is still running")
				}
			}
		})
	}
}

func TestAcceleratorMethods(t *testing.T) {
	registry := &Registry{
		Registry: map[string]Accelerator{},
	}

	SetRegistry(registry)

	devices.RegisterMockDevice()

	acc := &accelerator{
		dev:     newMockDevice(),
		running: true,
	}
	registry.MustRegister(acc)

	devType := acc.dev.HwType()

	if got := acc.Device(); got.HwType() != devType {
		t.Errorf("expected device type %v, got %v", devType, got.DevType())
	}
	if got := acc.Device().HwType(); got != devType {
		t.Errorf("expected device type %v, got %v", devType, got)
	}
	if got := acc.IsRunning(); !got {
		t.Errorf("expected accelerator to be running, got %v", got)
	}
	if got := acc.AccType().String(); got != devType {
		t.Errorf("expected accelerator type devices.DeviceType(0), got %v", got)
	}
	Shutdown()
}

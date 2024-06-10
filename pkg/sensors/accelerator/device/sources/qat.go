/*
Copyright 2023.

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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jaypipes/ghw"
	"github.com/sustainable-computing-io/kepler/pkg/sensors/accelerator/device"
	dev "github.com/sustainable-computing-io/kepler/pkg/sensors/accelerator/device"
	"github.com/sustainable-computing-io/kepler/pkg/sensors/accelerator/device/utils"
	"k8s.io/klog/v2"
)

const (
	// Turn off telemetry
	off = iota
	// Turn on telemetry
	on
	qatDevice = "qat"
)

var (
	// telemetry base path
	teleBasePath = "/sys/kernel/debug/qat_%s_%s/telemetry/"

	// control telemetry switch path
	controlPath = filepath.Join(teleBasePath, "control")

	// obtain device utilization data path
	deviceDataPath = filepath.Join(teleBasePath, "device_data")
	// List of QAT qatDevInfo for the device

)

type qatDevInfo struct {
	bdf      string
	driver   string
	datafile *os.File
}

type QATTelemetry struct {
	collectionSupported bool
	devices             map[string]qatDevInfo
}

func init() {
	if _, err := getQATDevices(); err != nil {
		klog.Errorf("Error initializing %s: %v", qatDevice, err)
		return
	}
	dev.AddDeviceInterface(qatDevice, qatDevice, qatDeviceStartup)
}

func qatDeviceStartup() (dev.AcceleratorInterface, error) {
	q := QATTelemetry{
		collectionSupported: false,
	}

	if err := q.Init(); err != nil {
		klog.Errorf("Error initializing %s: %v", qatDevice, err)
		return nil, err
	}

	klog.Info("QAT power metrics collection initialized")

	return &q, nil
}

func (q *QATTelemetry) GetName() string {
	return qatDevice
}

func (q *QATTelemetry) GetType() string {
	return qatDevice
}

func (q *QATTelemetry) GetHwType() string {
	return qatDevice
}

func (q *QATTelemetry) InitLib() error {
	return nil
}

// Init initialize and start the QAT metric collector
func (q *QATTelemetry) Init() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("could not init telemetry:%s", err)
		}
	}()

	// get qat devices
	q.devices, err = getQATDevices()
	if err != nil {
		q.collectionSupported = false
		return err
	}

	// turn on telemetry
	if err = controlTelemetry(q.devices, on); err != nil {
		klog.Errorf("failed to start telemetry: %v\n", err)
		return err
	}

	// open the telemetry data file
	q.devices, err = openDataFile(q.devices)
	if err != nil {
		klog.Errorf("failed to open telemetry data file: %v\n", err)
		return err
	}

	klog.Infof("found %d QAT devices\n", len(q.devices))
	q.collectionSupported = true
	return nil
}

func (q *QATTelemetry) Shutdown() bool {
	var err error
	// close telemetry data file
	if err = closeDataFile(q.devices); err != nil {
		return false
	}
	// turn off telemetry
	if err = controlTelemetry(q.devices, off); err != nil {
		return false
	}
	return true
}

func (q *QATTelemetry) GetAbsEnergyFromDevice() []uint32 {
	return nil
}

func (q *QATTelemetry) GetDevicesByID() map[int]any {
	devices := make(map[int]interface{})
	return devices
}

func (q *QATTelemetry) GetDevicesByName() map[string]any {
	devices := make(map[string]interface{})
	for k, v := range q.devices {
		devices[k] = v
	}
	return devices
}

func (q *QATTelemetry) GetDeviceInstances() map[int]map[int]any {
	instances := make(map[int]map[int]interface{})
	return instances
}

func (q *QATTelemetry) GetProcessResourceUtilizationPerDevice(device any, since time.Duration) (map[uint32]interface{}, error) {
	pam := make(map[uint32]interface{}) // Process Accelerator Metrics
	return pam, nil
}

func (q *QATTelemetry) GetDeviceUtilizationStats(qat any) (map[any]interface{}, error) {
	qatMetrics := map[string]dev.QATUtilizationSample{}
	qam := make(map[any]interface{})

	switch d := qat.(type) {
	case qatDevInfo:
		file := d.datafile
		deviceUtil, err := getUtilization(file)
		if err != nil {
			klog.Errorf("failed to get qat utilization on device %s: %v\n", d.bdf, err)
			return qam, err
		}
		qatMetrics[d.bdf] = deviceUtil
		for k, v := range qatMetrics {
			qam[k] = v
		}
		return qam, nil
	default:
		klog.Error("expected qatDevInfo but got come other type")
		return qam, errors.New("invalid device type")
	}
}

func (q *QATTelemetry) IsDeviceCollectionSupported() bool {
	return q.collectionSupported
}

func (q *QATTelemetry) SetDeviceCollectionSupported(supported bool) {
	q.collectionSupported = supported
}

// getQATDevices obtain available qat devices and search for ID
func getQATDevices() (map[string]qatDevInfo, error) {
	devices := make(map[string]qatDevInfo)
	pci, err := ghw.PCI()
	if err != nil {
		klog.Errorf("Error getting PCI info: %v", err)
		return nil, err
	}
	pci_devices := pci.ListDevices()
	if len(pci_devices) == 0 {
		fmt.Printf("error: could not retrieve PCI devices\n")
		return nil, fmt.Errorf("unable to find an available QAT device. Please check the status of QAT")
	}

	// for _, device := range pci_devices {
	// 	vendor := device.Vendor
	// 	vendorName := vendor.Name
	// 	if len(vendor.Name) > 20 {
	// 		vendorName = string([]byte(vendorName)[0:17]) + "..."
	// 	}
	// 	product := device.Product
	// 	productName := product.Name
	// 	if len(product.Name) > 40 {
	// 		productName = string([]byte(productName)[0:37]) + "..."
	// 	}
	// 	fmt.Printf("%-12s\t%-20s\t%-40s\n", device.Address, vendorName, productName)
	// }

	for _, device := range pci_devices {
		product := device.Product
		productName := product.Name
		if len(product.Name) > 40 {
			productName = string([]byte(productName)[0:37]) + "..."
		}
		if strings.Contains(productName, string([]byte("QAT"))) {
			if utils.IsQATStatusUp(device.Address) {
				devices[productName] = qatDevInfo{
					bdf:    device.Address,
					driver: device.Driver,
				}
			} else {
				klog.Errorf("found a QAT device %s but its status is down, skipping", device.Address)
			}
		}
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("unable to find an available QAT device. Please check the status of QAT")
	}

	return devices, nil
}

// controlTelemetry obtain control paths based on QAT information, then turn on/off telemetry
func controlTelemetry(devices map[string]qatDevInfo, mode int) error {
	var err error
	for qatDev, qatInfo := range devices {
		// path to control the telemetry switch
		bsf := qatInfo.bdf
		driver := qatInfo.driver
		path := fmt.Sprintf(controlPath, driver, bsf)

		// turn on/off telemetry
		err = switchTelemetry(path, mode)
		if err != nil {
			klog.Errorf("failed to control %s with mode %d: %s ", qatDev, mode, err)
			delete(devices, qatDev)
		}
	}

	if len(devices) == 0 {
		return fmt.Errorf("unable to control any QAT device. Please check the status of QAT")
	}

	return err
}

// switchTelemetry turn on/off telemetry
func switchTelemetry(filename string, mode int) error {
	file, err := os.OpenFile(filename, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// turn on/off telemetry
	_, err = file.WriteString(strconv.Itoa(mode))
	if err != nil {
		return err
	}
	return nil
}

// openDataFile open qat telemetry data file, and obtain available devices
func openDataFile(devices map[string]qatDevInfo) (map[string]qatDevInfo, error) {
	// available devices
	availableDev := make(map[string]qatDevInfo)
	for qatDev, qatinfo := range devices {
		// dataPath that can read data from telemetry
		bsf := qatinfo.bdf

		dataPath := fmt.Sprintf(deviceDataPath, bsf, bsf)

		f, err := os.OpenFile(dataPath, os.O_RDONLY, 0444)
		if err != nil {
			klog.Errorf("failed to open %s telemetry data file: %v\n", qatDev, err)
			delete(devices, qatDev)
			continue
		}
		availableDev[qatDev] = qatDevInfo{bdf: bsf, datafile: f}
	}

	if len(availableDev) == 0 {
		return nil, fmt.Errorf("unable to open any telemetry data file for QAT. Please check the status of QAT")
	}
	return availableDev, nil
}

// closeDataFile close qat telemetry data file
func closeDataFile(devices map[string]qatDevInfo) error {
	var err error
	if len(devices) == 0 {
		return nil
	}
	for qatDev, info := range devices {
		file := info.datafile
		if err = file.Close(); err != nil {
			return fmt.Errorf("failed to close %s telemetry data file: %v", qatDev, err)
		}
	}
	return nil
}

// getUtilization calculate utilization from each qat device
func getUtilization(file *os.File) (device.QATUtilizationSample, error) {
	utilizationSample := device.QATUtilizationSample{}

	// reset file pointer to the beginning
	_, err := file.Seek(0, 0)
	if err != nil {
		return utilizationSample, fmt.Errorf("failed to reset file pointer: %s", err)
	}

	// get all data from telemetry
	data, err := io.ReadAll(file)
	if err != nil {
		return utilizationSample, fmt.Errorf("failed to read device_data file: %s", err)
	}
	if len(data) == 0 {
		return utilizationSample, fmt.Errorf("failed to get device_data, the length of data is zero")
	}

	return processData(strings.Fields(string(data))), nil
}

// processData calculate all telemetry data
func processData(data []string) device.QATUtilizationSample {
	var (
		// compression utilization of all slices
		cprSum uint64
		// decompression utilization of all slices
		dcprSum uint64
		// translator utilization of all slices
		xltSum uint64
		// cipher utilization of on all slices
		cphSum uint64
		// authentication utilization of all slices
		athSum uint64
	)

	// preprocess data by converting []string into map[string]uint64
	output := make(map[string]uint64)

	for i := 0; i < len(data)-1; i += 2 {
		key := data[i]
		value, _ := strconv.ParseUint(data[i+1], 10, 64)
		output[key] = value
	}

	// calculate the utilization of different functions(compression, decopression...)
	for key, value := range output {
		if strings.HasPrefix(key, "util_cpr") {
			cprSum += value
			continue
		}
		if strings.HasPrefix(key, "util_dcpr") {
			dcprSum += value
			continue
		}
		if strings.HasPrefix(key, "util_xlt") {
			xltSum += value
			continue
		}
		if strings.HasPrefix(key, "util_cph") {
			cphSum += value
			continue
		}
		if strings.HasPrefix(key, "util_ath") {
			athSum += value
			continue
		}
	}

	return device.QATUtilizationSample{
		SampleCnt:   output["sample_cnt"],
		PciTransCnt: output["pci_trans_cnt"],
		Latency:     output["lat_acc_avg"],
		BwIn:        output["bw_in"],
		BwOut:       output["bw_out"],
		CprUtil:     cprSum,
		DcprUtil:    dcprSum,
		XltUtil:     xltSum,
		CphUtil:     cphSum,
		AthUtil:     athSum,
	}
}

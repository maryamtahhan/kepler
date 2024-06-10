// Copyright 2018 Intel Corp. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	sysBusPci = "/sys/bus/pci/devices"
)

const (
	totalVfFile      = "sriov_totalvfs"
	configuredVfFile = "sriov_numvfs"
)

// GetVFList returns a List containing PCI addr for all VF discovered in a given PF
func GetVFList(pf string) (vfList []string, err error) {
	vfList = make([]string, 0)
	pfDir := filepath.Join(sysBusPci, pf)
	_, err = os.Lstat(pfDir)
	if err != nil {
		err = fmt.Errorf("Error. Could not get PF directory information for device: %s, Err: %v", pf, err)
		return
	}

	vfDirs, err := filepath.Glob(filepath.Join(pfDir, "virtfn*"))

	if err != nil {
		err = fmt.Errorf("error reading VF directories %v", err)
		return
	}

	//Read all VF directory and get add VF PCI addr to the vfList
	for _, dir := range vfDirs {
		dirInfo, err := os.Lstat(dir)
		if err == nil && (dirInfo.Mode()&os.ModeSymlink != 0) {
			linkName, err := filepath.EvalSymlinks(dir)
			if err == nil {
				vfLink := filepath.Base(linkName)
				vfList = append(vfList, vfLink)
			}
		}
	}
	return
}

// GetPciAddrFromVFID returns PCI address for VF ID
func GetPciAddrFromVFID(pf string, vf int) (pciAddr string, err error) {
	vfDir := fmt.Sprintf("%s/%s/virtfn%d", sysBusPci, pf, vf)
	dirInfo, err := os.Lstat(vfDir)
	if err != nil {
		err = fmt.Errorf("Error. Could not get directory information for device: %s, VF: %v. Err: %v", pf, vf, err)
		return "", err
	}

	if (dirInfo.Mode() & os.ModeSymlink) == 0 {
		err = fmt.Errorf("Error. No symbolic link between virtual function and PCI - Device: %s, VF: %v", pf, vf)
		return
	}

	pciInfo, err := os.Readlink(vfDir)
	if err != nil {
		err = fmt.Errorf("Error. Cannot read symbolic link between virtual function and PCI - Device: %s, VF: %v. Err: %v", pf, vf, err)
		return
	}

	pciAddr = pciInfo[len("../"):]
	return
}

// GetSriovVFcapacity returns SRIOV VF capacity
func GetSriovVFcapacity(pf string) int {
	totalVfFilePath := filepath.Join(sysBusPci, pf, totalVfFile)
	vfs, err := ioutil.ReadFile(totalVfFilePath)
	if err != nil {
		return 0
	}
	totalvfs := bytes.TrimSpace(vfs)
	numvfs, err := strconv.Atoi(string(totalvfs))
	if err != nil {
		return 0
	}
	return numvfs
}

// IsQATStatusUp returns 'false' if 'operstate' is not "up" for a QAT device.
// This function will only return 'false' if the 'operstate' file of the device is readable
// and holds value anything other than "up". Or else we assume QAT device is up.
func IsQATStatusUp(dev string) bool {

	if opsFiles, err := filepath.Glob(filepath.Join(sysBusPci, dev, "qat", "*", "operstate")); err == nil {
		for _, f := range opsFiles {
			bytes, err := ioutil.ReadFile(f)
			if err != nil || strings.TrimSpace(string(bytes)) != "up" {
				return false
			}
		}
	}
	return true
}

// SriovConfigured returns true if sriov_numvfs reads > 0 else false
func SriovConfigured(addr string) bool {
	if GetVFconfigured(addr) > 0 {
		return true
	}
	return false
}

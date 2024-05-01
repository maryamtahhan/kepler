/*
Copyright 2021.

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

package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime/debug"
	"time"

	"github.com/sustainable-computing-io/kepler/pkg/bpf"
	"github.com/sustainable-computing-io/kepler/pkg/collector/stats"
	"github.com/sustainable-computing-io/kepler/pkg/config"
	"github.com/sustainable-computing-io/kepler/pkg/manager"
	acc "github.com/sustainable-computing-io/kepler/pkg/sensors/accelerator"
	"github.com/sustainable-computing-io/kepler/pkg/sensors/components"
	"github.com/sustainable-computing-io/kepler/pkg/sensors/platform"
	kversion "github.com/sustainable-computing-io/kepler/pkg/version"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/klog/v2"
)

const (
	// to change these msg, you also need to update the e2e test
	finishingMsg = "Exiting..."
	startedMsg   = "Started Kepler in %s"
)

var (
	address                      = flag.String("address", "0.0.0.0:8888", "bind address")
	metricsPath                  = flag.String("metrics-path", "/metrics", "metrics path")
	enableGPU                    = flag.Bool("enable-gpu", false, "whether enable gpu (need to have libnvidia-ml installed)")
	enableQAT                    = flag.Bool("enable-qat", false, "whether enable qat (need to have Intel QAT driver installed)")
	enabledEBPFCgroupID          = flag.Bool("enable-cgroup-id", true, "whether enable eBPF to collect cgroup id (must have kernel version >= 4.18 and cGroup v2)")
	exposeHardwareCounterMetrics = flag.Bool("expose-hardware-counter-metrics", true, "whether expose hardware counter as prometheus metrics")
	enabledMSR                   = flag.Bool("enable-msr", false, "whether MSR is allowed to obtain energy data")
	kubeconfig                   = flag.String("kubeconfig", "", "absolute path to the kubeconfig file, if empty we use the in-cluster configuration")
	apiserverEnabled             = flag.Bool("apiserver", true, "if apiserver is disabled, we collect pod information from kubelet")
	redfishCredFilePath          = flag.String("redfish-cred-file-path", "", "path to the redfish credential file")
	exposeEstimatedIdlePower     = flag.Bool("expose-estimated-idle-power", false, "estimated idle power is meaningful only if Kepler is running on bare-metal or when there is only one virtual machine on the node")
)

func healthProbe(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(`ok`))
	if err != nil {
		klog.Fatalf("%s", fmt.Sprintf("failed to write response: %v", err))
	}
}

func finalizing() {
	if config.EnabledGPU {
		if gpus, err := acc.GetActiveAcceleratorsByType("gpu"); err == nil {
			for _, a := range gpus {
				a.StopAccelerator()
			}
		}
	}
	if config.EnabledQAT {
		if qats, err := acc.GetActiveAcceleratorsByType("qat"); err == nil {
			for _, a := range qats {
				a.StopAccelerator()
			}
		}
	}
	stack := "exit stack: \n" + string(debug.Stack())
	klog.Infof(stack)
	exitCode := 10
	klog.Infoln(finishingMsg)
	klog.FlushAndExit(klog.ExitFlushTimeout, exitCode)
}

func main() {
	start := time.Now()
	defer finalizing()
	klog.InitFlags(nil)
	flag.Parse()

	klog.Infof("Kepler running on version: %s", kversion.Version)

	config.SetEnabledEBPFCgroupID(*enabledEBPFCgroupID)
	config.SetEnabledHardwareCounterMetrics(*exposeHardwareCounterMetrics)
	config.SetEnabledGPU(*enableGPU)
	config.SetEnabledQAT(*enableQAT)
	config.EnabledMSR = *enabledMSR
	config.SetEnabledIdlePower(*exposeEstimatedIdlePower || components.IsSystemCollectionSupported())

	config.SetKubeConfig(*kubeconfig)
	config.SetEnableAPIServer(*apiserverEnabled)

	// set redfish credential file path
	if *redfishCredFilePath != "" {
		config.SetRedfishCredFilePath(*redfishCredFilePath)
	}

	config.LogConfigs()

	components.InitPowerImpl()
	platform.InitPowerImpl()

	bpfExporter, err := bpf.NewExporter()
	if err != nil {
		klog.Fatalf("failed to create eBPF exporter: %v", err)
	}
	defer bpfExporter.Detach()

	stats.InitAvailableParamAndMetrics()

	if config.EnabledGPU {
		if err := acc.InitAcc("gpu", true); err != nil {
			klog.Fatalf("failed to init GPU accelerators: %v", err)
		}
	}

	if config.IsExposeQATMetricsEnabled() {
		if err := acc.InitAcc("qat", false); err != nil {
			klog.Fatalf("failed to init QAT accelerators: %v", err)
		}
	}

	m := manager.New(bpfExporter)
	reg := m.PrometheusCollector.RegisterMetrics()
	defer components.StopPower()

	// starting a new gorotine to collect data and report metrics
	// BPF is attached here
	if startErr := m.Start(); startErr != nil {
		klog.Infof("%s", fmt.Sprintf("failed to start : %v", startErr))
	}
	metricPathConfig := config.GetMetricPath(*metricsPath)
	bindAddressConfig := config.GetBindAddress(*address)

	http.Handle(metricPathConfig, promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{
			Registry: reg,
		},
	))
	http.HandleFunc("/healthz", healthProbe)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, httpErr := w.Write([]byte(`<html>
                        <head><title>Energy Stats Exporter</title></head>
                        <body>
                        <h1>Energy Stats Exporter</h1>
                        <p><a href="` + metricPathConfig + `">Metrics</a></p>
                        </body>
                        </html>`))
		if err != nil {
			klog.Fatalf("%s", fmt.Sprintf("failed to write response: %v", httpErr))
		}
	})

	klog.Infof("starting to listen on %s", bindAddressConfig)
	ch := make(chan error)
	go func() {
		ch <- http.ListenAndServe(bindAddressConfig, nil)
	}()

	klog.Infof(startedMsg, time.Since(start))
	klog.Flush() // force flush to parse the start msg in the e2e test
	err = <-ch
	klog.Fatalf("%s", fmt.Sprintf("failed to bind on %s: %v", bindAddressConfig, err))
}

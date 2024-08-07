// Code generated by bpf2go; DO NOT EDIT.
//go:build mips || mips64 || ppc64 || s390x

package bpf

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"

	"github.com/cilium/ebpf"
)

type keplerProcessMetricsT struct {
	CgroupId       uint64
	Pid            uint64
	ProcessRunTime uint64
	CpuCycles      uint64
	CpuInstr       uint64
	CacheMiss      uint64
	PageCacheHit   uint64
	VecNr          [10]uint16
	Comm           [16]int8
	_              [4]byte
}

// loadKepler returns the embedded CollectionSpec for kepler.
func loadKepler() (*ebpf.CollectionSpec, error) {
	reader := bytes.NewReader(_KeplerBytes)
	spec, err := ebpf.LoadCollectionSpecFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("can't load kepler: %w", err)
	}

	return spec, err
}

// loadKeplerObjects loads kepler and converts it into a struct.
//
// The following types are suitable as obj argument:
//
//	*keplerObjects
//	*keplerPrograms
//	*keplerMaps
//
// See ebpf.CollectionSpec.LoadAndAssign documentation for details.
func loadKeplerObjects(obj interface{}, opts *ebpf.CollectionOptions) error {
	spec, err := loadKepler()
	if err != nil {
		return err
	}

	return spec.LoadAndAssign(obj, opts)
}

// keplerSpecs contains maps and programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type keplerSpecs struct {
	keplerProgramSpecs
	keplerMapSpecs
}

// keplerSpecs contains programs before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type keplerProgramSpecs struct {
	KeplerIrqTrace         *ebpf.ProgramSpec `ebpf:"kepler_irq_trace"`
	KeplerReadPageTrace    *ebpf.ProgramSpec `ebpf:"kepler_read_page_trace"`
	KeplerSchedSwitchTrace *ebpf.ProgramSpec `ebpf:"kepler_sched_switch_trace"`
	KeplerWritePageTrace   *ebpf.ProgramSpec `ebpf:"kepler_write_page_trace"`
}

// keplerMapSpecs contains maps before they are loaded into the kernel.
//
// It can be passed ebpf.CollectionSpec.Assign.
type keplerMapSpecs struct {
	CacheMiss                  *ebpf.MapSpec `ebpf:"cache_miss"`
	CacheMissEventReader       *ebpf.MapSpec `ebpf:"cache_miss_event_reader"`
	CpuCycles                  *ebpf.MapSpec `ebpf:"cpu_cycles"`
	CpuCyclesEventReader       *ebpf.MapSpec `ebpf:"cpu_cycles_event_reader"`
	CpuInstructions            *ebpf.MapSpec `ebpf:"cpu_instructions"`
	CpuInstructionsEventReader *ebpf.MapSpec `ebpf:"cpu_instructions_event_reader"`
	PidTimeMap                 *ebpf.MapSpec `ebpf:"pid_time_map"`
	Processes                  *ebpf.MapSpec `ebpf:"processes"`
}

// keplerObjects contains all objects after they have been loaded into the kernel.
//
// It can be passed to loadKeplerObjects or ebpf.CollectionSpec.LoadAndAssign.
type keplerObjects struct {
	keplerPrograms
	keplerMaps
}

func (o *keplerObjects) Close() error {
	return _KeplerClose(
		&o.keplerPrograms,
		&o.keplerMaps,
	)
}

// keplerMaps contains all maps after they have been loaded into the kernel.
//
// It can be passed to loadKeplerObjects or ebpf.CollectionSpec.LoadAndAssign.
type keplerMaps struct {
	CacheMiss                  *ebpf.Map `ebpf:"cache_miss"`
	CacheMissEventReader       *ebpf.Map `ebpf:"cache_miss_event_reader"`
	CpuCycles                  *ebpf.Map `ebpf:"cpu_cycles"`
	CpuCyclesEventReader       *ebpf.Map `ebpf:"cpu_cycles_event_reader"`
	CpuInstructions            *ebpf.Map `ebpf:"cpu_instructions"`
	CpuInstructionsEventReader *ebpf.Map `ebpf:"cpu_instructions_event_reader"`
	PidTimeMap                 *ebpf.Map `ebpf:"pid_time_map"`
	Processes                  *ebpf.Map `ebpf:"processes"`
}

func (m *keplerMaps) Close() error {
	return _KeplerClose(
		m.CacheMiss,
		m.CacheMissEventReader,
		m.CpuCycles,
		m.CpuCyclesEventReader,
		m.CpuInstructions,
		m.CpuInstructionsEventReader,
		m.PidTimeMap,
		m.Processes,
	)
}

// keplerPrograms contains all programs after they have been loaded into the kernel.
//
// It can be passed to loadKeplerObjects or ebpf.CollectionSpec.LoadAndAssign.
type keplerPrograms struct {
	KeplerIrqTrace         *ebpf.Program `ebpf:"kepler_irq_trace"`
	KeplerReadPageTrace    *ebpf.Program `ebpf:"kepler_read_page_trace"`
	KeplerSchedSwitchTrace *ebpf.Program `ebpf:"kepler_sched_switch_trace"`
	KeplerWritePageTrace   *ebpf.Program `ebpf:"kepler_write_page_trace"`
}

func (p *keplerPrograms) Close() error {
	return _KeplerClose(
		p.KeplerIrqTrace,
		p.KeplerReadPageTrace,
		p.KeplerSchedSwitchTrace,
		p.KeplerWritePageTrace,
	)
}

func _KeplerClose(closers ...io.Closer) error {
	for _, closer := range closers {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Do not access this directly.
//
//go:embed kepler_bpfeb.o
var _KeplerBytes []byte

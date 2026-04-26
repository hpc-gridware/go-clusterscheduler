package qhost_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qhost "github.com/hpc-gridware/go-clusterscheduler/pkg/qhost/v9.1"
)

var _ = Describe("Parsers v9.1", func() {

	// Plain qhost output (no -F flags, no attribute lines).
	qhostPlain := `HOSTNAME                ARCH         NCPU NSOC NCOR NTHR  LOAD  MEMTOT  MEMUSE  SWAPTO  SWAPUS
----------------------------------------------------------------------------------------------
global                  -               -    -    -    -     -       -       -       -       -
master                  lx-amd64        4    1    4    4  0.31   15.6G  422.9M    1.5G     0.0
exec                    lx-amd64        8    1    8    8  0.50   31.2G  800.0M    3.0G     0.0
`

	// v9.1 qhost -F output: 4-space indent, integer num_proc/socket/core/thread,
	// no m_topology_inuse, hc:slots as a custom consumable resource.
	qhostFV91 := `HOSTNAME                ARCH         NCPU NSOC NCOR NTHR  LOAD  MEMTOT  MEMUSE  SWAPTO  SWAPUS
----------------------------------------------------------------------------------------------
global                  -               -    -    -    -     -       -       -       -       -
master                  lx-amd64        4    1    4    4  0.31   15.6G  422.9M    1.5G     0.0
    hl:arch=lx-amd64
    hl:num_proc=4
    hl:mem_total=15.617G
    hl:swap_total=1.500G
    hl:virtual_total=17.117G
    hl:load_avg=0.310000
    hl:load_short=0.450000
    hl:load_medium=0.310000
    hl:load_long=0.280000
    hl:mem_free=15.197G
    hl:swap_free=1.500G
    hl:virtual_free=16.697G
    hl:mem_used=422.900M
    hl:swap_used=0.000
    hl:virtual_used=422.900M
    hl:cpu=0.100000
    hl:m_topology=SCCCC
    hl:m_socket=1
    hl:m_core=4
    hl:m_thread=4
    hl:np_load_avg=0.077500
    hl:np_load_short=0.112500
    hl:np_load_medium=0.077500
    hl:np_load_long=0.070000
    hc:slots=4.000000
exec                    lx-amd64        8    1    8    8  0.50   31.2G  800.0M    3.0G     0.0
    hl:arch=lx-amd64
    hl:num_proc=8
    hl:mem_total=31.200G
    hl:swap_total=3.000G
    hl:virtual_total=34.200G
    hl:load_avg=0.500000
    hl:load_short=0.600000
    hl:load_medium=0.500000
    hl:load_long=0.400000
    hl:mem_free=30.400G
    hl:swap_free=3.000G
    hl:virtual_free=33.400G
    hl:mem_used=800.000M
    hl:swap_used=0.000
    hl:virtual_used=800.000M
    hl:cpu=0.300000
    hl:m_topology=SCCCCCCCC
    hl:m_socket=1
    hl:m_core=8
    hl:m_thread=8
    hl:np_load_avg=0.062500
    hl:np_load_short=0.075000
    hl:np_load_medium=0.062500
    hl:np_load_long=0.050000
    hc:slots=8.000000
`

	Context("ParseHosts", func() {

		It("parses v9.1 plain qhost output (global row included)", func() {
			hosts, err := qhost.ParseHosts(qhostPlain)
			Expect(err).To(BeNil())
			Expect(hosts).To(HaveLen(3))
			Expect(hosts[0].Name).To(Equal("global"))
			Expect(hosts[0].NCPU).To(Equal(0))
			Expect(hosts[0].NSOC).To(Equal(0))
			Expect(hosts[0].LOAD).To(Equal(float64(0)))
			Expect(hosts[1].Name).To(Equal("master"))
			Expect(hosts[1].NCPU).To(Equal(4))
			Expect(hosts[1].NSOC).To(Equal(1))
			Expect(hosts[1].NCOR).To(Equal(4))
			Expect(hosts[1].NTHR).To(Equal(4))
			Expect(hosts[2].Name).To(Equal("exec"))
			Expect(hosts[2].NCPU).To(Equal(8))
		})

		It("tolerates exec rows whose values are all '-'", func() {
			input := `HOSTNAME                ARCH         NCPU NSOC NCOR NTHR  LOAD  MEMTOT  MEMUSE  SWAPTO  SWAPUS
----------------------------------------------------------------------------------------------
global                  -               -    -    -    -     -       -       -       -       -
unreachable             -               -    -    -    -     -       -       -       -       -
master                  lx-amd64        4    1    4    4  0.31   15.6G  422.9M    1.5G     0.0
`
			hosts, err := qhost.ParseHosts(input)
			Expect(err).To(BeNil())
			Expect(hosts).To(HaveLen(3))
			Expect(hosts[1].Name).To(Equal("unreachable"))
			Expect(hosts[1].NCPU).To(Equal(0))
		})

	})

	Context("ParseHostsRaw", func() {

		It("returns column tokens verbatim including '-'", func() {
			raw, err := qhost.ParseHostsRaw(qhostPlain)
			Expect(err).To(BeNil())
			Expect(raw).To(HaveLen(3))
			Expect(raw[0].Name).To(Equal("global"))
			Expect(raw[0].Cols).To(Equal([]string{"-", "-", "-", "-", "-", "-", "-", "-", "-", "-"}))
			Expect(raw[1].Name).To(Equal("master"))
			Expect(raw[1].Cols[0]).To(Equal("lx-amd64"))
			Expect(raw[1].Cols[6]).To(Equal("15.6G"))
		})

	})

	Context("ParseHostFullMetrics", func() {

		It("parses v9.1 qhost -F output with 4-space indent and integer fields", func() {
			hosts, err := qhost.ParseHostFullMetrics(qhostFV91)
			Expect(err).To(BeNil())
			// global + master + exec
			Expect(hosts).To(HaveLen(3))

			Expect(hosts[0].Name).To(Equal("global"))

			master := hosts[1]
			Expect(master.Name).To(Equal("master"))
			Expect(master.NumProc).To(Equal(float64(4)))
			Expect(master.Socket).To(Equal(int64(1)))
			Expect(master.Core).To(Equal(int64(4)))
			Expect(master.Thread).To(Equal(int64(4)))
			// m_topology_inuse absent - should remain empty string
			Expect(master.TopologyInuse).To(Equal(""))
			// hc:slots is a custom consumable resource
			Expect(master.Resources).To(HaveKey("slots"))
			Expect(master.Resources["slots"].FloatValue).To(Equal(float64(4)))

			exec := hosts[2]
			Expect(exec.Name).To(Equal("exec"))
			Expect(exec.NumProc).To(Equal(float64(8)))
			Expect(exec.Socket).To(Equal(int64(1)))
			Expect(exec.Core).To(Equal(int64(8)))
			Expect(exec.Thread).To(Equal(int64(8)))
			Expect(exec.Resources).To(HaveKey("slots"))
			Expect(exec.Resources["slots"].FloatValue).To(Equal(float64(8)))
		})

	})

})

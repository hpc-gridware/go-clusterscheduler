package qhost_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	qhost "github.com/hpc-gridware/go-clusterscheduler/pkg/qhost/v9.0"
)

var _ = Describe("Parsers", func() {

	sample := `
HOSTNAME                ARCH         NCPU NSOC NCOR NTHR  LOAD  MEMTOT  MEMUSE  SWAPTO  SWAPUS
----------------------------------------------------------------------------------------------
global                  -               -    -    -    -     -       -       -       -       -
master                  lx-amd64        4    1    4    4  0.31   15.6G  422.9M    1.5G     0.0
exec                    lx-amd64        4    1    4    4  0.31   15.6G  422.9M    1.5G     0.0
`
	Context("ParseQhostOutput", func() {

		It("should return error if output is invalid", func() {
			hosts, err := qhost.ParseHosts(sample)
			Expect(err).To(BeNil())
			Expect(hosts).To(HaveLen(2))
			Expect(hosts[0].Name).To(Equal("master"))
			Expect(hosts[0].Arch).To(Equal("lx-amd64"))
			Expect(hosts[0].NCPU).To(Equal(4))
			Expect(hosts[0].NSOC).To(Equal(1))
			Expect(hosts[0].NCOR).To(Equal(4))
			Expect(hosts[0].NTHR).To(Equal(4))
			Expect(hosts[0].LOAD).To(Equal(0.31))
			Expect(hosts[0].MEMTOT).To(Equal(int64(156 * 1024 * 1024 * 1024 / 10)))
			Expect(hosts[0].MEMUSE).To(Equal(int64(4229 * 1024 * 1024 / 10)))
			Expect(hosts[0].SWAPTO).To(Equal(int64(1.5 * 1024 * 1024 * 1024)))
			Expect(hosts[0].SWAPUS).To(Equal(int64(0.0)))
			Expect(hosts[1].Name).To(Equal("exec"))
			Expect(hosts[1].Arch).To(Equal("lx-amd64"))
			Expect(hosts[1].NCPU).To(Equal(4))
			Expect(hosts[1].NSOC).To(Equal(1))
			Expect(hosts[1].NCOR).To(Equal(4))
			Expect(hosts[1].NTHR).To(Equal(4))
			Expect(hosts[1].LOAD).To(Equal(0.31))
			Expect(hosts[1].MEMTOT).To(Equal(int64(156 * 1024 * 1024 * 1024 / 10)))
			Expect(hosts[1].MEMUSE).To(Equal(int64(4229 * 1024 * 1024 / 10)))
			Expect(hosts[1].SWAPTO).To(Equal(int64(1.5 * 1024 * 1024 * 1024)))
			Expect(hosts[1].SWAPUS).To(Equal(int64(0.0)))
		})

	})

	Context("ParseHostFullMetrics", func() {

		qhostFOutput1 := `HOSTNAME                ARCH         NCPU NSOC NCOR NTHR  LOAD  MEMTOT  MEMUSE  SWAPTO  SWAPUS
----------------------------------------------------------------------------------------------
global                  -               -    -    -    -     -       -       -       -       -
master                  lx-amd64        4    1    4    4  0.60   15.6G  465.8M    1.5G     0.0
   hl:arch=lx-amd64
   hl:num_proc=4.000000
   hl:mem_total=15.617G
   hl:swap_total=1.500G
   hl:virtual_total=17.117G
   hl:load_avg=0.600000
   hl:load_short=0.700000
   hl:load_medium=0.600000
   hl:load_long=0.440000
   hl:mem_free=15.162G
   hl:swap_free=1.500G
   hl:virtual_free=16.662G
   hl:mem_used=465.824M
   hl:swap_used=0.000
   hl:virtual_used=465.824M
   hl:cpu=0.200000
   hl:m_topology=SCCCC
   hl:m_topology_inuse=SCCCC
   hl:m_socket=1.000000
   hl:m_core=4.000000
   hl:m_thread=4.000000
   hl:np_load_avg=0.150000
   hl:np_load_short=0.175000
   hl:np_load_medium=0.150000
   hl:np_load_long=0.110000
sim1                    lx-amd64        4    1    4    4  0.60   15.6G  465.8M    1.5G     0.0
   hl:load_avg=0.600000
   hl:load_short=0.700000
   hl:load_medium=0.600000
   hl:load_long=0.440000
   hl:arch=lx-amd64
   hl:num_proc=4.000000
   hl:mem_free=15.162G
   hl:swap_free=1.500G
   hl:virtual_free=16.662G
   hl:mem_total=15.617G
   hl:swap_total=1.500G
   hl:virtual_total=17.117G
   hl:mem_used=465.824M
   hl:swap_used=0.000
   hl:virtual_used=465.824M
   hl:cpu=0.200000
   hl:m_topology=SCCCC
   hl:m_topology_inuse=SCCCC
   hl:m_socket=1.000000
   hl:m_core=4.000000
   hl:m_thread=4.000000
   hl:np_load_avg=0.150000
   hl:np_load_short=0.175000
   hl:np_load_medium=0.150000
   hl:np_load_long=0.110000
   hf:load_report_host=master
sim10                   lx-amd64        4    1    4    4  0.60   15.6G  465.8M    1.5G     0.0
   hl:load_avg=0.600000
   hl:load_short=0.700000
   hl:load_medium=0.600000
   hl:load_long=0.440000
   hl:arch=lx-amd64
   hl:num_proc=4.000000
   hl:mem_free=15.162G
   hl:swap_free=1.500G
   hl:virtual_free=16.662G
   hl:mem_total=15.617G
   hl:swap_total=1.500G
   hl:virtual_total=17.117G
   hl:mem_used=465.824M
   hl:swap_used=0.000
   hl:virtual_used=465.824M
   hl:cpu=0.200000
   hl:m_topology=SCCCC
   hl:m_topology_inuse=SCCCC
   hl:m_socket=1.000000
   hl:m_core=4.000000
   hl:m_thread=4.000000
   hl:np_load_avg=0.150000
   hl:np_load_short=0.175000
   hl:np_load_medium=0.150000
   hl:np_load_long=0.110000
   hf:load_report_host=master
sim11                   lx-amd64        4    1    4    4  0.60   15.6G  465.8M    1.5G     0.0
   hl:load_avg=0.600000
   hl:load_short=0.700000
   hl:load_medium=0.600000
   hl:load_long=0.440000
   hl:arch=lx-amd64
   hl:num_proc=4.000000
   hl:mem_free=15.162G
   hl:swap_free=1.500G
   hl:virtual_free=16.662G
   hl:mem_total=15.617G
   hl:swap_total=1.500G
   hl:virtual_total=17.117G
   hl:mem_used=465.824M
   hl:swap_used=0.000
   hl:virtual_used=465.824M
   hl:cpu=0.200000
   hl:m_topology=SCCCC
   hl:m_topology_inuse=SCCCC
   hl:m_socket=1.000000
   hl:m_core=4.000000
   hl:m_thread=4.000000
   hl:np_load_avg=0.150000
   hl:np_load_short=0.175000
   hl:np_load_medium=0.150000
   hl:np_load_long=0.110000
   hf:load_report_host=master
sim12                   lx-amd64        4    1    4    4  0.60   15.6G  465.8M    1.5G     0.0
   hl:load_avg=0.600000
   hl:load_short=0.700000
   hl:load_medium=0.600000
   hl:load_long=0.440000
   hl:arch=lx-amd64
   hl:num_proc=4.000000
   hl:mem_free=15.162G
   hl:swap_free=1.500G
   hl:virtual_free=16.662G
   hl:mem_total=15.617G
   hl:swap_total=1.500G
   hl:virtual_total=17.117G
   hl:mem_used=465.824M
   hl:swap_used=0.000
   hl:virtual_used=465.824M
   hl:cpu=0.200000
   hl:m_topology=SCCCC
   hl:m_topology_inuse=SCCCC
   hl:m_socket=1.000000
   hl:m_core=4.000000
   hl:m_thread=4.000000
   hl:np_load_avg=0.150000
   hl:np_load_short=0.175000
   hl:np_load_medium=0.150000
   hl:np_load_long=0.110000
   hf:load_report_host=master
sim2                    lx-amd64        4    1    4    4  0.60   15.6G  465.8M    1.5G     0.0
   hl:load_avg=0.600000
   hl:load_short=0.700000
   hl:load_medium=0.600000
   hl:load_long=0.440000
   hl:arch=lx-amd64
   hl:num_proc=4.000000
   hl:mem_free=15.162G
   hl:swap_free=1.500G
   hl:virtual_free=16.662G
   hl:mem_total=15.617G
   hl:swap_total=1.500G
   hl:virtual_total=17.117G
   hl:mem_used=465.824M
   hl:swap_used=0.000
   hl:virtual_used=465.824M
   hl:cpu=0.200000
   hl:m_topology=SCCCC
   hl:m_topology_inuse=SCCCC
   hl:m_socket=1.000000
   hl:m_core=4.000000
   hl:m_thread=4.000000
   hl:np_load_avg=0.150000
   hl:np_load_short=0.175000
   hl:np_load_medium=0.150000
   hl:np_load_long=0.110000
   hf:load_report_host=master
sim3                    lx-amd64        4    1    4    4  0.60   15.6G  465.8M    1.5G     0.0
   hl:load_avg=0.600000
   hl:load_short=0.700000
   hl:load_medium=0.600000
   hl:load_long=0.440000
   hl:arch=lx-amd64
   hl:num_proc=4.000000
   hl:mem_free=15.162G
   hl:swap_free=1.500G
   hl:virtual_free=16.662G
   hl:mem_total=15.617G
   hl:swap_total=1.500G
   hl:virtual_total=17.117G
   hl:mem_used=465.824M
   hl:swap_used=0.000
   hl:virtual_used=465.824M
   hl:cpu=0.200000
   hl:m_topology=SCCCC
   hl:m_topology_inuse=SCCCC
   hl:m_socket=1.000000
   hl:m_core=4.000000
   hl:m_thread=4.000000
   hl:np_load_avg=0.150000
   hl:np_load_short=0.175000
   hl:np_load_medium=0.150000
   hl:np_load_long=0.110000
   hf:load_report_host=master
sim4                    lx-amd64        4    1    4    4  0.60   15.6G  465.8M    1.5G     0.0
   hl:load_avg=0.600000
   hl:load_short=0.700000
   hl:load_medium=0.600000
   hl:load_long=0.440000
   hl:arch=lx-amd64
   hl:num_proc=4.000000
   hl:mem_free=15.162G
   hl:swap_free=1.500G
   hl:virtual_free=16.662G
   hl:mem_total=15.617G
   hl:swap_total=1.500G
   hl:virtual_total=17.117G
   hl:mem_used=465.824M
   hl:swap_used=0.000
   hl:virtual_used=465.824M
   hl:cpu=0.200000
   hl:m_topology=SCCCC
   hl:m_topology_inuse=SCCCC
   hl:m_socket=1.000000
   hl:m_core=4.000000
   hl:m_thread=4.000000
   hl:np_load_avg=0.150000
   hl:np_load_short=0.175000
   hl:np_load_medium=0.150000
   hl:np_load_long=0.110000
   hf:load_report_host=master
sim5                    lx-amd64        4    1    4    4  0.60   15.6G  465.8M    1.5G     0.0
   hl:load_avg=0.600000
   hl:load_short=0.700000
   hl:load_medium=0.600000
   hl:load_long=0.440000
   hl:arch=lx-amd64
   hl:num_proc=4.000000
   hl:mem_free=15.162G
   hl:swap_free=1.500G
   hl:virtual_free=16.662G
   hl:mem_total=15.617G
   hl:swap_total=1.500G
   hl:virtual_total=17.117G
   hl:mem_used=465.824M
   hl:swap_used=0.000
   hl:virtual_used=465.824M
   hl:cpu=0.200000
   hl:m_topology=SCCCC
   hl:m_topology_inuse=SCCCC
   hl:m_socket=1.000000
   hl:m_core=4.000000
   hl:m_thread=4.000000
   hl:np_load_avg=0.150000
   hl:np_load_short=0.175000
   hl:np_load_medium=0.150000
   hl:np_load_long=0.110000
   hf:load_report_host=master
sim6                    lx-amd64        4    1    4    4  0.60   15.6G  465.8M    1.5G     0.0
   hl:load_avg=0.600000
   hl:load_short=0.700000
   hl:load_medium=0.600000
   hl:load_long=0.440000
   hl:arch=lx-amd64
   hl:num_proc=4.000000
   hl:mem_free=15.162G
   hl:swap_free=1.500G
   hl:virtual_free=16.662G
   hl:mem_total=15.617G
   hl:swap_total=1.500G
   hl:virtual_total=17.117G
   hl:mem_used=465.824M
   hl:swap_used=0.000
   hl:virtual_used=465.824M
   hl:cpu=0.200000
   hl:m_topology=SCCCC
   hl:m_topology_inuse=SCCCC
   hl:m_socket=1.000000
   hl:m_core=4.000000
   hl:m_thread=4.000000
   hl:np_load_avg=0.150000
   hl:np_load_short=0.175000
   hl:np_load_medium=0.150000
   hl:np_load_long=0.110000
   hf:load_report_host=master
sim7                    lx-amd64        4    1    4    4  0.60   15.6G  465.8M    1.5G     0.0
   hl:load_avg=0.600000
   hl:load_short=0.700000
   hl:load_medium=0.600000
   hl:load_long=0.440000
   hl:arch=lx-amd64
   hl:num_proc=4.000000
   hl:mem_free=15.162G
   hl:swap_free=1.500G
   hl:virtual_free=16.662G
   hl:mem_total=15.617G
   hl:swap_total=1.500G
   hl:virtual_total=17.117G
   hl:mem_used=465.824M
   hl:swap_used=0.000
   hl:virtual_used=465.824M
   hl:cpu=0.200000
   hl:m_topology=SCCCC
   hl:m_topology_inuse=SCCCC
   hl:m_socket=1.000000
   hl:m_core=4.000000
   hl:m_thread=4.000000
   hl:np_load_avg=0.150000
   hl:np_load_short=0.175000
   hl:np_load_medium=0.150000
   hl:np_load_long=0.110000
   hf:load_report_host=master
sim8                    lx-amd64        4    1    4    4  0.60   15.6G  465.8M    1.5G     0.0
   hl:load_avg=0.600000
   hl:load_short=0.700000
   hl:load_medium=0.600000
   hl:load_long=0.440000
   hl:arch=lx-amd64
   hl:num_proc=4.000000
   hl:mem_free=15.162G
   hl:swap_free=1.500G
   hl:virtual_free=16.662G
   hl:mem_total=15.617G
   hl:swap_total=1.500G
   hl:virtual_total=17.117G
   hl:mem_used=465.824M
   hl:swap_used=0.000
   hl:virtual_used=465.824M
   hl:cpu=0.200000
   hl:m_topology=SCCCC
   hl:m_topology_inuse=SCCCC
   hl:m_socket=1.000000
   hl:m_core=4.000000
   hl:m_thread=4.000000
   hl:np_load_avg=0.150000
   hl:np_load_short=0.175000
   hl:np_load_medium=0.150000
   hl:np_load_long=0.110000
   hf:load_report_host=master
sim9                    lx-amd64        4    1    4    4  0.60   15.6G  465.8M    1.5G     0.0
   hl:load_avg=0.600000
   hl:load_short=0.700000
   hl:load_medium=0.600000
   hl:load_long=0.440000
   hl:arch=lx-amd64
   hl:num_proc=4.000000
   hl:mem_free=15.162G
   hl:swap_free=1.500G
   hl:virtual_free=16.662G
   hl:mem_total=15.617G
   hl:swap_total=1.500G
   hl:virtual_total=17.117G
   hl:mem_used=465.824M
   hl:swap_used=0.000
   hl:virtual_used=465.824M
   hl:cpu=0.200000
   hl:m_topology=SCCCC
   hl:m_topology_inuse=SCCCC
   hl:m_socket=1.000000
   hl:m_core=4.000000
   hl:m_thread=4.000000
   hl:np_load_avg=0.150000
   hl:np_load_short=0.175000
   hl:np_load_medium=0.150000
   hl:np_load_long=0.110000
   hf:load_report_host=master`

		qhostFOutput2 := `HOSTNAME                ARCH         NCPU NSOC NCOR NTHR  LOAD  MEMTOT  MEMUSE  SWAPTO  SWAPUS
----------------------------------------------------------------------------------------------
global                  -               -    -    -    -     -       -       -       -       -
   gc:testc=100000.000000
master                  lx-amd64       14    1   14   14  1.50    7.7G    2.0G 1024.0M   12.0K
   gc:testc=100000.000000
   hl:load_avg=1.500000
   hl:load_short=1.670000
   hl:load_medium=1.500000
   hl:load_long=1.100000
   hl:arch=lx-amd64
   hl:num_proc=14.000000
   hl:mem_free=5.621G
   hl:swap_free=1023.984M
   hl:virtual_free=6.621G
   hl:mem_total=7.653G
   hl:swap_total=1023.996M
   hl:virtual_total=8.653G
   hl:mem_used=2.032G
   hl:swap_used=12.000K
   hl:virtual_used=2.032G
   hl:cpu=0.500000
   hl:m_topology=SCCCCCCCCCCCCCC
   hl:m_topology_inuse=SCCCCCCCCCCCCCC
   hl:m_socket=1.000000
   hl:m_core=14.000000
   hl:m_thread=14.000000
   hl:np_load_avg=0.107143
   hl:np_load_short=0.119286
   hl:np_load_medium=0.107143
   hl:np_load_long=0.078571
   `

		It("should return error if output is invalid", func() {
			hosts, err := qhost.ParseHostFullMetrics(sample)
			Expect(err).To(BeNil())
			Expect(hosts).To(HaveLen(3))
		})

		It("should parse host full metrics", func() {
			hosts, err := qhost.ParseHostFullMetrics(qhostFOutput1)
			Expect(err).To(BeNil())
			Expect(hosts).To(HaveLen(14))
			Expect(hosts[0].Name).To(Equal("global"))
			Expect(hosts[1].Name).To(Equal("master"))
			Expect(hosts[12].Name).To(Equal("sim8"))
		})

		It("should parse host full metrics with global host values", func() {
			hosts, err := qhost.ParseHostFullMetrics(qhostFOutput2)
			Expect(err).To(BeNil())
			Expect(hosts).To(HaveLen(2))
			Expect(hosts[0].Name).To(Equal("global"))
			Expect(hosts[1].Name).To(Equal("master"))
			Expect(len(hosts[0].Resources)).To(Equal(1))
			Expect(hosts[0].Resources["testc"]).To(Equal(
				qhost.ResourceAvailability{
					Name:                          "testc",
					StringValue:                   "100000.000000",
					FloatValue:                    100000.000000,
					ResourceAvailabilityLimitedBy: "g",
					Source:                        "c",
					FullString:                    "gc:testc=100000.000000",
				},
			))
		})

	})

})

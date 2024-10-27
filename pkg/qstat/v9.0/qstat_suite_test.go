package qstat_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestQstat(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Qstat Suite")
}

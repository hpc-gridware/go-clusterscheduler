package qhost_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestQhost(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Qhost Suite")
}

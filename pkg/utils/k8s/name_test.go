package k8s

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("ToDNSName", func() {
	table.DescribeTable("is DNS name compliant",
		func(prefix string, name string, suffix string, expected string) {
			result := ToDNSName(prefix, name, suffix)
			Expect(result).To(Equal(expected))
			Expect(len(result) <= 63).Should(BeTrue())
		},
		Entry("<= 63", "kidle", "shortname", "idle", "kidle-shortname-idle"),
		Entry("<= 63", "kidle", "shortname", "", "kidle-shortname"),
		Entry("== 63", "kidle", "name-length-is-63-yessssssssssssssssssssssssssssss", "wakeup", "kidle-name-length-is-63-yessssssssssssssssssssssssssssss-wakeup"),
		Entry("> 63", "kidle", "very-toooooooooooooooooooooooooooooooooooooooooooooooooo-long", "idle", "kidle-very-tooooooooooooooooooooooooooooooooooooooo-dmvyes-idle"),
	)
})

package rslog_test

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rightscale/rslog"
	"gopkg.in/inconshreveable/log15.v2"

	"testing"
)

func TestLogger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Logger Suite")
}

var _ = Describe("Logger", func() {
	var logger log15.Logger

	Describe("NewSyslog", func() {
		var pkg string
		var tag string

		JustBeforeEach(func() {
			logger = rslog.NewSyslog(pkg, tag)
		})

		Describe("with a valid setup", func() {
			BeforeEach(func() {
				tag = "foo"
				pkg = "PACKAGE"
			})

			It("creates a valid logger", func() {
				Ω(logger).ShouldNot(BeNil())
			})

			// XXX TBD: mock syslog??
		})

	})

	Describe("NewFile", func() {
		var pkg string
		var file string

		JustBeforeEach(func() {
			logger = rslog.NewFile(pkg, file)
		})

		Describe("with a valid filename", func() {
			var f *os.File
			var logContent string

			BeforeEach(func() {
				var err error
				f, err = ioutil.TempFile("", "")
				Ω(err).ShouldNot(HaveOccurred())
				file = f.Name()
				pkg = "PACKAGE"
			})

			AfterEach(func() {
				os.Remove(f.Name())
			})

			JustBeforeEach(func() {
				Ω(logger).ShouldNot(BeNil())
				logger.Info("42", "context", "foo")
				logC, err := ioutil.ReadAll(f)
				Ω(err).ShouldNot(HaveOccurred())
				logContent = string(logC)
			})

			It("creates a valid logger", func() {
				expected := `INFO 42`
				Ω(string(logContent)).Should(ContainSubstring(expected))
			})

			It("logs the package name", func() {
				expected := `PACKAGE`
				Ω(string(logContent)).Should(ContainSubstring(expected))
			})

			It("logs the timestamp", func() {
				expected := `\[201[0-9]-[0-9]{2}-[0-9]{2}.*\]`
				Ω(string(logContent)).Should(MatchRegexp(expected))
			})

			It("logs the context", func() {
				expected := "context=foo"
				Ω(string(logContent)).Should(ContainSubstring(expected))
			})
		})

		Describe("with an invalid filename", func() {
			var exitStatus int

			BeforeEach(func() {
				file = ""
				rslog.OSExit = func(s int) { exitStatus = s }
			})

			It("exits the process with status code 1", func() {
				Ω(exitStatus).Should(Equal(1))
			})
		})
	})
})

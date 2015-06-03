package rslog

import (
	"fmt"
	"io/ioutil"
	"log/syslog"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
		var err error

		JustBeforeEach(func() {
			logger, err = NewSyslog(pkg, tag)
		})

		Describe("with a valid setup", func() {
			var usedTag string
			var usedPriority syslog.Priority

			BeforeEach(func() {
				tag = "foo"
				pkg = "PACKAGE"
				syslogNew = func(p syslog.Priority, t string) (*syslog.Writer, error) {
					usedTag = t
					usedPriority = p
					return &syslog.Writer{}, nil
				}
			})

			It("creates a valid logger", func() {
				Ω(err).ShouldNot(HaveOccurred())
				Ω(logger).ShouldNot(BeNil())
				Ω(usedTag).Should(Equal(tag))
				Ω(usedPriority).Should(Equal(syslog.LOG_NOTICE | syslog.LOG_LOCAL0))
			})
		})

		Describe("when syslog connection fails", func() {
			BeforeEach(func() {
				syslogNew = func(_ syslog.Priority, _ string) (*syslog.Writer, error) {
					return nil, fmt.Errorf("kaboom")
				}
			})

			It("reports an error", func() {
				Ω(err).Should(HaveOccurred())
			})

		})

	})

	Describe("NewFile", func() {
		var pkg string
		var file string
		var err error

		JustBeforeEach(func() {
			logger, err = NewFile(pkg, file)
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
				logger.Info("42", "true", true, "false", false, "float32", 3.14,
					"float64", float64(3.15), "int", 1, "string", "foo",
					"other", struct{ val string }{val: "bar"})
				logC, err := ioutil.ReadAll(f)
				Ω(err).ShouldNot(HaveOccurred())
				logContent = string(logC)
			})

			It("creates a valid logger", func() {
				Ω(err).ShouldNot(HaveOccurred())
				expected := `INFO 42`
				Ω(string(logContent)).Should(ContainSubstring(expected))
			})

			It("logs the package name", func() {
				Ω(err).ShouldNot(HaveOccurred())
				expected := `PACKAGE`
				Ω(string(logContent)).Should(ContainSubstring(expected))
			})

			It("logs the timestamp", func() {
				Ω(err).ShouldNot(HaveOccurred())
				expected := `\[201[0-9]-[0-9]{2}-[0-9]{2}.*\]`
				Ω(string(logContent)).Should(MatchRegexp(expected))
			})

			It("logs the context", func() {
				Ω(err).ShouldNot(HaveOccurred())
				expected := "true=true"
				Ω(string(logContent)).Should(ContainSubstring(expected))
				expected = "false=false"
				Ω(string(logContent)).Should(ContainSubstring(expected))
				expected = "float32=3.14"
				Ω(string(logContent)).Should(ContainSubstring(expected))
				expected = "float64=3.15"
				Ω(string(logContent)).Should(ContainSubstring(expected))
				expected = "int=1"
				Ω(string(logContent)).Should(ContainSubstring(expected))
				expected = "string=foo"
				Ω(string(logContent)).Should(ContainSubstring(expected))
				expected = "other={val:bar}"
				Ω(string(logContent)).Should(ContainSubstring(expected))
			})

			Describe("with nil context data", func() {
				JustBeforeEach(func() {
					logger.Info("oops", "nil", error(nil))
					logC, err := ioutil.ReadAll(f)
					Ω(err).ShouldNot(HaveOccurred())
					logContent = string(logC)
				})

				It("logs nil", func() {
					Ω(err).ShouldNot(HaveOccurred())
					expected := "nil=nil"
					Ω(string(logContent)).Should(ContainSubstring(expected))
				})
			})

		})

		Describe("with an invalid filename", func() {

			BeforeEach(func() {
				file = ""
			})

			It("reports an error", func() {
				Ω(err).Should(HaveOccurred())
			})
		})
	})
})

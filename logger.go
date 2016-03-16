package rslog

import (
	"bytes"
	"fmt"
	"io"
	"log/syslog"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gopkg.in/inconshreveable/log15.v2"
)

// NewFileHandler creates a file based logger.
func NewFileHandler(file string) (log15.Handler, error) {
	h, err := log15.FileHandler(file, SimpleFormat(true))
	if err != nil {
		// Don't try to use log as that could panic
		return nil, fmt.Errorf("failed to create log file %s: %s", file, err)
	}
	return h, nil
}

// NewSyslogHandler creates a syslog based logger.
// tag is used to prefix all log entries.
// Use an empty tag to prefix log entries with the process name (os.Arg[0]).
func NewSyslogHandler(tag string) (log15.Handler, error) {
	sysWr, err := SyslogNew(syslog.LOG_NOTICE|syslog.LOG_LOCAL0, tag)
	if err != nil {
		// Don't try to use log as that could panic
		return nil, fmt.Errorf("failed to connect to syslog: %s", err)
	}
	return newSyslogHandler(sysWr), nil
}

// NewTCPSyslogHandler creates a new syslog based handler that talks to
// syslog on the provided address using TCP protocol.
func NewTCPSyslogHandler(addr string, tag string) (log15.Handler, error) {
	sysWr, err := SyslogNewTCP(addr, syslog.LOG_NOTICE|syslog.LOG_LOCAL0, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to syslog: %s", err)
	}
	return newSyslogHandler(sysWr), nil
}

// NewUDPSyslogHandler creates a new syslog based handler that talks to
// syslog on the provided address using UDP protocol.
func NewUDPSyslogHandler(addr string, tag string) (log15.Handler, error) {
	sysWr, err := SyslogNewUDP(addr, syslog.LOG_NOTICE|syslog.LOG_LOCAL0, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to syslog: %s", err)
	}
	return newSyslogHandler(sysWr), nil
}

func newSyslogHandler(sysWr *syslog.Writer) log15.Handler {
	return log15.FuncHandler(func(r *log15.Record) error {
		var syslogFn = sysWr.Info
		switch r.Lvl {
		case log15.LvlCrit:
			syslogFn = sysWr.Crit
		case log15.LvlError:
			syslogFn = sysWr.Err
		case log15.LvlWarn:
			syslogFn = sysWr.Warning
		case log15.LvlInfo:
			syslogFn = sysWr.Info
		case log15.LvlDebug:
			syslogFn = sysWr.Debug
		}
		fmtr := SimpleFormat(false)
		s := strings.TrimSpace(string(fmtr.Format(r)))
		return syslogFn(s)
	})
}

// SimpleFormat returns a log15 formatter that uses a logfmt like output.
// The timestamps switch can be used to toggle prefixing each entry with the current time.
// (see https://brandur.org/logfmt)
func SimpleFormat(timestamps bool) log15.Format {
	return formatter(timestamps, true)
}

// TerseFormat removes all additional metadata (timestampts, level) on the
// assumption that the underlying sink (syslog, etc.) already provides and/or
// does not require them.
func TerseFormat() log15.Format {
	return formatter(false, false)
}

// a formatter with optional parts.
func formatter(timestamps, level bool) log15.Format {
	return log15.FormatFunc(func(r *log15.Record) []byte {
		b := &bytes.Buffer{}

		// time
		if timestamps {
			b.WriteByte('[')
			b.WriteString(r.Time.Format(simpleTimeFormat))
			b.WriteByte(']')
		}

		// level
		if level {
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(strings.ToUpper(r.Lvl.String()))
		}

		// special case for the 'empty tag' in order to support intermixing legacy
		// logging with log15 style. the empty tag's value is thus free-form and
		// always preceeds the message as a string literal. the empty tag, if any,
		// must appear first in context.
		context := r.Ctx
		contextOffset := 0
		contextLength := len(context)
		if contextLength > 0 {
			k, ok := context[0].(string)
			if ok && len(k) == 0 {
				contextOffset = 2
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				v, ok := context[1].(string)
				if ok {
					b.WriteString(v)
				} else {
					b.WriteString("LOG_ERR=\"\"")
				}
			}
		}

		// message
		message := r.Msg
		if len(message) > 0 {
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(message)
		}

		// remaining context, if any.
		if contextLength > contextOffset {
			// try to justify the log output for short messages
			messageLength := len(message)
			if messageLength < simpleMsgJust {
				b.Write(bytes.Repeat([]byte{' '}, simpleMsgJust-messageLength))
			}

			// print the keys logfmt style
			for i := contextOffset; i < contextLength; i += 2 {
				var v string
				k, ok := context[i].(string)
				if ok {
					v = formatLogfmtValue(context[i+1])
				} else {
					k, v = "LOG_ERR", formatLogfmtValue(context[i])
				}
				if b.Len() > 0 {
					b.WriteByte(' ')
				}
				b.WriteString(k)
				b.WriteByte('=')
				b.WriteString(v)
			}
		}
		b.WriteByte('\n')
		return b.Bytes()
	})
}

const simpleTimeFormat = "2006-01-02 15:04:05"
const simpleMsgJust = 40

// copied from log15 https://github.com/inconshreveable/log15/blob/master/format.go#L203-L223
func formatLogfmtValue(value interface{}) string {
	if value == nil {
		return "nil"
	}

	value = formatShared(value)
	switch v := value.(type) {
	case bool:
		return strconv.FormatBool(v)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', 3, 64)
	case float64:
		return strconv.FormatFloat(v, 'f', 3, 64)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", value)
	case string:
		return escapeString(v)
	default:
		return escapeString(fmt.Sprintf("%+v", value))
	}
}

// copied from log15 https://github.com/inconshreveable/log15/blob/master/format.go
func formatShared(value interface{}) (result interface{}) {
	defer func() {
		if err := recover(); err != nil {
			if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr && v.IsNil() {
				result = "nil"
			} else {
				panic(err)
			}
		}
	}()

	switch v := value.(type) {
	case time.Time:
		return v.Format(simpleTimeFormat)

	case error:
		return v.Error()

	case fmt.Stringer:
		return v.String()

	default:
		return v
	}
}

// copied from log15 https://github.com/inconshreveable/log15/blob/master/format.go
func escapeString(s string) string {
	needQuotes := false
	e := bytes.Buffer{}
	e.WriteByte('"')
	for _, r := range s {
		if r <= ' ' || r == '=' || r == '"' {
			needQuotes = true
		}

		switch r {
		case '\\', '"':
			e.WriteByte('\\')
			e.WriteByte(byte(r))
		case '\n':
			e.WriteByte('\\')
			e.WriteByte('n')
		case '\r':
			e.WriteByte('\\')
			e.WriteByte('r')
		case '\t':
			e.WriteByte('\\')
			e.WriteByte('t')
		default:
			e.WriteRune(r)
		}
	}
	e.WriteByte('"')
	start, stop := 0, e.Len()
	if !needQuotes {
		start, stop = 1, stop-1
	}
	return string(e.Bytes()[start:stop])
}

// copied from log15
type closingHandler struct {
	io.WriteCloser
	log15.Handler
}

// copied from log15
func (h *closingHandler) Close() error {
	return h.WriteCloser.Close()
}

// Override for testing
var (
	SyslogNew    = syslog.New
	SyslogNewTCP = func(addr string, p syslog.Priority, t string) (*syslog.Writer, error) {
		return syslog.Dial("tcp", addr, p, t)
	}
	SyslogNewUDP = func(addr string, p syslog.Priority, t string) (*syslog.Writer, error) {
		return syslog.Dial("udp", addr, p, t)
	}
)

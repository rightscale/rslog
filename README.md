# rslog
[![GoDoc](https://godoc.org/github.com/rightscale/rslog?status.svg)](https://godoc.org/github.com/rightscale/rslog)

rslog is a thin wrapper on top of [log15](https://github.com/inconshreveable/log15) that provides
a standard format for log entries based on [logfmt](https://brandur.org/logfmt).

rslog makes it possible to create file based or syslog loggers. Usage:
```go
l := rslog.NewFile("mypackage", "/var/log/mypackage.log")
l.info("message", "key1", "value1", "key2", "value2")
```
The logger methods accept an arbitrary number of key/value pairs (the log "context"). These pairs
get logged using the logfmt format, that is: `key1=value1 key2=value2`.

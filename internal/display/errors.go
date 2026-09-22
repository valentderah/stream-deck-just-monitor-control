package display

import "errors"

var ErrUnsupported = errors.New("display: unsupported on this platform")
var ErrMonitorNotFound = errors.New("display: monitor not found")
var ErrNoMonitorsSelected = errors.New("display: no monitors selected")

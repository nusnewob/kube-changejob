// internal/config/defaults.go
package config

import "time"

var DefaultControllerConfig = ControllerConfig{
	PollInterval: 60 * time.Second,
}

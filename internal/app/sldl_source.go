package app

import (
	"fmt"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// NewSldlSource builds the sldl-backed Soulseek source behind the status,
// connection test and install screens. It is never registered with the
// source manager: downloads go through Core's native Soulseek client.
func NewSldlSource(config *core.Config, logBuffer *core.LogBuffer) *core.SoulseekSource {
	sldlPath := config.SoulseekBinaryPath
	if sldlPath == "" {
		sldlPath = DefaultSldlPath()
	}
	if err := EnsureSldlExecutable(sldlPath); err != nil && logBuffer != nil {
		logBuffer.Warn(fmt.Sprintf("sldl binary might not be runnable: %v", err))
	}
	src := core.NewSoulseekSource(sldlPath, config.SoulseekUsername, config.SoulseekPassword)
	src.SetLogger(logBuffer)
	return src
}

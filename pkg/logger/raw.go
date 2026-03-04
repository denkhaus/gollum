package logger

import (
	"bytes"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// ============================================================================
// Custom Console Encoder with \r\n line endings for raw terminal mode
// ============================================================================

// rawModeConsoleEncoder is a custom console encoder that uses \r\n instead of \n
// for proper line endings in raw terminal mode.
type rawModeConsoleEncoder struct {
	zapcore.Encoder
}

// newRawModeConsoleEncoder creates a new console encoder with \r\n line endings.
func newRawModeConsoleEncoder(encoderConfig zapcore.EncoderConfig) zapcore.Encoder {
	return &rawModeConsoleEncoder{
		Encoder: zapcore.NewConsoleEncoder(encoderConfig),
	}
}

// Clone creates a copy of the encoder.
func (e *rawModeConsoleEncoder) Clone() zapcore.Encoder {
	return &rawModeConsoleEncoder{
		Encoder: e.Encoder.Clone(),
	}
}

// EncodeEntry encodes a log entry and replaces \n with \r\n for raw terminal mode.
func (e *rawModeConsoleEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf, err := e.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return nil, err
	}

	// Replace all \n with \r\n for proper raw terminal mode handling
	// We need to be careful not to double-replace existing \r\n
	str := buf.String()
	var result bytes.Buffer
	result.Grow(len(str) + len(str)/10) // Pre-allocate with some extra space

	for i := 0; i < len(str); i++ {
		if str[i] == '\n' {
			// Check if this is already \r\n
			if i > 0 && str[i-1] == '\r' {
				// Already \r\n, just write the \n
				result.WriteByte('\n')
			} else {
				// Standalone \n, convert to \r\n
				result.WriteString("\r\n")
			}
		} else {
			result.WriteByte(str[i])
		}
	}

	// Create a new buffer from the pool and write our processed string to it
	newBuf := buffer.NewPool().Get()
	newBuf.WriteString(result.String())
	return newBuf, nil
}

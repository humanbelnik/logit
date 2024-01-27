package logit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"golang.org/x/exp/slog"
)

const (
	timeFormat = "[15:04:05.000]"

	reset = "\033[0m"

	black        = 30
	red          = 31
	green        = 32
	yellow       = 33
	blue         = 34
	magenta      = 35
	cyan         = 36
	lightGray    = 37
	darkGray     = 90
	lightRed     = 91
	lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	lightCyan    = 96
	white        = 97
)

// child: nested handler.
// buffer: pipe child's output here.
// mut: make buffer thread-safe across multiple Goroutines.
type Handler struct {
	child  slog.Handler
	buffer *bytes.Buffer
	mut    *sync.Mutex
}

// withColor applies given color to a string.
func withColor(code int, s string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(code), s, reset)
}

// Enabled returns true if child handler is enabled for specified level of logging.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.child.Enabled(ctx, level)
}

// WithAttrs returns Handler with specified attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		child:  h.child.WithAttrs(attrs),
		buffer: h.buffer,
		mut:    h.mut,
	}
}

// Handle process record if it's enabled
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	level := r.Level.String() + ":"
	switch r.Level {
	case slog.LevelInfo:
		level = withColor(lightGreen, level)
	case slog.LevelDebug:
		level = withColor(blue, level)
	case slog.LevelWarn:
		level = withColor(yellow, level)
	case slog.LevelError:
		level = withColor(red, level)
	}

	childAttrs, err := h.extractChildAttrs(ctx, r)
	if err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(childAttrs, "", " ")
	if err != nil {
		return err
	}

	fmt.Println(withColor(lightGray, r.Time.Format(timeFormat)), level, withColor(white, r.Message), withColor(darkGray, string(bytes)))

	return nil
}

// WithGroup returns Handler with named Group.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		child:  h.child.WithGroup(name),
		buffer: h.buffer,
		mut:    h.mut,
	}
}

// extractChildAttrs takes child handler's attributes, writes it to a main Handlers and atteches to the main Handler's attributes.
func (h *Handler) extractChildAttrs(ctx context.Context, r slog.Record) (map[string]any, error) {
	h.mut.Lock()
	defer func() {
		h.buffer.Reset()
		h.mut.Unlock()
	}()

	if err := h.child.Handle(ctx, r); err != nil {
		return nil, fmt.Errorf("cannot handle child's attributes: %w", err)
	}

	var childAttrs map[string]any
	if err := json.Unmarshal(h.buffer.Bytes(), &childAttrs); err != nil {
		return nil, fmt.Errorf("cannot unmarshall from buffer to map: %w", err)
	}

	return childAttrs, nil
}

// emitDefault supresses child's TimeStamp and so on.
func emitDefaults(next func([]string, slog.Attr) slog.Attr) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey || a.Key == slog.LevelKey || a.Key == slog.MessageKey {
			return slog.Attr{}
		}

		if next == nil {
			return a
		}
		return next(groups, a)
	}
}

func NewHandler(level slog.Level) *Handler {
	buffer := &bytes.Buffer{}

	return &Handler{
		buffer: buffer,
		child: slog.NewJSONHandler(buffer, &slog.HandlerOptions{
			Level:       level,
			ReplaceAttr: emitDefaults(slog.HandlerOptions{}.ReplaceAttr),
		}),
		mut: &sync.Mutex{},
	}
}

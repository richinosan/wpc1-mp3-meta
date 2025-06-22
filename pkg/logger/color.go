package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

// ANSI カラーコード
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorGray   = "\033[37m"
	ColorCyan   = "\033[36m"
)

// ColorHandler は、カラー付きでログを出力するslog.Handlerの実装
type ColorHandler struct {
	writer io.Writer
	level  slog.Level
}

// NewColorHandler は新しいColorHandlerを作成
func NewColorHandler(w io.Writer, level slog.Level) *ColorHandler {
	return &ColorHandler{
		writer: w,
		level:  level,
	}
}

// Enabled は指定されたレベルのログが有効かどうかを返す
func (h *ColorHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle はログレコードを処理する
func (h *ColorHandler) Handle(_ context.Context, r slog.Record) error {
	// レベルに応じた色を選択
	var levelColor string
	var levelText string

	switch r.Level {
	case slog.LevelDebug:
		levelColor = ColorGray
		levelText = "DEBUG"
	case slog.LevelInfo:
		levelColor = ColorCyan
		levelText = "INFO "
	case slog.LevelWarn:
		levelColor = ColorYellow
		levelText = "WARN "
	case slog.LevelError:
		levelColor = ColorRed
		levelText = "ERROR"
	default:
		levelColor = ColorReset
		levelText = "UNKNOWN"
	}

	// 時刻のフォーマット
	timestamp := r.Time.Format("15:04:05")

	// メッセージの構築
	message := fmt.Sprintf("%s%s%s %s[%s]%s %s",
		ColorBlue, timestamp, ColorReset,
		levelColor, levelText, ColorReset,
		r.Message,
	)

	// 属性の追加
	if r.NumAttrs() > 0 {
		attrs := make([]string, 0, r.NumAttrs())
		r.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value))
			return true
		})
		if len(attrs) > 0 {
			message += fmt.Sprintf(" %s{%s}%s", ColorGray,
				fmt.Sprintf("%v", attrs), ColorReset)
		}
	}

	message += "\n"

	_, err := h.writer.Write([]byte(message))
	return err
}

// WithAttrs は属性を追加した新しいHandlerを返す
func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 簡単な実装として、同じハンドラーを返す
	return h
}

// WithGroup はグループを追加した新しいHandlerを返す
func (h *ColorHandler) WithGroup(name string) slog.Handler {
	// 簡単な実装として、同じハンドラーを返す
	return h
}

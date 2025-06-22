package logger

import (
	"log/slog"
	"os"
)

// グローバルなカラーハンドラー
var colorHandler *ColorHandler

// init 関数でデフォルトのカラーハンドラーを初期化
func init() {
	// jqを読むために、stderrに出力する
	colorHandler = NewColorHandler(os.Stderr, slog.LevelDebug)
	logger := slog.New(colorHandler)
	slog.SetDefault(logger)
}

// GetHandler はグローバルなカラーハンドラーを返す
func GetHandler() *ColorHandler {
	return colorHandler
}

// SetLevel はログレベルを設定する
func SetLevel(level slog.Level) {
	// jqを読むために、stderrに出力する
	colorHandler = NewColorHandler(os.Stderr, level)
	logger := slog.New(colorHandler)
	slog.SetDefault(logger)
}

// SetupColorLogger はカラー付きロガーを設定する（後方互換性のため）
func SetupColorLogger(level slog.Level) {
	SetLevel(level)
}

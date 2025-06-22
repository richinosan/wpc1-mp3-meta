package cmd

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/goaux/headline"
	"github.com/richinosan/wpc1-mp3-meta/pkg/cmd/opts"
	_ "github.com/richinosan/wpc1-mp3-meta/pkg/logger" // グローバルなカラーハンドラーを初期化
	"github.com/spf13/cobra"
)

//go:embed help.txt
var help string

var Command = &cobra.Command{
	Short: headline.Get(help),
	Long:  help,
	RunE:  Run,
}

func Run(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	slog.DebugContext(cmd.Context(), "attr",
		slog.String("input", opts.InputDir),
		slog.String("output", opts.OutputDir),
		slog.Int("kbps", opts.Kbps),
		slog.String("artist", opts.AlbumArtist),
		slog.String("album", opts.Album),
		slog.String("release", opts.Release),
		slog.String("genre", opts.Genre),
		slog.String("jacket", opts.JacketImagePath),
		slog.String("metadata", opts.MetadataCsvPath),
		slog.String("title", opts.TitleColumn),
		slog.String("tracknumber", opts.TrackNumberColumn),
		slog.String("artistcolumn", opts.ArtistColumn),
		slog.Bool("dryrun", opts.DryRun),
	)

	files, err := os.ReadDir(opts.InputDir)
	if err != nil {
		return err
	}

	musics := make(map[string]Music)
	for _, file := range files {
		if !file.IsDir() {
			filename := file.Name()
			// 拡張子を除いたファイル名を取得
			name := strings.TrimSuffix(filename, filepath.Ext(filename))
			// nameを数値に変換（安全な方法）
			nameInt := 0
			if len(name) > 0 {
				// 先頭の数字部分のみを抽出
				var numStr strings.Builder
				for _, r := range name {
					if r >= '0' && r <= '9' {
						numStr.WriteRune(r)
					} else {
						break
					}
				}
				if numStr.Len() > 0 {
					if parsed, err := strconv.Atoi(numStr.String()); err == nil {
						nameInt = parsed
					}
				}
			}
			musics[name] = Music{
				TrackNumber:     nameInt,
				AlbumArtist:     opts.AlbumArtist,
				Artist:          "", // CSVから読み込まれる
				Album:           opts.Album,
				Release:         opts.Release,
				Genre:           opts.Genre,
				JacketImagePath: opts.JacketImagePath,
			}
		}
	}

	err = GenerateTitle(&musics)
	if err != nil {
		return err
	}

	if opts.DryRun {
		executionTime := time.Since(startTime)
		slog.DebugContext(cmd.Context(), "benchmark",
			slog.String("execution_time", executionTime.String()),
			slog.Float64("seconds", executionTime.Seconds()),
		)
		slog.Info("dry run")
		json, err := json.Marshal(musics)
		if err != nil {
			return err
		}
		fmt.Println(string(json))
		return nil
	}

	// 出力ディレクトリを作成
	err = os.MkdirAll(opts.OutputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 各音楽ファイルを処理
	for filename, music := range musics {
		inputPath := filepath.Join(opts.InputDir, filename+".wav") // 元のファイル拡張子を想定
		outputPath := filepath.Join(opts.OutputDir, fmt.Sprintf("%s「%s」%s.mp3", filename, music.Title, music.Artist))

		// 入力ファイルが存在するかチェック
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			// .wavがない場合、他の拡張子も試す
			possibleExts := []string{".flac", ".m4a", ".aac", ".ogg"}
			found := false
			for _, ext := range possibleExts {
				testPath := filepath.Join(opts.InputDir, filename+ext)
				if _, err := os.Stat(testPath); err == nil {
					inputPath = testPath
					found = true
					break
				}
			}
			if !found {
				slog.WarnContext(cmd.Context(), "input file not found", slog.String("filename", filename))
				continue
			}
		}

		err = convertToMp3WithMetadata(cmd.Context(), inputPath, outputPath, music)
		if err != nil {
			slog.ErrorContext(cmd.Context(), "failed to convert file",
				slog.String("input", inputPath),
				slog.String("output", outputPath),
				slog.String("error", err.Error()),
			)
			continue
		}

		slog.InfoContext(cmd.Context(), "converted successfully",
			slog.String("input", inputPath),
			slog.String("output", outputPath),
		)
	}

	return nil
}

// convertToMp3WithMetadata はffmpegを使って音楽ファイルをMP3に変換し、メタデータを挿入する
func convertToMp3WithMetadata(ctx context.Context, inputPath, outputPath string, music Music) error {
	args := []string{"-i", inputPath}

	// ジャケット画像がある場合は入力ファイルとして追加
	hasJacket := false
	if music.JacketImagePath != "" {
		if _, err := os.Stat(music.JacketImagePath); err == nil {
			args = append(args, "-i", music.JacketImagePath)
			hasJacket = true
			slog.DebugContext(ctx, "jacket image found", slog.String("path", music.JacketImagePath))
		} else {
			slog.WarnContext(ctx, "jacket image not found",
				slog.String("path", music.JacketImagePath),
				slog.String("error", err.Error()))
		}
	}

	// 出力オプションを追加
	args = append(args,
		"-codec:a", "libmp3lame",
		"-b:a", fmt.Sprintf("%dk", opts.Kbps),
		"-metadata", fmt.Sprintf("title=%s", music.Title),
		"-metadata", fmt.Sprintf("artist=%s", music.Artist),
		"-metadata", fmt.Sprintf("album_artist=%s", music.AlbumArtist),
		"-metadata", fmt.Sprintf("album=%s", music.Album),
		"-metadata", fmt.Sprintf("date=%s", music.Release),
		"-metadata", fmt.Sprintf("genre=%s", music.Genre),
		"-metadata", fmt.Sprintf("track=%d", music.TrackNumber),
	)

	// ジャケット画像がある場合はマッピングとエンコーディングオプションを追加
	if hasJacket {
		args = append(args,
			"-map", "0:a", // 音声ストリーム
			"-map", "1:v", // 画像ストリーム
			"-c:v", "copy", // 画像コーデック（jpegの場合はcopyが効率的）
			"-disposition:v", "attached_pic", // 添付画像として設定
		)
		slog.DebugContext(ctx, "adding jacket image to ffmpeg command")
	}

	args = append(args, "-y", outputPath) // -y で上書き許可

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	slog.DebugContext(ctx, "executing ffmpeg",
		slog.String("command", cmd.String()),
		slog.Bool("has_jacket", hasJacket),
		slog.String("jacket_path", music.JacketImagePath))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w, output: %s", err, string(output))
	}

	return nil
}

func init() {
	Command.Flags().StringVarP(&opts.InputDir, "input", "i", opts.InputDir, "input directory")
	Command.Flags().StringVarP(&opts.OutputDir, "output", "o", opts.OutputDir, "output directory")
	Command.Flags().IntVarP(&opts.Kbps, "kbps", "k", 320, "kbps")
	Command.Flags().StringVar(&opts.AlbumArtist, "artist", opts.AlbumArtist, "artist")
	Command.Flags().StringVar(&opts.Album, "album", opts.Album, "album")
	Command.Flags().StringVar(&opts.Release, "release", opts.Release, "release")
	Command.Flags().StringVar(&opts.Genre, "genre", opts.Genre, "genre")
	Command.Flags().StringVar(&opts.JacketImagePath, "jacket", opts.JacketImagePath, "jacket image path")
	Command.Flags().StringVar(&opts.MetadataCsvPath, "metadata", opts.MetadataCsvPath, "metadata csv path")
	Command.Flags().StringVar(&opts.TitleColumn, "title", opts.TitleColumn, "title column")
	Command.Flags().StringVar(&opts.TrackNumberColumn, "tracknumber", opts.TrackNumberColumn, "track number column")
	Command.Flags().StringVar(&opts.ArtistColumn, "artistcolumn", opts.ArtistColumn, "artist column")
	Command.Flags().BoolVarP(&opts.DryRun, "dryrun", "d", opts.DryRun, "dry run")
}

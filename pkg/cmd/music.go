package cmd

import (
	"encoding/csv"
	"os"

	"github.com/richinosan/wpc1-mp3-meta/pkg/cmd/opts"
)

// Music は、 mp3 ファイルのメタデータを表す。
// すべてstring
type Music struct {
	Title           string
	TrackNumber     int
	AlbumArtist     string
	Artist          string
	Album           string
	Release         string
	Genre           string
	JacketImagePath string
}

// GenerateTitle は、 csv ファイルを読み込んで、曲名をMusics[TrackNumber].Titleに設定する。
// 曲名は、 opts.TitleColumn の列にある。
// マップのキーは、 opts.TrackNumberColumn の列にある。
func GenerateTitle(musics *map[string]Music) error {
	csvFile, err := os.Open(opts.MetadataCsvPath)
	if err != nil {
		return err
	}
	defer csvFile.Close()

	csvReader := csv.NewReader(csvFile)
	records, err := csvReader.ReadAll()
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return nil
	}

	// ヘッダー行から列のインデックスを取得
	header := records[0]
	titleColumnIndex := -1
	trackNumberColumnIndex := -1
	artistColumnIndex := -1
	for i, columnName := range header {
		if columnName == opts.TitleColumn {
			titleColumnIndex = i
		}
		if columnName == opts.TrackNumberColumn {
			trackNumberColumnIndex = i
		}
		if columnName == opts.ArtistColumn {
			artistColumnIndex = i
		}
	}

	// 必要な列が見つからない場合はエラーを返す
	if titleColumnIndex == -1 || trackNumberColumnIndex == -1 || artistColumnIndex == -1 {
		return nil // または適切なエラーを返す
	}

	// データ行を処理（ヘッダー行をスキップ）
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) > titleColumnIndex && len(record) > trackNumberColumnIndex {
			trackNumber := record[trackNumberColumnIndex]
			if music, exists := (*musics)[trackNumber]; exists {
				music.Title = record[titleColumnIndex]
				music.Artist = record[artistColumnIndex]
				(*musics)[trackNumber] = music
			}
		}
	}

	return nil
}

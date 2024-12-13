package main

// ZIPファイルを解凍する処理
// 元ネタ
// https://qiita.com/kakiyuta/items/71016e5f96dd0cde81f5

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// Zipファイルを解凍
// OS特有のファイル,ディレクトリは解凍対象外(__MACOSX, .DS_Store ..... etc)
func Unzip(src string, dest string) ([]string, error) {
	var fileNames []string
	r, err := zip.OpenReader(src)
	if err != nil {
		return fileNames, err
	}
	defer r.Close()

	for _, f := range r.File {
		// 不要なファイルは除去
		if IsExcludedFileOrDir(f.Name) {
			continue
		}

		if !utf8.ValidString(f.Name) {
			// utf8に変換
			fname, err := ConvertToUtf8FromShiftJis(f.Name)
			if err != nil {
				return fileNames, err
			}
			f.Name = fname
		}
		fpath := filepath.Join(dest, f.Name)

		// 脆弱性Zip Slipの対策 https://snyk.io/research/zip-slip-vulnerability#go
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fileNames, fmt.Errorf("%s: illegal file path", fpath)
		}

		if f.FileInfo().IsDir() {
			// ディレクトリ作成
			os.MkdirAll(fpath, os.ModePerm)
			continue
		} else {
			fileNames = append(fileNames, f.Name)
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fileNames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fileNames, err
		}

		rc, err := f.Open()
		if err != nil {
			return fileNames, err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return fileNames, err
		}
	}

	return fileNames, nil
}

// 解凍の対象外のファイル,ディレクトリかチェック
func IsExcludedFileOrDir(checkTarget string) bool {

	// macOS特有のディレクトリは除去
	if strings.HasPrefix(checkTarget, "__MACOSX") {
		return true
	}

	// macOS特有のファイルは除去
	if strings.Contains(checkTarget, ".DS_Store") {
		return true
	}

	return false
}

// ShiftJisの文字列をUtf8に変換する
// Utf8の文字列を渡した場合は変換せずそのまま返す
func ConvertToUtf8FromShiftJis(sjis string) (string, error) {
	if utf8.ValidString(sjis) {
		// 元々utf8のため変換しない
		return sjis, nil
	}
	utf8str, _, err := transform.String(japanese.ShiftJIS.NewDecoder(), sjis)
	return utf8str, err
}

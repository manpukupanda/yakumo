package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// 対象の書類かどうかを判定（内国法人の有報(3号様式）のみ対象）
func isValidForProcessing(result *Result) bool {
	return result.DocTypeCode == "120" && result.FormCode == "030000" && result.OrdinanceCode == "010"
}

// メイン処理
func main() {
	// テーブルとインデックスを作成する（なければなにもしない）
	err := createTableAndIndex()
	if err != nil {
		log.Fatal(err)
	}

	// 直近１年分処理（当日から遡って１日ずつ）
	currentTime := time.Now()
	for i := 0; i < 365; i++ {
		exexOneDay(currentTime.AddDate(0, 0, -i).Format("2006-01-02"))
	}
}

// 1日分の処理。APIから1日分のリストを取得して、
// 取得したデータ分を処理する
func exexOneDay(date string) {

	docs, err := GetDocuments(date)
	if err != nil {
		log.Fatal(err)
		return
	}

	for _, v := range docs.Results {
		if !isValidForProcessing(&v) {
			continue
		}

		exist, err := exists(date, v)
		if err != nil {
			log.Fatal(err)
		}

		if exist {
			log.Printf("%s %s is already exist.\n", date, v.DocID)
			continue
		}

		log.Printf("%s %s %s %s %s\n", date, v.DocID, v.EdinetCode, v.FilerName, v.DocDescription)

		err = resultToText(v)
		if err != nil {
			log.Print("テキスト変換エラー")
			log.Fatal(err)
		}

		err = save(date, v)
		if err != nil {
			log.Print("DB保存に失敗")
			log.Fatal(err)
		}
	}
}

// Result データから、そのデータのzipを取得して検索用のテキストを作成する
func resultToText(result Result) error {
	// tempファイルを作成するだけして閉じる
	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, "edinet_*.zip")
	if err != nil {
		return err
	}
	tempFile.Close()
	tempFileName := tempFile.Name()
	// 処理後にtempファイルを削除する
	defer os.Remove(tempFileName)
	// 作成したtempファイルを上書きするようにzipをダウンロードする
	err = DownloadZip(result.DocID, tempFileName)
	if err != nil {
		return err
	}

	// zipファイルからテキスト作成
	err = zipToText(tempFileName)
	if err != nil {
		return err
	}
	return nil
}

// zipファイルから検索用のテキストを作成する
func zipToText(zipfile string) error {

	// zipを安全に解凍するワークディレクトリを作成
	tempDir := os.TempDir()
	workDir, err := os.MkdirTemp(tempDir, "zipext_*")
	if err != nil {
		return err
	}

	// 作業後にワークディレクトリを削除
	defer os.RemoveAll(workDir)

	// zipを解凍
	_, err = Unzip(zipfile, workDir)
	if err != nil {
		return err
	}

	// テキストを作成
	err = htmlsToText(workDir)
	if err != nil {
		return err
	}

	return nil
}

// 目次ごとのタイトルとパンくずと本文
type Heading struct {
	title      string
	breadcrumb string
	text       string
}

// 目次スライス
var headings []Heading

// htmlファイル群から検索用のテキストを作成する
func htmlsToText(dirpath string) error {
	// htmlファイルのリストを取得
	htmls, err := listHtmlFiles(dirpath)
	if err != nil {
		return err
	}

	// 並び変え
	sortHtmlList(htmls)

	// 空白文字、全角スペース、ノーブレークスペースが１つ以上連続するパターン
	rep := regexp.MustCompile(`[\s　\xA0\n]+`)

	// 目次スライスの初期化
	headings = make([]Heading, 0)

	// 各ファイルを順次処理して目次スライスに設定していく
	var inAudit bool
	for _, v := range *htmls {
		fp, err := os.Open(v)
		if err != nil {
			return err
		}
		defer fp.Close()

		// 監査報告書の最初のhtmlであるか
		firstHtmlOfAuditDoc := false
		if !inAudit && strings.Contains(v, "AuditDoc") {
			firstHtmlOfAuditDoc = true
			inAudit = true
		}

		err = htmlToText(fp, firstHtmlOfAuditDoc)
		if err != nil {
			return err
		}
	}

	// 目次ごとのテキストから余分なスペースを除外する
	for i := range headings {
		// 空白文字、全角スペース、ノーブレークスペースが１つ以上連続する箇所半角スペース１つに置き換える。
		t := rep.ReplaceAllString(headings[i].text, " ")
		// 前後の半角スペースは削除
		headings[i].text = strings.Trim(t, " ")
	}

	// パンくず設定
	setBreadcrumb()
	return nil
}

// 処理したい順番（本文->監査報告書）にソートする
func sortHtmlList(list *[]string) {
	sort.Slice(*list, func(i, j int) bool {
		cmpSrc := func(src string) string {
			tmp := strings.Replace(src, "AuditDoc", "ZZZ", 1)
			return strings.Replace(tmp, "PublicDoc", "AAA", 1)
		}
		t1 := cmpSrc((*list)[i])
		t2 := cmpSrc((*list)[j])
		return strings.Compare(t1, t2) < 0
	})
}

// 特定のディレクトリ内のhtmlファイルをリストする
func listHtmlFiles(dirPath string) (*[]string, error) {
	var paths []string
	// WalkDirを使ってディレクトリを再帰的に探索
	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// ディレクトリを除外
		if !d.IsDir() {
			// 拡張子でhtmlファイルを識別
			if strings.HasSuffix(path, ".htm") ||
				strings.HasSuffix(path, ".html") {
				paths = append(paths, path)
			}
		}
		return nil
	})
	return &paths, err
}

// htmlから検索用のテキストを作成する
func htmlToText(r io.Reader, firstHtmlOfAuditDoc bool) error {

	// UTF8のBOM付き対応
	// https://qiita.com/ssc-ynakamura/items/e05dc9bfacee063f3471
	fallback := unicode.UTF8.NewDecoder()
	r2 := transform.NewReader(r, unicode.BOMOverride(fallback))

	documentNode, err := html.Parse(r2)
	if err != nil {
		return err
	}

	// h1～h6、td、th、br タグのテキストは前後に半角スペースを付けるので、そのタグを識別するために使用する。
	// 例えば、 <div>あああ<h1>見出し１</h1>いいい</div> のテキストは「あああ 見出し１ いいい」となる。
	// h1がspanだった場合 <div>あああ<span>見出し１</span>いいい</div> のテキストは「あああ見出し１いいい」となる。（spanタグの前後にスペースがつかない）
	patternSpacedTags := "(h[1-6]|td|th|br)"
	reSpacedTags := regexp.MustCompile(patternSpacedTags)

	// テキスト中のスペースを詰めるためのパターン
	// アルファベット数字いくつかの記号に挟まれたスペースは残すが、それ以外の文字（ひらがなカタカナ漢字等）
	// に挟まれたスペースは除外するために使用する。
	// 有価証券報告書では氏名等がスペースで幅調整されているので、そのスペースを消すために使用する。
	patternSpaceMidKanjiEtc := `([^0-9０-９a-zA-Zａ-ｚＡ-Ｚ\,\.\!\?\(\)\%])([\s　\xA0]+)([^0-9０-９a-zA-Zａ-ｚＡ-Ｚ\,\.\!\?\(\)\%])`
	reSpaceMidKanjiEtc := regexp.MustCompile(patternSpaceMidKanjiEtc)

	// 表紙のHTMLかどうか
	// 最初のHTMLを表紙として扱う。
	// 表紙のHTMLは目次で区切らない
	isCoverPage := len(headings) == 0

	if firstHtmlOfAuditDoc {
		headings = append(headings, Heading{title: "監査報告書"})
	} else if isCoverPage {
		headings = append(headings, Heading{title: "表紙"})
	}

	var sb strings.Builder

	// HTMLをルートから深さ優先探索していく
	// 探索しながら目次を見つけたらheadingsにappendしていくことで、目次ごとの文字列を作成する
	var traverse func(*html.Node)
	traverse =
		func(n *html.Node) {
			if n.Type == html.DocumentNode {
				for child := n.FirstChild; child != nil; child = child.NextSibling {
					traverse(child)
				}
			} else if n.Type == html.ElementNode {
				if n.Data == "ix:header" {
					// InlineXBRLのheaderタグ以下は非表示項目なのでスキップする
					return
				}
				if n.Data == "head" {
					// headタグ以下は非表示項目なのでスキップする
					return
				}

				if isHeading(n) && !isCoverPage {
					// 【目次】処理。表紙の場合は目次で区切らない。
					// 目次の直前までのテキストを前の目次のテキストにセット
					headings[len(headings)-1].text = headings[len(headings)-1].text + " " + sb.String()

					// 新しい目次の処理
					sb = strings.Builder{}
					for child := n.FirstChild; child != nil; child = child.NextSibling {
						traverse(child)
					}
					title := sb.String()
					headings = append(headings, Heading{title: title, text: title})
					sb = strings.Builder{}
				} else {
					spacing := reSpacedTags.MatchString(n.Data)
					if spacing {
						sb.WriteString(" ")
					}
					for child := n.FirstChild; child != nil; child = child.NextSibling {
						traverse(child)
					}
					if spacing {
						sb.WriteString(" ")
					}
				}
			} else if n.Type == html.TextNode {
				// 日本語文字の間のスペースを除去する
				// 氏名などで幅調整をスペースでやっている場合を想定
				// 例：「監 査 法 人」を「監査法人」に
				s := n.Data
				for m := reSpaceMidKanjiEtc.MatchString(s); m; m = reSpaceMidKanjiEtc.MatchString(s) {
					s = reSpaceMidKanjiEtc.ReplaceAllString(s, "$1$3")
				}
				sb.WriteString(s)
			}
		}

	traverse(documentNode)
	headings[len(headings)-1].text = headings[len(headings)-1].text + " " + sb.String()

	return nil
}

// 要素内のテキストを返す
func innerText(element *html.Node) string {
	if element.Type == html.TextNode {
		return element.Data
	}

	if element.Type == html.ElementNode ||
		element.Type == html.DocumentNode {
		var sb strings.Builder
		for c := element.FirstChild; c != nil; c = c.NextSibling {
			sb.WriteString(innerText(c))
		}
		return sb.String()
	}
	return ""
}

// 目次のパターン
var reHeading = regexp.MustCompile(`^.{0,5}【(.*)】[\s　\xA0]*$`)

// 目次項目であるかどうか
func isHeading(element *html.Node) bool {
	text := innerText(element)
	matches := reHeading.FindAllString(text, 1)
	return len(matches) > 0
}

// breadcrumb を設定する
func setBreadcrumb() {
	var headingTypeStack []int
	var headingStack []string

	for i, s := range headings {
		htype := headingType(s.title)
		if htype == 0 {
			headings[i].breadcrumb = s.title
			continue
		}

		for slices.Contains(headingTypeStack, htype) {
			// 末尾の要素を削除
			headingTypeStack = headingTypeStack[:len(headingTypeStack)-1]
			headingStack = headingStack[:len(headingStack)-1]
		}
		headingTypeStack = append(headingTypeStack, htype)
		headingStack = append(headingStack, s.title)

		var sb strings.Builder
		sb.WriteString("本文")
		for _, h := range headingStack {
			m := reHeading.FindStringSubmatch(h)
			sb.WriteString(" > " + m[1])
		}
		headings[i].breadcrumb = sb.String()
	}
}

// 目次の種別を判定する
var reHeadingPrefix = regexp.MustCompile(`^(.*)【`)
var reHeadingPattern1 = regexp.MustCompile(`第.*部`)
var reHeadingPattern2 = regexp.MustCompile(`第[0-9０-９]`)
var reHeadingPattern3 = regexp.MustCompile(`[\(（][0-9０-９]+[\)）]`)
var reHeadingPattern4 = regexp.MustCompile(`[0-9０-９]`)
var reHeadingPattern5 = regexp.MustCompile(`[①-⑳]`)
var reHeadingPattern6 = regexp.MustCompile(`[\(（][ア-ンｱ-ﾝ]+[\)）]`)
var reHeadingPattern7 = regexp.MustCompile(`[ア-ンｱ-ﾝ]+`)
var reHeadingPattern8 = regexp.MustCompile(`[\(（][a-zａ-ｚ+[\)）]`)
var reHeadingPattern9 = regexp.MustCompile(`[a-zａ-ｚ]+`)

func headingType(heading string) int {
	match := reHeadingPrefix.FindStringSubmatch(heading)
	if len(match) == 0 {
		return 0
	}
	prefix := match[1]
	if reHeadingPattern1.MatchString(prefix) {
		return 1
	}
	if reHeadingPattern2.MatchString(prefix) {
		return 2
	}
	if reHeadingPattern3.MatchString(prefix) {
		return 3
	}
	if reHeadingPattern4.MatchString(prefix) {
		return 4
	}
	if reHeadingPattern5.MatchString(prefix) {
		return 5
	}
	if reHeadingPattern6.MatchString(prefix) {
		return 6
	}
	if reHeadingPattern7.MatchString(prefix) {
		return 7
	}
	if reHeadingPattern8.MatchString(prefix) {
		return 8
	}
	if reHeadingPattern9.MatchString(prefix) {
		return 9
	}
	return 99
}

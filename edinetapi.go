package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

// APIキー未設定エラー
var ErrApikey error = errors.New("ApiKey is empty")

// 書類一覧取得APIのURLの書式
var apiDocumentsUrlFormat string = "https://api.edinet-fsa.go.jp/api/v2/documents.json?date=%s&type=2&Subscription-Key=%s"

// 日付を指定して書類一覧取得APIのURL
func urlOfDocuments(date string) string {
	return fmt.Sprintf(apiDocumentsUrlFormat, date, apiKey)
}

// 書類一覧取得
func GetDocuments(date string) (*Documents, error) {
	if apiKey == "" {
		return nil, ErrApikey
	}
	url := urlOfDocuments(date)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	byteArray, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	data := new(Documents)
	if err := json.Unmarshal(byteArray, data); err != nil {
		return nil, err
	}

	return data, nil
}

// 本文ZIP取得APIのURLの書式
var apiDownloadZipUrlFormat string = "https://api.edinet-fsa.go.jp/api/v2/documents/%s?type=1&Subscription-Key=%s"

// 本文ZIP取得APIのURL
func urlOfTheZip(docID string) string {
	return fmt.Sprintf(apiDownloadZipUrlFormat, docID, apiKey)
}

// 本文ZIPを取得する
func DownloadZip(docID string, filepath string) error {
	if apiKey == "" {
		return ErrApikey
	}

	url := urlOfTheZip(docID)

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// 書類一覧JSONの構造体
type Documents struct {
	Metadata `json:"metadata"`
	Results  []Result `json:"results"`
}

type Metadata struct {
	Title     string `json:"title"`
	Parameter struct {
		Date string `json:"date"`
		Type string `json:"type"`
	} `json:"parameter"`
	Resultset struct {
		Count int `json:"count"`
	} `json:"resultset"`
	ProcessDateTime string `json:"processDateTime"`
	Status          string `json:"status"`
	Message         string `json:"message"`
}

type Result struct {
	SeqNumber            int    `json:"seqNumber"`
	DocID                string `json:"docID"`
	EdinetCode           string `json:"edinetCode"`
	SecCode              string `json:"secCode"`
	Jcn                  string `json:"JCN"`
	FilerName            string `json:"filerName"`
	FundCode             string `json:"fundCode"`
	OrdinanceCode        string `json:"ordinanceCode"`
	FormCode             string `json:"formCode"`
	DocTypeCode          string `json:"docTypeCode"`
	PeriodStart          string `json:"periodStart"`
	PeriodEnd            string `json:"periodEnd"`
	SubmitDateTime       string `json:"submitDateTime"`
	DocDescription       string `json:"docDescription"`
	IssuerEdinetCode     string `json:"issuerEdinetCode"`
	SubjectEdinetCode    string `json:"subjectEdinetCode"`
	SubsidiaryEdinetCode string `json:"subsidiaryEdinetCode"`
	CurrentReportReason  string `json:"currentReportReason"`
	ParentDocID          string `json:"parentDocID"`
	OpeDateTime          string `json:"opeDateTime"`
	WithdrawalStatus     string `json:"withdrawalStatus"`
	DocInfoEditStatus    string `json:"docInfoEditStatus"`
	DisclosureStatus     string `json:"disclosureStatus"`
	XbrlFlag             string `json:"xbrlFlag"`
	PdfFlag              string `json:"pdfFlag"`
	AttachDocFlag        string `json:"attachDocFlag"`
	EnglishDocFlag       string `json:"englishDocFlag"`
	CsvFlag              string `json:"csvFlag"`
	LegalStatus          string `json:"legalStatus"`
}

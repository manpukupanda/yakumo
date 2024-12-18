package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/lib/pq"
)

// ドライバー名
const dbDriver string = "postgres"

// データソース：DockerのPostgreSQLへの接続情報
// デフォルトのDockerコンテナを使わない場合は修正
var dbSource string = "user=PGroonga password=PGroonga dbname=PGroonga sslmode=disable"

// テーブルとインデックスを作成する。
// すでに存在する場合はそのまま。（エラーにもしない）
func createTableAndIndex() error {

	db, err := sql.Open(dbDriver, dbSource)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS documents (
			date char(10) NOT NULL,
			seqNumber int NOT NULL,
			docID char(8) NOT NULL,
			submitDateTime char(16) NULL,
			edinetCode char(6) NULL,
			secCode char(5) NULL,
			filerName text NULL,
			periodStart  char(10) NULL,
			periodEnd char(10) NULL,
			docDescription text NULL,
			PRIMARY KEY (date, seqNumber)
		);

		CREATE TABLE IF NOT EXISTS document_texts (
			docID char(8) NOT NULL,
			seq int NOT NULL,
			title text NOT NULL,
			breadcrumb text NOT NULL,
			content text NOT NULL,
			PRIMARY KEY (docID, seq)
		);

		CREATE EXTENSION IF NOT EXISTS pgroonga;
		CREATE INDEX IF NOT EXISTS pgroonga_content_index ON document_texts USING pgroonga (breadcrumb, content);
		`)
	if err != nil {
		return err
	}
	return nil
}

// データベースに登録済みかをチェックする
func exists(date string, result Result) (bool, error) {
	db, err := sql.Open(dbDriver, dbSource)
	if err != nil {
		return false, err
	}
	defer db.Close()

	// documentsテーブルの存在チェック
	rows, err := db.Query(`
		SELECT submitDateTime,edinetCode,secCode,filerName,periodStart,periodEnd,docDescription
		FROM documents 
		WHERE date = $1 AND seqNumber = $2
		`, date, result.SeqNumber)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	if rows.Next() {
		// データがあり変更なければtrue（存在する）
		var submitDateTime string
		var edinetCode string
		var secCode string
		var filerName string
		var periodStart string
		var periodEnd string
		var docDescription string
		err = rows.Scan(&submitDateTime, &edinetCode, &secCode, &filerName, &periodStart, &periodEnd, &docDescription)
		if err != nil {
			return false, err
		}

		if submitDateTime == result.SubmitDateTime &&
			edinetCode == result.EdinetCode &&
			strings.Trim(secCode, " ") == result.SecCode &&
			filerName == result.FilerName &&
			periodStart == result.PeriodStart &&
			periodEnd == result.PeriodEnd &&
			docDescription == result.DocDescription {
			return true, nil
		}
		fmt.Printf("[%s] [%s]\n", secCode, result.SecCode)
	}
	return false, nil
}

// データベースに保存する
func save(date string, result Result) error {

	db, err := sql.Open(dbDriver, dbSource)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Print("トランザクション開始エラー")
		return err
	}

	// documentsテーブルの更新
	// 同一キーレコードがあった場合、そのレコードの値と比較して変更があればアップデートする。変更なければなにもしない。
	// 同一キーレコードがなければインサートする
	rows, err := tx.Query(`
		SELECT submitDateTime,edinetCode,secCode,filerName,periodStart,periodEnd,docDescription
		FROM documents 
		WHERE date = $1 AND seqNumber = $2
		`, date, result.SeqNumber)
	if err != nil {
		log.Print("documentsテーブル selectエラー")
		tx.Rollback()
		return err
	}

	if rows.Next() {
		// データあり、変更あればupdate、なければなにもしない
		var submitDateTime string
		var edinetCode string
		var secCode string
		var filerName string
		var periodStart string
		var periodEnd string
		var docDescription string
		err = rows.Scan(&submitDateTime, &edinetCode, &secCode, &filerName, &periodStart, &periodEnd, &docDescription)
		if err != nil {
			log.Print("documentsテーブル scanエラー")
			tx.Rollback()
			return err
		}
		rows.Close()

		if submitDateTime == result.SubmitDateTime &&
			edinetCode == result.EdinetCode &&
			secCode == result.SecCode &&
			filerName == result.FilerName &&
			periodStart == result.PeriodStart &&
			periodEnd == result.PeriodEnd &&
			docDescription == result.DocDescription {
			// 変更なし、なにもしない
		} else {
			// 変更あり、アップデート
			log.Print("UPDATE documents")
			rows, err = tx.Query(`
			UPDATE documents
			SET submitDateTime = $1,
			    edinetCode = $2,
				secCode = $3,
				filerName = $4,
				periodStart = $5,
				periodEnd = $6,
				docDescription = $7
			WHERE date = $8 AND seqNumber = $9
			`, result.SubmitDateTime, result.EdinetCode, result.SecCode,
				result.FilerName, result.PeriodStart, result.PeriodEnd,
				result.DocDescription, date, result.SeqNumber)

			if err != nil {
				log.Print("documentsテーブル 更新エラー")
				tx.Rollback()
				return err
			}
			rows.Close()
		}
	} else {
		rows.Close()
		// データなし、インサート
		stmt, err := tx.Prepare("INSERT INTO documents(date,seqNumber,docID,submitDateTime,edinetCode,secCode,filerName,periodStart,periodEnd,docDescription) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)")
		if err != nil {
			log.Print("documentsテーブル insert prepareエラー")
			tx.Rollback()
			return err
		}
		_, err = stmt.Exec(date, result.SeqNumber, result.DocID,
			result.SubmitDateTime, result.EdinetCode, result.SecCode,
			result.FilerName, result.PeriodStart, result.PeriodEnd,
			result.DocDescription)
		if err != nil {
			log.Print("documentsテーブル insert execエラー")
			tx.Rollback()
			return err
		}
		stmt.Close()
	}

	// document_textsテーブルの更新
	// 同一キーレコードがあった場合、なにもしない。
	// 同一キーレコードがなければインサートする
	seq := 1
	rows2, err := tx.Query(`
		SELECT docID
		FROM document_texts 
		WHERE docID = $1
		`, result.DocID)
	if err != nil {
		log.Print("document_textsテーブル select エラー")
		tx.Rollback()
		return err
	}
	if rows2.Next() {
		rows2.Close()
		// レコードがあればなにもしない
	} else {
		rows2.Close()
		// レコードがないのでインサート
		stmt2, err := tx.Prepare("INSERT INTO document_texts(docID,seq,title,breadcrumb,content) VALUES($1,$2,$3,$4,$5)")
		if err != nil {
			log.Print("document_textsテーブル insert エラー")
			tx.Rollback()
			return err
		}

		for _, s := range headings {
			_, err = stmt2.Exec(result.DocID, seq, s.title, s.breadcrumb, s.content)
			if err != nil {
				tx.Rollback()
				return err
			}
			seq++
		}
		stmt2.Close()
	}
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

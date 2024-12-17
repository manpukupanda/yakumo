# Yakumo

## 概要
Yakumoは有価証券報告書全文検索システムです。

## 必要なソフトウェア
* Go 1.18以降
* Docker（PostgreSQL, PHP実行環境）

## 環境変数一覧

| 変数名           | 値の例               | 説明                             |
|------------------|----------------------|-------------------------------|
| `YAKUMO_EDINET_API_KEY`     | `s3cr3t...`     | EDINET API キー ※1|

※1：  
YakumoはEDINET APIを利用してデータを取得しています。EDINET APIを利用するにはEDINET API キーが必要です。  
[EDINET API仕様書](https://disclosure2dl.edinet-fsa.go.jp/guide/static/disclosure/WZEK0110.html)を参照のうえ、取得してください。

## インストール方法
githubからcloneして、goのソースをコンパイルして実行モジュールを作成します。  
windowsの場合はyakumoをyakumo.exeとしてください。
```bash
git clone https://github.com/manpukupanda/yakumo.git
cd yakumo
go build -o yakumo
```

## 使用方法

環境変数を設定する。
```bash
export YAKUMO_EDINET_API_KEY=<EDINET API キー>
```

Dockerのコンテナを起動します。  
コンテナはデータベース（PostgreSQL）とPHPの実行用の２つが起動します。
```bash
cd docker
docker-compose up -d
```

プログラムを実行する。  
初回は１年分のデータを取得して処理するため時間がかかります。
```bash
cd ..
yakumo
```

プログラムが終了したら、ブラウザで http://localhost:8000/index.php にアクセスして利用してください。

## ライセンス
このプロジェクトは Apache-2.0 license に基づいています。


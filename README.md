# Yakumo

## 概要
Yakumoは有価証券報告書全文検索システムです。

## 必要なソフトウェア
* Go 1.18以降
* Docker（PostgreSQL, PHP実行環境）

## 環境変数一覧

| 変数名           | 値の例               | 説明                             |
|------------------|----------------------|-------------------------------|
| `YAKUMO_DBSOURCE`   | `user=...`  | データベースの接続文字列 ※１    |
| `YAKUMO_EDINET_API_KEY`     | `s3cr3t...`     | EDINET API キー|


※１：DockerのPostgreSQLをそのまま使う場合のデータベース接続文字列は以下の通り
```
user=PGroonga password=PGroonga dbname=PGroonga sslmode=disable
```

## インストール方法
goのソースをコンパイルし、実行モジュールを作成します。  
windowsの場合はyakumoをyakumo.exeとしてください。
```bash
cd ..
go build -o yakumo
```

## 使用方法

環境変数を設定する。
```bash
export YAKUMO_EDINET_API_KEY=<EDINET API キー>
export YAKUMO_DBSOURCE=<データベースの接続文字列>
```

Dockerのコンテナを起動します。  
コンテナはデータベース（PostgreSQL）とPHPの実行用の２つが起動します。
```bash
git clone https://github.com/manpukupanda/yakumo.git
cd yakumo/docker
docker-compose up -d
```

プログラムを実行する。  
初回は１年分のデータを取得して処理するため時間がかかります。
```bash
cd yakumo
yakumo
```

プログラムが終了したら、ブラウザで http://localhost:8000/index.php にアクセスして利用してください。

## ライセンス
このプロジェクトは [Apache-2.0 license](LICENCE) に基づいています。


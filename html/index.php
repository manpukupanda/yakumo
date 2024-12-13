<!--
目次別全文検索の参考イメージ画面用PHPです。
-->

<?php
$search_query="";
$breadcrumb_query="";
if (isset($_GET['q'])){
    $search_query = $_GET['q'];
}
if (isset($_GET['b'])){
    $breadcrumb_query = $_GET['b'];
}
?>

<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <title>有価証券報告書 全文検索</title>
    <link href="https://unpkg.com/@primer/css@^20.2.4/dist/primer.css" rel="stylesheet" />
    <style>
        span.keyword{
            color:red;
            font-weight:bold;
        }
    </style>
</head>
<body>

<?php
    if ($search_query == ""){
?>
    <!-- 初期画面 -->
    <div class="container-sm text-center p-6">
        <form action="index.php" method="GET">
            <header>
                <h1>有価証券報告書 全文検索</h1>
            </header>
            <input class="mt-5 mb-5 form-control input-block" type="text" name="q" size="60" placeholder="検索キーワードを入力" value="<?php echo $search_query; ?>">
            <input type="submit" class="btn btn-primary" value="検索">
        </form>
    </div>
<?php
    } else {
?>
    <!-- 検索パラメータあり画面 -->
    <div class="container-sm text-center">
        <form action="index.php" method="GET">
            <header>
                <h1>有価証券報告書 全文検索</h1>
            </header>
            <input class="mt-5 mb-1 form-control input-block" type="text" name="q" size="60" placeholder="検索キーワードを入力" value="<?php echo $search_query; ?>">
            <input class="mb-5 form-control input-block" type="text" name="b" size="60" placeholder="目次で絞り込み" value="<?php echo $breadcrumb_query; ?>">
            <input type="submit" class="btn btn-primary" value="検索">
        </form>
    </div>
<?php
    }
?>

    <!-- 検索結果表示 -->
    <div class="container-lg p-6">

<?php
try {
    if ($search_query != ""){
        // Dockerの groonga/pgroonga イメージより作成したデータベースを想定
        $dsn = 'pgsql:dbname=PGroonga;host=PGroonga';
        $pdo = new PDO($dsn, 'PGroonga', 'PGroonga');

        $sql = <<<SQL

SELECT M.docid, M.filername, M.docdescription, M.submitdatetime, D.breadcrumb,
        (pgroonga_snippet_html(content,pgroonga_query_extract_keywords (:search_query), 400))[1] AS highlighted_content
FROM documents M, document_texts D
WHERE M.docid = D.docid
AND   (:search_query != '' AND D.content &@~ :search_query)
AND   (:breadcrumb_query = '' OR D.breadcrumb &@~ :breadcrumb_query)
ORDER BY M.submitdatetime DESC;

SQL;

        $stmt = $pdo->prepare($sql);
        $stmt->execute(['search_query' => $search_query, 'breadcrumb_query' => $breadcrumb_query]);

        $prevdocid = "";

        if ($stmt->rowCount() > 0) {
            while ($row = $stmt->fetch(PDO::FETCH_ASSOC)) {
                if ($prevdocid != $row["docid"]){
                    if ($prevdocid != ""){
                        echo "</div>";
                    }
                    echo "<div class='container-md mt-4 border color-border-accent p-2 rounded mb-2'>";
                    echo "<div class='text-bold f2'><a target='_blank' href='https://disclosure2.edinet-fsa.go.jp/WZEK0040.aspx?" . $row["docid"] . "'>" . $row["filername"] . "</a></div> ";
                    echo "<div class='f6 color-fg-subtle'>" . $row["docdescription"]. "／" .$row["submitdatetime"]."</div> ";
                }

                echo "<div class='container-md mt-2 border color-border-accent p-2 rounded mb-2'>";
                echo "<div class='f6 color-fg-subtle'>" . $row["breadcrumb"] . "</div> ";
                echo "<div class='container-lg mt-2 f5'>";
                echo $row["highlighted_content"];
                echo "</div>";
                echo "</div>";
                
                $prevdocid = $row["docid"];
            }
            echo "</div>";
        } else {
            echo "一致する結果がありませんでした";
        }
        $pdo = null;
    }
} catch (PDOException $e) {
    echo "接続エラー: " . $e->getMessage();
    $pdo = null;
}
?>

    </div>
</body>
</html>

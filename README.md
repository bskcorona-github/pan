# pan

このリポジトリに関する手順を以下に示します。

1. ソースコード内の `accessToken = ""` の部分に、あなたのアクセストークンを記述してください。

2. 以下のコード行で、PostgreSQL データベースへの接続情報を設定します。これを、ご自身の PostgreSQL の接続情報に合わせて変更してください。

   ```go
   connStr := "user=postgres dbname=postgres sslmode=disable password=tkz2001r"
   ```

3. ターミナルで以下のコマンドを実行して、データの同期を行ってください。

   ```
   go run main.go sync
   ```

4.  `entries` という名前のテーブルが正常に作成されて、パンのデータが３個入っていれば成功です。


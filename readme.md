# このプログラムの説明
TwitterのTimelineとlistを取得してGoogle Cloud Storageに保存し、Bigqueryに投入するプログラム。
main.goとmain2.goはTwitter APIと格闘していた残骸ですが、一応残しています。

GoogleにはService Account認証で認証させていますのでService AccountのIDを取得して下さい。
Twitterは、アプリケーションのAccess Tokenを使っているので同じく取得して下さい。※公開されていますが、リフレッシュ済みです。

## 必要事項
$ go get code.google.com/p/goauth2/oauth/jwt
$ go get code.google.com/p/google-api-go-client/bigquery/v2
$ go get code.google.com/p/google-api-go-client/storage/v1
$ go get github.com/ChimeraCoder/anaconda

anacondaがListの取得に対応していないので、以下のコードをtimeline.go辺りに追加してください。
```go
func (a TwitterApi) GetListStatus(v url.Values) (tweets []Tweet, err error) {
	response_ch := make(chan response)
	a.queryQueue <- query{BaseUrl + "/lists/statuses.json", v, &tweets, _GET, response_ch}
	return tweets, (<-response_ch).err
}
```

## 注意事項
1. 実行した後、何が起こっても怒らないで下さい。
1. 著作権はT.Yokoyamaにありますが、MIT Licenseにしていますので参考にしていただいてかまいません。

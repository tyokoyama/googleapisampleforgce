package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"code.google.com/p/goauth2/oauth/jwt"
	"code.google.com/p/google-api-go-client/bigquery/v2"
	"code.google.com/p/google-api-go-client/storage/v1"
	"github.com/ChimeraCoder/anaconda"
)

type data struct {
	ScreenName string `json:"screenname"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	Text       string `json:"text"`
	Favorite   int64  `json:"favorite"`
	Retweet    int64  `json:"retweet"`
}

type cache struct {
	Exist         bool  `json:"-"`
	Home_Since_Id int64 `json:"home_since_id"`
	List_Since_Id int64 `json:"list_since_id"`
}

type AuthSetting struct {
	TwitterConsumerKey string `json:"twitter_consumerkey"`
	TwitterConsumerSecret string `json:"twitter_consumersecret"`
	TwitterAccessToken string `json:"twitter_accesstoken"`
	TwitterAccessTokenSecret string `json:"twitter_accesstokensecret"`
	GoogleClientId string `json:"google_client_id"`
	GoogleEmailAddress string `json:"google_email_address"`
}

const (
	// 認証関係
	authFileName = "auth.json"

	// Twitter
	cacheFileName = "cache.json"

	// Google
	projectID    = "tksyokoyama"
	BucketName   = "chugokudb6sample"
	FolderName   = "twitter"

	scope       = storage.DevstorageFull_controlScope
	scope_bq    = bigquery.BigqueryScope
	authURL     = "https://accounts.google.com/o/oauth2/auth"
	tokenURL    = "https://accounts.google.com/o/oauth2/token"
	entityName  = "allUsers"
	redirectURL = "urn:ietf:wg:oauth:2.0:oob"

	// pemファイルを作る時は、Developers ConsoleでService AccountのKeyを作り、
	// p12ファイルをダウンロードした後、コマンドを実行。（opensslのパスワードはp12ファイルのダウンロード時に表示される）
	// openssl pkcs12 -in tksyokoyama-a366f0cc5805.p12 -nocerts -out key.pem -nodes
	googleSecretFileName = "key.pem"
	googleCacheFileName  = "gcache.json"
)

/*
BigQueryに投入するときのSchemaのフォーマット。
 [
 	{
 	 "name": "screenname",
 	 "type": "string"
 	 },
 	{
 	 "name": "name",
 	 "type": "string"
 	 },
 	{
 	 "name": "created_at",
 	 "type": "string"
 	 },
 	{
 	 "name": "text",
 	 "type": "string"
 	 },
 	{
 	 "name": "favorite",
 	 "type": "integer"
 	 },
 	{
 	 "name": "retweet",
 	 "type": "integer"
 	 }
 ]
*/
func main() {
	var output []data
	var c cache
	var setting AuthSetting

	bRead, err := ioutil.ReadFile(authFileName)
	if err != nil {
		log.Fatalf("Auth File Not Found. %v\n", err)
	}
	err = json.Unmarshal(bRead, &setting)
	if err != nil {
		log.Fatalf("Auth File Format Error. %v \n", err)
	}

	// goauth2の認証（Service Account認証）
	gKey, err := ioutil.ReadFile(googleSecretFileName)
	if err != nil {
		log.Fatalln(err)
	}

	gToken := jwt.NewToken(setting.GoogleEmailAddress, scope, gKey)
	bqToken := jwt.NewToken(setting.GoogleEmailAddress, scope_bq, gKey)

	transport, err := jwt.NewTransport(gToken)
	if err != nil {
		log.Fatalln(err)
	}

	trasport_bq, err := jwt.NewTransport(bqToken)
	if err != nil {
		log.Fatalln(err)
	}

	c.Exist = false
	// cacheの読み込み
	bRead, err = ioutil.ReadFile(cacheFileName)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalln(err)
		}
	} else {
		// キャッシュあり
		err = json.Unmarshal(bRead, &c)
		if err != nil {
			log.Fatalln(err)
		}
		c.Exist = true
	}

	anaconda.SetConsumerKey(setting.TwitterConsumerKey)
	anaconda.SetConsumerSecret(setting.TwitterConsumerSecret)

	api := anaconda.NewTwitterApi(setting.TwitterAccessToken, setting.TwitterAccessTokenSecret)
	defer api.Close()

	// タイムラインを取得
	homeParam := url.Values{}
	homeParam.Add("count", "200")
	if c.Exist {
		homeParam.Add("since_id", fmt.Sprintf("%d", c.Home_Since_Id))
	}
	tweet, err := api.GetHomeTimeline(homeParam)
	if err != nil {
		log.Fatalln(err)
	}

	for _, t := range tweet {
		output = append(output, tweetToData(t))

		if c.Home_Since_Id < t.Id {
			c.Home_Since_Id = t.Id
		}
	}

	// リストのTweetを取得
	statusParam := url.Values{}
	statusParam.Add("include_rts", "1")
	statusParam.Add("list_id", "59668871")
	statusParam.Add("slug", "samdbox")
	statusParam.Add("count", "200")
	if c.Exist {
		statusParam.Add("since_id", fmt.Sprintf("%d", c.List_Since_Id))
	}

	// Listの取得APIに対応していないのでローカルで拡張。（そんなに難しくない）
	tweet, err = api.GetListStatus(statusParam)
	if err != nil {
		log.Fatalln(err)
	}

	for _, t := range tweet {
		output = append(output, tweetToData(t))

		if c.List_Since_Id < t.Id {
			c.List_Since_Id = t.Id
		}
	}

	// データが無ければ何もしない。
	if len(output) <= 0 {
		log.Println("No Tweet.")
		return
	}

	// BigQuery用のデータ作成
	b, err := json.Marshal(output)
	if err != nil {
		log.Fatalln(err)
	}
	convstr := strings.Replace(string(b), "},", "}\n", -1)
	log.Println(convstr)

	tweetFileName := "data" + time.Now().Format("20060102150405") + ".txt"
	err = ioutil.WriteFile("data"+time.Now().Format("20060102150405")+".txt", []byte(convstr[1:len(convstr)-1]), 0755)
	if err != nil {
		log.Fatalln(err)
	}

	// cacheデータをファイルに保存
	bCache, err := json.Marshal(c)
	if err != nil {
		log.Fatalln(err)
	}
	err = ioutil.WriteFile(cacheFileName, bCache, 0755)
	if err != nil {
		log.Fatalln(err)
	}

	// Cloud Storageにファイルを保存
	gcs, err := storage.New(transport.Client())
	if err != nil {
		log.Fatalln(err)
	}

	// gcsでフォルダを指定したい場合はObjectのNameにパスを記入する。
	gcsfile := &storage.Object{Name: FolderName + "/" + tweetFileName}
	f, err := os.Open(tweetFileName)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	_, err = gcs.Objects.Insert(BucketName, gcsfile).Media(f).Do()
	if err != nil {
		log.Fatalf("Insert Failed to GCS. %v\n", err)
	}

	// BigQueryに追加
	bq, err := bigquery.New(trasport_bq.Client())
	if err != nil {
		log.Fatalln(err)
	}

	var job bigquery.Job
	job.Configuration = new(bigquery.JobConfiguration)
	job.Configuration.Load = new(bigquery.JobConfigurationLoad)
	job.Configuration.Load.Schema = &bigquery.TableSchema{
		Fields: []*bigquery.TableFieldSchema{
			&bigquery.TableFieldSchema{Name: "screenname", Type: "string"},
			&bigquery.TableFieldSchema{Name: "name", Type: "string"},
			&bigquery.TableFieldSchema{Name: "created_at", Type: "string"},
			&bigquery.TableFieldSchema{Name: "text", Type: "string"},
			&bigquery.TableFieldSchema{Name: "favorite", Type: "integer"},
			&bigquery.TableFieldSchema{Name: "retweet", Type: "integer"},
		},
	}
	job.Configuration.Load.SourceUris = []string{"gs://chugokudb6sample/" + FolderName + "/" + tweetFileName}
	job.Configuration.Load.DestinationTable = &bigquery.TableReference{DatasetId: BucketName, ProjectId: projectID, TableId: FolderName}
	job.Configuration.Load.SourceFormat = "NEWLINE_DELIMITED_JSON"

	res, err := bq.Jobs.Insert(projectID, &job).Do()
	if err != nil {
		log.Fatalf("Insert Failed to Bigquery. %v\n", err)
	}

	for res.Status.State != "DONE" {
		log.Println(res.Status)
		res, err = bq.Jobs.Get(res.JobReference.ProjectId, res.JobReference.JobId).Do()
		if err != nil {
			log.Printf("Job Status Update Error %v\n", err)
		}
		time.Sleep(1 * time.Second)
	}

	log.Println(res.Status.ErrorResult)

	for _, detail := range res.Status.Errors {
		log.Println(detail)
	}

	err = os.Remove(tweetFileName)
	if err != nil {
		log.Fatalln(err)
	}
}

func tweetToData(t anaconda.Tweet) (d data) {
	d.ScreenName = t.User.ScreenName
	d.Name = t.User.Name
	d.Text = t.Text
	d.Favorite = int64(t.FavoriteCount)
	d.Retweet = int64(t.RetweetCount)

	ti, err := time.Parse(time.RubyDate, t.CreatedAt)
	if err != nil {
		log.Fatalln(err)
	}
	ti = ti.Add(9 * time.Hour) // 標準時のようなので9時間足しておく。
	d.CreatedAt = ti.Format("2006/01/02 15:04:05.000")

	return
}

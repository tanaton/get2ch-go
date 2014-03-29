#get2ch-go
##インストール
`
$ go get github.com/tanaton/get2ch-go
`
##簡単な使い方
````Go
// 使用前の準備（一回呼べばOK）
get2ch.Start(get2ch.NewFileCache("/2ch/dat"), nil)
// 取得したいスレッドを指定して
get := get2ch.NewGet2ch("huga", "1234567890")
// データを取得
data, err := get.GetData()
````
大体の使い方はexample/readd.goを見れば分かるかも

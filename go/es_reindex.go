package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/hidu/go-speed"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type IndexInfo struct {
	BaseUrl string            `json:"url"`
	Index   string            `json:"index"`
	Type    string            `json:"type"`
	Header  map[string]string `json:"header"`
}

func (i *IndexInfo) IndexUri() string {
	return fmt.Sprintf("%s%s/%s", i.BaseUrl, i.Index, i.Type)
}

type Config struct {
	OriginIndex   IndexInfo              `json:"origin_index"`
	NewIndex      IndexInfo              `json:"new_index"`
	ScanQuery     map[string]interface{} `json:"scan_query"`
	ScanTime      string                 `json:"scan_time"`
	FieldsDefault map[string]interface{} `json:"fields_default"`
	sameIndex     bool                   `json:"-"`
}

func (c *Config) String() string {
	bf, _ := json.Marshal(c)
	return string(bf)
}

var conf_name = flag.String("conf", "es_reindex.json", "reindex config file name")
var loop_sleep = flag.Int64("loop_sleep", 0, "each loop sleep time")

var scroll_speed = speed.NewSpeed("scroll", 5, func(msg string, sp *speed.Speed) {
	log.Println("speed", msg)
})

func main() {
	flag.Parse()
	config, err := readConf(*conf_name)
	if err != nil {
		fmt.Println("parser config failed:", err)
		os.Exit(2)
	}

	reIndex(config)
	scroll_speed.Stop()
}

func readConf(conf_name string) (*Config, error) {
	bs, err := ioutil.ReadFile(conf_name)
	if err != nil {
		return nil, err
	}
	var conf *Config
	err = json.Unmarshal(bs, &conf)
	if err != nil {
		return nil, err
	}
	if conf.ScanTime == "" {
		conf.ScanTime = "60s"
	}

	if conf.OriginIndex.BaseUrl == "" {
		return nil, fmt.Errorf("origin_index url is empty")
	}
	if conf.OriginIndex.Index == "" {
		return nil, fmt.Errorf("origin_index index is empty")
	}

	if conf.NewIndex.BaseUrl == "" {
		conf.NewIndex.BaseUrl = conf.OriginIndex.BaseUrl
	}
	if conf.NewIndex.Index == "" {
		conf.NewIndex.Index = conf.OriginIndex.Index
	}
	if conf.NewIndex.Type == "" && conf.OriginIndex.Type != "" {
		conf.NewIndex.Type = conf.OriginIndex.Type
	}

	if conf.NewIndex.Type != "" && conf.OriginIndex.Type == "" {
		return nil, fmt.Errorf("when origin_index.type is empty,new_index.type must empty")
	}

	if conf.ScanQuery == nil {
		conf.ScanQuery = make(map[string]interface{})
		conf.ScanQuery["size"] = 1
	}

	if conf.FieldsDefault == nil {
		conf.FieldsDefault = make(map[string]interface{})
	}

	conf.sameIndex = conf.OriginIndex.IndexUri() == conf.NewIndex.IndexUri()

	return conf, nil
}

type ScanResult struct {
	ScrollID string `json:"_scroll_id"`
	Token    int    `json:"took"`
	TimedOut bool   `json:"timed_out"`
	Hits     struct {
		Total uint64 `json:"total"`
	}
}

type DataItem struct {
	Index  string                 `json:"_index"`
	Type   string                 `json:"_type"`
	ID     string                 `json:"_id"`
	Source map[string]interface{} `json:"_source"`
}

func (item *DataItem) String() string {
	header := map[string]interface{}{
		"index": map[string]string{
			"_index": item.Index,
			"_type":  item.Type,
			"_id":    item.ID,
		},
	}
	hd, _ := json.Marshal(header)
	bd, _ := json.Marshal(item.Source)
	return string(hd) + "\n" + string(bd) + "\n"
}

type ScrollResult struct {
	ScrollID string `json:"_scroll_id"`
	Token    int    `json:"took"`
	TimedOut bool   `json:"timed_out"`
	Hits     struct {
		Total uint64      `json:"total"`
		Hits  []*DataItem `json:"hits"`
	} `json:"hits"`
}

func (c *ScrollResult) String() string {
	bf, _ := json.MarshalIndent(c, " ", "  ")
	return string(bf)
}

func checkErr(msg string, err error) {
	if err != nil {
		log.Fatal(msg, err)
	}
}

func reIndex(conf *Config) {
	urlStr := conf.OriginIndex.BaseUrl + "/" + conf.OriginIndex.Index
	if conf.OriginIndex.Type != "" {
		urlStr += "/" + conf.OriginIndex.Type
	}
	urlStr += "/_search?search_type=scan&scroll=" + conf.ScanTime

	bs, err := json.Marshal(conf.ScanQuery)
	checkErr("scan_query is not json", err)

	log.Println("scan_url:", urlStr)
	req, _ := http.NewRequest("GET", urlStr, bytes.NewReader(bs))

	if conf.OriginIndex.Header != nil {
		for k, v := range conf.OriginIndex.Header {
			req.Header.Add(k, v)
		}
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	checkErr("scan failed", err)

	body, err := ioutil.ReadAll(resp.Body)
	checkErr("scan failed", err)

	var scanResult *ScanResult
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	err = dec.Decode(&scanResult)
	checkErr("scan failed,result is :"+string(body), err)

	if scanResult.Hits.Total < 1 {
		log.Println("no result,done")
		return
	}
	scroll_id := scanResult.ScrollID

	var scrollResult *ScrollResult

	for {
		urlScroll := req.URL.Scheme + "://" + req.URL.Host + "/_search/scroll?scroll=" + conf.ScanTime
		reqScroll, _ := http.NewRequest("GET", urlScroll, strings.NewReader(scroll_id))

		if conf.OriginIndex.Header != nil {
			for k, v := range conf.OriginIndex.Header {
				reqScroll.Header.Add(k, v)
			}
		}
		resp, err := client.Do(reqScroll)
		checkErr("scroll failed", err)

		body, err := ioutil.ReadAll(resp.Body)
		checkErr("scroll failed", err)

		err = json.Unmarshal(body, &scrollResult)
		checkErr("scroll failed,result is :"+string(body), err)

		scroll_speed.Inc(len(scrollResult.Hits.Hits), len(body), 1)

		if len(scrollResult.Hits.Hits) < 1 {
			break
		}

		//		fmt.Println("scroll",scrollResult)
		reBulk(client, conf, scrollResult)
		//		break
		if *loop_sleep > 0 {
			time.Sleep(time.Duration(*loop_sleep) * time.Second)
		}
	}
}

func reBulk(client *http.Client, conf *Config, scrollResult *ScrollResult) {
	var datas []string
	for _, item := range scrollResult.Hits.Hits {
		item.Index = conf.NewIndex.Index

		if conf.NewIndex.Type != "" {
			item.Type = conf.NewIndex.Type
		}

		_hasChange := false
		for k, v := range conf.FieldsDefault {
			if _, has := item.Source[k]; !has {
				item.Source[k] = v
				_hasChange = true
			}
		}

		if !conf.sameIndex || _hasChange {
			datas = append(datas, item.String())
		}
	}

	if len(datas) < 1 {
		log.Println("not change,skip reindex")
		return
	}

	urlStr := conf.NewIndex.BaseUrl + "/_bulk"
	breq, _ := http.NewRequest("POST", urlStr, strings.NewReader(strings.Join(datas, "\n")))
	if conf.NewIndex.Header != nil {
		for k, v := range conf.NewIndex.Header {
			breq.Header.Add(k, v)
		}
	}
	resp, err := client.Do(breq)
	checkErr("bulk failed", err)

	body, err := ioutil.ReadAll(resp.Body)
	checkErr("bulk failed", err)
	log.Println("bulk resp:", string(body))
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/clbanning/x2j"
	"github.com/hidu/goutils/assest"
	"github.com/hidu/goutils/jsonutils"
	"log"
	"net/http"
	"strings"
	"time"
)

const VERSION = "20150712"

var port = flag.Int("port", 8100, "addr port")

func main() {
	flag.Parse()

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/xml2json", handleXml2json)
	http.HandleFunc("/jsonfix", handleJsonFix)
	http.HandleFunc("/getxmljsonschema", handleGetXmlJsonSchema)
	http.HandleFunc("/getjsonjsonschema", handleGetJsonJsonSchema)
	http.Handle("/res/", assest.Assest.HttpHandler("/"))

	addr := fmt.Sprintf(":%d", *port)
	log.Println("listen :", addr)

	err := http.ListenAndServe(addr, nil)
	log.Println("exit error:", err)
}

type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
	Used float64     `json:"used"`

	Status    int                 `json:"-"`
	rw        http.ResponseWriter `json:"-"`
	req       *http.Request       `json:"-"`
	startTime time.Time           `json:"-"`
}

func NewResult(rw http.ResponseWriter, req *http.Request) *Result {
	res := &Result{rw: rw, Status: http.StatusOK, req: req}
	res.startTime = time.Now()
	return res
}

func (res *Result) WriteData(code int, msg string, data interface{}) {
	res.Code = code
	res.Msg = msg
	res.Data = data
	res.Used = time.Now().Sub(res.startTime).Seconds()
	bs, _ := json.Marshal(res)
	res.rw.Header().Set("Content-Type", "application/json;charset=utf-8")
	res.rw.WriteHeader(res.Status)
	res.rw.Write(bs)
	log.Println(res.req.RemoteAddr, res.req.URL.Path, "code:", code, "used:", res.Used)
}

func handleXml2json(rw http.ResponseWriter, req *http.Request) {
	xml := strings.TrimSpace(req.PostFormValue("xml"))
	json_schema := strings.TrimSpace(req.PostFormValue("json_schema"))
	res := NewResult(rw, req)
	if xml == "" {
		res.WriteData(1, "xml is empty", nil)
		return
	}
	jsonStr, err := x2j.ToJson(strings.NewReader(xml))
	if err != nil {
		res.WriteData(2, err.Error(), nil)
		return
	}
	jsonFix(res, jsonStr, json_schema)
}

func jsonFix(res *Result, jsonStr string, json_schema string) {
	var jsonData interface{}
	json.Unmarshal([]byte(jsonStr), &jsonData)

	if json_schema != "" {
		var schema interface{}
		err := json.Unmarshal([]byte(json_schema), &schema)
		if err != nil {
			res.WriteData(2, err.Error(), nil)
			return
		}
		jsonData, err = jsonutils.FixDataWithSchema(jsonData, schema)
		if err != nil {
			res.WriteData(2, err.Error(), nil)
			return
		}
	}
	res.Status = http.StatusOK
	res.WriteData(0, "success", jsonData)

}

func handleGetXmlJsonSchema(rw http.ResponseWriter, req *http.Request) {
	xml := strings.TrimSpace(req.PostFormValue("xml"))
	res := NewResult(rw, req)
	if xml == "" {
		res.WriteData(1, "xml is empty", nil)
		return
	}
	jsonObj, err := x2j.ToMap(strings.NewReader(xml))
	if err != nil {
		res.WriteData(1, err.Error(), nil)
		return
	}
	schema, err := jsonutils.GenJsonSchema(jsonObj)
	if err != nil {
		res.WriteData(1, err.Error(), nil)
		return
	}
	res.Status = http.StatusOK
	res.WriteData(0, "success", schema)
}

func handleGetJsonJsonSchema(rw http.ResponseWriter, req *http.Request) {
	jsonStr := strings.TrimSpace(req.PostFormValue("json"))
	res := NewResult(rw, req)
	if jsonStr == "" {
		res.WriteData(1, "json str is empty", nil)
		return
	}
	var jsonObj interface{}
	err := json.Unmarshal([]byte(jsonStr), &jsonObj)
	if err != nil {
		res.WriteData(1, err.Error(), nil)
		return
	}
	schema, err := jsonutils.GenJsonSchema(jsonObj)
	if err != nil {
		res.WriteData(1, err.Error(), nil)
		return
	}
	res.Status = http.StatusOK
	res.WriteData(0, "success", schema)
}

func handleJsonFix(rw http.ResponseWriter, req *http.Request) {
	res := NewResult(rw, req)
	jsonStr := req.PostFormValue("json")
	json_schema := req.PostFormValue("json_schema")
	if json_schema == "" {
		res.WriteData(400, "json_schema is empty", nil)
		return
	}
	jsonFix(res, jsonStr, json_schema)
}

func handleIndex(rw http.ResponseWriter, req *http.Request) {
	rw.Write([]byte(indexHtml))
}

var indexHtml = `
<html>
<head>
<title>xml2json</title>
<meta charset="utf-8">
<script type="text/javascript" src="/res/js/jquery.min.js"></script>
</head>
<body>

<h2>xml2json:</h2>
<form action="/xml2json" method="post">
<p>xml:</p>
<textarea name="xml" style="width:95%;height:200px" id="xml"></textarea>

<p>json_schema:
&nbsp;<a href="#" onclick='genXmlSchema();return false;'>genJsonSchema</a>
&nbsp;<span id="emsg1"></span>
</p>
<textarea name="json_schema" id="json_schema" style="width:95%;height:150px"></textarea>

<p><input type="submit" value="convert"></p>
</form>

<h2>jsonFix:</h2>
<form action="/jsonfix" method="post">
<p>json:</p>
<textarea name="json" id="json_str" style="width:95%;height:200px"></textarea>

<p>json_schema:
&nbsp;<a href="#" onclick='genJsonSchema();return false;'>genJsonSchema</a>
&nbsp;<span id="emsg2"></span>
</p>
<textarea name="json_schema" id="json_schema_2" style="width:95%;height:150px"></textarea>

<p><input type="submit" value="fix"></p>
</form>
<script>
function genXmlSchema(){
	$.post("/getxmljsonschema",{xml:$("#xml").val()},function(data){
		if(data.code!=0){
			$("#emsg1").html("<font color=red>"+data.msg+"</font>");
			return;
		}else{
			$("#emsg1").html(data.msg);
			var jsonStr=JSON.stringify(data.data,null,"  ")
			$("#json_schema").val(jsonStr)
		}
	})
}
function genJsonSchema(){
	$.post("/getjsonjsonschema",{json:$("#json_str").val()},function(data){
		if(data.code!=0){
			$("#emsg2").html("<font color=red>"+data.msg+"</font>");
			return;
		}else{
			$("#emsg2").html(data.msg);
			var jsonStr=JSON.stringify(data.data,null,"  ")
			$("#json_schema_2").val(jsonStr)
		}
	})
}
</script>

</body>
</html>

`

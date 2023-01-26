package main

import (
	"time"
	
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"math/rand"
	"strings"
	"encoding/json"
)

func main() {
	rand.Seed(time.Now().UnixNano())
		
	httpposturl := "http://model-internal:8094/v2/models/ctr-lgbm/versions/v0.1.0/infer"

	features := []string{}
	for i:=0;i<17;i++ {
		features = append(features, fmt.Sprintf("%v", rand.Intn(2)))
	}

	str_features := strings.Join(features, ", ")

	jsonStr := `{"inputs": [{"name": "predict-prob", "shape": [1, 17], "datatype": "FP32", "data": [[` + str_features + `]]}]}`

	jsonData := []byte(jsonStr)

	httpReq, error := http.NewRequest("POST", httpposturl, bytes.NewBuffer(jsonData))
	httpReq.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	httpResp, error := client.Do(httpReq)
	if error != nil {
		panic(error)
	}
	defer httpResp.Body.Close()

	// fmt.Println("response Status:", response.Status)
	// fmt.Println("response Headers:", response.Header)
	body, _ := ioutil.ReadAll(httpResp.Body)

	var result map[string][]interface{}
	json.Unmarshal([]byte(body), &result)
	outputs := result["outputs"]
	output := outputs[0]
	arr := output.(map[string]interface{})["data"]

	fmt.Println("predict: ", arr)

}
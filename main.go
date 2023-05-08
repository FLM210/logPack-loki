package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize"
)

var query = flag.String("query", "", "The LogQL query to perform")
var limit = flag.String("limit", "2000", "The max number of entries to return.")
var startTimestamp = flag.String("start", "", "The start time for the query (default 1 hours ago)")
var endTimestamp = flag.String("end", "", "The end time for the query (default now)")
var baseUrl = flag.String("url", "127.0.0.1:3100", "The loki address")

// var startTimestamp = flag.String("start", strconv.FormatInt(time.Now().Add(-2*time.Hour).Unix(), 10), "The start time for the query,default 2 hours ago.")
// var endTimestamp = flag.String("end", strconv.FormatInt(time.Now().Unix(), 10), "The end time for the query")

func main() {
	flag.Parse()
	if len(*query) == 0 {
		fmt.Println("Please input the LogQL query")
		return
	}
	if len(*startTimestamp) != 0 {
		startTime, err := time.Parse(time.RFC3339, *startTimestamp)
		if err != nil {
			fmt.Println("Error parsing time: ", err)
			return
		} else {
			fmt.Println(startTime.Unix())
			*startTimestamp = fmt.Sprint(startTime.Unix())
		}
	}
	if len(*endTimestamp) != 0 {
		endTime, err := time.Parse(time.RFC3339, *endTimestamp)
		if err != nil {
			fmt.Println("Error parsing time: ", err)
			return
		} else {
			*endTimestamp = string(rune(endTime.Unix()))
		}
	}
	// 构造Loki API查询URL

	url := fmt.Sprintf("http://%s/loki/api/v1/query_range?query=%s&start=%s&end=%s&limit=%s", *baseUrl, *query, *startTimestamp, *endTimestamp, *limit)
	fmt.Println(url)
	// 执行HTTP请求并处理响应
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error executing HTTP request:", err)
		return
	}
	//defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: received HTTP status code %d\n", resp.StatusCode)
		return
	}

	// 解析响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	//	fmt.Println(body)
	// 处理响应内容并生成Excel文件
	f := excelize.NewFile()
	sheetName := "logs"
	f.SetSheetName(f.GetSheetName(1), sheetName)
	f.SetCellValue(sheetName, "A1", "PodName")
	f.SetCellValue(sheetName, "B1", "Time")
	f.SetCellValue(sheetName, "C1", "Log Message")

	data := make(map[string]interface{})
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON data:", err)
		return
	}
	//fmt.Println(data)
	rows := data["data"].(map[string]interface{})["result"].([]interface{})
	rowWriteNum := 1
	for _, row := range rows {
		podName := row.(map[string]interface{})["stream"].(map[string]interface{})["pod"].(string)
		values := row.(map[string]interface{})["values"].([]interface{})
		for _, value := range values {
			timeStr := value.([]interface{})[0].(string)
			timeInt, err := strconv.ParseInt(timeStr, 10, 64)
			if err != nil {
				fmt.Println("Error parsing string:", err)
			}
			//time := time.Unix(timeInt/1000000000, (timeInt%1000000000)*int64(time.Millisecond))
			time := time.Unix(0, timeInt)
			regex := regexp.MustCompile("\x1b\\[(\\d|;)+[m|K]")
			logMessage := regex.ReplaceAllString(value.([]interface{})[1].(string), "")
			rowWriteNum = rowWriteNum + 1
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowWriteNum), podName)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowWriteNum), time)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowWriteNum), logMessage)
		}
	}

	if err := f.SaveAs("logs.xlsx"); err != nil {
		fmt.Println("Error saving Excel file:", err)
		return
	}

	fmt.Println("Logs saved to logs.xlsx")
}

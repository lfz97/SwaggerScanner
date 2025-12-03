package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"swaggerScanner/myutils"
	"swaggerScanner/swaggerParser"
	"sync"

	"github.com/go-resty/resty/v2"
)

// 汇总所有Swagger文件中的URL信息，并发处理提升效率
func GroupUrlsFromAllSwaggerFiles(fileList []string) []swaggerParser.UrlInfo {
	var UrlInfo_s []swaggerParser.UrlInfo
	wg := sync.WaitGroup{}
	ch_UrlInfo_s := make(chan []swaggerParser.UrlInfo)
	for _, filePath := range fileList {
		wg.Add(1)
		go func(fp string) {
			UrlInfo_s_p, err := swaggerParser.SwaggerParser(fp)
			if err != nil {
				fmt.Println(err)
				wg.Done()
				return
			}
			ch_UrlInfo_s <- *UrlInfo_s_p
			wg.Done()
		}(filePath)
	}
	go func() {
		wg.Wait()
		close(ch_UrlInfo_s)
	}()

	for U_s := range ch_UrlInfo_s {
		for _, item := range U_s {
			UrlInfo_s = append(UrlInfo_s, item)
		}
	}
	return UrlInfo_s
}

// 获取指定目录下的所有Swagger文件名
func GetSwaggerFileNamesFromDir(dirPath string) ([]string, error, bool) {
	dirExists := false
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) == true {
		er := os.Mkdir(dirPath, 0777)
		if er != nil {
			return nil, fmt.Errorf("directory does not exist ,and created failed: %s", dirPath), dirExists
		}
		return nil, fmt.Errorf("directory does not exist , created successfully!: %s", dirPath), dirExists
	} else if err == nil {
		dirExists = true
	} else {
		return nil, fmt.Errorf("directory exists but error occurred: %s", err), dirExists
	}
	fileList := []string{}
	files, err := os.ReadDir(dirPath)
	if err != nil {
		fmt.Println("Read dir failed:", err)
		return nil, err, dirExists
	}
	for _, file := range files {
		if !file.IsDir() {
			fileList = append(fileList, dirPath+"/"+file.Name())

		}
	}
	return fileList, nil, dirExists
}

type ReqResult struct {
	RequstUrl        string
	Method           string
	FullUrl          string
	ReqBody          string
	StatusCode       int
	ContentLength    int
	ContentPrefix250 string
}

func (r ReqResult) GetHeader() []string {
	return []string{"RequstUrl", "Method", "FullUrl", "ReqBody", "StatusCode", "ContentLength", "ContentPrefix250"}
}
func (r ReqResult) GetRow() []string {
	return []string{
		r.RequstUrl,
		r.Method,
		r.FullUrl,
		r.ReqBody,
		fmt.Sprintf("%d", r.StatusCode),
		fmt.Sprintf("%d", r.ContentLength),
		r.ContentPrefix250,
	}
}

func DoBatchRequestWithParam(UrlInfo_s []swaggerParser.UrlInfo) []ReqResult {
	// helper: fake data generator
	generateFakeData := func(t string) any {
		switch strings.ToLower(t) {
		case "boolean":
			return true
		case "integer", "number":
			return 888
		case "string":
			return "test_string"
		default:
			return "888"
		}
	}

	// helper: build request body recursively from schema
	var buildRequestBody func(s swaggerParser.UrlInfoParameterSchema) any
	buildRequestBody = func(s swaggerParser.UrlInfoParameterSchema) any {
		if s.Type == "object" && len(s.Properties) > 0 {
			m := make(map[string]any)
			for k, v := range s.Properties {
				if v.Type == "array" && v.Items != nil {
					m[k] = buildRequestBody(*v.Items)
				} else if v.Type == "object" && v.Items != nil { // nested object via items (edge case)
					m[k] = buildRequestBody(*v.Items)
				} else {
					m[k] = generateFakeData(v.Type)
				}
			}
			return m
		}
		if s.Type == "array" && s.Items != nil {
			// create one element for array
			return []any{buildRequestBody(*s.Items)}
		}
		return generateFakeData(s.Type)
	}

	var results []ReqResult
	client := resty.New().SetDebug(true)

	for _, urlInfo := range UrlInfo_s {
		r := ReqResult{}
		req := client.R()
		var err error
		var resp *resty.Response

		// replace path params
		requestPath := urlInfo.FullPath
		for _, p := range urlInfo.Parameters {
			if p.In == "path" {
				ph := "{" + p.Name + "}"
				requestPath = strings.ReplaceAll(requestPath, ph, fmt.Sprintf("%v", generateFakeData(p.Type)))
			}
		}

		method := strings.ToLower(urlInfo.Method)
		switch method {
		case "get":
			req.SetHeader("Content-Type", urlInfo.ContentType)
			for _, p := range urlInfo.Parameters {
				if p.In == "query" {
					req.SetQueryParam(p.Name, fmt.Sprintf("%v", generateFakeData(p.Type)))
				}
			}
			resp, err = req.Get(requestPath)
		case "post":
			req.SetHeader("Content-Type", urlInfo.ContentType)
			var bodyParam *swaggerParser.UrlInfoParameter
			for i := range urlInfo.Parameters {
				p := &urlInfo.Parameters[i]
				if p.In == "body" {
					bodyParam = p
				} else if p.In == "query" {
					req.SetQueryParam(p.Name, fmt.Sprintf("%v", generateFakeData(p.Type)))
				}
			}
			if bodyParam != nil {
				req.SetBody(buildRequestBody(bodyParam.Schema))
			}
			resp, err = req.Post(requestPath)
		default:
			// unsupported method, record and continue
			r.RequstUrl = urlInfo.FullPath
			r.Method = urlInfo.Method
			results = append(results, r)
			continue
		}

		if err != nil {
			r.RequstUrl = urlInfo.FullPath
			r.Method = urlInfo.Method
			r.StatusCode = 0
			r.ContentLength = 0
			r.ContentPrefix250 = "Request failed: " + err.Error()
			results = append(results, r)
			continue
		}

		r.RequstUrl = urlInfo.FullPath
		r.Method = urlInfo.Method
		r.FullUrl = (*resp.Request).URL
		// capture body we sent (if any)
		if resp.Request.RawRequest != nil && resp.Request.RawRequest.Body != nil {
			bodyReader, getErr := resp.Request.RawRequest.GetBody()
			if getErr == nil {
				bodyBytes, readErr := io.ReadAll(bodyReader)
				if readErr == nil {
					r.ReqBody = string(bodyBytes)
				}
			}
		}
		r.StatusCode = resp.StatusCode()
		r.ContentLength = int(resp.Size())
		bodyStr := resp.String()
		if len(bodyStr) < 250 {
			r.ContentPrefix250 = bodyStr
		} else {
			r.ContentPrefix250 = bodyStr[:250]
		}
		results = append(results, r)
	}
	return results
}

type ReqResultWithoutParam struct {
	RequstUrl        string
	Method           string
	StatusCode       int
	ContentLength    int
	ContentPrefix250 string
}

func (r ReqResultWithoutParam) GetHeader() []string {
	return []string{"RequstUrl", "Method", "StatusCode", "ContentLength", "ContentPrefix250"}
}
func (r ReqResultWithoutParam) GetRow() []string {
	return []string{
		r.RequstUrl,
		r.Method,
		fmt.Sprintf("%d", r.StatusCode),
		fmt.Sprintf("%d", r.ContentLength),
		r.ContentPrefix250,
	}
}
func DoBatchRequestWithoutParam(UrlInfo_s []swaggerParser.UrlInfo) []ReqResultWithoutParam {
	var results []ReqResultWithoutParam
	client := resty.New().SetDebug(true)
	for _, urlInfo := range UrlInfo_s {
		ReqResultWithoutParamTmp := ReqResultWithoutParam{}
		req := client.R()
		var err error
		var resp_p *resty.Response

		// 处理路径参数, 即使是无参数请求，路径参数也需要填充
		requestPath := urlInfo.FullPath
		for _, param := range urlInfo.Parameters {
			if param.In == "path" {
				placeholder := "{" + param.Name + "}"
				// 对于无参数扫描，我们依然用一个通用值填充路径参数以避免404
				requestPath = strings.Replace(requestPath, placeholder, "888", -1)
			}
		}

		if strings.ToLower(urlInfo.Method) == "get" {
			req.SetHeader("Content-Type", urlInfo.ContentType)
			resp_p, err = req.Get(requestPath)
			if err != nil {
				fmt.Println("line 93 Request failed:", err)
				ReqResultWithoutParamTmp.RequstUrl = urlInfo.FullPath
				ReqResultWithoutParamTmp.Method = urlInfo.Method
				ReqResultWithoutParamTmp.StatusCode = 0
				ReqResultWithoutParamTmp.ContentLength = 0
				ReqResultWithoutParamTmp.ContentPrefix250 = "Request failed: " + err.Error()
				results = append(results, ReqResultWithoutParamTmp)
				continue
			}

			ReqResultWithoutParamTmp.RequstUrl = urlInfo.FullPath
			ReqResultWithoutParamTmp.Method = urlInfo.Method
			ReqResultWithoutParamTmp.StatusCode = resp_p.StatusCode()
			ReqResultWithoutParamTmp.ContentLength = int(resp_p.Size())
			if len(resp_p.String()) < 250 {
				ReqResultWithoutParamTmp.ContentPrefix250 = resp_p.String()
			} else {
				ReqResultWithoutParamTmp.ContentPrefix250 = resp_p.String()[0:250]
			}

		} else if strings.ToLower(urlInfo.Method) == "post" {
			req.SetHeader("Content-Type", urlInfo.ContentType)
			resp_p, err = req.Post(requestPath)
			if err != nil {
				fmt.Println("line 161 Request failed:", err)
				ReqResultWithoutParamTmp.RequstUrl = urlInfo.FullPath
				ReqResultWithoutParamTmp.Method = urlInfo.Method
				ReqResultWithoutParamTmp.StatusCode = 0
				ReqResultWithoutParamTmp.ContentLength = 0
				ReqResultWithoutParamTmp.ContentPrefix250 = "Request failed: " + err.Error()
				results = append(results, ReqResultWithoutParamTmp)
				continue
			}

			ReqResultWithoutParamTmp.RequstUrl = urlInfo.FullPath
			ReqResultWithoutParamTmp.Method = urlInfo.Method
			ReqResultWithoutParamTmp.StatusCode = resp_p.StatusCode()
			ReqResultWithoutParamTmp.ContentLength = int(resp_p.Size())
			if len(resp_p.String()) < 250 {
				ReqResultWithoutParamTmp.ContentPrefix250 = resp_p.String()
			} else {
				ReqResultWithoutParamTmp.ContentPrefix250 = resp_p.String()[0:250]
			}
		} else {
			ReqResultWithoutParamTmp.RequstUrl = urlInfo.FullPath
			ReqResultWithoutParamTmp.Method = urlInfo.Method
			ReqResultWithoutParamTmp.StatusCode = 0
			ReqResultWithoutParamTmp.ContentLength = 0
			ReqResultWithoutParamTmp.ContentPrefix250 = ""
			results = append(results, ReqResultWithoutParamTmp)

		}
		results = append(results, ReqResultWithoutParamTmp)
	}
	return results
}

func ScanAllUrls(UrlInfo_s []swaggerParser.UrlInfo, goroutineNum int) ([]ReqResult, []ReqResultWithoutParam) {
	AllUrlResults := []ReqResult{}
	AllUrlWithoutParamResults := []ReqResultWithoutParam{}

	UrlInfo_s_s := myutils.SplitSliceEqualParts[swaggerParser.UrlInfo](UrlInfo_s, goroutineNum)
	wg_Worker := sync.WaitGroup{}
	wg_Collector := sync.WaitGroup{}
	ch_results_s := make(chan []ReqResult)
	ch_resultsWithoutParam_s := make(chan []ReqResultWithoutParam)
	for _, UrlInfo_s := range UrlInfo_s_s {
		wg_Worker.Add(2)
		go func(U_s []swaggerParser.UrlInfo, ch chan []ReqResult) {
			result_s := DoBatchRequestWithParam(U_s)
			ch_results_s <- result_s
			wg_Worker.Done()
		}(UrlInfo_s, ch_results_s)

		go func(U_s []swaggerParser.UrlInfo, ch chan []ReqResultWithoutParam) {
			result_s := DoBatchRequestWithoutParam(U_s)
			ch_resultsWithoutParam_s <- result_s
			wg_Worker.Done()
		}(UrlInfo_s, ch_resultsWithoutParam_s)
	}

	wg_Collector.Add(1)
	go func() {
		for result_s := range ch_results_s {
			AllUrlResults = append(AllUrlResults, result_s...)

		}
		wg_Collector.Done()
	}()

	wg_Collector.Add(1)
	go func() {
		for resultWithoutParam_s := range ch_resultsWithoutParam_s {
			AllUrlWithoutParamResults = append(AllUrlWithoutParamResults, resultWithoutParam_s...)
		}
		wg_Collector.Done()

	}()

	go func() {
		wg_Worker.Wait()
		close(ch_results_s)
		close(ch_resultsWithoutParam_s)
	}()

	wg_Collector.Wait()

	return AllUrlResults, AllUrlWithoutParamResults
}

type CsvRecord interface {
	GetHeader() []string
	GetRow() []string
}

func ExportResultsToCsvFile[T CsvRecord](Results_s []T, filePath string) error {
	fd, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	defer fd.Close()
	Writter_p := csv.NewWriter(fd)
	// 写入CSV头
	header := Results_s[0].GetHeader()
	Writter_p.Write(header)
	// 写入数据行
	for _, result := range Results_s {
		row := result.GetRow()
		Writter_p.Write(row)
	}
	Writter_p.Flush()
	return nil

}

func main() {

	fileList, err, dirExists := GetSwaggerFileNamesFromDir("请将所有Swagger.json放入此文件夹")
	if err != nil {
		fmt.Println(err)
		return
	}
	if dirExists == false {
		fmt.Println(err)
		return
	}
	if len(fileList) == 0 {
		fmt.Println("没有找到Swagger文件，请将Swagger.json放入指定文件夹")
		return
	}
	UrlInfo_s := GroupUrlsFromAllSwaggerFiles(fileList)
	if len(UrlInfo_s) == 0 {
		fmt.Println("没有找到有效的URL信息，请检查Swagger文件格式")
		return
	}

	AllUrlResults_s, AllUrlWithoutParamResults_s := ScanAllUrls(UrlInfo_s, 8)
	err = ExportResultsToCsvFile(AllUrlResults_s, "扫描结果.csv")
	if err != nil {
		fmt.Println("导出CSV文件失败:", err)
		return
	}
	err = ExportResultsToCsvFile(AllUrlWithoutParamResults_s, "扫描结果_无参数请求.csv")
	if err != nil {
		fmt.Println("导出CSV文件失败:", err)
		return

	}

}

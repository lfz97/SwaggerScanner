# swaggerScanner

一个用于 **解析和扫描 Swagger (OpenAPI) JSON** 的 Go 工具，帮助快速发现 **未授权访问接口**。

## **功能概述**
- 解析 Swagger (OpenAPI) JSON 文件
- 提取接口信息（路径、方法、参数）
- 自动生成请求并并发访问所有接口
- 输出扫描结果到 CSV 文件，包含：
  - `RequestUrl`：请求路径
  - `Method`：请求方法
  - `FullUrl`：完整 URL
  - `ReqBody`：请求体
  - `StatusCode`：响应状态码
  - `ContentLength`：响应长度
  - `ContentPrefix250`：响应正文前 250 字节

## **项目结构**
```
go.mod
main.go
myutils/
    SplitSliceEqualParts.go      # 切片分割工具
swaggerParser/
    SwaggerJson.go               # Swagger JSON 结构体定义
    swaggerParser.go             # Swagger JSON 解析逻辑
    UrlInfo.go                   # 接口 URL 信息及处理
```

## **使用步骤**
1. **准备 Swagger 文件**
   - 确保 `swagger.json` 中包含以下字段：
     ```json
     {
       "basePath": "/",
       "host": "daa-api.test.com.cn",
       "schemes": ["https"]
     }
     ```
   - 当前仅支持单协议（`http` 或 `https`）。

2. **放置文件**
   - 将所有需要扫描的 `swagger.json` 文件放入指定文件夹（工具会自动创建）。

3. **安装依赖**
   ```bash
   go mod tidy
   ```

4. **运行扫描**
   ```bash
   go run main.go
   ```

## **输出结果**
扫描完成后，会生成一个 CSV 文件，方便后续分析和处理。

## **环境要求**
- Go 1.18 及以上

## **贡献**
欢迎提交 Issue 和 PR，共同改进本项目。

## **License**
MIT

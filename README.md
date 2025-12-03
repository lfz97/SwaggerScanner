# swaggerScanner

一个用于解析和扫描 Swagger (OpenAPI) JSON 的 Go 项目。

## 项目结构

```
go.mod
main.go
myutils/
    SplitSliceEqualParts.go
swaggerParser/
    SwaggerJson.go
    swaggerParser.go
    UrlInfo.go
```

## 功能简介
- 解析 Swagger (OpenAPI) JSON 文件
- 提取接口信息、路径、方法等
- 支持对接口信息的扫描和处理
- 提供常用切片工具函数

## 主要模块说明
- `main.go`：程序入口，调用解析和扫描逻辑
- `swaggerParser/`：
  - `SwaggerJson.go`：定义 Swagger JSON 结构体
  - `swaggerParser.go`：实现 Swagger JSON 的解析逻辑
  - `UrlInfo.go`：接口 URL 信息结构体及相关处理
- `myutils/`：
  - `SplitSliceEqualParts.go`：切片分割工具函数

## 使用方法

1. 配置 `swagger.json`
   - 请在每个 `swagger.json` 中正确填写以下字段：
     - `basePath`：示例 `"/"`
     - `host`：示例 `"pmall-api.dominos.com.cn"`
     - `schemes`：当前仅支持写一个协议，`http` 或 `https`，示例：
       ```json
       "schemes": [
         "https"
       ]
       ```
   - 完整示例：
     ```json
     {
       "basePath": "/",
       "host": "pmall-api.dominos.com.cn",
       "schemes": [
         "https"
       ]
     }
     ```
2. 放置文件
   - 将所有需要扫描的 `swagger.json` 文件放入项目中的文件夹：`请将所有Swagger.json放入此文件夹`
3. 安装依赖
   ```powershell
   go mod tidy
   ```
4. 运行
   ```powershell
   go run main.go
   ```

## 依赖
- Go 1.18 及以上

## 贡献
欢迎提交 issue 和 PR 改进本项目。

## License
MIT

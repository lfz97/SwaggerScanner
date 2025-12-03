// 说明:
// 该文件负责从 Swagger(OpenAPI 2.0) JSON 中抽取接口的:
//  1. 完整请求路径 (前缀 Schemes://Host + BasePath + path)
//  2. 请求方法 (GET / POST 等)
//  3. Content-Type (优先 consumes[0], 默认 application/json)
//  4. 参数列表 (包含 query / path / body)
//
// 并将复杂的 body schema (仅处理 object 与 array) 转换为内部结构 UrlInfoParameterSchema。
// 注意: 目前未处理以下高级特性: $ref, allOf, enum, format, additionalProperties。
// 如果 Swagger 使用了这些特性, 当前解析将丢失更详细结构, 可后续扩展。
// 设计取舍: 将 body 参数保持为一个整体的 UrlInfoParameter, 不再拆分其子属性为多个参数, 便于后续统一构造请求体。
// 未来扩展建议: 添加 $ref 解析 + primitive 类型直接支持 + allOf 合并。
package swaggerParser

import (
	"encoding/json"
	"errors"
	"os"
)

// convertSwaggerSchemaToUrlInfoSchema
// 输入: Swagger 中 body 参数的 schema (Schema)
// 输出: 内部使用的 UrlInfoParameterSchema (只深入解析 object / array)
// 行为:
//   - object: 遍历 properties, 记录每个属性的类型与描述; 若属性本身是 array, 递归处理其 items
//   - array : 递归处理 items (items 可是 object/primitive); 当前未处理 items 为再嵌套 array 的复杂链条, 但逻辑可扩展
//   - 其它类型 (string/number/boolean/integer): 只保留 Type 字段
//
// 限制: 不处理 $ref / allOf, 因此遇到这些时会丢失具体属性信息
func convertSwaggerSchemaToUrlInfoSchema(s Schema) UrlInfoParameterSchema {
	urlInfoSchema := UrlInfoParameterSchema{ // 初始化内部 schema 结构
		Type: s.Type, // 保存原始类型
	}

	if s.Type == "object" { // 如果是对象类型，展开其属性
		urlInfoSchema.Properties = make(map[string]UrlInfoParameterSchemaProperty) // 为属性映射分配空间
		for propName, prop := range s.Properties {                                 // 遍历每个属性
			newProp := UrlInfoParameterSchemaProperty{ // 构建属性描述
				Type:        prop.Type,        // 属性类型
				Description: prop.Description, // 属性描述
			}
			if prop.Type == "array" { // 若属性本身是数组类型
				itemsSchema := convertSwaggerItemsToUrlInfoSchema(prop.Items) // 递归转换其 items
				newProp.Items = &itemsSchema                                  // 挂载到属性的 Items 指针
			}
			urlInfoSchema.Properties[propName] = newProp // 写入属性集合
		}
	} else if s.Type == "array" { // 如果顶层就是数组
		itemsSchema := convertSwaggerItemsToUrlInfoSchema(s.Items) // 转换数组元素类型
		urlInfoSchema.Items = &itemsSchema                         // 挂载元素 schema
	}

	return urlInfoSchema // 返回转换结果
}

// convertSwaggerItemsToUrlInfoSchema
// 作用: 专用于 array 的 items 或属性内部是 object 的情况
// 输入: Items (Swagger 中 array.items 或 object.properties 下的子结构)
// 输出: UrlInfoParameterSchema
// 行为:
//   - object: 展开其 properties
//   - 非 object: 只保留 Type
//     (若需要支持更深层级 array 嵌套或 $ref, 可在此处递归扩展)
func convertSwaggerItemsToUrlInfoSchema(i Items) UrlInfoParameterSchema {
	urlInfoSchema := UrlInfoParameterSchema{ // 初始化 items 对应的内部结构
		Type: i.Type, // 记录 items 的类型
	}
	if i.Type == "object" { // 若 items 是对象类型
		urlInfoSchema.Properties = make(map[string]UrlInfoParameterSchemaProperty) // 分配属性映射
		for propName, prop := range i.Properties {                                 // 遍历对象的属性
			urlInfoSchema.Properties[propName] = UrlInfoParameterSchemaProperty{ // 填充属性描述
				Type:        prop.Type,        // 属性类型
				Description: prop.Description, // 属性描述
			}
		}
	}
	return urlInfoSchema // 返回转换后的 items schema
}

// SwaggerParser
// 输入: swaggerPath (Swagger JSON 文件路径)
// 输出: *[]UrlInfo (接口抽取结果列表)
// 主流程:
//  1. 读取文件并反序列化为 SwaggerJson
//  2. 构建公共前缀 Prefix = scheme://host + basePath (若未声明 schemes 用 https)
//  3. 遍历 paths -> methods; 为每个 method 构建一个 UrlInfo
//  4. 遍历 parameters:
//     - body: 使用 convertSwaggerSchemaToUrlInfoSchema 转换其结构
//     - 其它 (query/path): 只记录基础 Type 方便后续填充参数
//
// 返回: 抽取出的 UrlInfo 列表指针, 供后续扫描函数使用
// 现有缺陷:
//   - 未处理 $ref/allOf 等; 若要提升准确度需在此增加引用展开与组合逻辑
//   - 未记录 required 列表, 后续可用于必填参数的测试覆盖
func SwaggerParser(swaggerPath string) (*[]UrlInfo, error) {
	jsonBytes, err := os.ReadFile(swaggerPath) // 读取 swagger 文件
	if err != nil {                            // 读取失败直接返回错误
		return nil, errors.New("Read swagger json failed:" + err.Error())
	}
	swagger := SwaggerJson{}                  // 初始化接收结构
	err = json.Unmarshal(jsonBytes, &swagger) // 反序列化 JSON
	if err != nil {                           // 反序列化失败
		return nil, errors.New("Unmarshal swagger json failed:" + err.Error())
	}

	Prefix := ""                  // 构造统一前缀 (协议 + 主机 + 基础路径)
	if len(swagger.Schemes) > 0 { // 优先使用声明的第一个 scheme
		Prefix = swagger.Schemes[0] + "://" + swagger.Host + swagger.BasePath
	} else { // 未声明 scheme 时默认 https
		Prefix = "https://" + swagger.Host + swagger.BasePath
	}

	finalUrlsInfo := []UrlInfo{} // 保存最终接口列表

	for path, methods := range swagger.Paths { // 遍历每个路径
		for method, info := range methods { // 遍历路径下的每个 HTTP 方法
			tmpUrlInfo := UrlInfo{}             // 初始化单个接口描述
			tmpUrlInfo.FullPath = Prefix + path // 拼接完整请求路径
			tmpUrlInfo.Method = method          // 保存方法
			tmpUrlInfo.Summary = info.Summary   // 保存摘要
			if len(info.Consumes) > 0 {         // 若声明了 consumes 列表
				tmpUrlInfo.ContentType = info.Consumes[0] // 使用第一个作为 Content-Type
			} else { // 未声明则默认 application/json
				tmpUrlInfo.ContentType = "application/json"
			}
			for _, param := range info.Parameters { // 遍历参数列表
				tmpParam := UrlInfoParameter{ // 初始化参数描述
					Name:        param.Name,        // 参数名
					In:          param.In,          // 参数位置(query / path / body)
					Description: param.Description, // 参数描述
				}
				if param.In == "body" { // body 参数结构化处理
					tmpParam.Schema = convertSwaggerSchemaToUrlInfoSchema(param.Schema) // 深度解析 object/array
				} else { // 非 body 参数只记录类型
					tmpParam.Type = param.Type
				}
				tmpUrlInfo.Parameters = append(tmpUrlInfo.Parameters, tmpParam) // 追加参数到接口定义
			}
			finalUrlsInfo = append(finalUrlsInfo, tmpUrlInfo) // 保存该接口
		}
	}
	return &finalUrlsInfo, nil // 返回所有接口信息

}

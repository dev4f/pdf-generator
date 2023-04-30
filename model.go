package main

type ExportRequest struct {
	Template string                 `json:"template"`
	FileName string                 `json:"file_name"`
	Data     map[string]interface{} `json:"data"`
}

type BaseResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

package cfgldr

type APIParamsMapper interface {
	APIParamsMap() *APIParamsMap
}

type APIParam interface {
	APIParam()
}

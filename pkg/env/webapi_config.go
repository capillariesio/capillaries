package env

type WebapiConfig struct {
	Port                     int    `json:"webapi_port" env:"CAPI_WEBAPI_PORT, overwrite"`
	AccessControlAllowOrigin string `json:"access_control_allow_origin" env:"CAPI_WEBAPI_ACCESS_CONTROL_ALLOW_ORIGIN, overwrite"`
}

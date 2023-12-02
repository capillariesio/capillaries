package env

type WebapiConfig struct {
	Port                     int    `json:"webapi_port"`
	AccessControlAllowOrigin string `json:"access_control_allow_origin"`
}

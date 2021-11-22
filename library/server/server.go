package serverConf

// ServerConfig server配置
type ServerConfig struct {
	IsDebug      bool   `json:"isDebug" toml:"isDebug" yaml:"isDebug"`
	AccessLog    bool   `json:"accessLog" toml:"accessLog" yaml:"accessLog"`
	Name         string `json:"name" toml:"name" yaml:"name"`
	Version      string `json:"version" toml:"version" yaml:"version"`
	Addr         string `json:"addr" toml:"addr" yaml:"addr"`
	Port         uint32 `json:"port" toml:"port" yaml:"port"`
	ReadTimeout  uint   `json:"readTimeout" toml:"readTimeout" yaml:"readTimeout" `
	WriteTimeout uint   `json:"writeTimeout" toml:"writeTimeout" yaml:"writeTimeout"`
	MaxHeaderMB  int    `json:"maxHeaderMB" toml:"maxHeaderMB" yaml:"maxHeaderMB"`
}

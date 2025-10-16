package config

import (
	"log"
	"strconv"
	"strings"
	"syscall"
)

var Cfg *config

type config struct {
	Enable_http     bool
	Uds             bool
	Uds_path        string
	Host            string
	Port            string
	Proxy           string
	Http_client_ver int
	Ipv6_only       bool
	Gluetun         struct {
		Gluetun_api            string
		Block_checker          bool
		Block_checker_cooldown int
	}
	Companion struct {
		Secret_key string
	}
}

func getenv(key string) string {
	// `YTPROXY_` as a prefix
	v, _ := syscall.Getenv("YTPROXY_" + key)
	return v
}

func getEnvBool(key string, def bool) bool {
	v := strings.ToLower(getenv(key))
	if v == "" {
		return def
	}
	return v == "true"
}

func getEnvString(key string, def string, tolower bool) string {
	var v string
	if tolower {
		v = strings.ToLower(getenv(key))
	}
	v = getenv(key)
	if v == "" {
		return def
	}
	return v
}

func getEnvInt(key string, def int) int {
	v := strings.ToLower(getenv(key))
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		log.Panicf("[FATAL] Failed to convert env variable '%s' to int", v)
	}
	return int(i)
}

func LoadConfig() {
	Cfg = &config{
		Enable_http: getEnvBool("ENABLE_HTTP", true),
		Uds:         getEnvBool("ENABLE_UDS", true),
		// I would use `/run/http3-proxy` here, but `/run` is not user writable
		// which is kinda anoying when developing.
		Uds_path:        getEnvString("UDS_PATH", "/tmp/http-ytproxy.sock", true),
		Host:            getEnvString("HOST", "0.0.0.0", true),
		Port:            getEnvString("PORT", "8080", true),
		Proxy:           getEnvString("PROXY", "", true),
		Http_client_ver: getEnvInt("HTTP_CLIENT_VER", 1),
		Ipv6_only:       getEnvBool("IPV6_ONLY", false),
		Gluetun: struct {
			Gluetun_api            string
			Block_checker          bool
			Block_checker_cooldown int
		}{
			Gluetun_api:            getEnvString("GLUETUN_API", "127.0.0.1:8000", true),
			Block_checker:          getEnvBool("BLOCK_CHECKER", true),
			Block_checker_cooldown: getEnvInt("BLOCK_CHECKER_COOLDOWN", 60),
		},
		Companion: struct{ Secret_key string }{
			Secret_key: getEnvString("SECRET_KEY", "", false),
		},
	}
	checkConfig()
}

func checkConfig() {
	if len(Cfg.Companion.Secret_key) > 16 {
		log.Fatalln("The value of 'companion.secret_key' (YTPROXY_SECRET_KEY) needs to be a size of 16 characters.")
	}
}

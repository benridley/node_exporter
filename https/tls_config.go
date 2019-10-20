// Copyright 2019 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package https allows the implementation of tls
package https

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"gopkg.in/yaml.v2"
)

type Config struct {
	TLSConfig TLSStruct `yaml:"tlsConfig"`
}

type TLSStruct struct {
	TLSCertPath string `yaml:"tlsCertPath"`
	TLSKeyPath  string `yaml:"tlsKeyPath"`
	ServerName  string `yaml:"serverName"`
	ClientAuth  string `yaml:"clientAuth"`
	ClientCAs   string `yaml:"clientCAs"`
}

func getTLSConfig(configPath string) (*tls.Config, error) {
	config, err := loadConfigFromYaml(configPath)
	if err != nil {
		log.Error("config failed to load from YAML: ", err)
		return nil, err
	}
	tlsc, err := configToTLSConfig(config)
	if err != nil {
		log.Error("failed to convert Config to tls.Config: ", err)
		return nil, err
	}
	return tlsc, nil
}

func loadConfigFromYaml(fileName string) (*Config, error) {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	c := &Config{}
	err = yaml.Unmarshal(content, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func configToTLSConfig(c *Config) (*tls.Config, error) {
	cfg := &tls.Config{}
	if len(c.TLSConfig.TLSCertPath) > 0 {
		cfg.GetCertificate = func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			cert, err := tls.LoadX509KeyPair(c.TLSConfig.TLSCertPath, c.TLSConfig.TLSKeyPath)
			if err != nil {
				return nil, err
			}
			return &cert, nil
		}
	}
	cfg.ServerName = c.TLSConfig.ServerName

	if len(c.TLSConfig.ClientCAs) > 0 {
		clientCAPool := x509.NewCertPool()
		clientCAFile, err := ioutil.ReadFile(c.TLSConfig.ClientCAs)
		if err != nil {
			return nil, err
		}
		clientCAPool.AppendCertsFromPEM(clientCAFile)
		cfg.ClientCAs = clientCAPool
	}
	if len(c.TLSConfig.ClientAuth) > 0 {
		switch s := (c.TLSConfig.ClientAuth); s {
		case "RequestClientCert":
			cfg.ClientAuth = tls.RequestClientCert
		case "RequireClientCert":
			cfg.ClientAuth = tls.RequireAnyClientCert
		case "VerifyClientCertIfGiven":
			cfg.ClientAuth = tls.VerifyClientCertIfGiven
		case "RequireAndVerifyClientCert":
			cfg.ClientAuth = tls.RequireAndVerifyClientCert
		default:
			return nil, errors.New("Invalid string provided to ClientAuth")
		}
	}
	return cfg, nil
}

// When the listen function is called if the tlsConfigPath is an empty string an HTTP server is started
// If the tlsConfigPath is a valid config file then an HTTPS server will be started
// The listen function also sets the GetConfigForClient method of the HTTPS server so that the config and certs are reloaded on new connections
func Listen(server *http.Server, tlsConfigPath string) error {
	if len(tlsConfigPath) > 0 {
		var err error
		server.TLSConfig, err = getTLSConfig(tlsConfigPath)
		if err != nil {
			return err
		}
		server.TLSConfig.GetConfigForClient = func(*tls.ClientHelloInfo) (*tls.Config, error) {
			return getTLSConfig(tlsConfigPath)
		}
		return server.ListenAndServeTLS("", "")
	} else {
		return server.ListenAndServe()
	}
}

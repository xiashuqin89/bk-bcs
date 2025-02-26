/*
Tencent is pleased to support the open source community by making Basic Service Configuration Platform available.
Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
Licensed under the MIT License (the "License"); you may not use this file except
in compliance with the License. You may obtain a copy of the License at
http://opensource.org/licenses/MIT
Unless required by applicable law or agreed to in writing, software distributed under
the License is distributed on an "as IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
either express or implied. See the License for the specific language governing permissions and
limitations under the License.
*/

package cc

import (
	"errors"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

// LoadSettings load service's configuration
func LoadSettings(sys *SysOption) error {
	if len(sys.ConfigFile) == 0 {
		return errors.New("service's configuration file path is not configured")
	}

	// configure file is configured, then load configuration from file.
	s, err := loadFromFile(sys.ConfigFile)
	if err != nil {
		return err
	}

	if err = s.trySetFlagBindIP(sys.BindIP); err != nil {
		return err
	}

	// s the default value if user not configured.
	s.trySetDefault()

	if err := s.Validate(); err != nil {
		return err
	}

	initRuntime(s)

	return nil
}

// loadFromFile load service's configuration from local config file.
func loadFromFile(filename string) (Setting, error) {
	if len(filename) == 0 {
		return nil, errors.New("file name is not set")
	}

	var s Setting
	switch ServiceName() {
	case APIServerName:
		s = new(ApiServerSetting)
	case AuthServerName:
		s = new(AuthServerSetting)
	case CacheServiceName:
		s = new(CacheServiceSetting)
	case ConfigServerName:
		s = new(ConfigServerSetting)
	case DataServiceName:
		s = new(DataServiceSetting)
	case FeedServerName:
		s = new(FeedServerSetting)
	case SidecarName:
		s = new(SidecarSetting)
	default:
		return nil, fmt.Errorf("unknown %s service name", ServiceName())
	}

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("load Setting from file: %s failed, err: %v", filename, err)
	}

	if err := yaml.Unmarshal(file, s); err != nil {
		return nil, fmt.Errorf("unmarshal Setting yaml from file: %s failed, err: %v", filename, err)
	}

	return s, nil
}

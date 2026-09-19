package config

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

//go:embed default.yaml
var defaultYAML []byte

type Config struct {
	Hosts    []Host    `mapstructure:"hosts"`
	Commands []Command `mapstructure:"commands"`
}
type Command struct {
	Label   string `mapstructure:"label"`
	Command string `mapstructure:"command"`
}
type Forward struct {
	Name       string `mapstructure:"name"`
	Type       string `mapstructure:"type"`
	LocalAddr  string `mapstructure:"local_addr"`
	RemoteAddr string `mapstructure:"remote_addr"`
	Enabled    bool   `mapstructure:"enabled"`
}
type Host struct {
	Name                  string    `mapstructure:"name"`
	Hostname              string    `mapstructure:"hostname"`
	User                  string    `mapstructure:"user"`
	Port                  int       `mapstructure:"port"`
	IdentityFile          string    `mapstructure:"identity_file"`
	Password              string    `mapstructure:"password"`
	KeyDeployed           bool      `mapstructure:"key_deployed"`
	AutoGenerateKey       bool      `mapstructure:"auto_generate_key"`
	Tags                  []string  `mapstructure:"tags"`
	ProxyJump             string    `mapstructure:"proxy_jump"`
	Notes                 string    `mapstructure:"notes"`
	Pinned                bool      `mapstructure:"pinned"`
	UseAgent              bool      `mapstructure:"use_agent"`
	ConnectTimeoutSeconds int       `mapstructure:"connect_timeout_seconds"`
	KeepAliveSeconds      int       `mapstructure:"keep_alive_seconds"`
	Forwards              []Forward `mapstructure:"forwards"`
}

var C Config

func Init(cfgFile string) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		configDir := filepath.Join(home, ".config", "vecna")
		_ = os.MkdirAll(configDir, 0755)
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}
	viper.SetDefault("hosts", []Host{})
	viper.SetDefault("commands", []Command{})
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			_ = viper.Unmarshal(&C)
			applyDefaultCommandsIfEmpty()
			return
		}
		if cfgFile == "" {
			if rerr := viper.ReadConfig(bytes.NewReader(defaultYAML)); rerr == nil {
				_ = viper.Unmarshal(&C)
				home, _ := os.UserHomeDir()
				configPath := filepath.Join(home, ".config", "vecna", "config.yaml")
				if err := viper.WriteConfigAs(configPath); err == nil {
					viper.SetConfigFile(configPath)
				}
			}
			applyDefaultCommandsIfEmpty()
			return
		}
	}
	_ = viper.Unmarshal(&C)
	applyDefaultCommandsIfEmpty()
}
func isOldDefaultCommands() bool {
	if len(C.Commands) != 2 {
		return false
	}
	labels := []string{C.Commands[0].Label, C.Commands[1].Label}
	old := map[string]bool{"disk usage": true, "memory": true}
	return old[labels[0]] && old[labels[1]]
}
func applyDefaultCommandsIfEmpty() {
	if len(C.Commands) > 0 && !isOldDefaultCommands() {
		return
	}
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewReader(defaultYAML)); err != nil {
		return
	}
	var def struct {
		Commands []Command `mapstructure:"commands"`
	}
	if err := v.Unmarshal(&def); err != nil || len(def.Commands) == 0 {
		return
	}
	C.Commands = def.Commands
	viper.Set("commands", def.Commands)
	_ = Save()
}
func Save() error {
	err := viper.WriteConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		home, _ := os.UserHomeDir()
		return viper.WriteConfigAs(filepath.Join(home, ".config", "vecna", "config.yaml"))
	}
	return err
}
func GetHosts() []Host { return C.Hosts }
func GetHostByName(name string) (Host, int, bool) {
	for i, h := range C.Hosts {
		if h.Name == name {
			return h, i, true
		}
	}
	return Host{}, -1, false
}
func GetCommands() []Command {
	if C.Commands == nil {
		return []Command{}
	}
	return C.Commands
}
func AddCommand(cmd Command) {
	C.Commands = append(C.Commands, cmd)
	viper.Set("commands", C.Commands)
	_ = Save()
}
func RemoveCommand(index int) {
	if index < 0 || index >= len(C.Commands) {
		return
	}
	C.Commands = append(C.Commands[:index], C.Commands[index+1:]...)
	viper.Set("commands", C.Commands)
	_ = Save()
}
func UpdateCommand(index int, cmd Command) {
	if index < 0 || index >= len(C.Commands) {
		return
	}
	C.Commands[index] = cmd
	viper.Set("commands", C.Commands)
	_ = Save()
}
func AddHost(h Host) { C.Hosts = append(C.Hosts, h); viper.Set("hosts", C.Hosts); _ = Save() }
func RemoveHost(index int) {
	if index < 0 || index >= len(C.Hosts) {
		return
	}
	C.Hosts = append(C.Hosts[:index], C.Hosts[index+1:]...)
	viper.Set("hosts", C.Hosts)
	_ = Save()
}
func UpdateHost(index int, host Host) {
	if index < 0 || index >= len(C.Hosts) {
		return
	}
	C.Hosts[index] = host
	viper.Set("hosts", C.Hosts)
	_ = Save()
}

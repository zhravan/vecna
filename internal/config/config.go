package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Hosts    []Host    `mapstructure:"hosts"`
	Commands []Command `mapstructure:"commands"`
}

type Command struct {
	Label   string `mapstructure:"label"`
	Command string `mapstructure:"command"`
}

type Host struct {
	Name            string   `mapstructure:"name"`
	Hostname        string   `mapstructure:"hostname"`
	User            string   `mapstructure:"user"`
	Port            int      `mapstructure:"port"`
	IdentityFile    string   `mapstructure:"identity_file"`
	Password        string   `mapstructure:"password"`          // encrypted
	KeyDeployed     bool     `mapstructure:"key_deployed"`      // true if key already deployed
	AutoGenerateKey bool     `mapstructure:"auto_generate_key"` // true if should auto-generate key
	Tags            []string `mapstructure:"tags"`
	ProxyJump       string   `mapstructure:"proxy_jump"` // name of another vecna host to use as jump/bastion
}

var C Config

func Init(cfgFile string) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		configDir := filepath.Join(home, ".config", "vecna")
		os.MkdirAll(configDir, 0755)

		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetDefault("hosts", []Host{})
	viper.SetDefault("commands", []Command{})

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return
		}
	}

	viper.Unmarshal(&C)
}

func Save() error {
	err := viper.WriteConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		home, _ := os.UserHomeDir()
		configPath := filepath.Join(home, ".config", "vecna", "config.yaml")
		return viper.WriteConfigAs(configPath)
	}
	return err
}

func GetHosts() []Host {
	return C.Hosts
}

// GetHostByName returns the host with the given name and its index, or (-1, false) if not found.
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
	Save()
}

func RemoveCommand(index int) {
	if index < 0 || index >= len(C.Commands) {
		return
	}
	C.Commands = append(C.Commands[:index], C.Commands[index+1:]...)
	viper.Set("commands", C.Commands)
	Save()
}

func AddHost(h Host) {
	C.Hosts = append(C.Hosts, h)
	viper.Set("hosts", C.Hosts)
}

func RemoveHost(index int) {
	if index < 0 || index >= len(C.Hosts) {
		return
	}
	C.Hosts = append(C.Hosts[:index], C.Hosts[index+1:]...)
	viper.Set("hosts", C.Hosts)
	Save()
}

func UpdateHost(index int, host Host) {
	if index < 0 || index >= len(C.Hosts) {
		return
	}
	C.Hosts[index] = host
	viper.Set("hosts", C.Hosts)
	Save()
}

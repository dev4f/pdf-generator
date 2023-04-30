package main

type Server struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

type Minio struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	UseSSL    bool   `mapstructure:"use_ssl"`
	Bucket    string `mapstructure:"bucket"`
	Path      string `mapstructure:"path"`
}

type StorageType int

const (
	StorageTypeInMemory StorageType = 0
	StorageTypeMinio    StorageType = 1
)

type Storage struct {
	Type  StorageType `mapstructure:"type"`
	Minio Minio       `mapstructure:"minio"`
}

type Config struct {
	Server  Server  `mapstructure:"server"`
	Storage Storage `mapstructure:"storage"`
	Test    string  `mapstructure:"TEST"`
}

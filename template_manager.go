package main

import (
	"bytes"
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"log"
)

type TemplateManager interface {
	GetByName(templateName string) (string, error)
	GetAll() (map[string]string, error)
	Add(templateName string, templateContent string) error
}

// InMemoryTemplateManager is a template manager that stores templates in memory
type InMemoryTemplateManager struct {
	templates map[string]string
}

func NewInMemoryTemplateManager() *InMemoryTemplateManager {
	return &InMemoryTemplateManager{templates: make(map[string]string)}
}

func (loader *InMemoryTemplateManager) GetByName(templateName string) (string, error) {
	return loader.templates[templateName], nil
}

func (loader *InMemoryTemplateManager) GetAll() (map[string]string, error) {
	return loader.templates, nil
}

func (loader *InMemoryTemplateManager) Add(templateName string, templateContent string) error {
	loader.templates[templateName] = templateContent
	return nil
}

// MinioTemplateManager is a template manager that stores templates in minio
type MinioTemplateManager struct {
	client *minio.Client
	bucket string
	path   string
}

func NewMinioTemplateManager(
	endpoint string,
	accessKey string,
	secretKey string,
	useSSL bool,
	bucket string,
	path string,
) (*MinioTemplateManager, error) {

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}

	return &MinioTemplateManager{
		client: client,
		bucket: bucket,
		path:   path,
	}, nil
}

func (loader *MinioTemplateManager) GetByName(templateName string) (string, error) {
	object, err := loader.client.GetObject(
		context.Background(),
		loader.bucket,
		loader.path+"/"+templateName,
		minio.GetObjectOptions{},
	)
	if err != nil {
		log.Fatalln(err)
		return "", err
	}
	defer object.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(object)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (loader *MinioTemplateManager) GetAll() (map[string]string, error) {
	objects := make(map[string]string)
	for object := range loader.client.ListObjects(
		context.Background(),
		loader.bucket,
		minio.ListObjectsOptions{
			Prefix:    loader.path,
			Recursive: true,
		},
	) {
		if object.Err != nil {
			log.Fatalln(object.Err)
			return nil, object.Err
		}
		objects[object.Key] = object.Key
	}
	return objects, nil
}

func (loader *MinioTemplateManager) Add(templateName string, templateContent string) error {
	_, err := loader.client.PutObject(
		context.Background(),
		loader.bucket,
		loader.path+"/"+templateName,
		bytes.NewReader([]byte(templateContent)),
		-1,
		minio.PutObjectOptions{},
	)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	return nil
}

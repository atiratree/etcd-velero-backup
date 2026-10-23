package plugin

import (
	"errors"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type FileObjectStore struct {
	log  logrus.FieldLogger
	root string
}

func NewFileObjectStore(logger logrus.FieldLogger) (interface{}, error) {
	return &FileObjectStore{log: logger}, nil
}

func (f *FileObjectStore) Init(config map[string]string) error {
	return nil
}

func (f *FileObjectStore) PutObject(bucket, key string, body io.Reader) error {
	return nil
}

func (f *FileObjectStore) ObjectExists(bucket, key string) (bool, error) {
	return false, nil
}

func (f *FileObjectStore) GetObject(bucket, key string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (f *FileObjectStore) ListCommonPrefixes(bucket, prefix, delimiter string) ([]string, error) {
	return nil, nil
}

func (f *FileObjectStore) ListObjects(bucket, prefix string) ([]string, error) {
	return nil, nil
}

func (f *FileObjectStore) DeleteObject(bucket, key string) error {
	return nil
}

func (f *FileObjectStore) CreateSignedURL(bucket, key string, ttl time.Duration) (string, error) {
	return "", errors.New("CreateSignedURL is not supported for the local file object store")
}

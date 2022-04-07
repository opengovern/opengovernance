package test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"path"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client/metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

type MockedS3Client struct {
	s3iface.S3API
	mu    sync.Mutex
	files map[string][]byte
	tags  map[string]map[string]string
}

func NewMockedS3Client() *MockedS3Client {
	return &MockedS3Client{
		files: map[string][]byte{},
		tags:  map[string]map[string]string{},
	}
}

func (m *MockedS3Client) HeadBucket(in *s3.HeadBucketInput) (*s3.HeadBucketOutput, error) {
	return &s3.HeadBucketOutput{}, nil
}

func (m *MockedS3Client) PutObject(in *s3.PutObjectInput) (out *s3.PutObjectOutput, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := path.Join(*in.Bucket, *in.Key)
	m.files[key], err = ioutil.ReadAll(in.Body)

	m.tags[key] = map[string]string{}
	if in.Tagging != nil {
		u, err := url.Parse("/?" + *in.Tagging)
		if err != nil {
			panic(fmt.Errorf("Unable to parse AWS S3 Tagging string %q: %w", *in.Tagging, err))
		}

		q := u.Query()
		for k := range q {
			m.tags[key][k] = q.Get(k)
		}
	}

	return &s3.PutObjectOutput{}, nil
}

func (m *MockedS3Client) GetObject(in *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := path.Join(*in.Bucket, *in.Key)
	if _, ok := m.files[key]; !ok {
		return nil, errors.New("Key does not exist")
	}

	return &s3.GetObjectOutput{
		Body: ioutil.NopCloser(bytes.NewReader(m.files[key])),
	}, nil
}

func (m *MockedS3Client) GetObjectRequest(in *s3.GetObjectInput) (*request.Request, *s3.GetObjectOutput) {
	m.mu.Lock()
	defer m.mu.Unlock()

	req := request.New(aws.Config{}, metadata.ClientInfo{Endpoint: "https://test.com"}, request.Handlers{}, nil, &request.Operation{}, nil, nil)
	return req, nil
}

func (m *MockedS3Client) DeleteObject(in *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.files[*in.Key]; ok {
		delete(m.files, *in.Key)
	}
	if _, ok := m.tags[*in.Key]; ok {
		delete(m.tags, *in.Key)
	}
	return &s3.DeleteObjectOutput{}, nil
}

package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	bucket = os.Getenv("ARTIFACTCACHE_BUCKET")
	client = s3.New(session.Must(session.NewSession()))
	memory = sync.Map{}
)

type UploadMemory struct {
	ObjectKey
	UploadId string
	ETags    []string
}

type ObjectKey struct {
	Audience string
	Scope    string
	Key      string
	Version  string
}

func (k ObjectKey) String() string {
	audience16 := md5.Sum([]byte(k.Audience))
	audience64 := base64.RawURLEncoding.EncodeToString(audience16[:])

	scope16 := md5.Sum([]byte(k.Scope))
	scope64 := base64.RawURLEncoding.EncodeToString(scope16[:])

	key16 := md5.Sum([]byte(k.Key))
	key64 := base64.RawURLEncoding.EncodeToString(key16[:])

	version16 := md5.Sum([]byte(k.Version))
	version64 := base64.RawURLEncoding.EncodeToString(version16[:])

	return fmt.Sprintf("%s/%s/%s/%s", audience64, scope64, key64, version64)
}

func headObject(objectKey ObjectKey) (time.Time, error) {
	req := s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey.String()),
	}

	res, err := client.HeadObject(&req)
	if err != nil {
		return time.Time{}, err
	}

	return aws.TimeValue(res.LastModified), nil
}

func presignGetObjectRequest(objectKey ObjectKey) (string, error) {
	req, _ := client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey.String()),
	})

	return req.Presign(time.Minute)
}

func createMultipartUpload(objectKey ObjectKey) (int, error) {
	req := s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey.String()),
	}

	res, err := client.CreateMultipartUpload(&req)
	if err != nil {
		return -1, err
	}

	cacheId := int(rand.Int31())

	memory.Store(cacheId, UploadMemory{
		ObjectKey: objectKey,
		UploadId:  aws.StringValue(res.UploadId),
		ETags:     []string{},
	})

	return cacheId, nil
}

func uploadPart(cacheId int, body io.Reader) error {
	value, loaded := memory.LoadAndDelete(cacheId)
	if !loaded {
		return fmt.Errorf("no memory of cache id %d", cacheId)
	}
	uploadMemory := value.(UploadMemory)

	// https://github.com/aws/aws-sdk-go/issues/142#issuecomment-257558022
	// https://github.com/aws/aws-sdk-go/issues/3063#issuecomment-571279246
	unchunked, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	req := s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(uploadMemory.ObjectKey.String()),
		UploadId:   aws.String(uploadMemory.UploadId),
		PartNumber: aws.Int64(int64(len(uploadMemory.ETags) + 1)),
		Body:       bytes.NewReader(unchunked),
	}
	log.Print(req)

	res, err := client.UploadPart(&req)
	if err != nil {
		return err
	}

	uploadMemory.ETags = append(uploadMemory.ETags, aws.StringValue(res.ETag))
	memory.Store(cacheId, uploadMemory)

	return nil
}

func completeMultipartUpload(cacheId int) error {
	value, loaded := memory.LoadAndDelete(cacheId)
	if !loaded {
		return fmt.Errorf("no memory of cache id %d", cacheId)
	}
	uploadMemory := value.(UploadMemory)

	completedMultipartUpload := &s3.CompletedMultipartUpload{}
	for i, eTag := range uploadMemory.ETags {
		completedMultipartUpload.Parts = append(completedMultipartUpload.Parts, &s3.CompletedPart{
			ETag:       aws.String(eTag),
			PartNumber: aws.Int64(int64(i + 1)),
		})
	}

	req := s3.CompleteMultipartUploadInput{
		Bucket:          aws.String(bucket),
		Key:             aws.String(uploadMemory.ObjectKey.String()),
		UploadId:        aws.String(uploadMemory.UploadId),
		MultipartUpload: completedMultipartUpload,
	}

	if _, err := client.CompleteMultipartUpload(&req); err != nil {
		return err
	}

	return nil
}

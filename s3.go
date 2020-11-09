package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
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

type ObjectKey struct {
	Audience string
	Scope    string
	Key      string
	Version  string
}

func (k ObjectKey) String() string {
	prefix16 := md5.Sum([]byte(k.Audience + k.Scope + k.Version))
	prefix64 := base64.RawURLEncoding.EncodeToString(prefix16[:])

	return fmt.Sprintf("%s/%s", prefix64, k.Key)
}

type ObjectHead struct {
	Key          string
	LastModified time.Time
}

func lookupObject(objectKey ObjectKey) (*ObjectHead, error) {
	req := s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(objectKey.String()),
	}

	res, err := client.ListObjects(&req)
	if err != nil {
		return nil, err
	}

	if len(res.Contents) == 0 {
		return nil, nil
	}

	var head ObjectHead
	for _, obj := range res.Contents {
		k := aws.StringValue(obj.Key)
		lm := aws.TimeValue(obj.LastModified)

		if eq := k == objectKey.Key; eq || lm.Before(head.LastModified) {
			head.Key = k
			head.LastModified = lm

			if eq {
				break
			}
		}
	}

	return &head, nil
}

func presignGetObjectRequest(objectHead ObjectHead) (string, error) {
	req, _ := client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectHead.Key),
	})

	return req.Presign(time.Minute)
}

type UploadMemory struct {
	ObjectKey
	UploadId string
	ETags    []string
	Cond     *sync.Cond
	Next     int
	Part     int
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

	memory.Store(cacheId, &UploadMemory{
		ObjectKey: objectKey,
		UploadId:  aws.StringValue(res.UploadId),
		ETags:     []string{},
		Cond:      sync.NewCond(&sync.Mutex{}),
		Next:      0,
		Part:      0,
	})

	return cacheId, nil
}

func uploadPart(cacheId int, rangeStart int, rangeEnd int, body io.Reader) error {
	value, loaded := memory.Load(cacheId)
	if !loaded {
		return fmt.Errorf("no memory of cache id %d", cacheId)
	}
	uploadMemory := value.(*UploadMemory)
	if uploadMemory == nil {
		return fmt.Errorf("no memory of cache id %d", cacheId)
	}

	uploadMemory.Cond.L.Lock()
	for uploadMemory.Next != rangeStart {
		uploadMemory.Cond.Wait()
	}
	uploadMemory.Part++
	req := s3.UploadPartInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(uploadMemory.ObjectKey.String()),
		UploadId:   aws.String(uploadMemory.UploadId),
		PartNumber: aws.Int64(int64(uploadMemory.Part)),
	}
	uploadMemory.Cond.L.Unlock()

	// https://github.com/aws/aws-sdk-go/issues/142#issuecomment-257558022
	// https://github.com/aws/aws-sdk-go/issues/3063#issuecomment-571279246
	unchunked, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	req.Body = bytes.NewReader(unchunked)

	res, err := client.UploadPart(&req)
	if err != nil {
		return err
	}

	uploadMemory.Cond.L.Lock()
	uploadMemory.ETags = append(uploadMemory.ETags, aws.StringValue(res.ETag))
	uploadMemory.Next = rangeEnd + 1
	uploadMemory.Cond.Broadcast()
	uploadMemory.Cond.L.Unlock()

	return nil
}

func completeMultipartUpload(cacheId int) error {
	value, loaded := memory.LoadAndDelete(cacheId)
	if !loaded {
		return fmt.Errorf("no memory of cache id %d", cacheId)
	}
	uploadMemory := value.(*UploadMemory)

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

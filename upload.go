package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/julienschmidt/httprouter"
)

const (
	indexRangeStart = 3
	indexRangeEnd   = 4
	groupLength     = 7
)

var (
	regexContentRange = regexp.MustCompile(`(?P<Unit>\w+) ((?P<RangeStart>\d+)-(?P<RangeEnd>\d+)|\*)/((?P<Size>\d+)|\*)`)
)

type ReserveCacheRequest struct {
	Key     string `json:"key,omitempty"`
	Version string `json:"version,omitempty"`
}

type ReserveCacheResponse struct {
	CacheId int `json:"cacheId,omitempty"`
}

func postReserveCache(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var req ReserveCacheRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, err := createMultipartUpload(ObjectKey{
		Audience: ps.ByName("audience"),
		Scope:    ps.ByName("scope"),
		Key:      req.Key,
		Version:  req.Version,
	})
	if err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)

	res := ReserveCacheResponse{
		CacheId: id,
	}
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func patchUploadCache(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	cacheId, err := strconv.Atoi(ps.ByName("cacheId"))
	if err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	contentRange := regexContentRange.FindStringSubmatch(r.Header.Get("Content-Range"))
	if contentRange == nil || len(contentRange) != groupLength {
		err := errors.New("contentRange == nil || len(contentRange) != groupLength")
		log.Print(err)

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rangeStart, err := strconv.Atoi(contentRange[indexRangeStart])
	if err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rangeEnd, err := strconv.Atoi(contentRange[indexRangeEnd])
	if err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := uploadPart(cacheId, rangeStart, rangeEnd, r.Body); err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func postCommitCache(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	cacheId, err := strconv.Atoi(ps.ByName("cacheId"))
	if err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := completeMultipartUpload(cacheId); err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type AssumeRoleUploadResponse struct {
	ObjectS3URI     string
	AccessKeyId     string
	SecretAccessKey string
	SessionToken    string
}

func getAssumeRoleUpload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	objectKey := ObjectKey{
		Audience: ps.ByName("audience"),
		Scope:    ps.ByName("scope"),
		Key:      ps.ByName("key"),
		Version:  "",
	}
	key := objectKey.String()

	sessionName := objectKey.Audience[strings.LastIndex(objectKey.Audience, ":")+1:]
	if len(sessionName) > 64 {
		sessionName = sessionName[:64]
	}

	credentials, err := assumeRole(sessionName, uploadPolicy(key))
	if err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := AssumeRoleUploadResponse{
		ObjectS3URI:     fmt.Sprintf("s3://%s/%s", BucketName, key),
		AccessKeyId:     aws.StringValue(credentials.AccessKeyId),
		SecretAccessKey: aws.StringValue(credentials.SecretAccessKey),
		SessionToken:    aws.StringValue(credentials.SessionToken),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

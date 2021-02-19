package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/julienschmidt/httprouter"
)

type CacheEntry struct {
	ArchiveLocation string `json:"archiveLocation,omitempty"`
	CacheKey        string `json:"cacheKey,omitempty"`
	CacheVersion    string `json:"cacheVersion,omitempty"`
	CreationTime    string `json:"creationTime,omitempty"`
	Scope           string `json:"scope,omitempty"`
}

func getCacheEntry(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	audience := ps.ByName("audience")
	scope := ps.ByName("scope")
	version := r.URL.Query().Get("version")

	keys := strings.Split(r.URL.Query().Get("keys"), ",")
	key := keys[0]
	restoreKeys := keys[1:]

	objectKey, objectHead, err := lookupObject(audience, scope, key, version, restoreKeys)
	if err != nil {
		log.Print(err)
	}
	if objectHead == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	url, err := presignGetObjectRequest(*objectHead)
	if err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	cacheEntry := CacheEntry{
		ArchiveLocation: url,
		CacheKey:        objectKey,
		CacheVersion:    version,
		CreationTime:    objectHead.LastModified.Format(time.RFC3339),
		Scope:           scope,
	}
	if err := json.NewEncoder(w).Encode(cacheEntry); err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type AssumeRoleDownloadResponse struct {
	CacheHit        bool
	ObjectS3URI     string
	AccessKeyId     string
	SecretAccessKey string
	SessionToken    string
}

func getAssumeRoleDownload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	audience := ps.ByName("audience")
	scope := ps.ByName("scope")
	key := ps.ByName("key")

	var restoreKeys []string
	if queryParam := r.URL.Query().Get("restoreKeys"); queryParam != "" {
		restoreKeys = strings.Split(queryParam, ",")
	}

	lookupKey, objectHead, err := lookupObject(audience, scope, key, "", restoreKeys)
	if err != nil {
		log.Print(err)
	}
	if objectHead == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	sessionName := audience[strings.LastIndex(audience, ":")+1:]
	if len(sessionName) > 64 {
		sessionName = sessionName[:64]
	}

	credentials, err := assumeRole(sessionName, downloadPolicy(objectHead.Key))
	if err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := AssumeRoleDownloadResponse{
		CacheHit:        lookupKey == key,
		ObjectS3URI:     fmt.Sprintf("s3://%s/%s", BucketName, objectHead.Key),
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

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

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
	objectKey := ObjectKey{
		Audience: ps.ByName("audience"),
		Scope:    ps.ByName("scope"),
		Version:  r.URL.Query().Get("version"),
	}

	var timestamp time.Time
	for _, key := range strings.Split(r.URL.Query().Get("keys"), ",") {
		objectKey.Key = key

		t, err := headObject(objectKey)
		if err != nil {
			log.Print(err)
			continue
		}

		timestamp = t
		break
	}

	if timestamp == (time.Time{}) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	url, err := presignGetObjectRequest(objectKey)
	if err != nil {
		log.Print(err)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	cacheEntry := CacheEntry{
		ArchiveLocation: url,
		CacheKey:        objectKey.Key,
		CacheVersion:    objectKey.Version,
		CreationTime:    timestamp.Format(time.RFC3339),
		Scope:           objectKey.Scope,
	}
	if err := json.NewEncoder(w).Encode(cacheEntry); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

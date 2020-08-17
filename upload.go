package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
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

	if err := uploadPart(cacheId, r.Body); err != nil {
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

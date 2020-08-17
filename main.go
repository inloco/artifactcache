package main

import (
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func main() {
	router := httprouter.New()
	router.GET("/:actionsCacheURL/_apis/artifactcache/cache", logged(authenticated(getCacheEntry)))
	router.POST("/:actionsCacheURL/_apis/artifactcache/caches", logged(authenticated(postReserveCache)))
	router.PATCH("/:actionsCacheURL/_apis/artifactcache/caches/:cacheId", logged(authenticated(patchUploadCache)))
	router.POST("/:actionsCacheURL/_apis/artifactcache/caches/:cacheId", logged(authenticated(postCommitCache)))

	log.Fatal(http.ListenAndServe(":8080", router))
}

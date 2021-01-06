package main

import (
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func main() {
	router := httprouter.New()

	// ADO Adapter
	router.GET("/:actionsCacheURL/_apis/artifactcache/cache", logged(authenticated(getCacheEntry)))
	router.POST("/:actionsCacheURL/_apis/artifactcache/caches", logged(authenticated(postReserveCache)))
	router.PATCH("/:actionsCacheURL/_apis/artifactcache/caches/:cacheId", logged(authenticated(patchUploadCache)))
	router.POST("/:actionsCacheURL/_apis/artifactcache/caches/:cacheId", logged(authenticated(postCommitCache)))

	// S3 Authenticator
	router.GET("/:actionsCacheURL/assumeRole/:key/upload", logged(authenticated(getAssumeRoleUpload)))
	router.GET("/:actionsCacheURL/assumeRole/:key/download", logged(authenticated(getAssumeRoleDownload)))

	log.Panic(http.ListenAndServe(":8080", router))
}

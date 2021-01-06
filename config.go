package main

import "os"

var (
	BucketName = os.Getenv("ARTIFACTCACHE_BUCKET_NAME")
	RoleARN    = os.Getenv("ARTIFACTCACHE_ROLE_ARN")
)

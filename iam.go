package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type Statement struct {
	Effect   string
	Action   []string
	Resource []string
}

type Policy struct {
	Version   string
	Statement []Statement
}

func (p Policy) String() string {
	bytes, err := json.Marshal(p)
	if err != nil {
		log.Panic(err)
	}

	return string(bytes)
}

func downloadPolicy(object string) Policy {
	return Policy{
		Version: "2012-10-17",
		Statement: []Statement{
			Statement{
				Effect: "Allow",
				Action: []string{
					"s3:GetObject",
				},
				Resource: []string{
					fmt.Sprintf("arn:aws:s3:::%s/%s", BucketName, object),
				},
			},
		},
	}
}

func uploadPolicy(object string) Policy {
	return Policy{
		Version: "2012-10-17",
		Statement: []Statement{
			Statement{
				Effect: "Allow",
				Action: []string{
					"s3:PutObject",
					"s3:AbortMultipartUpload",
					"s3:ListMultipartUploadParts",
				},
				Resource: []string{
					fmt.Sprintf("arn:aws:s3:::%s/%s", BucketName, object),
				},
			},
		},
	}
}

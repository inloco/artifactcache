package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

var (
	stsClient = sts.New(session.Must(session.NewSession()))
)

func assumeRole(sessionName string, policy Policy) (*sts.Credentials, error) {
	out, err := stsClient.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         aws.String(RoleARN),
		RoleSessionName: aws.String(sessionName),
		Policy:          aws.String(policy.String()),
		DurationSeconds: aws.Int64(900),
	})
	if err != nil {
		return nil, err
	}

	return out.Credentials, nil
}

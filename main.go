package main

import (
	_ "github.com/Nerzal/gocloak/v13"
	_ "github.com/aws/aws-sdk-go-v2/aws"
	_ "github.com/aws/aws-sdk-go-v2/service/s3"
	_ "github.com/aws/aws-sdk-go-v2/service/s3/types"
	_ "github.com/aws/smithy-go"
	_ "github.com/caarlos0/env/v6"
	_ "github.com/go-chi/chi/v5"
	_ "github.com/go-chi/cors"
	_ "github.com/go-chi/telemetry"
	_ "github.com/golang-jwt/jwt/v5"
	_ "github.com/stretchr/testify/assert"
	_ "github.com/swaggo/http-swagger"
	_ "github.com/swaggo/swag"
	_ "github.com/twmb/franz-go/pkg/kgo"
	_ "google.golang.org/grpc"
	_ "google.golang.org/protobuf/types/known/emptypb"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/api/rbac/v1"
	_ "k8s.io/apimachinery/pkg/api/errors"
	_ "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/rest"
)

func main() {}

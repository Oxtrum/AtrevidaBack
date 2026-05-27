package main

import (
	"context"
	"encoding/json"
	"os"

	"atrevida-agenda-api/app"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
)

//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -o docs

// @title Atrevida Fit - Agenda API
// @version v0.1
// @description API REST para reservas, servicios, combos y administracion de Atrevida Fit.
// @BasePath /
// @schemes http https
func main() {
	routerAtrevida, err := app.Build()
	if err != nil {
		panic(err)
	}

	println(app.Name + "(" + app.Version + ")\n" + "Running")

	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		adapterV1 := ginadapter.New(routerAtrevida)
		adapterV2 := ginadapter.NewV2(routerAtrevida)

		lambda.Start(func(ctx context.Context, payload json.RawMessage) (any, error) {
			var eventVersion struct {
				Version string `json:"version"`
			}
			if err := json.Unmarshal(payload, &eventVersion); err != nil {
				return nil, err
			}

			if eventVersion.Version == "2.0" {
				var req events.APIGatewayV2HTTPRequest
				if err := json.Unmarshal(payload, &req); err != nil {
					return nil, err
				}
				return adapterV2.ProxyWithContext(ctx, req)
			}

			var req events.APIGatewayProxyRequest
			if err := json.Unmarshal(payload, &req); err != nil {
				return nil, err
			}
			return adapterV1.ProxyWithContext(ctx, req)
		})
		return
	}

	if err := routerAtrevida.Run(":8080"); err != nil {
		panic(err)
	}
}

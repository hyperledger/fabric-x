/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package routes

import (
	"context"

	"github.com/hyperledger/fabric-samples/token-sdk/endorser/service"
)

//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=./oapi-server.yaml ../../swagger.yaml

type Server struct {
	fsc *service.FabricSmartClient
}

func NewServer(fsc *service.FabricSmartClient) Server {
	return Server{fsc: fsc}
}

// Returns 200 if the service is healthy
// (POST /endorser/init)
func (s Server) Init(ctx context.Context, request InitRequestObject) (InitResponseObject, error) {
	err := s.fsc.Init(ctx)
	if err != nil {
		return nil, err
	}
	return Init200JSONResponse{HealthSuccessJSONResponse{
		Message: "ok",
	}}, err
}

// Returns 200 if the service is healthy
// (GET /healthz)
func (s Server) Healthz(ctx context.Context, request HealthzRequestObject) (HealthzResponseObject, error) {
	return Healthz200JSONResponse{HealthSuccessJSONResponse{Message: "ok"}}, nil
}

// Returns 200 if the service is ready to accept calls
// (GET /readyz)
func (s Server) Readyz(ctx context.Context, request ReadyzRequestObject) (ReadyzResponseObject, error) {
	return Readyz200JSONResponse{HealthSuccessJSONResponse{Message: "ok"}}, nil
}

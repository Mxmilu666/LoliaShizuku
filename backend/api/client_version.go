package api

import (
	"context"
	"net/http"

	"github.com/Mxmilu666/LoliaShizuku/backend/httpclient"
	"github.com/Mxmilu666/LoliaShizuku/backend/models"
)

type ClientVersionAPI struct {
	client *httpclient.Client
}

func NewClientVersionAPI(client *httpclient.Client) *ClientVersionAPI {
	return &ClientVersionAPI{client: client}
}

func (a *ClientVersionAPI) GetLatestClientVersion(ctx context.Context) (*models.ClientVersionInfo, error) {
	var data models.ClientVersionInfo
	if err := a.client.DoJSON(ctx, http.MethodGet, "/client/version", nil, nil, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

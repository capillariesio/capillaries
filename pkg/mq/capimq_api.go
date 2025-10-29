package mq

import "github.com/capillariesio/capillaries/pkg/wfmodel"

type CapimqApiGenericResponse struct {
	Data  any    `json:"data"`
	Error string `json:"error"`
}

type CapimqApiClaimResponse struct {
	Data  *wfmodel.Message `json:"data"`
	Error string           `json:"error"`
}

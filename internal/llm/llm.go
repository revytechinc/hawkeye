// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/revytechinc/hawkeye/internal/headroom"
	"github.com/revytechinc/hawkeye/internal/redact"
)

var (
	ErrNoModel     = errors.New("local llm model is not configured")
	ErrGPURequired = errors.New("gpu required for this job but none is present")
)

type Request struct {
	Prompt  string
	NeedGPU bool
	NeedRAM bool
}

type Response struct {
	Text    string `json:"text"`
	Backend string `json:"backend"`
	UsedGPU bool   `json:"used_gpu"`
}

type Completer interface {
	Complete(ctx context.Context, req Request) (Response, error)
}

type Local struct {
	Backend    string
	ModelPath  string
	PreferGPU  bool
	RequireGPU bool
	GPUPresent bool
	Headroom   headroom.Snapshot
	RAMMin     *int64
	VRAMMin    *int64
}

func (l Local) Complete(ctx context.Context, req Request) (Response, error) {
	_ = ctx
	req.Prompt = redact.String(req.Prompt)
	needGPU := req.NeedGPU || l.RequireGPU
	if needGPU && !l.GPUPresent {
		return Response{}, ErrGPURequired
	}
	job := headroom.Job{NeedRAM: true, NeedGPU: needGPU}
	if err := headroom.Allow(job, l.Headroom, l.RAMMin, nil, nil, l.VRAMMin); err != nil {
		return Response{}, err
	}
	useGPU := l.PreferGPU && l.GPUPresent
	if l.ModelPath == "" {
		return Response{Backend: l.Backend, UsedGPU: useGPU}, ErrNoModel
	}
	return Response{
		Text:    fmt.Sprintf("local %s skeleton; GPU=%v", l.Backend, useGPU),
		Backend: l.Backend,
		UsedGPU: useGPU,
	}, nil
}

type None struct{}

func (None) Complete(context.Context, Request) (Response, error) {
	return Response{}, ErrNoModel
}

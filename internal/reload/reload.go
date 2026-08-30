// Copyright (c) 2026 REVYTECH, Inc.
// SPDX-License-Identifier: BSD-3-Clause

package reload

import (
	"sync/atomic"

	"github.com/revytechinc/hawkeye/internal/config"
)

// Holder stores the live config. SIGHUP validates then swaps; bad config keeps the old value.
type Holder struct {
	v atomic.Value
}

func New(c config.Config) *Holder {
	h := &Holder{}
	h.v.Store(c)
	return h
}

func (h *Holder) Get() config.Config {
	if h == nil {
		return config.Default()
	}
	c, _ := h.v.Load().(config.Config)
	return c
}

func (h *Holder) ReloadFile(path string) error {
	c, err := config.Load(path)
	if err != nil {
		return err
	}
	if err := config.Validate(c); err != nil {
		return err
	}
	h.v.Store(c)
	return nil
}

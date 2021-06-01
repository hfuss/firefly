// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package networkmap

import (
	"context"

	"github.com/kaleido-io/firefly/internal/broadcast"
	"github.com/kaleido-io/firefly/pkg/database"
	"github.com/kaleido-io/firefly/pkg/dataexchange"
	"github.com/kaleido-io/firefly/pkg/identity"
)

type Manager interface {
	JoinNetwork(ctx context.Context) error
}

type networkMap struct {
	ctx       context.Context
	database  database.Plugin
	broadcast broadcast.Manager
	exchange  dataexchange.Plugin
	identity  identity.Plugin
	// orgIdentity  fftypes.Identity
	// nodeIdentity fftypes.Identity
}

func NewNetworkMap(ctx context.Context, di database.Plugin, bm broadcast.Manager, dx dataexchange.Plugin, ii identity.Plugin) (Manager, error) {
	nm := &networkMap{
		ctx:       ctx,
		database:  di,
		broadcast: bm,
		exchange:  dx,
		identity:  ii,
	}

	return nm, nil
}

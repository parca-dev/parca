// Copyright 2021 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metastore

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
)

func TestMappingKeyBytes(t *testing.T) {
	m := &pb.Mapping{
		Start:  0,
		Limit:  1,
		Offset: 2,
	}

	require.Equal(t, []byte{
		0x76,
		0x31,
		0x2f,
		0x6d,
		0x61,
		0x70,
		0x70,
		0x69,
		0x6e,
		0x67,
		0x73,
		0x2f,
		0x62,
		0x79,
		0x2d,
		0x6b,
		0x65,
		0x79,
		0x2f,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x10,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x2,
	}, MakeMappingKey(m))

	m = &pb.Mapping{
		Start:  0,
		Limit:  1,
		Offset: 2,
		File:   "a",
	}

	require.Equal(t, []byte{
		0x76,
		0x31,
		0x2f,
		0x6d,
		0x61,
		0x70,
		0x70,
		0x69,
		0x6e,
		0x67,
		0x73,
		0x2f,
		0x62,
		0x79,
		0x2d,
		0x6b,
		0x65,
		0x79,
		0x2f,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x10,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x2,
		0x61,
	}, MakeMappingKey(m))
}

func TestFunctionKeyBytes(t *testing.T) {
	f := &pb.Function{
		StartLine:  3,
		Name:       "a",
		SystemName: "b",
		Filename:   "c",
	}

	require.Equal(t, MakeFunctionKey(f), []byte{
		0x76,
		0x31,
		0x2f,
		0x66,
		0x75,
		0x6e,
		0x63,
		0x74,
		0x69,
		0x6f,
		0x6e,
		0x73,
		0x2f,
		0x62,
		0x79,
		0x2d,
		0x6b,
		0x65,
		0x79,
		0x2f,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x3,
		0x61,
		0x62,
		0x63,
	})
}

func TestLocationKeyBytes(t *testing.T) {
	l := &Location{
		Address: 3,
		Mapping: &pb.Mapping{
			Id: []byte{
				0x02,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x00,
				0x03,
			},
		},
	}

	require.Equal(t, MakeLocationKey(l), []byte{
		0x76,
		0x31,
		0x2f,
		0x6c,
		0x6f,
		0x63,
		0x61,
		0x74,
		0x69,
		0x6f,
		0x6e,
		0x73,
		0x2f,
		0x62,
		0x79,
		0x2d,
		0x6b,
		0x65,
		0x79,
		0x2f,
		0x2,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x3,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x3,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
		0x0,
	})
}

func mappingStoreTest(t *testing.T, s MappingStore) {
	ctx := context.Background()
	var err error

	m := &pb.Mapping{
		Start:           1,
		Limit:           10,
		Offset:          5,
		File:            "file",
		BuildId:         "buildID0",
		HasFunctions:    false,
		HasFilenames:    false,
		HasLineNumbers:  false,
		HasInlineFrames: false,
	}
	m.Id, err = s.CreateMapping(ctx, m)
	require.NoError(t, err)

	m1 := &pb.Mapping{
		Start:           12,
		Limit:           110,
		Offset:          51,
		File:            "file1",
		BuildId:         "buildID1",
		HasFunctions:    true,
		HasFilenames:    true,
		HasLineNumbers:  false,
		HasInlineFrames: true,
	}
	m1.Id, err = s.CreateMapping(ctx, m1)
	require.NoError(t, err)

	mapByKey, err := s.GetMappingByKey(ctx, m)
	require.NoError(t, err)
	require.Equal(t, m.Id, mapByKey.Id)
	require.Equal(t, m.Start, mapByKey.Start)
	require.Equal(t, m.Limit, mapByKey.Limit)
	require.Equal(t, m.Offset, mapByKey.Offset)
	require.Equal(t, m.File, mapByKey.File)
	require.Equal(t, m.BuildId, mapByKey.BuildId)
	require.Equal(t, m.HasFunctions, mapByKey.HasFunctions)
	require.Equal(t, m.HasFilenames, mapByKey.HasFilenames)
	require.Equal(t, m.HasLineNumbers, mapByKey.HasLineNumbers)
	require.Equal(t, m.HasInlineFrames, mapByKey.HasInlineFrames)
}

func functionStoreTest(t *testing.T, s FunctionStore) {
	ctx := context.Background()
	var err error

	f := &pb.Function{
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  22,
	}
	f.Id, err = s.CreateFunction(ctx, f)
	require.NoError(t, err)

	f1 := &pb.Function{
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  42,
	}
	f1.Id, err = s.CreateFunction(ctx, f1)
	require.NoError(t, err)

	funcByID, err := s.GetFunctionByKey(ctx, f)
	require.NoError(t, err)
	require.Equal(t, f.Id, funcByID.Id)
	require.Equal(t, f.Name, funcByID.Name)
	require.Equal(t, f.SystemName, funcByID.SystemName)
	require.Equal(t, f.Filename, funcByID.Filename)
	require.Equal(t, f.StartLine, funcByID.StartLine)

	funcs, err := s.GetFunctions(context.Background())
	require.NoError(t, err)

	// Order is not guaranteed, so make sure it's one of the two possibilities.

	if funcs[0].StartLine == 22 {
		require.True(t, proto.Equal(funcs[0], f))
		require.True(t, proto.Equal(funcs[1], f1))
	}

	if funcs[0].StartLine == 42 {
		require.True(t, proto.Equal(funcs[0], f1))
		require.True(t, proto.Equal(funcs[1], f))
	}
}

func LocationLinesStoreTest(t *testing.T, s LocationLineStore) {
	ctx := context.Background()

	locID := uuid.New()
	f1ID := uuid.New()
	ll := []LocationLine{{
		Line: 2,
		Function: &pb.Function{
			Id:   f1ID[:],
			Name: "f1",
		},
	}}
	err := s.CreateLocationLines(ctx, locID[:], ll)
	require.NoError(t, err)

	llRetrieved, functionIDs, err := s.GetLinesByLocationIDs(ctx, locID[:])
	require.NoError(t, err)
	require.Equal(t, [][]byte{f1ID[:]}, functionIDs)
	require.Equal(t, map[string][]*pb.Line{
		string(locID[:]): {
			{
				Line:       2,
				FunctionId: f1ID[:],
			},
		},
	}, llRetrieved)
}

func LocationStoreTest(t *testing.T, s ProfileMetaStore) {
	ctx := context.Background()

	l := &Location{
		Address: uint64(42),
	}
	lID, err := s.CreateLocation(ctx, l)
	require.NoError(t, err)

	lUUID, err := uuid.FromBytes(lID)
	require.NoError(t, err)

	l.ID = lUUID

	l1 := &Location{
		Address: uint64(421),
	}
	l1ID, err := s.CreateLocation(ctx, l1)
	require.NoError(t, err)

	l1UUID, err := uuid.FromBytes(l1ID)
	require.NoError(t, err)

	l1.ID = l1UUID

	locs, err := GetLocations(context.Background(), s)
	require.NoError(t, err)

	if locs[0].Address == 42 {
		require.Equal(t, locs[0].Address, l.Address)
		require.Equal(t, locs[1].Address, l1.Address)
	} else {
		require.Equal(t, locs[1].Address, l.Address)
		require.Equal(t, locs[0].Address, l1.Address)
	}

	l1, err = GetLocationByKey(ctx, s, l1)
	require.NoError(t, err)

	locByID, err := GetLocationsByIDs(ctx, s, l1.ID[:])
	require.NoError(t, err)

	require.Equal(t, l1, locByID[string(l1.ID[:])])

	f := &pb.Function{
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  22,
	}
	l1.Lines = []LocationLine{
		{Line: 1, Function: f},
		{Line: 5, Function: f},
	}

	err = s.Symbolize(ctx, l1)
	require.NoError(t, err)

	locByID, err = GetLocationsByIDs(ctx, s, l1.ID[:])
	require.NoError(t, err)
	res := locByID[string(l1.ID[:])]
	requireEqualLocation(t, l1, res)
}

func requireEqualLocation(t *testing.T, expected, compared *Location) {
	require.Equal(t, expected.ID, compared.ID)
	require.Equal(t, expected.Address, compared.Address)
	require.Equal(t, expected.IsFolded, compared.IsFolded)
	if expected.Mapping != nil {
		require.NotNil(t, compared.Mapping)
		require.True(t, proto.Equal(expected.Mapping, compared.Mapping))
	}
	require.Equal(t, len(expected.Lines), len(compared.Lines))
	for i := range expected.Lines {
		require.Equal(t, expected.Lines[i].Line, compared.Lines[i].Line)
		require.True(t, proto.Equal(expected.Lines[i].Function, compared.Lines[i].Function))
	}
}

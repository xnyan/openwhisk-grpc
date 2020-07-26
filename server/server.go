package main

import (
	"context"

	"github.com/DCsunset/openwhisk-grpc/db"
	"github.com/DCsunset/openwhisk-grpc/storage"
)

type Server struct{}

var store = storage.Store{}

func (s *Server) Get(ctx context.Context, in *db.GetRequest) (*db.GetResponse, error) {
	data := store.Get(in.SessionId, in.Keys, in.Loc, in.VirtualLoc)
	return &db.GetResponse{Data: data}, nil
}

func (s *Server) Set(ctx context.Context, in *db.SetRequest) (*db.SetResponse, error) {
	loc := store.Set(in.SessionId, in.Data, in.VirtualLoc, in.Dep, in.VirtualDep)
	return &db.SetResponse{Loc: loc}, nil
}

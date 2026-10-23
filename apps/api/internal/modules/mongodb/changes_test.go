package mongodb

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"noxis/internal/constants"
	"noxis/internal/entity"
	"noxis/internal/mocks"
)

func TestApplyReturnsOutcomeSaveFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	connections := mocks.NewMockConnectionRepository(ctrl)
	changes := mocks.NewMockChangeRepository(ctrl)
	engine := mocks.NewMockMongoModule(ctrl)
	connection := entity.Connection{Name: "demo"}
	request := entity.MongoRequest{Op: constants.MongoInsert, Database: "demo", Collection: "docs", Value: `{"name":"nox"}`}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	change := entity.Change{
		ID: "mongo-change", Connection: connection.Name, Kind: constants.KindMongo,
		Status: constants.ChangeApproved, Query: string(encoded),
	}
	engineErr := errors.New("engine unavailable")
	storeErr := errors.New("sqlite unavailable")

	connections.EXPECT().Get(gomock.Any(), connection.Name).Return(&connection, nil)
	changes.EXPECT().ClaimChange(gomock.Any(), change.ID, gomock.Any()).Return(true, nil)
	engine.EXPECT().Apply(gomock.Any(), connection, request).Return(entity.Result{}, engineErr)
	changes.EXPECT().SaveChange(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, saved entity.Change) error {
		if saved.Status != constants.ChangeFailed {
			t.Fatalf("saved status=%q want failed", saved.Status)
		}
		return storeErr
	})

	flow := NewChanges(connections, changes, engine, nil)
	_, err = flow.Apply(context.Background(), &change)
	if err == nil || !strings.Contains(err.Error(), engineErr.Error()) || !strings.Contains(err.Error(), storeErr.Error()) {
		t.Fatalf("Apply error=%v, want engine and save errors", err)
	}
}

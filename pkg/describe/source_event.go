package describe

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
)

type SourceAction string

const (
	SourceCreate SourceAction = "CREATE"
	SourceUpdate SourceAction = "UPDATE"
	SourceDelete SourceAction = "DELETE"
)

type SourceEvent struct {
	Action     SourceAction
	SourceID   uuid.UUID
	AccountID  string
	SourceType api.SourceType
	ConfigRef  string
}

func ProcessSourceAction(db Database, event SourceEvent) error {
	fmt.Printf("Processing SourceEvent[%s] for Source[%s] with type %s\n", event.Action, event.SourceID.String(), event.SourceType)
	switch event.Action {
	case SourceCreate:
		err := CreateSource(db, event)
		if err != nil {
			return err
		}
	case SourceUpdate:
		err := UpdateSource(db, event)
		if err != nil {
			return err
		}
	case SourceDelete:
		err := DeleteSource(db, event)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("source action (%s) is invalid", event.Action)
	}

	return nil
}

func CreateSource(db Database, event SourceEvent) error {
	switch {
	case len(event.SourceID) == 0 || event.SourceID.Variant() == uuid.Invalid:
		return fmt.Errorf("source has invalid uuid format")
	case len(event.AccountID) == 0:
		return fmt.Errorf("account id must be provided")
	case !api.IsValidSourceType(event.SourceType):
		return fmt.Errorf("source has invalid source type")
	case event.ConfigRef == "": // TODO: should check if the config ref exists?
		return fmt.Errorf("source has invalid config ref")
	}

	err := db.CreateSource(&Source{
		ID:             event.SourceID,
		AccountID:      event.AccountID,
		Type:           event.SourceType,
		ConfigRef:      event.ConfigRef,
		NextDescribeAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	if err != nil {
		return err
	}

	return nil
}

func UpdateSource(db Database, event SourceEvent) error {
	switch {
	case len(event.SourceID) == 0 || event.SourceID.Variant() == uuid.Invalid:
		return fmt.Errorf("source has invalid uuid format")
	case len(event.AccountID) == 0:
		return fmt.Errorf("account id must be provided")
	case event.SourceType != "" && !api.IsValidSourceType(event.SourceType):
		return fmt.Errorf("source has invalid source type")
	case event.ConfigRef == "": // TODO: should check if the config ref exists?
		return fmt.Errorf("source has invalid credentials")
	}

	err := db.UpdateSource(&Source{
		ID:        event.SourceID,
		AccountID: event.AccountID,
		Type:      event.SourceType,
		ConfigRef: event.ConfigRef,
	})
	if err != nil {
		return fmt.Errorf("update source: %w", err)
	}

	return nil
}

func DeleteSource(db Database, event SourceEvent) error {
	switch {
	case len(event.SourceID) == 0 || event.SourceID.Variant() == uuid.Invalid:
		return fmt.Errorf("source has invalid uuid format")
	}

	err := db.DeleteSource(Source{
		ID: event.SourceID,
	})
	if err != nil {
		return fmt.Errorf("delete source: %w", err)
	}

	return nil
}

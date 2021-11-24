package describe

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SourceAction string

const (
	SourceCreate SourceAction = "CREATE"
	SourceUpdate SourceAction = "UPDATE"
	SourceDelete SourceAction = "DELETE"
)

type SourceEvent struct {
	Action            SourceAction
	SourceID          uuid.UUID
	SourceType        SourceType
	SourceCredentials []byte
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
	case !IsValidSourceType(event.SourceType):
		return fmt.Errorf("source has invalid source type")
	case !IsCredentialsValid(event.SourceCredentials, event.SourceType):
		return fmt.Errorf("source has invalid credentials")
	}

	err := db.CreateSource(Source{
		ID:             event.SourceID,
		Type:           event.SourceType,
		Credentials:    event.SourceCredentials,
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
	case event.SourceType != "" && !IsValidSourceType(event.SourceType):
		return fmt.Errorf("source has invalid source type")
	case len(event.SourceCredentials) > 0 && !IsCredentialsValid(event.SourceCredentials, event.SourceType):
		return fmt.Errorf("source has invalid credentials")
	}

	err := db.UpdateSource(Source{
		ID:          event.SourceID,
		Type:        event.SourceType,
		Credentials: event.SourceCredentials,
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

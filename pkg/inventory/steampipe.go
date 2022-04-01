package inventory

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
	"regexp"
)

type SteampipeOption struct {
	Host string
	Port string
	User string
	Pass string
	Db   string
}

type SteampipeDatabase struct {
	conn *pgx.Conn
}

type SteampipeResult struct {
	headers []string
	data    [][]interface{}
}

func NewSteampipeDatabase(option SteampipeOption) (*SteampipeDatabase, error) {
	var err error
	dsn := fmt.Sprintf(`host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=GMT`,
		option.Host,
		option.Port,
		option.User,
		option.Pass,
		option.Db,
	)
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return nil, err
	}

	return &SteampipeDatabase{conn: conn}, nil
}

func (s *SteampipeDatabase) Query(query string, from, size int, orderBy string,
	orderDir api.DirectionType) (*SteampipeResult, error) {

	// parameterize order by is not supported by steampipe.
	// in order to prevent SQL Injection, we ensure that orderby field is only consists of
	// characters and underline.
	if ok, err := regexp.Match("(\\w|_)+", []byte(orderBy)); err != nil || !ok {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("invalid orderby field:" + orderBy)
	}

	orderStr := ""
	if orderBy != "" {
		if orderDir == api.DirectionAscending {
			orderStr = " order by " + orderBy + " asc"
		} else if orderDir == api.DirectionDescending {
			orderStr = " order by " + orderBy + " desc"
		} else {
			return nil, errors.New("invalid order direction:" + string(orderDir))
		}
	}

	query = query + orderStr + " LIMIT $1 OFFSET $2;"

	r, err := s.conn.Query(context.Background(),
		query, size, from)
	defer r.Close()
	if err != nil {
		return nil, err
	}

	var headers []string
	for _, field := range r.FieldDescriptions() {
		headers = append(headers, string(field.Name))
	}
	var result [][]interface{}
	for r.Next() {
		v, err := r.Values()
		if err != nil {
			return nil, err
		}

		result = append(result, v)
	}

	return &SteampipeResult{
		headers: headers,
		data:    result,
	}, nil
}

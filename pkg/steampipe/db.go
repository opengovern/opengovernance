package steampipe

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v4/pgxpool"
)

type DirectionType string

const (
	DirectionAscending  DirectionType = "asc"
	DirectionDescending DirectionType = "desc"
)

type Option struct {
	Host string
	Port string
	User string
	Pass string
	Db   string
}

type Database struct {
	conn *pgxpool.Pool
}

type Result struct {
	Headers []string
	Data    [][]interface{}
}

func NewSteampipeDatabase(option Option) (*Database, error) {
	var err error
	connString := fmt.Sprintf(`host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=GMT`,
		option.Host,
		option.Port,
		option.User,
		option.Pass,
		option.Db,
	)

	conn, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(context.Background()); err != nil {
		return nil, err
	}

	return &Database{conn: conn}, nil
}

func (s *Database) Query(query string, from, size int, orderBy string,
	orderDir DirectionType) (*Result, error) {

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
		if orderDir == DirectionAscending {
			orderStr = " order by " + orderBy + " asc"
		} else if orderDir == DirectionDescending {
			orderStr = " order by " + orderBy + " desc"
		} else {
			return nil, errors.New("invalid order direction:" + string(orderDir))
		}
	}

	query = query + orderStr + " LIMIT $1 OFFSET $2;"

	fmt.Println("query is: ", query)
	fmt.Println("size: ", size, "from:", from)

	r, err := s.conn.Query(context.Background(),
		query, size, from)
	if err != nil {
		return nil, err
	}
	defer r.Close()

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

	return &Result{
		Headers: headers,
		Data:    result,
	}, nil
}

func (s *Database) Count(query string) (*Result, error) {
	r, err := s.conn.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer r.Close()

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

	return &Result{
		Headers: headers,
		Data:    result,
	}, nil
}

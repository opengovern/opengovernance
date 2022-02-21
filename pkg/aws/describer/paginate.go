package describer

func PaginateRetrieveAll(fn func(prevToken *string) (nextToken *string, err error)) error {
	first, token := true, (*string)(nil)
	for first || (token != nil && *token != "") {
		var err error
		if token, err = fn(token); err != nil {
			return err
		}

		first = false
	}

	return nil
}

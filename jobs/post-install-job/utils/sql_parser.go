package utils

import pg_query "github.com/pganalyze/pg_query_go/v4"

// extractTableRefs is a recursive AST-walker for pg_query_go. It detects
// tables in top-level SELECT, subqueries, JOINs, UNION, etc.
func extractTableRefs(node *pg_query.Node) []string {
	var tables []string
	if node == nil {
		return tables
	}

	switch n := node.Node.(type) {

	//---------------------------------------------------------------------
	// SELECT
	//---------------------------------------------------------------------
	case *pg_query.Node_SelectStmt:
		sel := n.SelectStmt

		// 1. FROM Clause
		if sel.FromClause != nil {
			for _, fromItem := range sel.FromClause {
				tables = append(tables, extractTableRefs(fromItem)...)
			}
		}

		// 2. WHERE Clause
		if sel.WhereClause != nil {
			tables = append(tables, extractTableRefs(sel.WhereClause)...)
		}

		// 3. GROUP BY / HAVING
		for _, gc := range sel.GroupClause {
			tables = append(tables, extractTableRefs(gc)...)
		}
		if sel.HavingClause != nil {
			tables = append(tables, extractTableRefs(sel.HavingClause)...)
		}

		// 4. TARGET LIST
		for _, target := range sel.TargetList {
			tables = append(tables, extractTableRefs(target)...)
		}

		// 5. UNION / INTERSECT / EXCEPT (Set Operations).
		//    If there's a set operation, the left subquery is in sel.Larg
		//    and the right subquery is in sel.Rarg. They are *SelectStmt,
		//    so we must wrap them in a Node before recursing.
		if sel.Larg != nil {
			leftNode := &pg_query.Node{
				Node: &pg_query.Node_SelectStmt{
					SelectStmt: sel.Larg,
				},
			}
			tables = append(tables, extractTableRefs(leftNode)...)
		}
		if sel.Rarg != nil {
			rightNode := &pg_query.Node{
				Node: &pg_query.Node_SelectStmt{
					SelectStmt: sel.Rarg,
				},
			}
			tables = append(tables, extractTableRefs(rightNode)...)
		}

	//---------------------------------------------------------------------
	// JOIN
	//---------------------------------------------------------------------
	case *pg_query.Node_JoinExpr:
		j := n.JoinExpr
		if j.Larg != nil {
			tables = append(tables, extractTableRefs(j.Larg)...)
		}
		if j.Rarg != nil {
			tables = append(tables, extractTableRefs(j.Rarg)...)
		}
		// ON/Quals
		if j.Quals != nil {
			tables = append(tables, extractTableRefs(j.Quals)...)
		}

	//---------------------------------------------------------------------
	// RANGE SUBSELECT
	//---------------------------------------------------------------------
	case *pg_query.Node_RangeSubselect:
		rs := n.RangeSubselect
		if rs.Subquery != nil {
			tables = append(tables, extractTableRefs(rs.Subquery)...)
		}

	//---------------------------------------------------------------------
	// SUBLINK (subquery in WHERE or SELECT list, e.g. "EXISTS(SELECT ...)")
	//---------------------------------------------------------------------
	case *pg_query.Node_SubLink:
		sl := n.SubLink
		if sl.Subselect != nil {
			tables = append(tables, extractTableRefs(sl.Subselect)...)
		}

	//---------------------------------------------------------------------
	// RANGEVAR (simple table reference)
	//---------------------------------------------------------------------
	case *pg_query.Node_RangeVar:
		rv := n.RangeVar
		if rv.Relname != "" {
			tables = append(tables, rv.Relname)
		}

	//---------------------------------------------------------------------
	// INSERT
	//---------------------------------------------------------------------
	case *pg_query.Node_InsertStmt:
		ins := n.InsertStmt
		if ins.Relation != nil && ins.Relation.Relname != "" {
			tables = append(tables, ins.Relation.Relname)
		}
		if ins.SelectStmt != nil {
			tables = append(tables, extractTableRefs(ins.SelectStmt)...)
		}

	//---------------------------------------------------------------------
	// UPDATE
	//---------------------------------------------------------------------
	case *pg_query.Node_UpdateStmt:
		upd := n.UpdateStmt
		if upd.Relation != nil && upd.Relation.Relname != "" {
			tables = append(tables, upd.Relation.Relname)
		}
		for _, fromItem := range upd.FromClause {
			tables = append(tables, extractTableRefs(fromItem)...)
		}
		if upd.WhereClause != nil {
			tables = append(tables, extractTableRefs(upd.WhereClause)...)
		}

	//---------------------------------------------------------------------
	// DELETE
	//---------------------------------------------------------------------
	case *pg_query.Node_DeleteStmt:
		del := n.DeleteStmt
		if del.Relation != nil && del.Relation.Relname != "" {
			tables = append(tables, del.Relation.Relname)
		}
		for _, usingItem := range del.UsingClause {
			tables = append(tables, extractTableRefs(usingItem)...)
		}
		if del.WhereClause != nil {
			tables = append(tables, extractTableRefs(del.WhereClause)...)
		}

	//---------------------------------------------------------------------
	// OTHER EXPRESSION NODES
	//---------------------------------------------------------------------
	default:
		// If you have more advanced node types that can contain sub-SELECTs
		// (e.g. function calls), you can either handle them directly or
		// reflect them. Most typical queries are covered above.
	}

	return tables
}

func ExtractTableRefsFromQuery(queryToExecute string) ([]string, error) {
	// Parse the SQL using pg_query_go
	parseResult, err := pg_query.Parse(queryToExecute)
	if err != nil {
		return nil, err
	}

	// Collect table references from each statement
	var tables []string
	for _, rawStmt := range parseResult.Stmts {
		stmtTables := extractTableRefs(rawStmt.Stmt)
		tables = append(tables, stmtTables...)
	}

	return tables, nil
}

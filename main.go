package main

import (
	"context"
	"database/sql"
	"fmt"
	"oso-go-df-sqlboiler/models"

	_ "github.com/mattn/go-sqlite3"

	"github.com/osohq/go-oso"
	"github.com/osohq/go-oso/types"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

const (
	policy = `
allow("steve", "get", org: Organization) if org.Name = "osohq";
allow("steve", "get", repo: Repository) if repo.Org.Name = "osohq";
`
)

type MyAdapter struct {
	ctx         context.Context
	db          *sql.DB
	table_names map[string]string
	field_names map[string]map[string]string
}

type MyQuery struct {
	Type string
	Mods []qm.QueryMod
}

func (a MyAdapter) toSql(datum types.Datum) (string, interface{}) {
	var sql string
	var value interface{}
	switch t := datum.DatumVarient.(type) {
	case types.Projection:
		table := a.table_names[t.TypeName]
		fields := a.field_names[t.TypeName]
		sql, value = fmt.Sprintf("%s.%s", table, fields[t.FieldName]), nil
	case types.Immediate:
		sql, value = "?", t.Value
	}
	return sql, value
}

// Ideally integrate with the codegeneration of sqlboiler to know which
// table to use for every model. Lacking that just hardcoding this example.
func (a MyAdapter) BuildQuery(filter *types.Filter) (interface{}, error) {
	mods := make([]qm.QueryMod, 0)
	table := a.table_names[filter.Root]
	fields := a.field_names[filter.Root]

	for _, filter_relation := range filter.Relations {
		relation := filter.Types[filter.Root][filter_relation.FromFieldName].(types.Relation)
		other_table := a.table_names[relation.OtherType]
		other_fields := a.field_names[relation.OtherType]
		join_sql := fmt.Sprintf("%s on %s.%s = %s.%s", other_table, table, fields[relation.MyField], other_table, other_fields[relation.OtherField])
		join := qm.InnerJoin(join_sql)
		mods = append(mods, join)
	}

	// todo handle the ORs, not sure how yet
	for i, conditions := range filter.Conditions {
		group := make([]qm.QueryMod, 0)
		for _, condition := range conditions {
			args := make([]interface{}, 0)
			lhs, arg := a.toSql(condition.Lhs)
			if arg != nil {
				args = append(args, arg)
			}
			rhs, arg := a.toSql(condition.Rhs)
			if arg != nil {
				args = append(args, arg)
			}
			var op string
			switch condition.Cmp {
			case types.Eq:
				op = "="
			}

			where_sql := fmt.Sprintf("%s %s %s", lhs, op, rhs)
			where := qm.Where(where_sql, args...)
			group = append(group, where)
		}
		expr := qm.Expr(group...)
		if i == 0 {
			mods = append(mods, expr)
		} else {
			mods = append(mods, qm.Or2(expr))
		}
	}
	return MyQuery{filter.Root, mods}, nil
}

func (a MyAdapter) ExecQuery(query interface{}) (interface{}, error) {
	mq := query.(MyQuery)
	var results interface{}
	var err error
	switch mq.Type {
	case "Organization":
		results, err = models.Organizations(mq.Mods...).All(a.ctx, a.db)
	case "Repository":
		results, err = models.Repositories(mq.Mods...).All(a.ctx, a.db)
	}
	return results, err
}

func main() {
	oso, err := oso.NewOso()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", oso)

	oso.RegisterClassWithNameAndFields(models.Organization{}, nil, "Organization", map[string]interface{}{
		"Name": "String",
	})

	oso.RegisterClassWithNameAndFields(models.Repository{}, nil, "Repository", map[string]interface{}{
		"Name": "String",
		"Org": types.Relation{
			Kind:       "one",
			OtherType:  "Organization",
			MyField:    "OrgName",
			OtherField: "Name",
		},
	})

	err = oso.LoadString(policy)
	if err != nil {
		panic(err)
	}

	db, err := sql.Open("sqlite3", "./example.db")
	if err != nil {
		panic(err)
	}

	boil.SetDB(db)
	ctx := context.Background()

	// This is probably all stuff that can be gotten from sqlboiler
	// but I dont know how so for now just hardcoding it.
	adapter := MyAdapter{
		ctx: ctx,
		db:  db,
		table_names: map[string]string{
			"Repository":   "repositories",
			"Organization": "organizations",
		},
		field_names: map[string]map[string]string{
			"Repository": {
				"Name":    "name",
				"OrgName": "org_name",
			},
			"Organization": {
				"Name": "name",
			},
		},
	}
	oso.SetDataFilteringAdapter(&adapter)

	// query for organizations to make sure it works
	// orgs, _ := models.Organizations().All(ctx, db)
	// for _, org := range orgs {
	// 	fmt.Println(org)
	// }

	// query for repositories to make sure it works
	// repos, _ := models.Repositories().All(ctx, db)
	// for _, repo := range repos {
	// 	fmt.Println(repo)
	// }

	//osohq, _ := models.Organizations(qm.Where("name = ?", "osohq")).One(ctx, db)

	//fmt.Println(oso.IsAllowed("steve", "get", osohq))
	results, err := oso.AuthorizedResources("steve", "get", "Repository")
	for _, repo := range results.(models.RepositorySlice) {
		fmt.Println(repo)
	}
}

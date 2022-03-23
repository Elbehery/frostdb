package logicalplan

import (
	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/memory"
)

// LogicalPlan is a logical representation of a query. Each LogicalPlan is a
// sub-tree of the query. It is built recursively.
type LogicalPlan struct {
	Input *LogicalPlan

	// Each LogicalPlan struct must only have one of the following.
	SchemaScan  *SchemaScan
	TableScan   *TableScan
	Filter      *Filter
	Distinct    *Distinct
	Projection  *Projection
	Aggregation *Aggregation
}

func (plan *LogicalPlan) String() string {
	switch {
	case plan.SchemaScan != nil:
		return plan.SchemaScan.String()
	case plan.TableScan != nil:
		return plan.TableScan.String()
	case plan.Filter != nil:
		return plan.Filter.String()
	case plan.Projection != nil:
		return plan.Projection.String()
	case plan.Aggregation != nil:
		return plan.Aggregation.String()
	default:
		return "Unknown LogicalPlan"
	}
}

type PlanVisitor interface {
	PreVisit(plan *LogicalPlan) bool
	PostVisit(plan *LogicalPlan) bool
}

func (plan *LogicalPlan) Accept(visitor PlanVisitor) bool {
	continu := visitor.PreVisit(plan)
	if !continu {
		return false
	}

	if plan.Input != nil {
		continu = plan.Input.Accept(visitor)
		if !continu {
			return false
		}
	}

	return visitor.PostVisit(plan)
}

type TableReader interface {
	Iterator(
		pool memory.Allocator,
		projection []ColumnMatcher,
		filter Expr,
		callback func(r arrow.Record) error,
	) error
}

type TableProvider interface {
	GetTable(name string) TableReader
}

type TableScan struct {
	TableProvider TableProvider
	TableName     string

	// projection in this case means the columns that are to be read by the
	// table scan.
	Projection []ColumnMatcher

	// filter is the predicate that is to be applied by the table scan to rule
	// out any blocks of data to be scanned at all.
	Filter Expr
}

func (scan *TableScan) String() string {
	return "TableScan"
}

type SchemaScan struct {
	TableProvider TableProvider
	TableName     string

	// projection in this case means the columns that are to be read by the
	// table scan.
	Projection []ColumnMatcher

	// filter is the predicate that is to be applied by the table scan to rule
	// out any blocks of data to be scanned at all.
	Filter Expr
}

func (scan *SchemaScan) String() string {
	return "SchemaScan"
}

type Filter struct {
	Expr Expr
}

func (filter *Filter) String() string {
	return "Filter"
}

type Distinct struct {
	Columns []ColumnExpr
}

func (distinct *Distinct) String() string {
	return "Distinct"
}

type Projection struct {
	Exprs []Expr
}

func (projection *Projection) String() string {
	return "Projection"
}

type Aggregation struct {
	GroupExprs []ColumnExpr
	AggExpr    Expr
}

func (aggregation *Aggregation) String() string {
	return "Aggregation"
}
